package istio

import (
	"context"
	"fmt"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"

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
			err := UpdateLastAppliedConfiguration(&istioCR, mockIstioTag)

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(istioCR.Annotations).To(Not(BeEmpty()))
			Expect(istioCR.Annotations[lastAppliedConfiguration]).To(Equal(fmt.Sprintf(`{"config":{"numTrustedProxies":1},"IstioTag":"%s"}`, mockIstioTag)))

			appliedConfig, err := getLastAppliedConfiguration(&istioCR)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(*appliedConfig.Config.NumTrustedProxies).To(Equal(1))
		})
	})
})

var _ = Describe("Ingress Gateway", func() {
	Context("restartIngressGatewayIfNeeded", func() {
		It("should restart when CR spec numTrustedProxies is different than in lastAppliedConfig", func() {
			//given
			client := createFakeClientWithDeployment()
			newNumTrustedProxies := 2
			istioCR := operatorv1alpha1.Istio{}
			istioCR.Spec.Config.NumTrustedProxies = &newNumTrustedProxies
			istioCR.Annotations = map[string]string{}
			istioCR.Annotations[lastAppliedConfiguration] = fmt.Sprintf(`{"config":{"numTrustedProxies":1},"IstioTag":"%s"}`, mockIstioTag)

			//when
			err := restartIngressGatewayIfNeeded(context.TODO(), client, &istioCR)

			//then
			Expect(err).To(Not(HaveOccurred()))

			deployment, err := getIstioIngressDeployment(client)
			Expect(err).To(Not(HaveOccurred()))
			Expect(deployment.Spec.Template.Annotations["istio-operator.kyma-project.io/restartedAt"]).ToNot(BeEmpty())
		})

		It("should restart when CR numTrustedProxies is nil and in lastAppliedConfig is 1", func() {
			//given
			client := createFakeClientWithDeployment()
			istioCR := operatorv1alpha1.Istio{}
			istioCR.Annotations = map[string]string{}
			istioCR.Annotations[lastAppliedConfiguration] = fmt.Sprintf(`{"config":{"numTrustedProxies":1},"IstioTag":"%s"}`, mockIstioTag)

			//when
			err := restartIngressGatewayIfNeeded(context.TODO(), client, &istioCR)

			//then
			Expect(err).To(Not(HaveOccurred()))

			deployment, err := getIstioIngressDeployment(client)
			Expect(err).To(Not(HaveOccurred()))
			Expect(deployment.Spec.Template.Annotations["istio-operator.kyma-project.io/restartedAt"]).ToNot(BeEmpty())
		})

		It("should restart when CR numTrustedProxies is 1 and in lastAppliedConfig is nil", func() {
			//given
			client := createFakeClientWithDeployment()
			newNumTrustedProxies := 1
			istioCR := operatorv1alpha1.Istio{}
			istioCR.Spec.Config.NumTrustedProxies = &newNumTrustedProxies
			istioCR.Annotations = map[string]string{}
			istioCR.Annotations[lastAppliedConfiguration] = fmt.Sprintf(`{"config":{},"IstioTag":"%s"}`, mockIstioTag)

			//when
			err := restartIngressGatewayIfNeeded(context.TODO(), client, &istioCR)

			//then
			Expect(err).To(Not(HaveOccurred()))

			deployment, err := getIstioIngressDeployment(client)
			Expect(err).To(Not(HaveOccurred()))
			Expect(deployment.Spec.Template.Annotations["istio-operator.kyma-project.io/restartedAt"]).ToNot(BeEmpty())
		})

		It("should not restart when CR numTrustedProxies is the same value as in lastAppliedConfig", func() {
			client := createFakeClientWithDeployment()
			newNumTrustedProxies := 1
			istioCR := operatorv1alpha1.Istio{}
			istioCR.Spec.Config.NumTrustedProxies = &newNumTrustedProxies
			istioCR.Annotations = map[string]string{}
			istioCR.Annotations[lastAppliedConfiguration] = fmt.Sprintf(`{"config":{"numTrustedProxies":1},"IstioTag":"%s"}`, mockIstioTag)

			//when
			err := restartIngressGatewayIfNeeded(context.TODO(), client, &istioCR)

			//then
			Expect(err).To(Not(HaveOccurred()))

			deployment, err := getIstioIngressDeployment(client)
			Expect(err).To(Not(HaveOccurred()))
			Expect(deployment.Spec.Template.Annotations).ToNot(HaveKey("istio-operator.kyma-project.io/restartedAt"))
		})

		It("should restart when CR has numTrustedProxy configured and lastAppliedConfig annotation is not set", func() {
			//given
			client := createFakeClientWithDeployment()
			newNumTrustedProxies := 1
			istioCR := operatorv1alpha1.Istio{}
			istioCR.Spec.Config.NumTrustedProxies = &newNumTrustedProxies

			//when
			err := restartIngressGatewayIfNeeded(context.TODO(), client, &istioCR)

			//then
			Expect(err).To(Not(HaveOccurred()))

			deployment, err := getIstioIngressDeployment(client)
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

func getIstioIngressDeployment(client client.Client) (appsv1.Deployment, error) {
	deployment := appsv1.Deployment{}
	err := client.Get(context.TODO(), types.NamespacedName{Namespace: "istio-system", Name: "istio-ingressgateway"}, &deployment)
	return deployment, err
}
