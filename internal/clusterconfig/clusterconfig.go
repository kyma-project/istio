package clusterconfig

import (
	"context"
	"regexp"
	"strings"

	"github.com/kyma-project/istio/operator/internal/clusterconfig/factory"
	"github.com/kyma-project/istio/operator/internal/clusterconfig/factory/aws"
	"github.com/kyma-project/istio/operator/internal/clusterconfig/factory/gke"
	"github.com/kyma-project/istio/operator/internal/clusterconfig/factory/k3d"
	"github.com/kyma-project/istio/operator/internal/clusterconfig/factory/openstack"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/imdario/mergo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	ProductionClusterCPUThreshold      int64 = 5
	ProductionClusterMemoryThresholdGi int64 = 10
)

type ClusterSize int

const (
	UnknownSize ClusterSize = iota
	Evaluation
	Production
)

func (s ClusterSize) String() string {
	switch s {
	case Evaluation:
		return "Evaluation"
	case Production:
		return "Production"
	case UnknownSize:
		fallthrough
	default:
		return "Unknown"
	}
}

// EvaluateClusterSize counts the entire capacity of cpu and memory in the cluster and returns Evaluation
// if the total capacity of the resources is lower than ProductionClusterCPUThreshold or ProductionClusterMemoryThresholdGi.
func EvaluateClusterSize(ctx context.Context, k8sClient client.Client) (ClusterSize, error) {
	nodeList := corev1.NodeList{}
	err := k8sClient.List(ctx, &nodeList)
	if err != nil {
		return UnknownSize, err
	}

	var cpuCapacity resource.Quantity
	var memoryCapacity resource.Quantity
	for _, node := range nodeList.Items {
		nodeCPUCap := node.Status.Capacity.Cpu()
		if nodeCPUCap != nil {
			cpuCapacity.Add(*nodeCPUCap)
		}
		nodeMemoryCap := node.Status.Capacity.Memory()
		if nodeMemoryCap != nil {
			memoryCapacity.Add(*nodeMemoryCap)
		}
	}
	if cpuCapacity.Cmp(*resource.NewQuantity(ProductionClusterCPUThreshold, resource.DecimalSI)) == -1 ||
		memoryCapacity.Cmp(*resource.NewScaledQuantity(ProductionClusterMemoryThresholdGi, resource.Giga)) == -1 {
		return Evaluation, nil
	}
	return Production, nil
}

type ClusterProvider int

const (
	Unknown ClusterProvider = iota
	K3d
	GKE
	AWS
	Openstack
)

func (c ClusterProvider) String() string {
	switch c {
	case K3d:
		return "K3d"
	case GKE:
		return "GKE"
	case AWS:
		return "AWS"
	case Openstack:
		return "Openstack"
	case Unknown:
		fallthrough
	default:
		return "Unknown"
	}
}

type ClusterConfiguration map[string]interface{}

// BuildFactory discovers the cluster provider and dual-stack/Garden flags,
// then constructs the matching Factory. This is the single place where
// LB/CNI strategy selection happens for a reconcile loop.
func BuildFactory(ctx context.Context, k8sClient client.Client) (factory.Factory, error) {
	provider, err := DiscoverClusterProvider(ctx, k8sClient)
	if err != nil {
		return nil, err
	}
	ctrl.Log.Info("Discovered cluster provider", "provider", provider)

	usesGardenOS, err := hasGardenOS(ctx, k8sClient)
	if err != nil {
		return nil, err
	}

	dualStackEnabled, err := IsDualStackEnabled(ctx, k8sClient)
	if err != nil {
		return nil, err
	}

	in := factory.Inputs{DualStackEnabled: dualStackEnabled, UsesGardenOS: usesGardenOS}

	switch provider {
	case AWS:
		return aws.NewFactory(ctx, k8sClient, in)
	case K3d:
		return k3d.NewFactory(in), nil
	case GKE:
		return gke.NewFactory(in), nil
	case Openstack:
		return openstack.NewFactory(in), nil
	default:
		return factory.DefaultFactory(in), nil
	}
}

// ClusterConfigurationFromFactory renders the Istio operator overrides
// from a pre-built Factory.
func ClusterConfigurationFromFactory(s factory.Factory) ClusterConfiguration {
	return clusterConfiguration(s)
}

func hasGardenOS(ctx context.Context, k8sClient client.Client) (bool, error) {
	nodeList := corev1.NodeList{}
	err := k8sClient.List(ctx, &nodeList)
	if err != nil {
		return false, err
	}

	for _, node := range nodeList.Items {
		if strings.Contains(node.Status.NodeInfo.OSImage, "Garden") {
			return true, nil
		}
	}
	return false, nil
}

var (
	regexpMatchK3D = regexp.MustCompile(`^v\d+\.\d+\.\d+\+k3s\d+$`)
	regexpMatchGKE = regexp.MustCompile(`^v\d+\.\d+\.\d+-gke\.\d+$`)
)

func DiscoverClusterProvider(ctx context.Context, k8sClient client.Client) (ClusterProvider, error) {
	nodeList := corev1.NodeList{}
	err := k8sClient.List(ctx, &nodeList)
	if err != nil {
		return Unknown, err
	}

	for _, node := range nodeList.Items {
		providerID := strings.ToLower(node.Spec.ProviderID)
		switch {
		// For GKE and K3D matching on providerID should be considered.
		// k3d -> providerID: "k3s://"
		// GKE -> providerID: "gce://"
		case regexpMatchGKE.MatchString(node.Status.NodeInfo.KubeletVersion):
			return GKE, nil
		case regexpMatchK3D.MatchString(node.Status.NodeInfo.KubeletVersion):
			return K3d, nil
		case strings.HasPrefix(providerID, "aws://"):
			return AWS, nil
		case strings.HasPrefix(providerID, "openstack://"):
			return Openstack, nil
		}
	}

	return Unknown, nil
}

func clusterConfiguration(s factory.Factory) ClusterConfiguration {
	values := map[string]interface{}{}

	if s == nil {
		return ClusterConfiguration{
			"spec": map[string]interface{}{
				"values": values,
			},
		}
	}

	if cni := s.CNI(); cni != nil {
		if v := cni.GetCNIValues(); v != nil {
			values["cni"] = v
		}
	}

	if lb := s.LB(); lb != nil {
		if ann := lb.GetLBAnnotations(); ann != nil {
			values["gateways"] = map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"serviceAnnotations": ann,
				},
			}
		}
	}

	return ClusterConfiguration{
		"spec": map[string]interface{}{
			"values": values,
		},
	}
}

func MergeOverrides(template []byte, overrides ClusterConfiguration) ([]byte, error) {
	var templateMap map[string]interface{}
	err := yaml.Unmarshal(template, &templateMap)
	if err != nil {
		return nil, err
	}

	err = mergo.Merge(&templateMap, map[string]interface{}(overrides), mergo.WithOverride)
	if err != nil {
		return nil, err
	}

	return yaml.Marshal(templateMap)
}

func IsDualStackEnabled(ctx context.Context, sClient client.Client) (bool, error) {
	if !isExperimentalEnabled() {
		return false, nil
	}
	var kymaProvisioningInfo corev1.ConfigMap
	err := sClient.Get(ctx, client.ObjectKey{Namespace: "kyma-system", Name: "kyma-provisioning-info"}, &kymaProvisioningInfo)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	if cmDetails, ok := kymaProvisioningInfo.Data["details"]; ok {
		var detailsMap map[string]interface{}
		err = yaml.Unmarshal([]byte(cmDetails), &detailsMap)
		if err != nil {
			return false, err
		}

		if networkDetails, ok := detailsMap["networkDetails"]; ok {
			networkDetailsMap, ok := networkDetails.(map[string]interface{})
			if !ok {
				return false, nil
			}

			dualStackIPEnabled, ok := networkDetailsMap["dualStackIPEnabled"].(bool)
			if !ok {
				return false, nil
			}

			return dualStackIPEnabled, nil
		}
	}
	return false, nil
}
