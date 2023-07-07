package steps

import (
	"context"
	"encoding/json"
	"fmt"
	apinetworkingv1alpha3 "istio.io/api/networking/v1alpha3"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"

	"github.com/avast/retry-go"
	"github.com/cucumber/godog"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	"github.com/kyma-project/istio/operator/tests/integration/manifests"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"github.com/mitchellh/mapstructure"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultIopName      string = "installed-state-default-operator"
	defaultIopNamespace string = "istio-system"
	crdListPath         string = "manifests/crd_list.yaml"
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
	lister, err := manifests.NewCRDListerFromFile(k8sClient, crdListPath)
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

func EnableIstioInjection(ctx context.Context, namespace string) error {
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
		ns.Labels["istio-injection"] = "enabled"
		return k8sClient.Update(context.TODO(), &ns)
	}, testcontext.GetRetryOpts()...)
}

func IstioComponentHasResourcesSetToCpuAndMemory(ctx context.Context, component, resourceType, cpu, memory string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	operator, err := getIstioOperatorFromCluster(k8sClient)
	if err != nil {
		return err
	}
	resources, err := getResourcesForComponent(operator, component, resourceType)
	if err != nil {
		return err
	}

	if resources.Cpu != cpu {
		return fmt.Errorf("cpu %s for component %s wasn't expected; expected=%s got=%s", resourceType, component, cpu, resources.Cpu)
	}

	if resources.Memory != memory {
		return fmt.Errorf("memory %s for component %s wasn't expected; expected=%s got=%s", resourceType, component, memory, resources.Memory)
	}

	return nil
}

func IstioServiceHasAnnotation(ctx context.Context, serviceName, annotationName, clusterFlavour string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}
	istioService := corev1.Service{}
	err = k8sClient.Get(context.TODO(), types.NamespacedName{Name: serviceName, Namespace: defaultIopNamespace}, &istioService)
	if err != nil {
		return fmt.Errorf("default Istio Gateway Service wasn't found err=%s", err)
	}
	flavour, err := clusterconfig.DiscoverClusterFlavour(ctx, k8sClient)
	if err != nil {
		return fmt.Errorf("unable to determine cluster flavour err=%s", err)
	}
	annotationValue, found := istioService.Annotations[annotationName]
	if !found && flavour.String() == clusterFlavour {
		return fmt.Errorf("expected annotation '%s' on Istio Gateway Service for %s cluster (%s) wasn't found", annotationName, clusterFlavour, flavour)
	}
	if found && flavour.String() != clusterFlavour && annotationValue != clusterconfig.LocalKymaDomain {
		return fmt.Errorf("unexpected annotation '%s' on Istio Gateway Service for non-%s cluster (%s) was found", annotationName, clusterFlavour, flavour)
	}
	return nil
}

func UninstallIstio(ctx context.Context) error {
	istioClient := istio.NewIstioClient()
	return istioClient.Uninstall(ctx)
}

func getIstioOperatorFromCluster(k8sClient client.Client) (*istioOperator.IstioOperator, error) {

	iop := istioOperator.IstioOperator{}

	err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: defaultIopName, Namespace: defaultIopNamespace}, &iop)
	if err != nil {
		return nil, fmt.Errorf("default Kyma IstioOperator CR wasn't found err=%s", err)
	}

	return &iop, nil
}

type ResourceStruct struct {
	Cpu    string
	Memory string
}

func getResourcesForComponent(operator *istioOperator.IstioOperator, component, resourceType string) (*ResourceStruct, error) {
	res := ResourceStruct{}

	switch component {
	case "proxy_init":
		fallthrough
	case "proxy":
		jsonResources, err := json.Marshal(operator.Spec.Values.Fields["global"].GetStructValue().Fields[component].GetStructValue().
			Fields["resources"].GetStructValue().Fields[resourceType].GetStructValue())

		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(jsonResources, &res)
		if err != nil {
			return nil, err
		}
		return &res, nil
	case "ingress-gateway":
		if resourceType == "limits" {
			err := mapstructure.Decode(operator.Spec.Components.IngressGateways[0].K8S.Resources.Limits, &res)
			if err != nil {
				return nil, err
			}
		} else {
			err := mapstructure.Decode(operator.Spec.Components.IngressGateways[0].K8S.Resources.Requests, &res)
			if err != nil {
				return nil, err
			}
		}

		return &res, nil
	case "egress-gateway":
		if resourceType == "limits" {
			err := mapstructure.Decode(operator.Spec.Components.EgressGateways[0].K8S.Resources.Limits, &res)
			if err != nil {
				return nil, err
			}
		} else {
			err := mapstructure.Decode(operator.Spec.Components.EgressGateways[0].K8S.Resources.Requests, &res)
			if err != nil {
				return nil, err
			}
		}
		return &res, nil
	case "pilot":
		if resourceType == "limits" {
			err := mapstructure.Decode(operator.Spec.Components.Pilot.K8S.Resources.Limits, &res)
			if err != nil {
				return nil, err
			}
		} else {
			err := mapstructure.Decode(operator.Spec.Components.Pilot.K8S.Resources.Requests, &res)
			if err != nil {
				return nil, err
			}
		}
		return &res, nil
	default:
		return nil, godog.ErrPending
	}
}

// CreateIstioGateway creates an Istio Gateway with http port 80 configured and any host
func CreateIstioGateway(ctx context.Context, name, namespace string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

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
									Number: 8000,
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
