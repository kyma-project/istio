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

	// general.
	ConditionReasonReconcileSucceeded        ConditionReason = "ReconcileSucceeded"
	ConditionReasonReconcileSucceededMessage                 = "Reconciliation succeeded"
	ConditionReasonReconcileUnknown          ConditionReason = "ReconcileUnknown"
	ConditionReasonReconcileUnknownMessage                   = "Module readiness is unknown. Either a reconciliation is progressing, or failed previously. Check status of other conditions"
	ConditionReasonReconcileRequeued         ConditionReason = "ReconcileRequeued"
	ConditionReasonReconcileRequeuedMessage                  = "Proxy reset is still ongoing. Reconciliation requeued"
	ConditionReasonReconcileFailed           ConditionReason = "ReconcileFailed"
	ConditionReasonReconcileFailedMessage                    = "Reconciliation failed"
	ConditionReasonValidationFailed          ConditionReason = "ValidationFailed"
	ConditionReasonValidationFailedMessage                   = "Reconciliation did not happen as Istio Custom Resource failed to validate"
	ConditionReasonOlderCRExists             ConditionReason = "OlderCRExists"
	ConditionReasonOlderCRExistsMessage                      = "This Istio custom resource is not the oldest one and does not represent the module state"
	ConditionReasonOldestCRNotFound          ConditionReason = "OldestCRNotFound"
	ConditionReasonOldestCRNotFoundMessage                   = "Oldest Istio custom resource could not be found"

	// install / uninstall.
	ConditionReasonIstioInstallNotNeeded               ConditionReason = "IstioInstallNotNeeded"
	ConditionReasonIstioInstallNotNeededMessage                        = "Istio installation is not needed"
	ConditionReasonIstioInstallSucceeded               ConditionReason = "IstioInstallSucceeded"
	ConditionReasonIstioInstallSucceededMessage                        = "Istio installation succeeded"
	ConditionReasonIstioUninstallSucceeded             ConditionReason = "IstioUninstallSucceeded"
	ConditionReasonIstioUninstallSucceededMessage                      = "Istio uninstallation succeded"
	ConditionReasonIstioInstallUninstallFailed         ConditionReason = "IstioInstallUninstallFailed"
	ConditionReasonIstioInstallUninstallFailedMessage                  = "Istio install or uninstall failed"
	ConditionReasonCustomResourceMisconfigured         ConditionReason = "IstioCustomResourceMisconfigured"
	ConditionReasonCustomResourceMisconfiguredMessage                  = "Istio custom resource has invalid configuration"
	ConditionReasonIstioCRsDangling                    ConditionReason = "IstioCustomResourcesDangling"
	ConditionReasonIstioCRsDanglingMessage                             = "Istio deletion blocked because of existing Istio custom resources"
	ConditionReasonIstioVersionUpdateNotAllowed        ConditionReason = "IstioVersionUpdateNotAllowed"
	ConditionReasonIstioVersionUpdateNotAllowedMessage                 = "Update to the new Istio version is not allowed"

	// Istio CRs.
	ConditionReasonCRsReconcileSucceeded        ConditionReason = "CustomResourcesReconcileSucceeded"
	ConditionReasonCRsReconcileSucceededMessage                 = "Custom resources reconciliation succeeded"
	ConditionReasonCRsReconcileFailed           ConditionReason = "CustomResourcesReconcileFailed"
	ConditionReasonCRsReconcileFailedMessage                    = "Custom resources reconciliation failed"

	// proxy reset.
	ConditionReasonProxySidecarRestartSucceeded                 ConditionReason = "ProxySidecarRestartSucceeded"
	ConditionReasonProxySidecarRestartSucceededMessage                          = "Proxy sidecar restart succeeded"
	ConditionReasonProxySidecarRestartFailed                    ConditionReason = "ProxySidecarRestartFailed"
	ConditionReasonProxySidecarRestartFailedMessage                             = "Proxy sidecar restart failed"
	ConditionReasonProxySidecarRestartPartiallySucceeded        ConditionReason = "ProxySidecarRestartPartiallySucceeded"
	ConditionReasonProxySidecarRestartPartiallySucceededMessage                 = "Proxy sidecar restart partially succeeded"
	ConditionReasonProxySidecarManualRestartRequired            ConditionReason = "ProxySidecarManualRestartRequired"
	ConditionReasonProxySidecarManualRestartRequiredMessage                     = "Proxy sidecar manual restart is required for some workloads"

	// ingress gateway.
	ConditionReasonIngressGatewayRestartSucceeded        ConditionReason = "IngressGatewayRestartSucceeded"
	ConditionReasonIngressGatewayRestartSucceededMessage                 = "Istio Ingress Gateway restart succeeded"
	ConditionReasonIngressGatewayRestartFailed           ConditionReason = "IngressGatewayRestartFailed"
	ConditionReasonIngressGatewayRestartFailedMessage                    = "Istio Ingress Gateway restart failed"

	// egress gateway.
	ConditionReasonEgressGatewayRestartSucceeded        ConditionReason = "EgressGatewayRestartSucceeded"
	ConditionReasonEgressGatewayRestartSucceededMessage                 = "Istio Egress Gateway restart succeeded"
	ConditionReasonEgressGatewayRestartFailed           ConditionReason = "EgressGatewayRestartFailed"
	ConditionReasonEgressGatewayRestartFailedMessage                    = "Istio Egress Gateway restart failed"
)

type ReasonWithMessage struct {
	Reason  ConditionReason
	Message string
}

// IstioSpec describes the desired specification for installing or updating Istio.
type IstioSpec struct {
	// +kubebuilder:validation:Optional
	Config Config `json:"config,omitempty"`
	// +kubebuilder:validation:Optional
	Components *Components `json:"components,omitempty"`
	// +kubebuilder:validation:Optional
	Experimental *Experimental `json:"experimental,omitempty"`
	// +kubebuilder:validation:Optional
	CompatibilityMode bool `json:"compatibilityMode,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:categories={kyma-modules,kyma-istio}
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:JSONPath=".status.state",name="State",type="string"
//+kubebuilder:storageversion

// Istio contains Istio CR specification and current status.
type Istio struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IstioSpec   `json:"spec,omitempty"`
	Status IstioStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IstioList contains a list of Istio's.
type IstioList struct {
	metav1.TypeMeta `        json:",inline"`
	metav1.ListMeta `        json:"metadata,omitempty"`
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

func init() { //nolint:gochecknoinits // this was scaffolded from kubebuilder. TODO: remove this init function
	SchemeBuilder.Register(&Istio{}, &IstioList{})
}
