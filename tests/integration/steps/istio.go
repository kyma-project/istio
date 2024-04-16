package steps

import (
	"context"
	"fmt"
	"strings"

	apinetworkingv1alpha3 "istio.io/api/networking/v1alpha3"
	apisecurityv1beta1 "istio.io/api/security/v1beta1"
	v1beta1 "istio.io/api/type/v1beta1"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	securityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/avast/retry-go"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	crds "github.com/kyma-project/istio/operator/tests/integration/pkg/crds"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	defaultIstioNamespace string = "istio-system"
)

func IstioCRDsBePresentOnCluster(ctx context.Context, should string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	shouldHave := true
	if should != "should" {
		shouldHave = false
	}
	lister, err := crds.NewCRDListerFromFile(k8sClient, "steps/istio_crd_list.yaml")
	if err != nil {
		return err
	}
	return retry.Do(func() error {
		wrongs, err := lister.CheckForCRDs(context.TODO(), shouldHave)
		if err != nil {
			return err
		}
		if len(wrongs) > 0 {
			if shouldHave {
				return fmt.Errorf("CRDs %s where not present", strings.Join(wrongs, ";"))
			} else {
				return fmt.Errorf("CRDs %s where present", strings.Join(wrongs, ";"))
			}
		}
		return nil
	}, testcontext.GetRetryOpts()...)
}

func SetIstioInjection(ctx context.Context, enabled, namespace string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	var ns corev1.Namespace
	return retry.Do(func() error {
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: namespace}, &ns)
		if err != nil {
			return err
		}
		ns.Labels["istio-injection"] = enabled
		return k8sClient.Update(context.TODO(), &ns)
	}, testcontext.GetRetryOpts()...)
}

func IstioComponentHasResourcesSetToCpuAndMemory(ctx context.Context, component, resourceType, cpu, memory string) error {
	switch component {
	case "ingress-gateway":
		return DeploymentHasPodWithContainerResourcesSetToCpuAndMemory(ctx, "istio-ingressgateway", defaultIstioNamespace, "istio-proxy", resourceType, cpu, memory)
	case "pilot":
		return DeploymentHasPodWithContainerResourcesSetToCpuAndMemory(ctx, "istiod", defaultIstioNamespace, "discovery", resourceType, cpu, memory)
	}

	return fmt.Errorf("resources for component %s are not implemented", component)
}

func UninstallIstio(ctx context.Context) error {
	istioClient := istio.NewIstioClient()
	return istioClient.Uninstall(ctx)
}

// CreateIstioGateway creates an Istio Gateway with http port 80 configured and any host
func CreateIstioGateway(ctx context.Context, name, namespace string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	gateway := &networkingv1alpha3.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: apinetworkingv1alpha3.Gateway{
			Selector: map[string]string{
				"app":   "istio-ingressgateway",
				"istio": "ingressgateway",
			},
			Servers: []*apinetworkingv1alpha3.Server{
				{
					Port: &apinetworkingv1alpha3.Port{
						Number:   80,
						Protocol: "HTTP",
						Name:     "http",
					},
					Hosts: []string{
						"*",
					},
				},
			},
		},
	}

	err = retry.Do(func() error {
		err := k8sClient.Create(context.TODO(), gateway)
		if err != nil {
			return err
		}
		ctx = testcontext.AddCreatedTestObjectInContext(ctx, gateway)
		return nil
	}, testcontext.GetRetryOpts()...)

	return ctx, err
}

// CreateVirtualService creates a VirtualService that exposes the given service on any host
func CreateVirtualService(ctx context.Context, name, exposedService, gateway, namespace string) (context.Context, error) {
	return CreateVirtualServiceWithPort(ctx, name, exposedService, 8000, gateway, namespace)
}

// CreateVirtualServiceWithPort creates a VirtualService that exposes the given service and port on any host
func CreateVirtualServiceWithPort(ctx context.Context, name, exposedService string, exposedPort int, gateway, namespace string) (context.Context, error) {

	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	vs := &networkingv1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: apinetworkingv1alpha3.VirtualService{
			Hosts: []string{
				"*",
			},
			Gateways: []string{
				gateway,
			},
			Http: []*apinetworkingv1alpha3.HTTPRoute{
				{
					Match: []*apinetworkingv1alpha3.HTTPMatchRequest{
						{
							Uri: &apinetworkingv1alpha3.StringMatch{
								MatchType: &apinetworkingv1alpha3.StringMatch_Prefix{
									Prefix: "/",
								},
							},
						},
					},
					Route: []*apinetworkingv1alpha3.HTTPRouteDestination{
						{
							Destination: &apinetworkingv1alpha3.Destination{
								Host: exposedService,
								Port: &apinetworkingv1alpha3.PortSelector{
									Number: uint32(exposedPort),
								},
							},
						},
					},
				},
			},
		},
	}

	err = retry.Do(func() error {
		err := k8sClient.Create(context.TODO(), vs)
		if err != nil {
			return err
		}
		ctx = testcontext.AddCreatedTestObjectInContext(ctx, vs)
		return nil
	}, testcontext.GetRetryOpts()...)

	return ctx, err
}

func CreateAuthorizationPolicyExtAuthz(ctx context.Context, name, namespace, selector, provider, operation string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	ap := &securityv1beta1.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: apisecurityv1beta1.AuthorizationPolicy{
			Selector: &v1beta1.WorkloadSelector{
				MatchLabels: map[string]string{"app": selector},
			},
			Action: apisecurityv1beta1.AuthorizationPolicy_CUSTOM,
			ActionDetail: &apisecurityv1beta1.AuthorizationPolicy_Provider{
				Provider: &apisecurityv1beta1.AuthorizationPolicy_ExtensionProvider{
					Name: provider,
				},
			},
			Rules: []*apisecurityv1beta1.Rule{
				{
					To: []*apisecurityv1beta1.Rule_To{
						{
							Operation: &apisecurityv1beta1.Operation{
								Paths: []string{operation},
							},
						},
					},
				},
			},
		},
	}

	err = retry.Do(func() error {
		err := k8sClient.Create(context.TODO(), ap)
		if err != nil {
			return err
		}
		ctx = testcontext.AddCreatedTestObjectInContext(ctx, ap)
		return nil
	}, testcontext.GetRetryOpts()...)

	return ctx, err
}
