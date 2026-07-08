package aws

import (
	"context"

	"github.com/kyma-project/istio/operator/internal/clusterconfig/factory"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ProxyProtocolAnnotation   = "service.beta.kubernetes.io/aws-load-balancer-proxy-protocol"
	ProxyProtocolValue        = "*"
	ConnIdleTimeoutAnnotation = "service.beta.kubernetes.io/aws-load-balancer-connection-idle-timeout"
	ConnIdleTimeoutValue      = "4000"
	LBTypeAnnotation          = "service.beta.kubernetes.io/aws-load-balancer-type"
	NLBType                   = "nlb"
	ExternalType              = "external"
	NlbTargetTypeAnnotation   = "service.beta.kubernetes.io/aws-load-balancer-nlb-target-type"
	NlbTargetTypeInstance     = "instance"
	SchemeAnnotation          = "service.beta.kubernetes.io/aws-load-balancer-scheme"
	InternetFacingScheme      = "internet-facing"

	istioIngressNamespace   = "istio-system"
	istioIngressServiceName = "istio-ingressgateway"
	loadBalancerType        = "nlb"

	elbCmName      = "elb-deprecated"
	elbCmNamespace = "istio-system"
)

type IPStackType string

const (
	IPv4      IPStackType = "ipv4"
	DualStack IPStackType = "dualstack"
)

type Type string

const (
	NLB Type = "nlb"
	ELB Type = "elb"
)

type LB struct {
	stackType IPStackType
	lbType    Type
}

func (s *LB) Annotations() map[string]string {

	if s.lbType == NLB {
		// AWS LBC
		// In case of running with DualStack IP family,
		// The annotation "service.beta.kubernetes.io/aws-load-balancer-ip-address-type=dualstack" is required.
		// AWS LB Controller-style annotations (type=external, ip-address-type=dualstack)
		// are intentionally NOT emitted here. On Gardener IPv6/dual-stack clusters those
		// are added by Gardener's shoot-service mutating webhook. See:
		// https://github.com/gardener/gardener-extension-provider-aws/blob/master/pkg/webhook/shootservice/mutator.go
		// Switching IPv4 clusters to LB type=external is a potential follow up.
		if s.stackType == DualStack {
			return map[string]string{
				LBTypeAnnotation:        ExternalType,
				SchemeAnnotation:        InternetFacingScheme,
				NlbTargetTypeAnnotation: NlbTargetTypeInstance,
				ProxyProtocolAnnotation: ProxyProtocolValue,
			}
		}
		// in-tree AWS CCM
		if s.stackType == IPv4 {
			return map[string]string{
				LBTypeAnnotation:        NLBType,
				SchemeAnnotation:        InternetFacingScheme,
				NlbTargetTypeAnnotation: NlbTargetTypeInstance,
			}
		}
	}

	// ELB
	return map[string]string{
		ProxyProtocolAnnotation:   ProxyProtocolValue,
		ConnIdleTimeoutAnnotation: ConnIdleTimeoutValue,
	}
}

type Factory struct {
	inputs factory.Inputs
	lb     *LB
}

// NewFactory builds an AWS Factory. The cluster reads needed to decide
// between NLB and ELB happen here so that Make* methods stay pure.
func NewFactory(ctx context.Context, k8sClient client.Client, in factory.Inputs) (*Factory, error) {
	lb := &LB{}

	useNLB, err := shouldUseNLB(ctx, k8sClient)
	if err != nil {
		return nil, err
	}

	if useNLB {
		lb.lbType = NLB
	} else {
		lb.lbType = ELB
	}

	if in.DualStackEnabled {
		lb.stackType = DualStack
	} else {
		lb.stackType = IPv4
	}

	return &Factory{inputs: in, lb: lb}, nil
}

func (f *Factory) LB() factory.LB {
	return f.lb
}

func (f *Factory) CNI() factory.CNI { return nil }

func (f *Factory) NeedsProxyProtocol() bool {
	if f.lb.lbType == ELB {
		return true
	}
	switch f.lb.stackType {
	case IPv4:
		return false
	case DualStack:
		return true
	default:
		return false
	}
}

func (f *Factory) DualStackEnabled() bool { return f.inputs.DualStackEnabled }

func shouldUseNLB(ctx context.Context, k8sClient client.Client) (bool, error) {
	var elbDeprecated corev1.ConfigMap
	err := k8sClient.Get(ctx, client.ObjectKey{Namespace: elbCmNamespace, Name: elbCmName}, &elbDeprecated)
	if err != nil {
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}

	var ingressGatewaySvc corev1.Service
	err = k8sClient.Get(ctx, client.ObjectKey{Namespace: istioIngressNamespace, Name: istioIngressServiceName}, &ingressGatewaySvc)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	if value, ok := ingressGatewaySvc.Annotations[LBTypeAnnotation]; ok && value == loadBalancerType {
		return true, nil
	}

	return false, nil
}
