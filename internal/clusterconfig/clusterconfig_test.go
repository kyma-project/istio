package clusterconfig_test

import (
	"context"
	"testing"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

func TestClusterSize_String(t *testing.T) {
	tests := []struct {
		name string
		size clusterconfig.ClusterSize
		want string
	}{
		{name: "evaluation", size: clusterconfig.Evaluation, want: "Evaluation"},
		{name: "production", size: clusterconfig.Production, want: "Production"},
		{name: "unknown size", size: clusterconfig.UnknownSize, want: "Unknown"},
		{name: "out-of-range falls back to Unknown", size: clusterconfig.ClusterSize(99), want: "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.size.String())
		})
	}
}

func TestClusterProvider_String(t *testing.T) {
	tests := []struct {
		name     string
		provider clusterconfig.ClusterProvider
		want     string
	}{
		{name: "k3d", provider: clusterconfig.K3d, want: "K3d"},
		{name: "gke", provider: clusterconfig.GKE, want: "GKE"},
		{name: "aws", provider: clusterconfig.AWS, want: "AWS"},
		{name: "openstack", provider: clusterconfig.Openstack, want: "Openstack"},
		{name: "unknown", provider: clusterconfig.Unknown, want: "Unknown"},
		{name: "out-of-range falls back to Unknown", provider: clusterconfig.ClusterProvider(99), want: "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.provider.String())
		})
	}
}

func TestDiscoverClusterProvider(t *testing.T) {
	tests := []struct {
		name  string
		nodes []client.Object
		want  clusterconfig.ClusterProvider
	}{
		{
			name:  "empty node list returns Unknown",
			nodes: nil,
			want:  clusterconfig.Unknown,
		},
		{
			name: "k3d kubelet version",
			nodes: []client.Object{&corev1.Node{
				ObjectMeta: v1.ObjectMeta{Name: "n1"},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{KubeletVersion: "v1.26.6+k3s1"},
				},
			}},
			want: clusterconfig.K3d,
		},
		{
			name: "gke kubelet version",
			nodes: []client.Object{&corev1.Node{
				ObjectMeta: v1.ObjectMeta{Name: "n1"},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{KubeletVersion: "v1.30.6-gke.1125000"},
				},
			}},
			want: clusterconfig.GKE,
		},
		{
			name: "aws provider id",
			nodes: []client.Object{&corev1.Node{
				ObjectMeta: v1.ObjectMeta{Name: "n1"},
				Spec:       corev1.NodeSpec{ProviderID: "aws://abc123"},
			}},
			want: clusterconfig.AWS,
		},
		{
			name: "openstack provider id",
			nodes: []client.Object{&corev1.Node{
				ObjectMeta: v1.ObjectMeta{Name: "n1"},
				Spec:       corev1.NodeSpec{ProviderID: "openstack:///abc"},
			}},
			want: clusterconfig.Openstack,
		},
		{
			name: "unknown node returns Unknown",
			nodes: []client.Object{&corev1.Node{
				ObjectMeta: v1.ObjectMeta{Name: "n1"},
			}},
			want: clusterconfig.Unknown,
		},
		{
			name: "second node matches when first is unrecognised",
			nodes: []client.Object{
				&corev1.Node{ObjectMeta: v1.ObjectMeta{Name: "first"}},
				&corev1.Node{
					ObjectMeta: v1.ObjectMeta{Name: "second"},
					Spec:       corev1.NodeSpec{ProviderID: "aws://x"},
				},
			},
			want: clusterconfig.AWS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := createFakeClient(t, tt.nodes...)
			got, err := clusterconfig.DiscoverClusterProvider(context.Background(), c)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMergeOverrides(t *testing.T) {
	t.Run("override replaces template scalar", func(t *testing.T) {
		template := []byte("foo: bar\nbaz: original\n")
		overrides := clusterconfig.ClusterConfiguration{
			"baz": "overridden",
		}

		out, err := clusterconfig.MergeOverrides(template, overrides)
		require.NoError(t, err)

		var got map[string]interface{}
		require.NoError(t, yaml.Unmarshal(out, &got))
		assert.Equal(t, "bar", got["foo"])
		assert.Equal(t, "overridden", got["baz"])
	})

	t.Run("override deep-merges nested maps", func(t *testing.T) {
		template := []byte("spec:\n  values:\n    keep: keepme\n    replace: old\n")
		overrides := clusterconfig.ClusterConfiguration{
			"spec": map[string]interface{}{
				"values": map[string]interface{}{
					"replace": "new",
					"added":   "hello",
				},
			},
		}

		out, err := clusterconfig.MergeOverrides(template, overrides)
		require.NoError(t, err)

		var got map[string]interface{}
		require.NoError(t, yaml.Unmarshal(out, &got))

		spec := got["spec"].(map[string]interface{})
		values := spec["values"].(map[string]interface{})
		assert.Equal(t, "keepme", values["keep"])
		assert.Equal(t, "new", values["replace"])
		assert.Equal(t, "hello", values["added"])
	})

	t.Run("returns error on invalid template yaml", func(t *testing.T) {
		// Tab indentation is invalid in YAML mappings.
		_, err := clusterconfig.MergeOverrides([]byte("foo:\n\tbar: baz\n"), clusterconfig.ClusterConfiguration{})
		assert.Error(t, err)
	})

	t.Run("empty overrides leaves template intact", func(t *testing.T) {
		template := []byte("foo: bar\n")
		out, err := clusterconfig.MergeOverrides(template, clusterconfig.ClusterConfiguration{})
		require.NoError(t, err)

		var got map[string]interface{}
		require.NoError(t, yaml.Unmarshal(out, &got))
		assert.Equal(t, "bar", got["foo"])
	})
}

const (
	k3sMockKubeletVersion             = "v1.26.6+k3s1"
	gkeMockKubeletVersion             = "v1.30.6-gke.1125000"
	gardenerMockOSImageOldScheme      = "Garden Linux 1877.10"
	gardenerMockOSImageNewScheme      = "Garden Linux 2150.2.0"
	gardenerMockOSImageWithoutVersion = "Garden Linux"
)

var awsNLBConfig = clusterconfig.ClusterConfiguration(map[string]interface{}{
	"spec": map[string]interface{}{
		"values": map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"serviceAnnotations": map[string]string{
						"service.beta.kubernetes.io/aws-load-balancer-scheme":          "internet-facing",
						"service.beta.kubernetes.io/aws-load-balancer-nlb-target-type": "instance",
						"service.beta.kubernetes.io/aws-load-balancer-type":            "nlb",
					},
				},
			},
		},
	},
})

var openstackLBProxyProtocolConfig = clusterconfig.ClusterConfiguration(map[string]interface{}{
	"spec": map[string]interface{}{
		"values": map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"serviceAnnotations": map[string]string{
						"loadbalancer.openstack.org/proxy-protocol": "v1",
					},
				},
			},
		},
	},
})

var emptyValuesConfig = clusterconfig.ClusterConfiguration{
	"spec": map[string]interface{}{
		"values": map[string]interface{}{},
	},
}

func createFakeClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	require.NoError(t, operatorv1alpha2.AddToScheme(scheme.Scheme))
	require.NoError(t, corev1.AddToScheme(scheme.Scheme))
	require.NoError(t, networkingv1alpha3.AddToScheme(scheme.Scheme))
	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()
}

func createKymaRuntimeConfigWithDualStack(t *testing.T, enabled bool) *corev1.ConfigMap {
	t.Helper()
	networkDetails := map[string]interface{}{
		"dualStackIPEnabled": enabled,
	}
	details := map[string]interface{}{
		"networkDetails": networkDetails,
	}
	detailsBytes, err := yaml.Marshal(details)
	require.NoError(t, err)

	return &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      "kyma-provisioning-info",
			Namespace: "kyma-system",
		},
		Data: map[string]string{
			"details": string(detailsBytes),
		},
	}
}

func TestEvaluateClusterConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		objects []client.Object
		want    clusterconfig.ClusterConfiguration
	}{
		{
			name: "k3d sets cni values",
			objects: []client.Object{&corev1.Node{
				ObjectMeta: v1.ObjectMeta{Name: "k3d-node-1"},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{KubeletVersion: k3sMockKubeletVersion},
				},
			}},
			want: clusterconfig.ClusterConfiguration{
				"spec": map[string]interface{}{
					"values": map[string]interface{}{
						"cni": map[string]interface{}{
							"cniBinDir":  "/var/lib/rancher/k3s/data/cni",
							"cniConfDir": "/var/lib/rancher/k3s/agent/etc/cni/net.d",
						},
					},
				},
			},
		},
		{
			name: "AWS uses NLB when no elb-deprecated ConfigMap is present",
			objects: []client.Object{&corev1.Node{
				ObjectMeta: v1.ObjectMeta{Name: "aws-123"},
				Spec:       corev1.NodeSpec{ProviderID: "aws://asdasdads"},
			}},
			want: awsNLBConfig,
		},
		{
			name: "AWS uses NLB when elb-deprecated ConfigMap is present but service is already NLB-annotated",
			objects: []client.Object{
				&corev1.Node{
					ObjectMeta: v1.ObjectMeta{Name: "aws-123"},
					Spec:       corev1.NodeSpec{ProviderID: "aws://asdasdads"},
				},
				&corev1.ConfigMap{
					ObjectMeta: v1.ObjectMeta{Name: "elb-deprecated", Namespace: "istio-system"},
				},
				&corev1.Service{
					ObjectMeta: v1.ObjectMeta{
						Name:      "istio-ingressgateway",
						Namespace: "istio-system",
						Annotations: map[string]string{
							"service.beta.kubernetes.io/aws-load-balancer-scheme":          "internet-facing",
							"service.beta.kubernetes.io/aws-load-balancer-nlb-target-type": "instance",
							"service.beta.kubernetes.io/aws-load-balancer-type":            "nlb",
						},
					},
				},
			},
			want: awsNLBConfig,
		},
		{
			name: "AWS uses ELB when elb-deprecated ConfigMap is present",
			objects: []client.Object{
				&corev1.Node{
					ObjectMeta: v1.ObjectMeta{Name: "aws-123"},
					Spec:       corev1.NodeSpec{ProviderID: "aws://asdasdads"},
				},
				&corev1.ConfigMap{
					ObjectMeta: v1.ObjectMeta{Name: "elb-deprecated", Namespace: "istio-system"},
				},
			},
			want: emptyValuesConfig,
		},
		{
			name: "GKE sets cni values",
			objects: []client.Object{&corev1.Node{
				ObjectMeta: v1.ObjectMeta{Name: "gke-123"},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{KubeletVersion: gkeMockKubeletVersion},
				},
			}},
			want: clusterconfig.ClusterConfiguration{
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
			},
		},
		{
			name: "Gardener OpenStack with old Gardener versioning scheme",
			objects: []client.Object{&corev1.Node{
				ObjectMeta: v1.ObjectMeta{Name: "shoot--kyma-test--abcd1234-cpu-worker-0-z1-abcde-12345"},
				Spec:       corev1.NodeSpec{ProviderID: "openstack:///abcd1234-ab12-cd34-ef56-abcdef123456"},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{OSImage: gardenerMockOSImageOldScheme},
				},
			}},
			want: openstackLBProxyProtocolConfig,
		},
		{
			name: "Gardener OpenStack with new Gardener versioning scheme",
			objects: []client.Object{&corev1.Node{
				ObjectMeta: v1.ObjectMeta{Name: "shoot--kyma-test--abcd1234-cpu-worker-0-z1-abcde-12345"},
				Spec:       corev1.NodeSpec{ProviderID: "openstack:///abcd1234-ab12-cd34-ef56-abcdef123456"},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{OSImage: gardenerMockOSImageNewScheme},
				},
			}},
			want: openstackLBProxyProtocolConfig,
		},
		{
			name: "Gardener OpenStack without Garden Linux version",
			objects: []client.Object{&corev1.Node{
				ObjectMeta: v1.ObjectMeta{Name: "shoot--kyma-test--abcd1234-cpu-worker-0-z1-abcde-12345"},
				Spec:       corev1.NodeSpec{ProviderID: "openstack:///abcd1234-ab12-cd34-ef56-abcdef123456"},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{OSImage: gardenerMockOSImageWithoutVersion},
				},
			}},
			want: openstackLBProxyProtocolConfig,
		},
		{
			name: "Gardener on AWS uses NLB",
			objects: []client.Object{&corev1.Node{
				ObjectMeta: v1.ObjectMeta{Name: "shoot--kyma-test--abcd1234-cpu-worker-0-z1-abcde-12345"},
				Spec:       corev1.NodeSpec{ProviderID: "aws://example"},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{OSImage: gardenerMockOSImageNewScheme},
				},
			}},
			want: awsNLBConfig,
		},
		{
			name: "non-Gardener OpenStack returns no overrides",
			objects: []client.Object{&corev1.Node{
				ObjectMeta: v1.ObjectMeta{Name: "os-node"},
				Spec:       corev1.NodeSpec{ProviderID: "openstack:///abc"},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{OSImage: "Ubuntu 22.04"},
				},
			}},
			want: emptyValuesConfig,
		},
		{
			name: "unknown cluster returns no overrides",
			objects: []client.Object{&corev1.Node{
				ObjectMeta: v1.ObjectMeta{Name: "unknown-123"},
			}},
			want: emptyValuesConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := createFakeClient(t, tt.objects...)

			str, err := clusterconfig.BuildFactory(context.Background(), c)
			require.NoError(t, err)
			got := clusterconfig.ClusterConfigurationFromFactory(str)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEvaluateClusterSize(t *testing.T) {
	tests := []struct {
		name  string
		nodes []client.Object
		want  clusterconfig.ClusterSize
	}{
		{
			name: "Evaluation when CPU capacity is below threshold",
			nodes: []client.Object{&corev1.Node{
				ObjectMeta: v1.ObjectMeta{Name: "k3d-node-1"},
				Status: corev1.NodeStatus{
					Capacity: map[corev1.ResourceName]resource.Quantity{
						"cpu":    *resource.NewQuantity(clusterconfig.ProductionClusterCPUThreshold-1, resource.DecimalSI),
						"memory": *resource.NewScaledQuantity(int64(32), resource.Giga),
					},
				},
			}},
			want: clusterconfig.Evaluation,
		},
		{
			name: "Evaluation when memory capacity is below threshold",
			nodes: []client.Object{&corev1.Node{
				ObjectMeta: v1.ObjectMeta{Name: "k3d-node-1"},
				Status: corev1.NodeStatus{
					Capacity: map[corev1.ResourceName]resource.Quantity{
						"cpu":    *resource.NewMilliQuantity(int64(12000), resource.DecimalSI),
						"memory": *resource.NewScaledQuantity(clusterconfig.ProductionClusterMemoryThresholdGi-1, resource.Giga),
					},
				},
			}},
			want: clusterconfig.Evaluation,
		},
		{
			name: "Production when summed capacity meets thresholds across two nodes",
			nodes: []client.Object{
				&corev1.Node{
					ObjectMeta: v1.ObjectMeta{Name: "k3d-node-1"},
					Status: corev1.NodeStatus{
						Capacity: map[corev1.ResourceName]resource.Quantity{
							"cpu":    *resource.NewQuantity(clusterconfig.ProductionClusterCPUThreshold, resource.DecimalSI),
							"memory": *resource.NewScaledQuantity(clusterconfig.ProductionClusterMemoryThresholdGi, resource.Giga),
						},
					},
				},
				&corev1.Node{
					ObjectMeta: v1.ObjectMeta{Name: "k3d-node-2"},
					Status: corev1.NodeStatus{
						Capacity: map[corev1.ResourceName]resource.Quantity{
							"cpu":    *resource.NewQuantity(clusterconfig.ProductionClusterCPUThreshold, resource.DecimalSI),
							"memory": *resource.NewScaledQuantity(clusterconfig.ProductionClusterMemoryThresholdGi, resource.Giga),
						},
					},
				},
			},
			want: clusterconfig.Production,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := createFakeClient(t, tt.nodes...)

			size, err := clusterconfig.EvaluateClusterSize(context.Background(), c)

			require.NoError(t, err)
			assert.Equal(t, tt.want, size)
		})
	}
}

func TestIsDualStackEnabled_NonExperimental(t *testing.T) {
	c := createFakeClient(t)

	ds, err := clusterconfig.IsDualStackEnabled(context.Background(), c)

	require.NoError(t, err)
	assert.False(t, ds)
}
