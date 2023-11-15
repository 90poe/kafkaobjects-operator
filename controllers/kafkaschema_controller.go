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
	"github.com/90poe/kafkaobjects-operator/internal/reporter"
	"github.com/90poe/kafkaobjects-operator/internal/schemaregistry"
	"github.com/go-logr/logr"
)

// KafkaSchemaReconciler reconciles a KafkaSchema object
type KafkaSchemaReconciler struct {
	client.Client
	Scheme                    *runtime.Scheme
	KafkaSchemaRegistryClient *schemaregistry.Client
	Messenger                 *reporter.Messenger
}

//+kubebuilder:rbac:groups=xo.90poe.io,resources=kafkaschemas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=xo.90poe.io,resources=kafkaschemas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=xo.90poe.io,resources=kafkaschemas/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KafkaSchema object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *KafkaSchemaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx).WithValues("kafkaschema", req.NamespacedName)

	// Fetch the KafkaSchema instance
	instance := &xov1alpha1.KafkaSchema{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("KafkaSchema resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get KafkaSchema.")
		return ctrl.Result{}, err
	}

	return r.upsertSchema(ctx, instance, reqLogger)
}

// SetupWithManager sets up the controller with the Manager.
func (r *KafkaSchemaReconciler) SetupWithManager(mgr ctrl.Manager) error {
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
	r.KafkaSchemaRegistryClient, err = schemaregistry.NewClient(
		schemaregistry.URL(config.SchemaRegistryURL),
	)
	if err != nil {
		return err
	}
	// Make slack Messenger
	r.Messenger, err = reporter.New(config.SlackToken,
		reporter.SlackChannel(config.SlackChannel))
	if err != nil {
		return err
	}
	// Init Kafka manager
	return ctrl.NewControllerManagedBy(mgr).
		For(&xov1alpha1.KafkaSchema{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: config.MaxConcurrentReconciles}).
		WithEventFilter(labelSelectorPredicate).
		WithEventFilter(ignoreUpdateDeletePredicate()).
		Complete(r)
}

// upsertSchema will update or insert schema
func (r *KafkaSchemaReconciler) upsertSchema(ctx context.Context, schema *xov1alpha1.KafkaSchema, reqLogger logr.Logger) (_ ctrl.Result, retErr error) {
	// Init status
	statusMessage := "Succeeded"
	status := metav1.ConditionTrue
	condition := ConditionsInsert
	reason := ConditionReasonCreateSchema

	// Defer function to update status
	defer func() {
		// Log status update
		reqLogger.Info(fmt.Sprintf("schema %s %s status: %s", schema.Spec.Name,
			reason, statusMessage))
		// Send message to slack
		if status == metav1.ConditionFalse {
			// send message only on error
			r.Messenger.Send(statusMessage, reporter.ErrorMessage)
		}
		// Remove last condition and set new one
		meta.RemoveStatusCondition(&schema.Status.Conditions, condition)
		meta.SetStatusCondition(&schema.Status.Conditions, metav1.Condition{
			Type:    condition,
			Status:  status,
			Reason:  reason,
			Message: statusMessage,
		})
		// we will return error of status update if it is not nil
		err := r.Status().Update(ctx, schema)
		if err != nil {
			reqLogger.V(0).Info(fmt.Sprintf("Failed to update schema status: %v", retErr))
			retErr = errors.Join(retErr, err)
		}
	}()

	// Check if schema exists in Kafka Schema Registry
	exists, err := r.KafkaSchemaRegistryClient.SchemaExists(schema.Spec.Name)
	if err != nil {
		status = metav1.ConditionFalse
		statusMessage = fmt.Sprintf("can't check if schema %s exists: %v", schema.Name, err)
		return ctrl.Result{}, nil
	}

	// Create or update schema
	if exists {
		condition = ConditionsUpdate
		reason = ConditionReasonUpdateSchema
	}
	err = r.KafkaSchemaRegistryClient.CreateSchema(&schema.Spec)
	if err != nil {
		status = metav1.ConditionFalse
		statusMessage = fmt.Sprintf("can't %s kafka schema %s: %v", reason, schema.Name, err)
		return ctrl.Result{}, nil
	}

	return ctrl.Result{
		RequeueAfter: RevisitIntervalSec * time.Second,
	}, nil
}
