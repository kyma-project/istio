package istio_resources

import (
	"bytes"
	"context"
	"github.com/kyma-project/istio/operator/internal/resources"
	"text/template"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Apply", func() {
	templateValues := map[string]string{}
	templateValues["DomainName"] = "example.com"

	owner := metav1.OwnerReference{
		APIVersion: "operator.kyma-project.io/v1alpha2",
		Kind:       "Istio",
		Name:       "owner-name",
		UID:        "owner-uid",
	}

	It("should return created if no resource was present", func() {
		//given
		client := resources.createFakeClient()
		sample := NewVirtualServiceHealthz(client)

		//when
		changed, err := sample.apply(context.TODO(), client, owner, templateValues)

		//then
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultCreated))

		var s networkingv1beta1.VirtualServiceList
		listErr := client.List(context.TODO(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(1))

		Expect(s.Items[0].Annotations).To(Not(BeNil()))
		Expect(s.Items[0].Annotations[resources.DisclaimerKey]).To(Not(BeNil()))
	})

	It("should return not changed if no change was applied", func() {
		//given
		resourceTemplate, err := template.New("tmpl").Option("missingkey=error").Parse(string(manifest_vs_healthz))
		Expect(err).To(Not(HaveOccurred()))

		var resourceBuffer bytes.Buffer
		err = resourceTemplate.Execute(&resourceBuffer, templateValues)
		Expect(err).To(Not(HaveOccurred()))

		var p networkingv1beta1.VirtualService
		err = yaml.Unmarshal(resourceBuffer.Bytes(), &p)
		Expect(err).To(Not(HaveOccurred()))

		client := resources.createFakeClient(&p)
		sample := NewVirtualServiceHealthz(client)

		//when
		changed, err := sample.apply(context.TODO(), client, owner, templateValues)

		//then
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultNone))

		var s networkingv1beta1.VirtualServiceList
		listErr := client.List(context.TODO(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(1))

		Expect(s.Items[0].Annotations).To(Not(BeNil()))
		Expect(s.Items[0].Annotations[resources.DisclaimerKey]).To(Not(BeNil()))
	})

	It("should return updated if change was applied", func() {
		//given
		resourceTemplate, err := template.New("tmpl").Option("missingkey=error").Parse(string(manifest_vs_healthz))
		Expect(err).To(Not(HaveOccurred()))

		var resourceBuffer bytes.Buffer
		err = resourceTemplate.Execute(&resourceBuffer, templateValues)
		Expect(err).To(Not(HaveOccurred()))

		var p networkingv1beta1.VirtualService
		err = yaml.Unmarshal(resourceBuffer.Bytes(), &p)
		Expect(err).To(Not(HaveOccurred()))

		p.Spec.Hosts = append(p.Spec.Hosts, "new-host.com")
		client := resources.createFakeClient(&p)

		sample := NewVirtualServiceHealthz(client)

		//when
		changed, err := sample.apply(context.TODO(), client, owner, templateValues)

		//then
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultUpdated))

		var s networkingv1beta1.VirtualServiceList
		listErr := client.List(context.TODO(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(1))

		Expect(s.Items[0].Annotations).To(Not(BeNil()))
		Expect(s.Items[0].Annotations[resources.DisclaimerKey]).To(Not(BeNil()))
	})
})
