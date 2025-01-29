/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	xov1alpha1 "github.com/90poe/kafkaobjects-operator/api/v1alpha1"
	"github.com/90poe/kafkaobjects-operator/internal/env"
	"github.com/90poe/kafkaobjects-operator/internal/kafka"
	"github.com/90poe/kafkaobjects-operator/internal/reporter"
	"github.com/go-logr/logr"
)

// KafkaTopicReconciler reconciles a KafkaTopic object
type KafkaTopicReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	KafkaClientConfig *kafka.ClusterConfig
	Messenger         *reporter.Messenger
}

//+kubebuilder:rbac:groups=xo.90poe.io,resources=kafkatopics,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=xo.90poe.io,resources=kafkatopics/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=xo.90poe.io,resources=kafkatopics/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KafkaTopic object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *KafkaTopicReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx).WithValues("kafkatopic", req.NamespacedName)

	// Fetch the KafkaTopic instance
	instance := &xov1alpha1.KafkaTopic{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.V(0).Info("KafkaTopic resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.V(0).Info(fmt.Sprintf("Failed to get KafkaTopic: %v", err))
		return ctrl.Result{}, err
	}

	// Get Kafka client
	kClient, err := r.KafkaClientConfig.GetClient()
	if err != nil {
		reqLogger.V(0).Info(fmt.Sprintf("Failed to get Kafka Client: %v", err))
		r.Messenger.Send(fmt.Sprintf("%v", err), reporter.ErrorMessage)
		return ctrl.Result{}, nil
	}
	defer kClient.Close()

	return r.upsertTopic(ctx, kClient, instance, reqLogger)
}

// SetupWithManager sets up the controller with the Manager.
func (r *KafkaTopicReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// init config
	config, err := env.NewConfig()
	if err != nil {
		return err
	}
	// make label selector
	labelSelectorPredicate, err := predicate.LabelSelectorPredicate(*config.LabelSelectors)
	if err != nil {
		return err
	}
	// init kafka client
	r.KafkaClientConfig, err = kafka.NewClusterConfig(
		config.KafkaTopicNameRegexp,
		kafka.Brokers(config.KafkaBrokers),
		kafka.MaxPartsPerTopic(config.MaxKafkaTopicsPartitions),
	)
	if err != nil {
		return err
	}
	// Make slack messanger
	r.Messenger, err = reporter.New(config.SlackToken,
		reporter.SlackChannel(config.SlackChannel))
	if err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&xov1alpha1.KafkaTopic{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: config.MaxConcurrentReconciles}).
		WithEventFilter(labelSelectorPredicate).
		WithEventFilter(ignoreUpdateDeletePredicate()).
		Complete(r)
}

// upsertTopic will update or insert topic
func (r *KafkaTopicReconciler) upsertTopic(ctx context.Context, kClient *kafka.ClusterClient, topic *xov1alpha1.KafkaTopic, reqLogger logr.Logger) (_ ctrl.Result, retErr error) {
	// Init status
	statusMessage := "Succeeded"
	status := metav1.ConditionTrue
	condition := ConditionsInsert
	reason := ConditionReasonCreateTopic

	// Defer function to update status
	defer func() {
		// Log status update
		reqLogger.Info(fmt.Sprintf("topic %s %s status: %s", topic.Spec.Name,
			reason, statusMessage))
		// Send message to slack
		if status == metav1.ConditionFalse {
			// send message only on error
			r.Messenger.Send(statusMessage, reporter.ErrorMessage)
		}
		// Remove last condition and set new one
		meta.RemoveStatusCondition(&topic.Status.Conditions, condition)
		meta.SetStatusCondition(&topic.Status.Conditions, metav1.Condition{
			Type:    condition,
			Status:  status,
			Reason:  reason,
			Message: statusMessage,
		})
		// we will return error of status update if it is not nil
		err := r.Status().Update(ctx, topic)
		if err != nil {
			reqLogger.V(0).Info(fmt.Sprintf("Failed to update topic status: %v", retErr))
			retErr = errors.Join(retErr, err)
		}
	}()

	// Check if topic exists in Kafka cluster
	exists, err := kClient.TopicExists(&topic.Spec)
	if err != nil {
		status = metav1.ConditionFalse
		statusMessage = fmt.Sprintf("can't check if topic %s exists: %v", topic.Name, err)
		return ctrl.Result{}, nil
	}

	// Create or update topic
	if exists {
		condition = ConditionsUpdate
		reason = ConditionReasonUpdateTopic
		err = kClient.UpdateTopic(&topic.Spec)
	} else {
		err = kClient.CreateTopic(&topic.Spec)
	}
	if err != nil {
		status = metav1.ConditionFalse
		statusMessage = fmt.Sprintf("can't %s kafka topic %s: %v", reason, topic.Name, err)
		return ctrl.Result{}, nil
	}

	return ctrl.Result{
		RequeueAfter: RevisitIntervalSec * time.Second,
	}, nil
}
