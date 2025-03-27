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
		//given
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
		p, err := clusterconfig.GetClusterProvider(context.TODO(), client)
		Expect(err).To(BeNil())
		Expect(p).To(Equal("other"))
	})
	It("should return 'openstack' for clusters provisioned on OpenStack nodes", func() {
		//given
		node := corev1.Node{
			ObjectMeta: v1.ObjectMeta{
				Name: "node-1",
			},
			Spec: corev1.NodeSpec{ProviderID: "openstack://example"},
		}
		client := createFakeClient(&node)
		p, err := clusterconfig.GetClusterProvider(context.TODO(), client)
		Expect(err).To(BeNil())
		Expect(p).To(Equal("openstack"))
	})
	It("should return 'aws' for clusters provisioned on AWS nodes", func() {
		//given
		node := corev1.Node{
			ObjectMeta: v1.ObjectMeta{
				Name: "node-1",
			},
			Spec: corev1.NodeSpec{ProviderID: "aws://asdadsads"},
		}
		client := createFakeClient(&node)
		p, err := clusterconfig.GetClusterProvider(context.TODO(), client)
		Expect(err).To(BeNil())
		Expect(p).To(Equal("aws"))
	})
	It("should return 'other' for clusters without nodes", func() {
		//given
		client := createFakeClient()
		p, err := clusterconfig.GetClusterProvider(context.TODO(), client)
		Expect(err).To(BeNil())
		Expect(p).To(Equal("other"))
	})
})

var _ = Describe("EvaluateClusterConfiguration", func() {
	Context("k3d", func() {
		It("should set cni values and serviceAnnotations to k3d configuration", func() {
			//given
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
			provider, err := clusterconfig.GetClusterProvider(context.TODO(), client)

			//when
			config, err := clusterconfig.EvaluateClusterConfiguration(context.TODO(), client, provider)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(config).To(Equal(clusterconfig.ClusterConfiguration(map[string]interface{}{
				"spec": map[string]interface{}{
					"values": map[string]interface{}{
						"cni": map[string]string{
							"cniBinDir":  "/bin",
							"cniConfDir": "/var/lib/rancher/k3s/agent/etc/cni/net.d",
						},
					},
				},
			})))
		})
	})

	Context("GKE", func() {
		It("should set cni values to GKE configuration", func() {
			//given
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
			provider, err := clusterconfig.GetClusterProvider(context.TODO(), client)

			//when
			config, err := clusterconfig.EvaluateClusterConfiguration(context.TODO(), client, provider)

			//then
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
			//given
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
			provider, err := clusterconfig.GetClusterProvider(context.TODO(), client)

			//when
			config, err := clusterconfig.EvaluateClusterConfiguration(context.TODO(), client, provider)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(config).To(Equal(clusterconfig.ClusterConfiguration(map[string]interface{}{
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
			})))
		})
	})

	Context("Gardener - unknown", func() {
		It("should return no overrides", func() {
			//given
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
			provider, err := clusterconfig.GetClusterProvider(context.TODO(), client)

			//when
			config, err := clusterconfig.EvaluateClusterConfiguration(context.TODO(), client, provider)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(config).To(Equal(clusterconfig.ClusterConfiguration{}))
		})
	})

	Context("Unknown cluster", func() {
		It("should return no overrides", func() {
			//given
			unkownNode := corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Name: "unknown-123",
				},
			}

			client := createFakeClient(&unkownNode)
			provider, err := clusterconfig.GetClusterProvider(context.TODO(), client)

			//when
			config, err := clusterconfig.EvaluateClusterConfiguration(context.TODO(), client, provider)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(config).To(Equal(clusterconfig.ClusterConfiguration{}))
		})
	})
})

var _ = Describe("EvaluateClusterSize", func() {
	It("should return Evaluation when cpu capacity is less than ProductionClusterCpuThreshold", func() {
		//given
		k3dNode := corev1.Node{
			ObjectMeta: v1.ObjectMeta{
				Name: "k3d-node-1",
			},
			Status: corev1.NodeStatus{
				Capacity: map[corev1.ResourceName]resource.Quantity{
					"cpu":    *resource.NewQuantity(clusterconfig.ProductionClusterCpuThreshold-1, resource.DecimalSI),
					"memory": *resource.NewScaledQuantity(int64(32), resource.Giga),
				},
			},
		}

		client := createFakeClient(&k3dNode)

		//when
		size, err := clusterconfig.EvaluateClusterSize(context.TODO(), client)

		//then
		Expect(err).To(Not(HaveOccurred()))
		Expect(size).To(Equal(clusterconfig.Evaluation))
	})

	It("should return Evaluation when memory capacity is less than ProductionClusterMemoryThresholdGi", func() {
		//given
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

		//when
		size, err := clusterconfig.EvaluateClusterSize(context.TODO(), client)

		//then
		Expect(err).To(Not(HaveOccurred()))
		Expect(size).To(Equal(clusterconfig.Evaluation))
	})

	It("should return Production when memory capacity is bigger than ProductionClusterMemoryThresholdGi and CPU capacity is bigger than ProductionClusterCpuThreshold", func() {
		//given
		k3dNode := corev1.Node{
			ObjectMeta: v1.ObjectMeta{
				Name: "k3d-node-1",
			},
			Status: corev1.NodeStatus{
				Capacity: map[corev1.ResourceName]resource.Quantity{
					"cpu":    *resource.NewQuantity(clusterconfig.ProductionClusterCpuThreshold, resource.DecimalSI),
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
					"cpu":    *resource.NewQuantity(clusterconfig.ProductionClusterCpuThreshold, resource.DecimalSI),
					"memory": *resource.NewScaledQuantity(clusterconfig.ProductionClusterMemoryThresholdGi, resource.Giga),
				},
			},
		}

		client := createFakeClient(&k3dNode, &k3dNode2)

		//when
		size, err := clusterconfig.EvaluateClusterSize(context.TODO(), client)

		//then
		Expect(err).To(Not(HaveOccurred()))
		Expect(size).To(Equal(clusterconfig.Production))
	})
})

func createFakeClient(objects ...client.Object) client.Client {
	err := operatorv1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())

	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()
}
