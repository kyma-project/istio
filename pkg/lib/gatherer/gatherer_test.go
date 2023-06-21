package gatherer_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kyma-project/istio/operator/internal/tests"
	"github.com/onsi/ginkgo/v2/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	"github.com/masterminds/semver"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	IstioResourceName string = "some-istio"
	IstioCRNamespace  string = "kyma-system"
	TestLabelKey      string = "test-key"
	TestLabelVal      string = "test-val"
	DefaultNamespace  string = "default"
	ImageVersion      string = "1.10.0"
	ImageVersionOld   string = "1.09.0"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Gatherer Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("gatherer-suite", report)
})

var _ = Describe("Gatherer", func() {

	It("GetIstioCR", func() {
		kymaSystem := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: IstioCRNamespace,
			},
		}

		istioKymaSystem := v1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:      IstioResourceName,
			Namespace: IstioCRNamespace,
			Labels: map[string]string{
				TestLabelKey: TestLabelVal,
			},
		}}

		client := createClientSet(&kymaSystem, &istioKymaSystem)

		istioCr, err := gatherer.GetIstioCR(context.TODO(), client, IstioResourceName, IstioCRNamespace)

		Expect(err).ShouldNot(HaveOccurred())
		Expect(istioCr.ObjectMeta.Labels[TestLabelKey]).To(Equal(istioKymaSystem.ObjectMeta.Labels[TestLabelKey]))

		noObjectClient := createClientSet(&kymaSystem)
		istioCrNoObject, err := gatherer.GetIstioCR(context.TODO(), noObjectClient, IstioResourceName, IstioCRNamespace)

		Expect(err).Should(HaveOccurred())
		Expect(istioCrNoObject).To(BeNil())
	})

	It("ListIstioCR", func() {
		kymaSystem := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: IstioCRNamespace,
			},
		}

		istioKymaSystem := v1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:      IstioResourceName,
			Namespace: IstioCRNamespace,
			Labels: map[string]string{
				TestLabelKey: TestLabelVal,
			},
		}}

		istioDefault := v1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:      IstioResourceName,
			Namespace: DefaultNamespace,
			Labels: map[string]string{
				TestLabelKey: TestLabelVal,
			},
		}}

		client := createClientSet(&kymaSystem, &istioKymaSystem, &istioDefault)

		istioCrNoNamespace, err := gatherer.ListIstioCR(context.TODO(), client)

		Expect(err).ShouldNot(HaveOccurred())
		Expect(istioCrNoNamespace.Items).To(HaveLen(2))

		istioCrKymaSystem, err := gatherer.ListIstioCR(context.TODO(), client, IstioCRNamespace)

		Expect(err).ShouldNot(HaveOccurred())
		Expect(istioCrKymaSystem.Items).To(HaveLen(1))

		istioBothNamespaces, err := gatherer.ListIstioCR(context.TODO(), client, IstioCRNamespace, "default")

		Expect(err).ShouldNot(HaveOccurred())
		Expect(istioBothNamespaces.Items).To(HaveLen(2))
	})

	Context("ListIstioCPPods", func() {
		istiodPod := createPodWith("istiod", gatherer.IstioNamespace, "discovery", "istio/pilot", ImageVersion, false)
		istiogwPod := createPodWith("istio-ingressgateway", gatherer.IstioNamespace, "istio-proxy", "istio/proxyv2", ImageVersion, false)
		appPod := createPodWith("application", "app-namespace", "istio-proxy", "istio/proxyv2", ImageVersion, false)

		It("should not get any pods in istio-system namespace if there are none", func() {
			istioSystem := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: gatherer.IstioNamespace,
				},
			}

			client := createClientSet(&istioSystem)

			istioNamespace, err := gatherer.ListIstioCPPods(context.TODO(), client)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(istioNamespace.Items).To(BeEmpty())
		})

		It("should get all pods in istio-system namespace", func() {
			istioSystem := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: gatherer.IstioNamespace,
				},
			}

			client := createClientSet(&istioSystem, istiodPod, istiogwPod, appPod)

			istioNamespace, err := gatherer.ListIstioCPPods(context.TODO(), client)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(istioNamespace.Items).To(HaveLen(2))
		})
	})

	Context("ListInstalledIstioRevisions", func() {
		It("should list all istio versions with revisions", func() {

			istioSystem := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: gatherer.IstioNamespace,
				},
			}

			istiodDefaultRevision := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "istiod",
					Labels: map[string]string{
						"app":                       "istiod",
						"istio.io/rev":              "default",
						"operator.istio.io/version": "1.16.1",
					},
				},
			}

			istiodOtherRevision := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "istiod-stable",
					Labels: map[string]string{
						"app":                       "istiod",
						"istio.io/rev":              "stable",
						"operator.istio.io/version": "1.15.4",
					},
				},
			}
			client := createClientSet(&istioSystem, &istiodDefaultRevision, &istiodOtherRevision)

			istioVersions, err := gatherer.ListInstalledIstioRevisions(context.TODO(), client)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(istioVersions).To(HaveKey("default"))
			Expect(istioVersions["default"]).To(Equal(semver.MustParse("1.16.1")))

			Expect(istioVersions).To(HaveKey("stable"))
			Expect(istioVersions["stable"]).To(Equal(semver.MustParse("1.15.4")))
		})

		It("should return empty map when there is no istio installed", func() {
			client := createClientSet()

			istioVersions, err := gatherer.ListInstalledIstioRevisions(context.TODO(), client)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(istioVersions).To(BeEmpty())
		})
	})

	Context("GetIstioPodsVersion", func() {
		istiodPod := createPodWith("istiod", gatherer.IstioNamespace, "discovery", "istio/pilot", ImageVersion, false)
		istiogwPod := createPodWith("istio-ingressgateway", gatherer.IstioNamespace, "istio-proxy", "istio/proxyv2", ImageVersion, false)
		istiogwPodTerm := createPodWith("istio-ingressgateway-old", gatherer.IstioNamespace, "istio-proxy", "istio/proxyv2", ImageVersionOld, true)
		istiocniPod := createPodWith("istio-cni-node", gatherer.IstioNamespace, "install-cni", "istio/install-cni", ImageVersion, false)
		appPod := createPodWith("application", "app-namespace", "istio-proxy", "istio/proxyv2", ImageVersion, false)

		It("should get Istio installed version based on pods in istio-system namespace", func() {
			istioSystem := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: gatherer.IstioNamespace,
				},
			}

			client := createClientSet(&istioSystem, istiodPod, istiogwPod, istiocniPod, appPod)

			version, err := gatherer.GetIstioPodsVersion(context.TODO(), client)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(version).To(Equal(ImageVersion))
		})

		It("should not consider terminating pods in istio-system namespace when getting Istio version", func() {
			istioSystem := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: gatherer.IstioNamespace,
				},
			}

			client := createClientSet(&istioSystem, istiodPod, istiogwPod, istiocniPod, istiogwPodTerm, appPod)

			version, err := gatherer.GetIstioPodsVersion(context.TODO(), client)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(version).To(Equal(ImageVersion))
		})

		It("should get Istio installed version when there is a pod with image prerelease version", func() {
			istioSystem := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: gatherer.IstioNamespace,
				},
			}
			istiocniPodDistroless := createPodWith("istio-cni-node", "istio-system", "install-cni", "istio/install-cni", ImageVersion+"-distroless", false)

			client := createClientSet(&istioSystem, istiodPod, istiogwPod, istiocniPodDistroless, appPod)

			version, err := gatherer.GetIstioPodsVersion(context.TODO(), client)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(version).To(Equal(ImageVersion))
		})

		It("should return error when there are no pods in istio-namespace", func() {
			istioSystem := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: gatherer.IstioNamespace,
				},
			}

			client := createClientSet(&istioSystem, appPod)

			version, err := gatherer.GetIstioPodsVersion(context.TODO(), client)

			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to obtain installed Istio image version"))
			Expect(version).To(Equal(""))
		})

		It("should return error when there is an inconsistent version state in istio-system namespace", func() {
			istioSystem := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: gatherer.IstioNamespace,
				},
			}
			istiocniPodOld := createPodWith("istio-cni-node", "istio-system", "install-cni", "istio/install-cni", "1.0.0", false)
			client := createClientSet(&istioSystem, istiodPod, istiogwPod, istiocniPodOld, appPod)

			version, err := gatherer.GetIstioPodsVersion(context.TODO(), client)

			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Image version of pod istio-ingressgateway: 1.10.0 do not match version: 1.0.0"))
			Expect(version).To(Equal(""))
		})

		It("should return error when there is a pod with wrong image", func() {
			istioSystem := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: gatherer.IstioNamespace,
				},
			}
			istiocniPodWrong := createPodWith("istio-cni-node", "istio-system", "install-cni", "istio/install-cni", "wrong", false)
			client := createClientSet(&istioSystem, istiodPod, istiogwPod, istiocniPodWrong, appPod)

			version, err := gatherer.GetIstioPodsVersion(context.TODO(), client)

			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Invalid Semantic Version"))
			Expect(version).To(Equal(""))
		})

		It("should return error when there is a pod with latest versioned image", func() {
			istioSystem := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: gatherer.IstioNamespace,
				},
			}
			istiocniPodLatest := createPodWith("istio-cni-node", "istio-system", "install-cni", "istio/install-cni", "latest", false)
			client := createClientSet(&istioSystem, istiodPod, istiogwPod, istiocniPodLatest, appPod)

			version, err := gatherer.GetIstioPodsVersion(context.TODO(), client)

			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Invalid Semantic Version"))
			Expect(version).To(Equal(""))
		})
	})
})

func createClientSet(objects ...client.Object) client.Client {
	err := v1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = appsv1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())

	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()
}

func createPodWith(name, namespace, containerName, image, imageVersion string, terminating bool) *corev1.Pod {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet"},
			},
			Annotations: map[string]string{"sidecar.istio.io/status": fmt.Sprintf(`{"containers":["%s"]}`, name+"-container")},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPhase(corev1.PodRunning),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  containerName,
					Image: image + ":" + imageVersion,
				},
			},
		},
	}
	if terminating {
		timestamp := metav1.Now()
		pod.ObjectMeta.DeletionTimestamp = &timestamp
	}
	return &pod
}
