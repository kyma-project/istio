package istio_test

import (
	"fmt"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"context"
	"encoding/json"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/ingressgateway"
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

var _ = Describe("CR configuration", func() {
	Context("lastAppliedConfiguration", func() {
		It("should update lastAppliedConfiguration and is able to retrieve it back from annotation", func() {
			// given
			numTrustedProxies := 1
			istio := operatorv1alpha1.Istio{Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{NumTrustedProxies: &numTrustedProxies},
			}}

			// when
			updatedCR, err := istio.UpdateLastAppliedConfiguration(istio, mockIstioTag)

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(updatedCR.Annotations).To(Not(BeEmpty()))
			Expect(updatedCR.Annotations[lastAppliedConfiguration]).To(Equal(fmt.Sprintf(`{"config":{},"IstioTag":"%s"}`, mockIstioTag)))

			appliedConfig, err := istio.GetLastAppliedConfiguration(updatedCR)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(appliedConfig.Config.NumTrustedProxies).To(Equal(1))
		})
	})
})

var _ = Describe("IngressGateway", func() {
	istio := operatorv1alpha1.Istio{}
	client := createFakeClient()

	Context("NeedsRestart with lastAppliedConfig", func() {

		It("should restart when CR numTrustedProxies is 2 and CM has configuration for 3", func() {
			//given
			newNumTrustedProxies := 2
			istio.Spec.Config.NumTrustedProxies = &newNumTrustedProxies
			applyLastAppliedConfiguration(istio, spec, mockIstioTag)

			//when
			does, err := ingressgateway.NeedsRestart(context.TODO(), client, istio)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(does).To(BeTrue())
		})

		It("should restart when CR numTrustedProxies is nil and CM has configuration for numTrustedProxies:3", func() {
			//given
			istio.Spec.Config.NumTrustedProxies = nil

			//when
			does, err := ingressgateway.NeedsRestart(context.TODO(), client, istio)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(does).To(BeTrue())
		})

		It("should not restart when CR numTrustedProxies is 3 and CM has configuration for numTrustedProxies:3", func() {
			//given
			sameNumTrustedProxies := 3
			istio.Spec.Config.NumTrustedProxies = &sameNumTrustedProxies

			//when
			does, err := ingressgateway.NeedsRestart(context.TODO(), client, istio)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(does).To(BeFalse())
		})
	})

	Context("NeedsRestart without lastAppliedConfig", func() {
		client := createFakeClient()

		It("should restart when CR has numTrustedProxy configured, and CM doesn't have configuration for numTrustedProxies", func() {
			//given
			newNumTrustedProxies := 2
			istio.Spec.Config.NumTrustedProxies = &newNumTrustedProxies

			//when
			does, err := ingressgateway.NeedsRestart(context.TODO(), client, istio)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(does).To(BeTrue())
		})

		It("should not restart when CR doesn't configure numTrustedProxies, and CM doesn't have configuration for numTrustedProxies", func() {
			//given
			istio.Spec.Config.NumTrustedProxies = nil

			//when
			does, err := ingressgateway.NeedsRestart(context.TODO(), client, istio)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(does).To(BeFalse())
		})

		It("should restart when there's no CM", func() {
			//given
			client := createFakeClient()

			//when
			does, err := ingressgateway.NeedsRestart(context.TODO(), client, istio)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(does).To(BeTrue())
		})
	})

	Context("RestartDeployment", func() {
		client := createFakeClient()

		It("should set annotation on Istio IG deployment when restart is needed", func() {
			//given
			istio.Spec.Config.NumTrustedProxies = nil

			//when
			err := ingressgateway.RestartDeployment(context.TODO(), client)
			Expect(err).To(Not(HaveOccurred()))

			dep := appsv1.Deployment{}
			err = client.Get(context.TODO(), types.NamespacedName{Namespace: "istio-system", Name: "istio-ingressgateway"}, &dep)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(dep.Spec.Template.Annotations["reconciler.kyma-project.io/lastRestartDate"]).ToNot(BeEmpty())
		})
	})
})

func createFakeClient() client.Client {
	deployment := appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}

	err := corev1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = appsv1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())

	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(&deployment).Build()
}

func applyLastAppliedConfiguration(istio *operatorv1alpha1.Istio, spec operatorv1alpha1.IstioSpec, istioTag string) error {
	lastAppliedConfig := istio.AppliedConfig{
		IstioSpec: spec,
		IstioTag:  istioTag,
	}

	config, err := json.Marshal(lastAppliedConfig)
	if err != nil {
		return err
	}

	istio.Annotations[LastAppliedConfiguration] = string(config)
	return nil
}
