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
	// Ready is reported when the Istio installation / upgrade process has completed successfully.
	Ready State = "Ready"
	// Processing is reported when the Istio installation / upgrade process is in progress.
	Processing State = "Processing"
	// Error is reported when the Istio installation / upgrade process has failed.
	Error State = "Error"
	// Deleting is reported when the Istio installation / upgrade process is being deleted.
	Deleting State = "Deleting"
	// Warning is reported when the Istio installation / upgrade process has completed with warnings.
	// This state warrants user attention, as some features may not work as expected.
	Warning State = "Warning"

	ConditionTypeReady                             ConditionType = "Ready"
	ConditionTypeProxySidecarRestartSucceeded      ConditionType = "ProxySidecarRestartSucceeded"
	ConditionTypeIngressTargetingUserResourceFound ConditionType = "IngressTargetingUserResourceFound"

	// General

	// Reconciliation finished with full success.
	ConditionReasonReconcileSucceeded        ConditionReason = "ReconcileSucceeded"
	ConditionReasonReconcileSucceededMessage                 = "Reconciliation succeeded"
	// Reconciliation is in progress or failed previously.
	ConditionReasonReconcileUnknown        ConditionReason = "ReconcileUnknown"
	ConditionReasonReconcileUnknownMessage                 = "Module readiness is unknown. Either a reconciliation is progressing, or failed previously. Check status of other conditions"
	// Reconciliation is requeued to be tried again later.
	ConditionReasonReconcileRequeued        ConditionReason = "ReconcileRequeued"
	ConditionReasonReconcileRequeuedMessage                 = "Proxy reset is still ongoing. Reconciliation requeued"
	// Reconciliation failed.
	ConditionReasonReconcileFailed        ConditionReason = "ReconcileFailed"
	ConditionReasonReconcileFailedMessage                 = "Reconciliation failed"
	// Reconciliation did not happen as validation of Istio Custom Resource failed.
	ConditionReasonValidationFailed        ConditionReason = "ValidationFailed"
	ConditionReasonValidationFailedMessage                 = "Reconciliation did not happen as Istio Custom Resource failed to validate"
	// Reconciliation did not happen as there exists an older Istio Custom Resource.
	ConditionReasonOlderCRExists        ConditionReason = "OlderCRExists"
	ConditionReasonOlderCRExistsMessage                 = "This Istio custom resource is not the oldest one and does not represent the module state"
	// Reconciliation did not happen as the oldest Istio Custom Resource could not be found.
	ConditionReasonOldestCRNotFound        ConditionReason = "OldestCRNotFound"
	ConditionReasonOldestCRNotFoundMessage                 = "Oldest Istio custom resource could not be found"

	// Istio installation / uninstallation

	// Istio installtion is not needed.
	ConditionReasonIstioInstallNotNeeded        ConditionReason = "IstioInstallNotNeeded"
	ConditionReasonIstioInstallNotNeededMessage                 = "Istio installation is not needed"
	// Istio installation or uninstallation succeeded.
	ConditionReasonIstioInstallSucceeded        ConditionReason = "IstioInstallSucceeded"
	ConditionReasonIstioInstallSucceededMessage                 = "Istio installation succeeded"
	// Istio uninstallation succeeded.
	ConditionReasonIstioUninstallSucceeded        ConditionReason = "IstioUninstallSucceeded"
	ConditionReasonIstioUninstallSucceededMessage                 = "Istio uninstallation succeded"
	// Istio installation or uninstallation failed.
	ConditionReasonIstioInstallUninstallFailed        ConditionReason = "IstioInstallUninstallFailed"
	ConditionReasonIstioInstallUninstallFailedMessage                 = "Istio install or uninstall failed"
	// Istio Custom Resource has invalid configuration.
	ConditionReasonCustomResourceMisconfigured        ConditionReason = "IstioCustomResourceMisconfigured"
	ConditionReasonCustomResourceMisconfiguredMessage                 = "Istio custom resource has invalid configuration"
	// Istio Custom Resources are blocking Istio uninstallation.
	ConditionReasonIstioCRsDangling        ConditionReason = "IstioCustomResourcesDangling"
	ConditionReasonIstioCRsDanglingMessage                 = "Istio deletion blocked because of existing Istio custom resources"
	// Istio version update is not allowed.
	ConditionReasonIstioVersionUpdateNotAllowed        ConditionReason = "IstioVersionUpdateNotAllowed"
	ConditionReasonIstioVersionUpdateNotAllowedMessage                 = "Update to the new Istio version is not allowed"

	// Istio CRs

	// Custom resources reconciliation succeeded.
	ConditionReasonCRsReconcileSucceeded        ConditionReason = "CustomResourcesReconcileSucceeded"
	ConditionReasonCRsReconcileSucceededMessage                 = "Custom resources reconciliation succeeded"
	// Custom resources reconciliation failed.
	ConditionReasonCRsReconcileFailed        ConditionReason = "CustomResourcesReconcileFailed"
	ConditionReasonCRsReconcileFailedMessage                 = "Custom resources reconciliation failed"

	// Proxy reset

	// Proxy sidecar restart succeeded.
	ConditionReasonProxySidecarRestartSucceeded        ConditionReason = "ProxySidecarRestartSucceeded"
	ConditionReasonProxySidecarRestartSucceededMessage                 = "Proxy sidecar restart succeeded"
	// Proxy sidecar restart failed.
	ConditionReasonProxySidecarRestartFailed        ConditionReason = "ProxySidecarRestartFailed"
	ConditionReasonProxySidecarRestartFailedMessage                 = "Proxy sidecar restart failed"
	// Proxy sidecar restart partially succeeded.
	ConditionReasonProxySidecarRestartPartiallySucceeded        ConditionReason = "ProxySidecarRestartPartiallySucceeded"
	ConditionReasonProxySidecarRestartPartiallySucceededMessage                 = "Proxy sidecar restart partially succeeded"
	// Proxy sidecar manual restart is required.
	ConditionReasonProxySidecarManualRestartRequired        ConditionReason = "ProxySidecarManualRestartRequired"
	ConditionReasonProxySidecarManualRestartRequiredMessage                 = "Proxy sidecar manual restart is required for some workloads"

	// Ingress gateway

	// Istio ingress gateway restart succeeded.
	ConditionReasonIngressGatewayRestartSucceeded        ConditionReason = "IngressGatewayRestartSucceeded"
	ConditionReasonIngressGatewayRestartSucceededMessage                 = "Istio Ingress Gateway restart succeeded"
	// Istio ingress gateway restart failed.
	ConditionReasonIngressGatewayRestartFailed        ConditionReason = "IngressGatewayRestartFailed"
	ConditionReasonIngressGatewayRestartFailedMessage                 = "Istio Ingress Gateway restart failed"

	// Egress gateway

	// Istio egress gateway restart succeeded.
	ConditionReasonEgressGatewayRestartSucceeded        ConditionReason = "EgressGatewayRestartSucceeded"
	ConditionReasonEgressGatewayRestartSucceededMessage                 = "Istio Egress Gateway restart succeeded"
	// Istio egress gateway restart failed.
	ConditionReasonEgressGatewayRestartFailed        ConditionReason = "EgressGatewayRestartFailed"
	ConditionReasonEgressGatewayRestartFailedMessage                 = "Istio Egress Gateway restart failed"

	// User resource

	// Resource targeting Istio Ingress Gateway found.
	ConditionReasonIngressTargetingUserResourceFound        ConditionReason = "IngressTargetingUserResourceFound"
	ConditionReasonIngressTargetingUserResourceFoundMessage                 = "Resource targeting Istio Ingress Gateway found"
	// No resources targeting Istio Ingress Gateway found.
	ConditionReasonIngressTargetingUserResourceNotFound        ConditionReason = "IngressTargetingUserResourceNotFound"
	ConditionReasonIngressTargetingUserResourceNotFoundMessage                 = "Resources targeting Istio Ingress Gateway not found"
	// Resource targeting Istio Ingress Gateway detection failed.
	ConditionReasonIngressTargetingUserResourceDetectionFailed        ConditionReason = "IngressTargetingUserResourceDetectionFailed"
	ConditionReasonIngressTargetingUserResourceDetectionFailedMessage                 = "Resource targeting Istio Ingress Gateway detection failed"
)

type ReasonWithMessage struct {
	Reason  ConditionReason
	Message string
}

// IstioSpec describes the desired specification for installing or updating Istio.
type IstioSpec struct {
	// Defines configuration of the Istio installation.
	// +kubebuilder:validation:Optional
	Config Config `json:"config,omitempty"`
	// Defines configuration of Istio components.
	// +kubebuilder:validation:Optional
	Components *Components `json:"components,omitempty"`
	// Defines experimental configuration options.
	// +kubebuilder:validation:Optional
	Experimental *Experimental `json:"experimental,omitempty"`
	// Enables compatibility mode for Istio installation.
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

	// Spec defines the desired state of the Istio installation.
	Spec IstioSpec `json:"spec,omitempty"`
	// Status represents the current state of the Istio installation.
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
	// Description of Istio status.
	Description string `json:"description,omitempty"`
}

//nolint:gochecknoinits // this is a scaffolded file. TODO: remove init function
func init() {
	SchemeBuilder.Register(&Istio{}, &IstioList{})
}
