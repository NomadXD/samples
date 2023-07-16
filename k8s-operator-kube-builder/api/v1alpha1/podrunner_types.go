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

// PodRunnerSpec defines the desired state of PodRunner
type PodRunnerSpec struct {
	// PodName is the name of the pod.
	PodName string `json:"podName,omitempty"`

	// ImageName is the name of the image used t
	ImageName string `json:"imageName,omitempty"`

	Namespace string `json:"namespace,omitempty"`
}

// PodRunnerStatus defines the observed state of PodRunner
type PodRunnerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	PodStatus string `json:"podStatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PodRunner is the Schema for the podrunners API
type PodRunner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodRunnerSpec   `json:"spec,omitempty"`
	Status PodRunnerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PodRunnerList contains a list of PodRunner
type PodRunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodRunner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PodRunner{}, &PodRunnerList{})
}
