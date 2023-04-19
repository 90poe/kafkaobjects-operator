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

package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
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
	"github.com/90poe/kafkaobjects-operator/internal/schemaregistry"
)

// KafkaSchemaReconciler reconciles a KafkaSchema object
type KafkaSchemaReconciler struct {
	client.Client
	Scheme                    *runtime.Scheme
	KafkaSchemaRegistryClient *schemaregistry.Client
}

//+kubebuilder:rbac:groups=xo.ninetypercent.io,resources=kafkaschemas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=xo.ninetypercent.io,resources=kafkaschemas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=xo.ninetypercent.io,resources=kafkaschemas/finalizers,verbs=update

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
	reqLogger.Info("Reconciling KafkaSchemas")
	// Fetch the KafkaSchema instance
	instance := &xov1alpha1.KafkaSchema{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
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

	// Check status conditions
	condition := meta.FindStatusCondition(instance.Status.Conditions, "Ready")
	if condition != nil {
		// we are running on changed or ready object, so we can exit now
		// as we don't update or delete topic
		return ctrl.Result{}, nil
	}

	statusMessage := "Succeeded"
	err = r.KafkaSchemaRegistryClient.CreateSchema(&instance.Spec)
	if err != nil {
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to create schema.")
		statusMessage = fmt.Sprintf("can't create schema %s: %v", instance.Name, err)
	}

	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionTrue,
		Reason:  "creation",
		Message: statusMessage,
	})
	err = r.Status().Update(ctx, instance)
	if err != nil {
		reqLogger.Error(err, "Failed to update instance status.")
	}
	reqLogger.Info(fmt.Sprintf("schema %s succefully created", instance.Spec.Name))

	return ctrl.Result{}, nil
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
	// Init Kafka manager
	return ctrl.NewControllerManagedBy(mgr).
		For(&xov1alpha1.KafkaSchema{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: config.MaxConcurrentReconciles}).
		WithEventFilter(labelSelectorPredicate).
		Complete(r)
}
