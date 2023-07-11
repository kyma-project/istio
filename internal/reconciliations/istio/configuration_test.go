package istio_test

import (
	"context"
	"fmt"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	istio "github.com/kyma-project/istio/operator/internal/reconciliations/istio"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	mockIstioTag             string = "1.16.1-distroless"
	lastAppliedConfiguration string = "operator.kyma-project.io/lastAppliedConfiguration"
)

var _ = Describe("Istio Configuration", func() {
	Context("LastAppliedConfiguration", func() {
		It("should update lastAppliedConfiguration and is able to unmarshal it back from annotation", func() {
			// given
			numTrustedProxies := 1
			istioCR := operatorv1alpha1.Istio{Spec: operatorv1alpha1.IstioSpec{Config: operatorv1alpha1.Config{NumTrustedProxies: &numTrustedProxies}}}

			// when
			updatedCR, err := istio.UpdateLastAppliedConfiguration(istioCR, mockIstioTag)

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(updatedCR.Annotations).To(Not(BeEmpty()))
			Expect(updatedCR.Annotations[lastAppliedConfiguration]).To(Equal(fmt.Sprintf(`{"config":{"numTrustedProxies":1},"IstioTag":"%s"}`, mockIstioTag)))

			appliedConfig, err := istio.GetLastAppliedConfiguration(updatedCR)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(*appliedConfig.Config.NumTrustedProxies).To(Equal(1))
		})
	})
})

var _ = Describe("Ingress Gateway", func() {
	Context("IngressGatewayNeedsRestart", func() {
		It("should restart when CR numTrustedProxies is 2 and in lastAppliedConfig is 1", func() {
			//given
			oldNumTrustedProxies := 1

			istioCR := operatorv1alpha1.Istio{}
			istioCR.Spec.Config.NumTrustedProxies = &oldNumTrustedProxies

			updatedCR, err := istio.UpdateLastAppliedConfiguration(istioCR, mockIstioTag)
			Expect(err).ShouldNot(HaveOccurred())

			newNumTrustedProxies := 2
			updatedCR.Spec.Config.NumTrustedProxies = &newNumTrustedProxies

			//when
			does, err := istio.IngressGatewayNeedsRestart(updatedCR)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(does).To(BeTrue())
		})

		It("should restart when CR numTrustedProxies is nil and in lastAppliedConfig is 1", func() {
			//given
			oldNumTrustedProxies := 1

			istioCR := operatorv1alpha1.Istio{}
			istioCR.Spec.Config.NumTrustedProxies = &oldNumTrustedProxies

			updatedCR, err := istio.UpdateLastAppliedConfiguration(istioCR, mockIstioTag)
			Expect(err).ShouldNot(HaveOccurred())

			updatedCR.Spec.Config.NumTrustedProxies = nil

			//when
			does, err := istio.IngressGatewayNeedsRestart(updatedCR)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(does).To(BeTrue())
		})

		It("should restart when CR numTrustedProxies is 1 and in lastAppliedConfig is nil", func() {
			//given
			istioCR := operatorv1alpha1.Istio{}

			updatedCR, err := istio.UpdateLastAppliedConfiguration(istioCR, mockIstioTag)
			Expect(err).ShouldNot(HaveOccurred())

			newNumTrustedProxies := 1
			updatedCR.Spec.Config.NumTrustedProxies = &newNumTrustedProxies

			//when
			does, err := istio.IngressGatewayNeedsRestart(updatedCR)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(does).To(BeTrue())
		})

		It("should not restart when CR numTrustedProxies is the same value as in lastAppliedConfig", func() {
			//given
			oldNumTrustedProxies := 2
			istioCR := operatorv1alpha1.Istio{}
			istioCR.Spec.Config.NumTrustedProxies = &oldNumTrustedProxies

			updatedCR, err := istio.UpdateLastAppliedConfiguration(istioCR, mockIstioTag)
			Expect(err).ShouldNot(HaveOccurred())

			newNumTrustedProxies := 2
			updatedCR.Spec.Config.NumTrustedProxies = &newNumTrustedProxies

			//when
			does, err := istio.IngressGatewayNeedsRestart(updatedCR)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(does).To(BeFalse())
		})

		It("should restart when CR has numTrustedProxy configured and lastAppliedConfig annotation is not set", func() {
			//given
			newNumTrustedProxies := 1

			istioCR := operatorv1alpha1.Istio{}
			istioCR.Spec.Config.NumTrustedProxies = &newNumTrustedProxies

			//when
			does, err := istio.IngressGatewayNeedsRestart(istioCR)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(does).To(BeTrue())
		})
	})

	Context("RestartIngressGateway", func() {
		client := createFakeClientWithDeployment()

		It("should set annotation on Istio IG deployment when restart is needed", func() {
			//given
			newNumTrustedProxies := 1

			istioCR := operatorv1alpha1.Istio{}
			istioCR.Spec.Config.NumTrustedProxies = &newNumTrustedProxies

			//when
			err := istio.RestartIngressGateway(context.TODO(), client)
			Expect(err).To(Not(HaveOccurred()))

			deployment := appsv1.Deployment{}
			err = client.Get(context.TODO(), types.NamespacedName{Namespace: "istio-system", Name: "istio-ingressgateway"}, &deployment)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(deployment.Spec.Template.Annotations["istio-operator.kyma-project.io/restartedAt"]).ToNot(BeEmpty())
		})
	})
})

func createFakeClientWithDeployment() client.Client {
	deployment := appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}

	err := corev1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = appsv1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())

	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(&deployment).Build()
}
