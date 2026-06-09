package aws

import (
	"context"

	"github.com/kyma-project/istio/operator/internal/clusterconfig/strategy"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IpAddressTypeAnnotation = "service.beta.kubernetes.io/aws-load-balancer-ip-address-type"
	IpAddressTypeDualStack  = "dualstack"
	LBTypeAnnotation        = "service.beta.kubernetes.io/aws-load-balancer-type"
	ExternalType            = "external"
	NLBType                 = "nlb"
	NlbTargetTypeAnnotation = "service.beta.kubernetes.io/aws-load-balancer-nlb-target-type"
	NlbTargetTypeInstance   = "instance"
	SchemeAnnotation        = "service.beta.kubernetes.io/aws-load-balancer-scheme"
	InternetFacingScheme    = "internet-facing"

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
	k8sClient client.Client
	ctx       context.Context
}

func NewStrategy(ctx context.Context, k8sClient client.Client, dualStackEnabled bool) (*strategy.Strategy, error) {
	lb := &LB{
		ctx:       ctx,
		k8sClient: k8sClient,
	}

	useNLB, err := lb.shouldUseNLB()
	if err != nil {
		return nil, err
	}

	if useNLB {
		lb.lbType = NLB
	} else {
		lb.lbType = ELB
	}

	if dualStackEnabled {
		lb.stackType = DualStack
	} else {
		lb.stackType = IPv4
	}

	return &strategy.Strategy{
		LB: lb,
	}, nil
}

func (s *LB) GetLBAnnotations() (map[string]string, bool) {
	if s.lbType == ELB {
		return nil, false
	}
	switch s.stackType {
	case IPv4:
		return map[string]string{
			LBTypeAnnotation:        NLBType,
			SchemeAnnotation:        InternetFacingScheme,
			NlbTargetTypeAnnotation: NlbTargetTypeInstance,
		}, true
	case DualStack:
		return map[string]string{
			LBTypeAnnotation:        ExternalType,
			SchemeAnnotation:        InternetFacingScheme,
			NlbTargetTypeAnnotation: NlbTargetTypeInstance,
			IpAddressTypeAnnotation: IpAddressTypeDualStack,
		}, true
	default:
		return map[string]string{
			LBTypeAnnotation:        NLBType,
			SchemeAnnotation:        InternetFacingScheme,
			NlbTargetTypeAnnotation: NlbTargetTypeInstance,
		}, true
	}
}

func (s *LB) RequiresProxyProtocolEnvoyFilter() bool {
	if s.lbType == ELB {
		return true
	}

	switch s.stackType {
	case IPv4:
		return false
	case DualStack:
		return true
	default:
		return false
	}
}

func (s *LB) shouldUseNLB() (bool, error) {
	var elbDeprecated corev1.ConfigMap
	err := s.k8sClient.Get(s.ctx, client.ObjectKey{Namespace: elbCmNamespace, Name: elbCmName}, &elbDeprecated)
	if err != nil {
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}

	var ingressGatewaySvc corev1.Service
	err = s.k8sClient.Get(s.ctx, client.ObjectKey{Namespace: istioIngressNamespace, Name: istioIngressServiceName}, &ingressGatewaySvc)
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
