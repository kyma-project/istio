package clusterconfig

import (
	"context"
	"k8s.io/apimachinery/pkg/api/resource"
	"regexp"

	"github.com/imdario/mergo"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type ClusterSize int

const (
	UnknownSize ClusterSize = iota
	Evaluation
	Production

	ProductionClusterCpuThreshold      int64 = 4
	ProductionClusterMemoryThresholdGi int64 = 10
)

func (s ClusterSize) String() string {
	switch s {
	case Evaluation:
		return "Evaluation"
	case Production:
		return "Production"
	default:
		return "Unknown"
	}
}

func EvaluateClusterSize(ctx context.Context, k8sclient client.Client) (ClusterSize, error) {
	nodeList := corev1.NodeList{}
	err := k8sclient.List(ctx, &nodeList)
	if err != nil {
		return UnknownSize, err
	}

	var cpuCapacity resource.Quantity
	var memoryCapacity resource.Quantity
	for _, node := range nodeList.Items {
		nodeCpuCap := node.Status.Capacity.Cpu()
		if nodeCpuCap != nil {
			cpuCapacity.Add(*nodeCpuCap)
		}
		nodeMemoryCap := node.Status.Capacity.Memory()
		if nodeMemoryCap != nil {
			memoryCapacity.Add(*nodeMemoryCap)
		}
	}
	if cpuCapacity.Cmp(*resource.NewQuantity(ProductionClusterCpuThreshold, resource.DecimalSI)) == -1 ||
		memoryCapacity.Cmp(*resource.NewScaledQuantity(ProductionClusterMemoryThresholdGi, resource.Giga)) == -1 {
		return Evaluation, nil
	}
	return Production, nil
}

type ClusterFlavour int

const (
	Unknown ClusterFlavour = iota
	k3d
	GKE
)

type ClusterConfiguration map[string]interface{}

func EvaluateClusterConfiguration(ctx context.Context, k8sclient client.Client) (ClusterConfiguration, error) {
	flavour, err := discoverClusterFlavour(ctx, k8sclient)
	if err != nil {
		return ClusterConfiguration{}, err
	}

	return flavour.clusterConfiguration(), nil
}

func discoverClusterFlavour(ctx context.Context, k8sclient client.Client) (ClusterFlavour, error) {
	matcherGKE, err := regexp.Compile(`^v\d+\.\d+\.\d+-gke\.\d+$`)
	if err != nil {
		return Unknown, err
	}

	matcherk3d, err := regexp.Compile(`^v\d+\.\d+\.\d+\+k3s\d+$`)
	if err != nil {
		return Unknown, err
	}

	nodeList := corev1.NodeList{}
	err = k8sclient.List(ctx, &nodeList)
	if err != nil {
		return Unknown, err
	}

	for _, node := range nodeList.Items {
		match := matcherGKE.MatchString(node.Status.NodeInfo.KubeProxyVersion)
		if match {
			return GKE, nil
		}
		match = matcherk3d.MatchString(node.Status.NodeInfo.KubeProxyVersion)
		if match {
			return k3d, nil
		}
	}

	return Unknown, nil
}

func (f ClusterFlavour) clusterConfiguration() ClusterConfiguration {
	switch f {
	case k3d:
		config := map[string]interface{}{
			"spec": map[string]interface{}{
				"values": map[string]interface{}{
					"cni": map[string]string{
						"cniBinDir":  "/bin",
						"cniConfDir": "/var/lib/rancher/k3s/agent/etc/cni/net.d",
					},
					"gateways": map[string]interface{}{
						"istio-ingressgateway": map[string]interface{}{
							"serviceAnnotations": map[string]string{
								"dns.gardener.cloud/dnsnames": "'*.local.kyma.dev'",
							},
						},
					},
				},
			},
		}
		return config
	case GKE:
		config := map[string]interface{}{
			"spec": map[string]interface{}{
				"values": map[string]interface{}{
					"cni": map[string]interface{}{
						"cniBinDir": "/home/kubernetes/bin",
						"resourceQuotas": map[string]bool{
							"enabled": true,
						},
					},
				},
			},
		}
		return config
	}
	return ClusterConfiguration{}
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
