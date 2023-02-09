package gatherer_test

import (
	"context"
	"testing"

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
	IstioResourceName    string = "some-istio"
	IstioCRNamespace     string = "kyma-system"
	IstioSystemNamespace string = "istio-system"
	TestLabelKey         string = "test-key"
	TestLabelVal         string = "test-val"
	DefaultNamespace     string = "default"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Gatherer Suite")
}

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

	Context("ListInstalledIstioRevisions", func() {
		It("Should list all istio versions with revisions", func() {

			istioSystem := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: IstioSystemNamespace,
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
		It("Should return empty map when there is no istio installed", func() {
			client := createClientSet()

			istioVersions, err := gatherer.ListInstalledIstioRevisions(context.TODO(), client)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(istioVersions).To(BeEmpty())
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
