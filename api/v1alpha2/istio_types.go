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

package v1alpha2

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

	ConditionTypeReady                        ConditionType = "Ready"
	ConditionTypeProxySidecarRestartSucceeded ConditionType = "ProxySidecarRestartSucceeded"

	// general
	ConditionReasonReconcileSucceeded        ConditionReason = "ReconcileSucceeded"
	ConditionReasonReconcileSucceededMessage                 = "Reconciliation succeeded"
	ConditionReasonReconcileFailed           ConditionReason = "ReconcileFailed"
	ConditionReasonReconcileFailedMessage                    = "Reconciliation failed"
	ConditionReasonOlderCRExists             ConditionReason = "OlderCRExists"
	ConditionReasonOlderCRExistsMessage                      = "This Istio custom resource is not the oldest one and does not represent the module state"

	// install / uninstall
	ConditionReasonIstioInstallNotNeeded              ConditionReason = "IstioInstallNotNeeded"
	ConditionReasonIstioInstallNotNeededMessage                       = "Istio installation is not needed"
	ConditionReasonIstioInstallSucceeded              ConditionReason = "IstioInstallSucceeded"
	ConditionReasonIstioInstallSucceededMessage                       = "Istio installation succeeded"
	ConditionReasonIstioUninstallSucceeded            ConditionReason = "IstioUninstallSucceeded"
	ConditionReasonIstioUninstallSucceededMessage                     = "Istio uninstallation succeded"
	ConditionReasonIstioInstallUninstallFailed        ConditionReason = "IstioInstallUninstallFailed"
	ConditionReasonIstioInstallUninstallFailedMessage                 = "Istio install or uninstall failed"
	ConditionReasonCustomResourceMisconfigured        ConditionReason = "IstioCustomResourceMisconfigured"
	ConditionReasonCustomResourceMisconfiguredMessage                 = "Istio custom resource has invalid configuration"
	ConditionReasonIstioCRsDangling                   ConditionReason = "IstioCustomResourcesDangling"
	ConditionReasonIstioCRsDanglingMessage                            = "Istio deletion blocked because of existing Istio custom resources"

	// Istio CRs
	ConditionReasonCRsReconcileSucceeded        ConditionReason = "CustomResourcesReconcileSucceeded"
	ConditionReasonCRsReconcileSucceededMessage                 = "Custom resources reconciliation succeeded"
	ConditionReasonCRsReconcileFailed           ConditionReason = "CustomResourcesReconcileFailed"
	ConditionReasonCRsReconcileFailedMessage                    = "Custom resources reconciliation failed"

	// proxy reset
	ConditionReasonProxySidecarRestartSucceeded             ConditionReason = "ProxySidecarRestartSucceeded"
	ConditionReasonProxySidecarRestartSucceededMessage                      = "Proxy sidecar restart succeeded"
	ConditionReasonProxySidecarRestartFailed                ConditionReason = "ProxySidecarRestartFailed"
	ConditionReasonProxySidecarRestartFailedMessage                         = "Proxy sidecar restart failed"
	ConditionReasonProxySidecarManualRestartRequired        ConditionReason = "ProxySidecarManualRestartRequired"
	ConditionReasonProxySidecarManualRestartRequiredMessage                 = "Proxy sidecar manual restart is required for some workloads"

	// ingress gateway
	ConditionReasonIngressGatewayReconcileSucceeded        ConditionReason = "IngressGatewayReconcileSucceeded"
	ConditionReasonIngressGatewayReconcileSucceededMessage                 = "Istio Ingress Gateway reconciliation succeeded"
	ConditionReasonIngressGatewayReconcileFailed           ConditionReason = "IngressGatewayReconcileFailed"
	ConditionReasonIngressGatewayReconcileFailedMessage                    = "Istio Ingress Gateway reconciliation failed"

	// external authorizer
	ConditionReasonExternalAuthorizerReconcileSucceeded        ConditionReason = "ExternalAuthorizerReconcileSucceeded"
	ConditionReasonExternalAuthorizerReconcileSucceededMessage                 = "External Authorizer reconciliation Succeeded"
	ConditionReasonExternalAuthorizerReconcileFailed           ConditionReason = "ExternalAuthorizerReconcileFailed"
	ConditionReasonExternalAuthorizerReconcileFailedMessage                    = "External Authorizer reconciliation Failed"
)

type ReasonWithMessage struct {
	Reason  ConditionReason
	Message string
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
//+kubebuilder:storageversion

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
	// State signifies the current state of CustomObject. Value
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

func ConditionFromReason(reason ReasonWithMessage) *metav1.Condition {
	condition, found := conditionReasons[reason.Reason]
	if found {
		message := condition.Message
		if reason.Message != "" {
			message = reason.Message
		}
		return &metav1.Condition{
			Type:    string(condition.Type),
			Status:  condition.Status,
			Reason:  string(reason.Reason),
			Message: message,
		}
	}
	return nil
}

var conditionReasons = map[ConditionReason]conditionMeta{
	ConditionReasonReconcileSucceeded: {Type: ConditionTypeReady, Status: metav1.ConditionTrue, Message: ConditionReasonReconcileSucceededMessage},
	ConditionReasonReconcileFailed:    {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonReconcileFailedMessage},
	ConditionReasonOlderCRExists:      {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonOlderCRExistsMessage},

	ConditionReasonIstioInstallNotNeeded:       {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIstioInstallNotNeededMessage},
	ConditionReasonIstioInstallSucceeded:       {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIstioInstallSucceededMessage},
	ConditionReasonIstioUninstallSucceeded:     {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIstioUninstallSucceededMessage},
	ConditionReasonIstioInstallUninstallFailed: {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIstioInstallUninstallFailedMessage},
	ConditionReasonCustomResourceMisconfigured: {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonCustomResourceMisconfiguredMessage},
	ConditionReasonIstioCRsDangling:            {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIstioCRsDanglingMessage},

	ConditionReasonCRsReconcileSucceeded: {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonCRsReconcileSucceededMessage},
	ConditionReasonCRsReconcileFailed:    {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonCRsReconcileFailedMessage},

	ConditionReasonProxySidecarRestartSucceeded:      {Type: ConditionTypeProxySidecarRestartSucceeded, Status: metav1.ConditionTrue, Message: ConditionReasonProxySidecarRestartSucceededMessage},
	ConditionReasonProxySidecarRestartFailed:         {Type: ConditionTypeProxySidecarRestartSucceeded, Status: metav1.ConditionFalse, Message: ConditionReasonProxySidecarRestartFailedMessage},
	ConditionReasonProxySidecarManualRestartRequired: {Type: ConditionTypeProxySidecarRestartSucceeded, Status: metav1.ConditionFalse, Message: ConditionReasonProxySidecarManualRestartRequiredMessage},

	ConditionReasonIngressGatewayReconcileSucceeded: {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIngressGatewayReconcileSucceededMessage},
	ConditionReasonIngressGatewayReconcileFailed:    {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIngressGatewayReconcileFailedMessage},

	ConditionReasonExternalAuthorizerReconcileSucceeded: {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonExternalAuthorizerReconcileSucceededMessage},
	ConditionReasonExternalAuthorizerReconcileFailed:    {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonExternalAuthorizerReconcileFailedMessage},
}

type conditionMeta struct {
	Type    ConditionType
	Status  metav1.ConditionStatus
	Message string
}

func NewReasonWithMessage(reason ConditionReason, customMessage ...string) ReasonWithMessage {
	message := ""
	if len(customMessage) > 0 {
		message = customMessage[0]
	}
	return ReasonWithMessage{
		Reason:  reason,
		Message: message,
	}
}

func (i *Istio) HasFinalizers() bool {
	return len(i.Finalizers) > 0
}

func IsReadyTypeCondition(reason ReasonWithMessage) bool {
	condition, found := conditionReasons[reason.Reason]
	if found && condition.Type == ConditionTypeReady {
		return true
	}
	return false
}
