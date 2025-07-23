package istioresources

import (
	"context"

	"github.com/kyma-project/istio/operator/internal/resources"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Apply", func() {
	templateValues := map[string]string{}
	owner := metav1.OwnerReference{
		APIVersion: "operator.kyma-project.io/v1alpha2",
		Kind:       "Istio",
		Name:       "owner-name",
		UID:        "owner-uid",
	}

	It("should return created if no resource was present", func() {
		client := createFakeClient()
		sample := NewProxyProtocolEnvoyFilter(client, false)

		//when
		changed, err := sample.reconcile(context.Background(), client, owner, templateValues)

		//then
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultCreated))

		var s networkingv1alpha3.EnvoyFilterList
		listErr := client.List(context.Background(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(1))

		Expect(s.Items[0].Annotations).To(Not(BeNil()))
		Expect(s.Items[0].Annotations[resources.DisclaimerKey]).To(Not(BeNil()))

		Expect(s.Items[0].GetLabels()).ToNot(BeNil())
		Expect(s.Items[0].GetLabels()).To(HaveLen(6))
		Expect(s.Items[0].GetLabels()).To(HaveKeyWithValue("kyma-project.io/module", "istio"))
		Expect(s.Items[0].GetLabels()).To(HaveKeyWithValue("app.kubernetes.io/name", "istio-operator"))
		Expect(s.Items[0].GetLabels()).To(HaveKeyWithValue("app.kubernetes.io/instance", "istio-operator-default"))
		Expect(s.Items[0].GetLabels()).To(HaveKeyWithValue("app.kubernetes.io/version", "dev"))
		Expect(s.Items[0].GetLabels()).To(HaveKeyWithValue("app.kubernetes.io/component", "operator"))
		Expect(s.Items[0].GetLabels()).To(HaveKeyWithValue("app.kubernetes.io/part-of", "istio"))
	})

	It("should return not changed if no change was applied", func() {
		//given
		client := createFakeClient()
		sample := NewProxyProtocolEnvoyFilter(client, false)

		//when
		changed, err := sample.reconcile(context.Background(), client, owner, templateValues)
		// first reconciliation required because we add app.kubernetes.io/version in runtime, so it's not present in the yaml
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultCreated))

		var s networkingv1alpha3.EnvoyFilterList
		listErr := client.List(context.Background(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(1))

		Expect(s.Items[0].Annotations).To(Not(BeNil()))
		Expect(s.Items[0].Annotations[resources.DisclaimerKey]).To(Not(BeNil()))

		// then
		// we check in the second reconciliation that nothing changed
		sample = NewProxyProtocolEnvoyFilter(client, false)
		changed, err = sample.reconcile(context.Background(), client, owner, templateValues)

		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultNone))

		listErr = client.List(context.Background(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(1))

		Expect(s.Items[0].Annotations).To(Not(BeNil()))
		Expect(s.Items[0].Annotations[resources.DisclaimerKey]).To(Not(BeNil()))
	})

	It("should return updated if change was applied", func() {
		//given
		var p networkingv1alpha3.EnvoyFilter
		err := yaml.Unmarshal(proxyProtocolEnvoyFilter, &p)
		Expect(err).To(Not(HaveOccurred()))

		p.Spec.Priority = 123
		client := createFakeClient(&p)

		sample := NewProxyProtocolEnvoyFilter(client, false)

		//when
		changed, err := sample.reconcile(context.Background(), client, owner, templateValues)

		//then
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultUpdated))

		var s networkingv1alpha3.EnvoyFilterList
		listErr := client.List(context.Background(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(1))

		Expect(s.Items[0].Annotations).To(Not(BeNil()))
		Expect(s.Items[0].Annotations[resources.DisclaimerKey]).To(Not(BeNil()))
	})
})
