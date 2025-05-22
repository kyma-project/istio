package clusterconfig_test

import (
	"context"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	k3sMockKubeletVersion string = "v1.26.6+k3s1"
	gkeMockKubeletVersion string = "v1.30.6-gke.1125000"
	gardenerMockOSImage   string = "Garden Linux 12.04"
)

var _ = Describe("GetClusterProvider", func() {
	It("should return other when cluster provider is unknown", func() {
		// given
		node := corev1.Node{
			ObjectMeta: v1.ObjectMeta{
				Name: "node-1",
			},
			Spec: corev1.NodeSpec{ProviderID: "kubernetes://asdadsads"},
			Status: corev1.NodeStatus{
				NodeInfo: corev1.NodeSystemInfo{
					KubeletVersion: k3sMockKubeletVersion,
				},
			},
		}
		client := createFakeClient(&node)
		p, err := clusterconfig.GetClusterProvider(context.Background(), client)
		Expect(err).To(BeNil())
		Expect(p).To(Equal(clusterconfig.Other))
	})
	It("should return 'openstack' for clusters provisioned on OpenStack nodes", func() {
		// given
		node := corev1.Node{
			ObjectMeta: v1.ObjectMeta{
				Name: "node-1",
			},
			Spec: corev1.NodeSpec{ProviderID: "openstack://example"},
		}
		client := createFakeClient(&node)
		p, err := clusterconfig.GetClusterProvider(context.Background(), client)
		Expect(err).To(BeNil())
		Expect(p).To(Equal(clusterconfig.Openstack))
	})
	It("should return 'aws' for clusters provisioned on AWS nodes", func() {
		// given
		node := corev1.Node{
			ObjectMeta: v1.ObjectMeta{
				Name: "node-1",
			},
			Spec: corev1.NodeSpec{ProviderID: "aws://asdadsads"},
		}
		client := createFakeClient(&node)
		p, err := clusterconfig.GetClusterProvider(context.Background(), client)
		Expect(err).To(BeNil())
		Expect(p).To(Equal(clusterconfig.Aws))
	})
	It("should return 'other' for clusters without nodes", func() {
		// given
		client := createFakeClient()
		p, err := clusterconfig.GetClusterProvider(context.Background(), client)
		Expect(err).To(BeNil())
		Expect(p).To(Equal(clusterconfig.Other))
	})
})

var _ = Describe("EvaluateClusterConfiguration", func() {
	Context("k3d", func() {
		It("should set cni values and serviceAnnotations to k3d configuration", func() {
			// given
			k3dNode := corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Name: "k3d-node-1",
				},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{
						KubeletVersion: k3sMockKubeletVersion,
					},
				},
			}

			client := createFakeClient(&k3dNode)
			provider, _ := clusterconfig.GetClusterProvider(context.Background(), client)

			// when
			config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), client, provider)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(config).To(Equal(clusterconfig.ClusterConfiguration(map[string]interface{}{
				"spec": map[string]interface{}{
					"values": map[string]interface{}{
						"cni": map[string]string{
							"cniBinDir":  "/var/lib/rancher/k3s/data/cni",
							"cniConfDir": "/var/lib/rancher/k3s/agent/etc/cni/net.d",
						},
					},
				},
			})))
		})
	})

	Context("AWS", func() {
		It("should use NLB when there is no elb-deprecated ConfigMap present", func() {
			// given
			awsNode := corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Name: "aws-123",
				},

				Spec: corev1.NodeSpec{
					ProviderID: "aws://asdasdads",
				},
			}

			client := createFakeClient(&awsNode)
			provider, _ := clusterconfig.GetClusterProvider(context.Background(), client)

			// when
			config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), client, provider)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(config).To(Equal(clusterconfig.AWSNLBConfig))
		})

		It("should use NLB load balancer on AWS if the elb-deprecated ConfigMap is present,"+
			" but the load balancer type was already switched", func() {
			// given
			awsNode := corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Name: "aws-123",
				},
				Spec: corev1.NodeSpec{
					ProviderID: "aws://asdasdads",
				},
			}

			elbDeprecatedConfigMap := corev1.ConfigMap{
				ObjectMeta: v1.ObjectMeta{
					Name:      "elb-deprecated",
					Namespace: "istio-system",
				},
			}

			ingressGatewayService := corev1.Service{
				ObjectMeta: v1.ObjectMeta{
					Name:      "istio-ingressgateway",
					Namespace: "istio-system",
					Annotations: map[string]string{
						"service.beta.kubernetes.io/aws-load-balancer-scheme":          "internet-facing",
						"service.beta.kubernetes.io/aws-load-balancer-nlb-target-type": "instance",
						"service.beta.kubernetes.io/aws-load-balancer-type":            "nlb",
					},
				},
			}

			client := createFakeClient(&awsNode, &elbDeprecatedConfigMap, &ingressGatewayService)
			provider, _ := clusterconfig.GetClusterProvider(context.Background(), client)

			// when
			config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), client, provider)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(config).To(Equal(clusterconfig.AWSNLBConfig))
		})

		It("should use ELB load balancer on AWS if elb-deprecated ConfigMap is present", func() {
			// given
			awsNode := corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Name: "aws-123",
				},
				Spec: corev1.NodeSpec{
					ProviderID: "aws://asdasdads",
				},
			}

			elbDeprecatedConfigMap := corev1.ConfigMap{
				ObjectMeta: v1.ObjectMeta{
					Name:      "elb-deprecated",
					Namespace: "istio-system",
				},
			}

			client := createFakeClient(&awsNode, &elbDeprecatedConfigMap)
			provider, _ := clusterconfig.GetClusterProvider(context.Background(), client)

			// when
			config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), client, provider)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(config).To(Equal(clusterconfig.ClusterConfiguration{}))
		})
	})

	Context("GKE", func() {
		It("should set cni values to GKE configuration", func() {
			// given
			gkeNode := corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Name: "gke-123",
				},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{
						KubeletVersion: gkeMockKubeletVersion,
					},
				},
			}

			client := createFakeClient(&gkeNode)
			provider, _ := clusterconfig.GetClusterProvider(context.Background(), client)

			// when
			config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), client, provider)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(config).To(Equal(clusterconfig.ClusterConfiguration(map[string]interface{}{
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
			})))
		})
	})

	Context("Gardener OpenStack", func() {
		It("should set istio-ingressgateway LoadBalancer service annotation value to Gardener OpenStack configuration", func() {
			// given
			gardenerNode := corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Name: "Garden Linux 1.23",
				},
				Spec: corev1.NodeSpec{ProviderID: "openstack://example"},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{
						OSImage: gardenerMockOSImage,
					},
				},
			}

			client := createFakeClient(&gardenerNode)
			provider, _ := clusterconfig.GetClusterProvider(context.Background(), client)

			// when
			config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), client, provider)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(config).To(Equal(clusterconfig.OpenStackLBProxyProtocolConfig))
		})
	})

	Context("Gardener - unknown", func() {
		It("should use NLB load balancer on AWS", func() {
			// given
			gardenerNode := corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Name: "Garden Linux 1.23",
				},
				Spec: corev1.NodeSpec{ProviderID: "aws://example"},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{
						OSImage: gardenerMockOSImage,
					},
				},
			}

			client := createFakeClient(&gardenerNode)
			provider, _ := clusterconfig.GetClusterProvider(context.Background(), client)

			// when
			config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), client, provider)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(config).To(Equal(clusterconfig.AWSNLBConfig))
		})
	})

	Context("Unknown cluster", func() {
		It("should return no overrides", func() {
			// given
			unkownNode := corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Name: "unknown-123",
				},
			}

			client := createFakeClient(&unkownNode)
			provider, _ := clusterconfig.GetClusterProvider(context.Background(), client)

			// when
			config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), client, provider)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(config).To(Equal(clusterconfig.ClusterConfiguration{}))
		})
	})
})

var _ = Describe("EvaluateClusterSize", func() {
	It("should return Evaluation when cpu capacity is less than ProductionClusterCPUThreshold", func() {
		// given
		k3dNode := corev1.Node{
			ObjectMeta: v1.ObjectMeta{
				Name: "k3d-node-1",
			},
			Status: corev1.NodeStatus{
				Capacity: map[corev1.ResourceName]resource.Quantity{
					"cpu":    *resource.NewQuantity(clusterconfig.ProductionClusterCPUThreshold-1, resource.DecimalSI),
					"memory": *resource.NewScaledQuantity(int64(32), resource.Giga),
				},
			},
		}

		client := createFakeClient(&k3dNode)

		// when
		size, err := clusterconfig.EvaluateClusterSize(context.Background(), client)

		// then
		Expect(err).To(Not(HaveOccurred()))
		Expect(size).To(Equal(clusterconfig.Evaluation))
	})

	It("should return Evaluation when memory capacity is less than ProductionClusterMemoryThresholdGi", func() {
		// given
		k3dNode := corev1.Node{
			ObjectMeta: v1.ObjectMeta{
				Name: "k3d-node-1",
			},
			Status: corev1.NodeStatus{
				Capacity: map[corev1.ResourceName]resource.Quantity{
					"cpu":    *resource.NewMilliQuantity(int64(12000), resource.DecimalSI),
					"memory": *resource.NewScaledQuantity(clusterconfig.ProductionClusterMemoryThresholdGi-1, resource.Giga),
				},
			},
		}

		client := createFakeClient(&k3dNode)

		// when
		size, err := clusterconfig.EvaluateClusterSize(context.Background(), client)

		// then
		Expect(err).To(Not(HaveOccurred()))
		Expect(size).To(Equal(clusterconfig.Evaluation))
	})

	It("should return Production when memory capacity is bigger than ProductionClusterMemoryThresholdGi and CPU capacity is bigger than ProductionClusterCPUThreshold", func() {
		// given
		k3dNode := corev1.Node{
			ObjectMeta: v1.ObjectMeta{
				Name: "k3d-node-1",
			},
			Status: corev1.NodeStatus{
				Capacity: map[corev1.ResourceName]resource.Quantity{
					"cpu":    *resource.NewQuantity(clusterconfig.ProductionClusterCPUThreshold, resource.DecimalSI),
					"memory": *resource.NewScaledQuantity(clusterconfig.ProductionClusterMemoryThresholdGi, resource.Giga),
				},
			},
		}

		k3dNode2 := corev1.Node{
			ObjectMeta: v1.ObjectMeta{
				Name: "k3d-node-2",
			},
			Status: corev1.NodeStatus{
				Capacity: map[corev1.ResourceName]resource.Quantity{
					"cpu":    *resource.NewQuantity(clusterconfig.ProductionClusterCPUThreshold, resource.DecimalSI),
					"memory": *resource.NewScaledQuantity(clusterconfig.ProductionClusterMemoryThresholdGi, resource.Giga),
				},
			},
		}

		client := createFakeClient(&k3dNode, &k3dNode2)

		// when
		size, err := clusterconfig.EvaluateClusterSize(context.Background(), client)

		// then
		Expect(err).To(Not(HaveOccurred()))
		Expect(size).To(Equal(clusterconfig.Production))
	})
})

func createFakeClient(objects ...client.Object) client.Client {
	err := operatorv1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = networkingv1alpha3.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()
}
