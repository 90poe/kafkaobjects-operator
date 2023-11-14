package controllers

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	ConditionsInsert            = "Insert"
	ConditionsUpdate            = "Update"
	ConditionReasonCreateTopic  = "CreateTopic"
	ConditionReasonUpdateTopic  = "UpdateTopic"
	ConditionReasonCreateSchema = "CreateSchema"
	ConditionReasonUpdateSchema = "UpdateSchema"
	RevisitIntervalSec          = 36000 // 10 hours
)

// ignoreUpdateDeletePredicater is brilliantly useful function, it will prevent multiple reconcile calls
func ignoreUpdateDeletePredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			genChanged := e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
			return genChanged
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted.
			return !e.DeleteStateUnknown
		},
	}
}
