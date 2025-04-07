package v1alpha2

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	ConditionReasonReconcileUnknown:   {Type: ConditionTypeReady, Status: metav1.ConditionUnknown, Message: ConditionReasonReconcileUnknownMessage},
	ConditionReasonReconcileRequeued:  {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonReconcileRequeuedMessage},
	ConditionReasonValidationFailed:   {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonValidationFailedMessage},
	ConditionReasonOlderCRExists:      {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonOlderCRExistsMessage},
	ConditionReasonOldestCRNotFound:   {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonOldestCRNotFoundMessage},

	ConditionReasonIstioInstallNotNeeded:        {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIstioInstallNotNeededMessage},
	ConditionReasonIstioInstallSucceeded:        {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIstioInstallSucceededMessage},
	ConditionReasonIstioUninstallSucceeded:      {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIstioUninstallSucceededMessage},
	ConditionReasonIstioInstallUninstallFailed:  {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIstioInstallUninstallFailedMessage},
	ConditionReasonCustomResourceMisconfigured:  {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonCustomResourceMisconfiguredMessage},
	ConditionReasonIstioCRsDangling:             {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIstioCRsDanglingMessage},
	ConditionReasonIstioVersionUpdateNotAllowed: {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIstioVersionUpdateNotAllowedMessage},

	ConditionReasonCRsReconcileSucceeded: {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonCRsReconcileSucceededMessage},
	ConditionReasonCRsReconcileFailed:    {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonCRsReconcileFailedMessage},

	ConditionReasonProxySidecarRestartSucceeded:          {Type: ConditionTypeProxySidecarRestartSucceeded, Status: metav1.ConditionTrue, Message: ConditionReasonProxySidecarRestartSucceededMessage},
	ConditionReasonProxySidecarRestartFailed:             {Type: ConditionTypeProxySidecarRestartSucceeded, Status: metav1.ConditionFalse, Message: ConditionReasonProxySidecarRestartFailedMessage},
	ConditionReasonProxySidecarRestartPartiallySucceeded: {Type: ConditionTypeProxySidecarRestartSucceeded, Status: metav1.ConditionFalse, Message: ConditionReasonProxySidecarRestartPartiallySucceededMessage},
	ConditionReasonProxySidecarManualRestartRequired:     {Type: ConditionTypeProxySidecarRestartSucceeded, Status: metav1.ConditionFalse, Message: ConditionReasonProxySidecarManualRestartRequiredMessage},

	ConditionReasonIngressGatewayRestartSucceeded: {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIngressGatewayRestartSucceededMessage},
	ConditionReasonIngressGatewayRestartFailed:    {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonIngressGatewayRestartFailedMessage},

	ConditionReasonEgressGatewayRestartSucceeded: {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonEgressGatewayRestartSucceededMessage},
	ConditionReasonEgressGatewayRestartFailed:    {Type: ConditionTypeReady, Status: metav1.ConditionFalse, Message: ConditionReasonEgressGatewayRestartFailedMessage},
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
