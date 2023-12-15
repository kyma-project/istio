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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type State string
type ConditionType string
type ConditionReason string

// Valid IstioCR States.
const (
	Ready      State = "Ready"
	Processing State = "Processing"
	Error      State = "Error"
	Deleting   State = "Deleting"
	Warning    State = "Warning"

	ConditionTypeReady ConditionType = "Ready"

	ConditionReasonReconcileSucceeded            ConditionReason = "ReconcileSucceeded"
	ConditionReasonReconcileFailed               ConditionReason = "ReconcileFailed"
	ConditionReasonUpdateCheckSucceeded          ConditionReason = "UpdateCheckSucceeded"
	ConditionReasonUpdateDone                    ConditionReason = "UpdateDone"
	ConditionReasonProcessing                    ConditionReason = "Processing"
	ConditionReasonUpdateCheck                   ConditionReason = "UpdateCheck"
	ConditionReasonIstioCRsDangling              ConditionReason = "IstioCustomResourcesDangling"
	ConditionReasonIstioCRsReconcileFailed       ConditionReason = "IstioCustomResourcesReconcileFailed"
	ConditionReasonProxyResetReconcileFailed     ConditionReason = "ProxyResetReconcileFailed"
	ConditionReasonIngressGatewayReconcileFailed ConditionReason = "IngressGatewayReconcileFailed"
	ConditionReasonCustomResourceMisconfigured   ConditionReason = "CustomResourceMisconfigured"
	ConditionReasonDeleting                      ConditionReason = "Deleting"
	ConditionReasonIstioInstallationFailed       ConditionReason = "IstioInstallationFailed"
	ConditionReasonOlderCRExists                 ConditionReason = "OlderCRExists"

	ConditionReasonReconcileSucceededMessage            = "Reconciled successfully"
	ConditionReasonReconcileFailedMessage               = "Reconciliation failed"
	ConditionReasonUpdateCheckSucceededMessage          = "Update not required"
	ConditionReasonUpdateDoneMessage                    = "Update done"
	ConditionReasonProcessingMessage                    = "Istio installation is proceeding"
	ConditionReasonUpdateCheckMessage                   = "Checking if update is required"
	ConditionReasonIstioCRsDanglingMessage              = "Istio deletion blocked because of existing Istio resources that are not default"
	ConditionReasonIstioCRsReconcileFailedMessage       = "Istio custom resources reconciliation failed"
	ConditionReasonProxyResetReconcileFailedMessage     = "Proxy reset reconciliation failed"
	ConditionReasonIngressGatewayReconcileFailedMessage = "Ingress Gateway reconciliation failed"
	ConditionReasonCustomResourceMisconfiguredMessage   = "Configuration present on Istio Custom Resource is not correct"
	ConditionReasonDeletingMessage                      = "Proceeding with uninstallation and deletion of Istio"
	ConditionReasonIstioInstallationFailedMessage       = "Failure during execution of Istio installation"
	ConditionReasonOlderCRExistsMessage                 = "This CR is not the oldest one so does not represent the module State"
)

type ConditionMeta struct {
	Type    ConditionType
	Status  metav1.ConditionStatus
	Message string
}

var ConditionReasons = map[ConditionReason]ConditionMeta{
	ConditionReasonReconcileSucceeded:            {Type: ConditionTypeReady, Status: metav1.ConditionTrue, Message: ConditionReasonReconcileSucceededMessage},
	ConditionReasonReconcileFailed:               {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonReconcileFailedMessage},
	ConditionReasonUpdateCheckSucceeded:          {Type: ConditionTypeReady, Status: metav1.ConditionTrue, Message: ConditionReasonUpdateCheckSucceededMessage},
	ConditionReasonUpdateDone:                    {Type: ConditionTypeReady, Status: metav1.ConditionTrue, Message: ConditionReasonUpdateDoneMessage},
	ConditionReasonProcessing:                    {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonProcessingMessage},
	ConditionReasonUpdateCheck:                   {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonUpdateCheckMessage},
	ConditionReasonIstioCRsDangling:              {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIstioCRsDanglingMessage},
	ConditionReasonIstioCRsReconcileFailed:       {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIstioCRsReconcileFailedMessage},
	ConditionReasonProxyResetReconcileFailed:     {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonProxyResetReconcileFailedMessage},
	ConditionReasonIngressGatewayReconcileFailed: {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIngressGatewayReconcileFailedMessage},
	ConditionReasonCustomResourceMisconfigured:   {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonCustomResourceMisconfiguredMessage},
	ConditionReasonDeleting:                      {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonDeletingMessage},
	ConditionReasonIstioInstallationFailed:       {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIstioInstallationFailedMessage},
	ConditionReasonOlderCRExists:                 {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonOlderCRExistsMessage},
}

// Defines the desired specification for installing or updating Istio.
type IstioSpec struct {
	// +kubebuilder:validation:Optional
	Config Config `json:"config,omitempty"`
	// +kubebuilder:validation:Optional
	Components *Components `json:"components,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:JSONPath=".status.state",name="State",type="string"

// Contains Istio CR specification and current status.
type Istio struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IstioSpec   `json:"spec,omitempty"`
	Status IstioStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// Contains a list of Istio's.
type IstioList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Istio `json:"items"`
}

// IstioStatus defines the observed state of IstioCR.
type IstioStatus struct {
	// State signifies current state of CustomObject. Value
	// can be one of ("Ready", "Processing", "Error", "Deleting", "Warning").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Processing;Deleting;Ready;Error;Warning
	State State `json:"state"`
	//  Conditions associated with IstioStatus.
	Conditions *[]metav1.Condition `json:"conditions,omitempty"`
	// Description of Istio status
	Description string `json:"description,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Istio{}, &IstioList{})
}

func (i *Istio) GetStatus() IstioStatus {
	return i.Status
}

func (i *Istio) SetStatus(status IstioStatus) {
	i.Status = status
}

func (i *Istio) ComponentName() string {
	return "istio"
}

func (i *Istio) HasFinalizer() bool {
	return len(i.Finalizers) > 0
}

func ConditionFromReason(reason ConditionReason, customMessage string) *metav1.Condition {
	condition, found := ConditionReasons[reason]
	if found {
		if customMessage == "" {
			customMessage = condition.Message
		}
		return &metav1.Condition{
			Type:    string(condition.Type),
			Status:  condition.Status,
			Reason:  string(reason),
			Message: customMessage,
		}
	}
	return nil
}
