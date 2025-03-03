package clusterconfig

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	"regexp"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"

	"github.com/imdario/mergo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	ProductionClusterCpuThreshold      int64 = 5
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
	default:
		return "Unknown"
	}
}

// EvaluateClusterSize counts the entire capacity of cpu and memory in the cluster and returns Evaluation
// if the total capacity of any of the resources is lower than ProductionClusterCpuThreshold or ProductionClusterMemoryThresholdGi
func EvaluateClusterSize(ctx context.Context, k8sClient client.Client) (ClusterSize, error) {
	nodeList := corev1.NodeList{}
	err := k8sClient.List(ctx, &nodeList)
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
	Gardener
	AWS
)

func (c ClusterFlavour) String() string {
	switch c {
	case k3d:
		return "k3d"
	case GKE:
		return "GKE"
	case Gardener:
		return "Gardener"
	case AWS:
		return "AWS"
	}
	return "Unknown"
}

type ClusterConfiguration map[string]interface{}

var AWSNLBConfig = ClusterConfiguration{
	"spec": map[string]interface{}{
		"values": map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"serviceAnnotations": map[string]string{
						loadBalancerTypeAnnotation:          loadBalancerType,
						loadBalancerSchemeAnnotation:        loadBalancerScheme,
						loadBalancerNlbTargetTypeAnnotation: loadBalancerNlbTargetType,
					},
				},
			},
		},
	},
}

const (
	elbCmName      = "elb-deprecated"
	elbCmNamespace = "istio-system"

	loadBalancerSchemeAnnotation        = "service.beta.kubernetes.io/aws-load-balancer-scheme"
	loadBalancerScheme                  = "internet-facing"
	loadBalancerNlbTargetTypeAnnotation = "service.beta.kubernetes.io/aws-load-balancer-nlb-target-type"
	loadBalancerNlbTargetType           = "instance"
	loadBalancerTypeAnnotation          = "service.beta.kubernetes.io/aws-load-balancer-type"
	loadBalancerType                    = "external"
)

func ShouldUseNLB(ctx context.Context, k8sClient client.Client) (bool, error) {
	var elbDeprecated corev1.ConfigMap
	err := k8sClient.Get(ctx, client.ObjectKey{Namespace: elbCmNamespace, Name: elbCmName}, &elbDeprecated)
	if err != nil {
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}

	var ingressGatewaySvc corev1.Service
	err = k8sClient.Get(ctx, client.ObjectKey{Namespace: "istio-system", Name: "istio-ingressgateway"}, &ingressGatewaySvc)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	if value, ok := ingressGatewaySvc.Annotations[loadBalancerTypeAnnotation]; ok && value == loadBalancerType {
		return true, nil
	}

	return false, nil
}

// awsConfig returns config specific to AWS cluster.
// The function evaluates whether to use NLB or ELB load balancer, based on:
//
// 1. Presence of "elb-deprecated" ConfigMap
//
// 2. If the ConfigMap is present, it checks the loadBalancerTypeAnnotation,
// to check if it was not already set to "nlb".
// This safeguards against switching back to ELB.
func awsConfig(ctx context.Context, k8sClient client.Client) (ClusterConfiguration, error) {
	useNLB, err := ShouldUseNLB(ctx, k8sClient)
	if err != nil {
		return ClusterConfiguration{}, err
	}

	if useNLB {
		return AWSNLBConfig, nil
	}

	return ClusterConfiguration{}, err
}

func EvaluateClusterConfiguration(ctx context.Context, k8sClient client.Client) (ClusterConfiguration, error) {
	flavour, err := DiscoverClusterFlavour(ctx, k8sClient)
	if err != nil {
		return ClusterConfiguration{}, err
	}

	if flavour == AWS {
		return awsConfig(ctx, k8sClient)
	}

	return flavour.clusterConfiguration()
}

// GetClusterProvider is a small hack that tries to determine the
// hyperscaler based on the first provider node.
func GetClusterProvider(ctx context.Context, k8sclient client.Client) (string, error) {
	nodes := corev1.NodeList{}
	err := k8sclient.List(ctx, &nodes)
	if err != nil {
		return "", err
	}
	// if we got OK response and node list is empty, we can't guess cloud provider
	// treat as "other" provider
	// in standard execution this should never be reached because if cluster
	// doesn't have any nodes, nothing can be run on it
	// this catches rare case where cluster doesn't have any nodes, but
	// client-go also doesn't return any error
	if len(nodes.Items) == 0 {
		ctrl.Log.Info("unable to determine cloud provider due to empty node list, using 'other' as provider")
		return "other", nil
	}

	// get 1st node since all nodes usually are backed by the same provider
	n := nodes.Items[0]
	provider := n.Spec.ProviderID
	switch {
	case strings.HasPrefix(provider, "aws://"):
		return "aws", nil
	default:
		return "other", nil
	}
}

func DiscoverClusterFlavour(ctx context.Context, k8sClient client.Client) (ClusterFlavour, error) {
	matcherGKE, err := regexp.Compile(`^v\d+\.\d+\.\d+-gke\.\d+$`)
	if err != nil {
		return Unknown, err
	}
	matcherk3d, err := regexp.Compile(`^v\d+\.\d+\.\d+\+k3s\d+$`)
	if err != nil {
		return Unknown, err
	}
	matcherGardener, err := regexp.Compile(`^Garden Linux \d+.\d+$`)
	if err != nil {
		return Unknown, err
	}
	matcherAws, err := regexp.Compile(`^aws://`)
	if err != nil {
		return Unknown, err
	}

	nodeList := corev1.NodeList{}
	err = k8sClient.List(ctx, &nodeList)
	if err != nil {
		return Unknown, err
	}

	for _, node := range nodeList.Items {
		if matcherGKE.MatchString(node.Status.NodeInfo.KubeletVersion) {
			return GKE, nil
		} else if matcherk3d.MatchString(node.Status.NodeInfo.KubeletVersion) {
			return k3d, nil
		} else if matcherAws.MatchString(node.Spec.ProviderID) {
			return AWS, nil
		} else if matcherGardener.MatchString(node.Status.NodeInfo.OSImage) {
			return Gardener, nil
		}
	}

	return Unknown, nil
}

func (c ClusterFlavour) clusterConfiguration() (ClusterConfiguration, error) {
	switch c {
	case k3d:
		config := map[string]interface{}{
			"spec": map[string]interface{}{
				"values": map[string]interface{}{
					"cni": map[string]string{
						"cniBinDir":  "/bin",
						"cniConfDir": "/var/lib/rancher/k3s/agent/etc/cni/net.d",
					},
				},
			},
		}
		return config, nil
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
		return config, nil
	case AWS:
		config := map[string]interface{}{
			"spec": map[string]interface{}{
				"values": map[string]interface{}{
					"gateways": map[string]interface{}{
						"istio-ingressgateway": map[string]interface{}{
							"serviceAnnotations": map[string]string{
								"service.beta.kubernetes.io/aws-load-balancer-type": "nlb",
							},
						},
					},
				},
			},
		}
		return config, nil
	case Gardener:
		return ClusterConfiguration{}, nil
	}
	return ClusterConfiguration{}, nil
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
