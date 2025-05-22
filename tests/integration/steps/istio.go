package steps

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	apinetworkingv1 "istio.io/api/networking/v1"
	apisecurityv1 "istio.io/api/security/v1"
	apiv1beta1 "istio.io/api/type/v1beta1"
	networkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	securityv1 "istio.io/client-go/pkg/apis/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/istio/operator/tests/testcontext"

	"github.com/avast/retry-go"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	crds "github.com/kyma-project/istio/operator/tests/integration/pkg/crds"
)

const (
	defaultIstioNamespace string = "istio-system"
)

func IstioCRDsBePresentOnCluster(ctx context.Context, should string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	shouldHave := should == "should"

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
			}
			return fmt.Errorf("CRDs %s where present", strings.Join(wrongs, ";"))
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

func IstioComponentHasResourcesSetToCPUAndMemory(ctx context.Context, component, resourceType, cpu, memory string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	resources, err := getResourcesForIstioComponent(k8sClient, component, resourceType)
	if err != nil {
		return ctx, err
	}

	if err := assertResources(*resources, cpu, memory); err != nil {
		return ctx, errors.Wrap(err, fmt.Sprintf("assert %s resources of Istio component %s ", resourceType, component))
	}
	return ctx, nil
}

func UninstallIstio(ctx context.Context) error {
	istioClient := istio.NewIstioClient()
	return istioClient.Uninstall(ctx)
}

func getResourcesForIstioComponent(k8sClient client.Client, component, resourceType string) (*resourceStruct, error) {
	res := resourceStruct{}

	switch component {
	case "ingress-gateway":
		var igDeployment appsv1.Deployment
		err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "istio-ingressgateway", Namespace: defaultIstioNamespace}, &igDeployment)
		if err != nil {
			return nil, err
		}

		if resourceType == "limits" {
			res.Memory = *igDeployment.Spec.Template.Spec.Containers[0].Resources.Limits.Memory()
			res.CPU = *igDeployment.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu()
		} else {
			res.Memory = *igDeployment.Spec.Template.Spec.Containers[0].Resources.Requests.Memory()
			res.CPU = *igDeployment.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu()
		}
		return &res, nil

	case "egress-gateway":
		var egDeployment appsv1.Deployment
		err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "istio-egressgateway", Namespace: defaultIstioNamespace}, &egDeployment)
		if err != nil {
			return nil, err
		}

		if resourceType == "limits" {
			res.Memory = *egDeployment.Spec.Template.Spec.Containers[0].Resources.Limits.Memory()
			res.CPU = *egDeployment.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu()
		} else {
			res.Memory = *egDeployment.Spec.Template.Spec.Containers[0].Resources.Requests.Memory()
			res.CPU = *egDeployment.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu()
		}
		return &res, nil

	case "pilot":
		var idDeployment appsv1.Deployment
		err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "istiod", Namespace: defaultIstioNamespace}, &idDeployment)
		if err != nil {
			return nil, err
		}

		if resourceType == "limits" {
			res.Memory = *idDeployment.Spec.Template.Spec.Containers[0].Resources.Limits.Memory()
			res.CPU = *idDeployment.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu()
		} else {
			res.Memory = *idDeployment.Spec.Template.Spec.Containers[0].Resources.Requests.Memory()
			res.CPU = *idDeployment.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu()
		}
		return &res, nil

	default:
		return nil, fmt.Errorf("resources for component %s are not implemented", component)
	}
}

// CreateIstioGateway creates an Istio Gateway with http port 80 configured and any host.
func CreateIstioGateway(ctx context.Context, name, namespace string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	ctx, err = CreateDummySecretWithCert(ctx, "dummy-cert", "istio-system")
	if err != nil {
		return ctx, err
	}

	gateway := &networkingv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: apinetworkingv1.Gateway{
			Selector: map[string]string{
				"app":   "istio-ingressgateway",
				"istio": "ingressgateway",
			},
			Servers: []*apinetworkingv1.Server{
				{
					Port: &apinetworkingv1.Port{
						Number:   80,
						Protocol: "HTTP",
						Name:     "http",
					},
					Hosts: []string{
						"*",
					},
				},
				{
					Port: &apinetworkingv1.Port{
						Number:   443,
						Protocol: "HTTPS",
						Name:     "https",
					},
					Hosts: []string{
						"*",
					},
					Tls: &apinetworkingv1.ServerTLSSettings{
						Mode:           apinetworkingv1.ServerTLSSettings_SIMPLE,
						CredentialName: "dummy-cert",
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
		testcontext.AddCreatedTestObjectInContext(ctx, gateway)
		return nil
	}, testcontext.GetRetryOpts()...)

	return ctx, err
}

// CreateVirtualService creates a VirtualService that exposes the given service on any host.
func CreateVirtualService(ctx context.Context, name, exposedService, gateway, namespace string) (context.Context, error) {
	return CreateVirtualServiceWithPort(ctx, name, exposedService, 8000, gateway, namespace)
}

// CreateVirtualServiceWithPort creates a VirtualService that exposes the given service and port on any host.
func CreateVirtualServiceWithPort(ctx context.Context, name, exposedService string, exposedPort int, gateway, namespace string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	vs := &networkingv1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: apinetworkingv1.VirtualService{
			Hosts: []string{
				"*",
			},
			Gateways: []string{
				gateway,
			},
			Http: []*apinetworkingv1.HTTPRoute{
				{
					Match: []*apinetworkingv1.HTTPMatchRequest{
						{
							Uri: &apinetworkingv1.StringMatch{
								MatchType: &apinetworkingv1.StringMatch_Prefix{
									Prefix: "/",
								},
							},
						},
					},
					Route: []*apinetworkingv1.HTTPRouteDestination{
						{
							Destination: &apinetworkingv1.Destination{
								Host: exposedService,
								Port: &apinetworkingv1.PortSelector{
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
		testcontext.AddCreatedTestObjectInContext(ctx, vs)
		return nil
	}, testcontext.GetRetryOpts()...)

	return ctx, err
}

func CreateAuthorizationPolicyExtAuthz(ctx context.Context, name, namespace, selector, provider, operation string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	ap := &securityv1.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: apisecurityv1.AuthorizationPolicy{
			Selector: &apiv1beta1.WorkloadSelector{
				MatchLabels: map[string]string{"app": selector},
			},
			Action: apisecurityv1.AuthorizationPolicy_CUSTOM,
			ActionDetail: &apisecurityv1.AuthorizationPolicy_Provider{
				Provider: &apisecurityv1.AuthorizationPolicy_ExtensionProvider{
					Name: provider,
				},
			},
			Rules: []*apisecurityv1.Rule{
				{
					To: []*apisecurityv1.Rule_To{
						{
							Operation: &apisecurityv1.Operation{
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
		testcontext.AddCreatedTestObjectInContext(ctx, ap)
		return nil
	}, testcontext.GetRetryOpts()...)

	return ctx, err
}
