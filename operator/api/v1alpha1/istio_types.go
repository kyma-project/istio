/*
Copyright 2022.

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
	"github.com/kyma-project/module-manager/operator/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IstioSpec defines the desired state of Istio
type IstioSpec struct {
	// +kubebuilder:validation:Optional
	ReleaseName string `json:"releaseName,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Istio is the Schema for the istio API
type Istio struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IstioSpec    `json:"spec,omitempty"`
	Status types.Status `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IstioList contains a list of Istio
type IstioList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Istio `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Istio{}, &IstioList{})
}

var _ types.CustomObject = &Istio{}

func (i *Istio) GetStatus() types.Status {
	return i.Status
}

func (i *Istio) SetStatus(status types.Status) {
	i.Status = status
}

func (i *Istio) ComponentName() string {
	return "istio"
}
