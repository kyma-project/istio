package clusterconfig_test

import (
	"context"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	k3sMockKubeProxyVersion string = "v1.25.6+k3s1"
	gkeMockKubeProxyVersion string = "v1.24.9-gke.3200"
	gardenerMockOSImage     string = "Garden Linux 934.8"
)

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
						KubeProxyVersion: k3sMockKubeProxyVersion,
					},
				},
			}

			client := createFakeClient(&k3dNode)

			//when
			config, err := clusterconfig.EvaluateClusterConfiguration(context.TODO(), client)

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
						KubeProxyVersion: gkeMockKubeProxyVersion,
					},
				},
			}

			client := createFakeClient(&gkeNode)

			//when
			config, err := clusterconfig.EvaluateClusterConfiguration(context.TODO(), client)

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

	Context("Gardener", func() {
		It("should set Istio GW annotation specific for Gardener clusters only", func() {
			//given
			gardenerNode := corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Name: "shoot-node-1",
				},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{
						OSImage: gardenerMockOSImage,
					},
				},
			}

			client := createFakeClient(&gardenerNode)

			//when
			config, err := clusterconfig.EvaluateClusterConfiguration(context.TODO(), client)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(config).To(Equal(clusterconfig.ClusterConfiguration(map[string]interface{}{
				"spec": map[string]interface{}{
					"values": map[string]interface{}{
						"gateways": map[string]interface{}{
							"istio-ingressgateway": map[string]interface{}{
								"serviceAnnotations": map[string]string{
									"dns.gardener.cloud/dnsnames": "*.example.com",
								},
							},
						},
					},
				},
			})))
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

			//when
			config, err := clusterconfig.EvaluateClusterConfiguration(context.TODO(), client)

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
	err := operatorv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())

	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()
}
