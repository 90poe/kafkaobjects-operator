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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Segment to note Kafka segment size
type Segment struct {
	// This configuration controls the segment file size for the log.
	// Retention and cleaning is always done a file at a time so a larger segment size
	// means fewer files but less granular control over retention.
	// +optional
	// +kubebuilder:default=0
	Bytes uint `json:"bytes,omitempty"`
	// This configuration controls the period of time after which Kafka will force the log
	// to roll even if the segment file isn't full to ensure that retention can delete or
	// compact old data.
	// +optional
	// +kubebuilder:default=0
	MS uint64 `json:"ms,omitempty"`
}

// KafkaTopicSpec defines the desired state of KafkaTopic
type KafkaTopicSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9\\._\\-]{1,255}$`
	Name string `json:"name"`
	// +optional
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^(delete|compact)$`
	// +kubebuilder:default=delete
	CleanupPolicy string  `json:"cleanuppolicy,omitempty"`
	Segment       Segment `json:"segment,omitempty"`
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	Partitions uint `json:"partitions"`
	// +optional
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=2
	MinInSyncReplicas uint `json:"mininsyncreplicas,omitempty"`
	// +optional
	// +kubebuilder:default=-1
	RetentionHours int `json:"retentionhours,omitempty"`
	// +optional
	// +kubebuilder:default=-1
	RetentionBytes int `json:"retentionbytes,omitempty"`
	// +optional
	// +kubebuilder:default=1048576
	MaxMessageBytes int64 `json:"maxmessagebytes,omitempty"`
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=2
	Replication uint `json:"replication"`
}

// KafkaTopicStatus defines the observed state of KafkaTopic
type KafkaTopicStatus struct {
	// Represents the observations of a KafkaTopic's current state.
	// KafkaTopic.status.conditions.type are: "Available", "Progressing", and "Degraded"
	// KafkaTopic.status.conditions.status are one of True, False, Unknown.
	// KafkaTopic.status.conditions.reason the value should be a CamelCase string and producers of specific
	// condition types may define expected values and meanings for this field, and whether the values
	// are considered a guaranteed API.
	// KafkaTopic.status.conditions.Message is a human readable message indicating details about the transition.
	// For further information see: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// Conditions store the status conditions of the KafkaTopic instances
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KafkaTopic is the Schema for the kafkatopics API
type KafkaTopic struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaTopicSpec   `json:"spec,omitempty"`
	Status KafkaTopicStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KafkaTopicList contains a list of KafkaTopic
type KafkaTopicList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaTopic `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KafkaTopic{}, &KafkaTopicList{})
}
