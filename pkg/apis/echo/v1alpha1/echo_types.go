package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EchoSpec defines the desired state of Echo
// +k8s:openapi-gen=true
type EchoSpec struct {
	// Message is the message which Echo should print to STDOUT
	Message string `json:"message"`
	// Replicas is the number of Echos which should exist
	Replicas int32 `json:"replicas,omitempty"`
	// Namespace is the namespace in which an Echo should be deployed
	Namespace string `json:"namespace"`
	// Version is the version tag on the application image to use, default is 'latest'
	Version string `json:"version,omitempty"`
}

// EchoStatus defines the observed state of Echo
// +k8s:openapi-gen=true
type EchoStatus struct {
	Nodes []string `json:"nodes"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Echo is the Schema for the echos API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Echo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EchoSpec   `json:"spec,omitempty"`
	Status EchoStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EchoList contains a list of Echo
type EchoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Echo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Echo{}, &EchoList{})
}
