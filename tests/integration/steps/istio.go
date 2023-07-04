package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/avast/retry-go"
	"github.com/cucumber/godog"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	"github.com/kyma-project/istio/operator/tests/integration/manifests"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"github.com/mitchellh/mapstructure"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
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

func IstioDeploymentHasAnnotation(ctx context.Context, deploymentName, annotationName, clusterFlavour string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}
	istioGWDeployment := appsv1.Deployment{}
	err = k8sClient.Get(context.TODO(), types.NamespacedName{Name: deploymentName, Namespace: defaultIopNamespace}, &istioGWDeployment)
	if err != nil {
		return fmt.Errorf("default Istio Gateway Deployment wasn't found err=%s", err)
	}
	flavour, err := clusterconfig.DiscoverClusterFlavour(ctx, k8sClient)
	if err != nil {
		return fmt.Errorf("unable to determine cluster flavour err=%s", err)
	}
	_, found := istioGWDeployment.Spec.Template.Annotations[annotationName]
	if !found && flavour.String() == clusterFlavour {
		return fmt.Errorf("expected annotation '%s' on Istio Gateway Deployment template for %s cluster wasn't found", annotationName, clusterFlavour)
	}
	if found && flavour.String() != clusterFlavour {
		return fmt.Errorf("unexpected annotation '%s' on Istio Gateway Deployment template for non-%s cluster was found", annotationName, clusterFlavour)
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
