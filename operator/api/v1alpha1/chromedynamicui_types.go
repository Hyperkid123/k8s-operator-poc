/*
Copyright 2024.

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

type RouteDefinition struct {
	Pathname   string `json:"pathname"`
	Exact      bool   `json:"exact,omitempty"`
	Module     string `json:"module"`
	ImportName string `json:"importName,omitempty"`
	// +kubebuilder:validation:XPreserveUnknownFields
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	Props map[string]string `json:"props,omitempty"`
}

type UiModule struct {
	ManifestLocation string            `json:"manifestLocation"`
	Routes           []RouteDefinition `json:"routes"`
}

// ChromeDynamicUISpec defines the desired state of ChromeDynamicUI
type ChromeDynamicUISpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ChromeDynamicUI. Edit chromedynamicui_types.go to remove/update
	Name   string   `json:"name"`
	Title  string   `json:"title"`
	Module UiModule `json:"module"`
}

// ChromeDynamicUIStatus defines the observed state of ChromeDynamicUI
type ChromeDynamicUIStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ChromeDynamicUI is the Schema for the chromedynamicuis API
type ChromeDynamicUI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChromeDynamicUISpec   `json:"spec,omitempty"`
	Status ChromeDynamicUIStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ChromeDynamicUIList contains a list of ChromeDynamicUI
type ChromeDynamicUIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ChromeDynamicUI `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ChromeDynamicUI{}, &ChromeDynamicUIList{})
}
