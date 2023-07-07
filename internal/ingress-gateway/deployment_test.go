package ingressgateway_test

import (
	"context"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	ingressgateway "github.com/kyma-project/istio/operator/internal/ingress-gateway"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Istio GW Deployment", func() {
	istio := operatorv1alpha1.Istio{}

	Context("NeedsRestart with CM", func() {
		client := CreateFakeClientWithIGW(TestConfigMap)

		It("should restart when CR numTrustedProxies is 2 and CM has configuration for 3", func() {
			//given
			newNumTrustedProxies := 2
			istio.Spec.Config.NumTrustedProxies = &newNumTrustedProxies

			//when
			does, err := ingressgateway.NeedsRestart(context.TODO(), client, &istio)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(does).To(BeTrue())
		})

		It("should restart when CR numTrustedProxies is nil and CM has configuration for numTrustedProxies:3", func() {
			//given
			istio.Spec.Config.NumTrustedProxies = nil

			//when
			does, err := ingressgateway.NeedsRestart(context.TODO(), client, &istio)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(does).To(BeTrue())
		})

		It("should not restart when CR numTrustedProxies is 3 and CM has configuration for numTrustedProxies:3", func() {
			//given
			sameNumTrustedProxies := 3
			istio.Spec.Config.NumTrustedProxies = &sameNumTrustedProxies

			//when
			does, err := ingressgateway.NeedsRestart(context.TODO(), client, &istio)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(does).To(BeFalse())
		})

		It("should not restart when Istio CR is nil", func() {
			//given
			newNumTrustedProxies := 2
			istio.Spec.Config.NumTrustedProxies = &newNumTrustedProxies

			//when
			does, err := ingressgateway.NeedsRestart(context.TODO(), client, nil)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(does).To(BeFalse())
		})
	})

	Context("NeedsRestart with empty or missing CM", func() {
		client := CreateFakeClientWithIGW(TestConfigMapEmpty)

		It("should restart when CR has numTrustedProxy configured, and CM doesn't have configuration for numTrustedProxies", func() {
			//given
			newNumTrustedProxies := 2
			istio.Spec.Config.NumTrustedProxies = &newNumTrustedProxies

			//when
			does, err := ingressgateway.NeedsRestart(context.TODO(), client, &istio)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(does).To(BeTrue())
		})

		It("should not restart when CR doesn't configure numTrustedProxies, and CM doesn't have configuration for numTrustedProxies", func() {
			//given
			istio.Spec.Config.NumTrustedProxies = nil

			//when
			does, err := ingressgateway.NeedsRestart(context.TODO(), client, &istio)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(does).To(BeFalse())
		})

		It("should restart when there's no CM", func() {
			//given
			client := CreateFakeClientWithIGW()

			//when
			does, err := ingressgateway.NeedsRestart(context.TODO(), client, &istio)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(does).To(BeTrue())
		})
	})

	Context("RestartDeployment", func() {
		client := CreateFakeClientWithIGW(TestConfigMap)

		It("should set annotation on Istio IG deployment when restart is needed", func() {
			//given
			istio.Spec.Config.NumTrustedProxies = nil

			//when
			err := ingressgateway.RestartDeployment(context.TODO(), client)
			Expect(err).To(Not(HaveOccurred()))

			dep := appsv1.Deployment{}
			err = client.Get(context.TODO(), types.NamespacedName{Namespace: "istio", Name: "istio-ingressgateway"}, &dep)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(dep.Spec.Template.Annotations["reconciler.kyma-project.io/lastRestartDate"]).ToNot(BeEmpty())
		})
	})
})
