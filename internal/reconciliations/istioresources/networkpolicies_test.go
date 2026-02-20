package istioresources

import (
	"context"

	"github.com/kyma-project/istio/operator/internal/resources"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

var _ = Describe("NetworkPolicies", func() {
	templateValues := map[string]string{}
	owner := metav1.OwnerReference{
		APIVersion: "operator.kyma-project.io/v1alpha2",
		Kind:       "Istio",
		Name:       "owner-name",
		UID:        "owner-uid",
	}

	expectedNames := []types.NamespacedName{
		{
			Namespace: "istio-system",
			Name:      "kyma-project.io--istio-cni-node",
		},
		{
			Namespace: "istio-system",
			Name:      "kyma-project.io--istio-ingressgateway-egress",
		},
		{
			Namespace: "istio-system",
			Name:      "kyma-project.io--istio-ingressgateway",
		},
		{
			Namespace: "kyma-system",
			Name:      "kyma-project.io--allow-istio-controller-manager",
		},
		{
			Namespace: "istio-system",
			Name:      "kyma-project.io--istio-pilot",
		},
		{
			Namespace: "istio-system",
			Name:      "kyma-project.io--istio-pilot-jwks",
		},
	}

	It("should return created if no resources were present", func() {
		client := createFakeClient()
		sample := NewNetworkPolicies(client)

		// when
		changed, err := sample.reconcile(context.Background(), client, owner, templateValues)

		// then
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultCreated))

		var policies v1.NetworkPolicyList
		listErr := client.List(context.Background(), &policies)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(policies.Items).To(HaveLen(len(expectedNames)))

		for _, policy := range policies.Items {
			Expect(expectedNames).To(ContainElement(types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}))
			Expect(policy.Annotations).To(Not(BeNil()))
			Expect(policy.Annotations[resources.DisclaimerKey]).To(Not(BeNil()))
			Expect(policy.GetLabels()).To(HaveKeyWithValue("kyma-project.io/module", "istio"))
			Expect(policy.GetLabels()).To(HaveKeyWithValue("kyma-project.io/managed-by", "kyma"))
			Expect(policy.GetLabels()).To(HaveKey("app.kubernetes.io/version"))
		}
	})

	It("should return not changed if no change was applied", func() {
		client := createFakeClient()
		sample := NewNetworkPolicies(client)

		changed, err := sample.reconcile(context.Background(), client, owner, templateValues)
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultCreated))

		sample = NewNetworkPolicies(client)
		changed, err = sample.reconcile(context.Background(), client, owner, templateValues)

		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultNone))

		var policies v1.NetworkPolicyList
		listErr := client.List(context.Background(), &policies)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(policies.Items).To(HaveLen(len(expectedNames)))
	})

	It("should return updated if change was applied", func() {
		// given
		var policy v1.NetworkPolicy
		err := yaml.Unmarshal(allowJwks, &policy)
		Expect(err).To(Not(HaveOccurred()))

		policy.Spec.Egress = nil
		client := createFakeClient(&policy)
		sample := NewNetworkPolicies(client)

		// when
		changed, err := sample.reconcile(context.Background(), client, owner, templateValues)

		// then
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultUpdated))
	})

	It("should delete existing resources when marked for deletion", func() {
		client := createFakeClient()
		sample := NewNetworkPolicies(client)

		changed, err := sample.reconcile(context.Background(), client, owner, templateValues)
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultCreated))

		sample.shouldDelete = true
		changed, err = sample.reconcile(context.Background(), client, owner, templateValues)

		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultUpdated))

		var policies v1.NetworkPolicyList
		listErr := client.List(context.Background(), &policies)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(policies.Items).To(BeEmpty())
	})
})
