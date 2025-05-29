package webhooks

import (
	"context"
	"testing"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/tests"
	. "github.com/onsi/ginkgo/v2"
	gingkoTypes "github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	networkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	"istio.io/istio/istioctl/pkg/tag"
	v1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	revLabelKey   = "istio.io/rev"
	tagLabelKey   = "istio.io/tag"
	defaultWhName = "istio-sidecar-injector"
	taggedWhName  = "istio-revision-tag-default"
)

func TestProxies(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Merge Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report gingkoTypes.Report) {
	tests.GenerateGinkgoJunitReport("istio-webhook-suite", report)
})

var validSelector = &metav1.LabelSelector{
	MatchExpressions: []metav1.LabelSelectorRequirement{{
		Key:      "istio-injection",
		Operator: "DoesNotExist",
	}},
}

var deactivatedSelector = &metav1.LabelSelector{
	MatchLabels: GetDeactivatedLabel(),
}

func createFakeClient(objects ...client.Object) client.Client {
	err := operatorv1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = appsv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = networkingv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()
}

func createMutatingWebhookWithSelector(whConfName string, labels map[string]string, selector *metav1.LabelSelector) *v1.MutatingWebhookConfiguration {
	return &v1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:   whConfName,
			Labels: labels,
		},
		Webhooks: []v1.MutatingWebhook{
			{
				Name:              "object.sidecar-injector.istio.io",
				NamespaceSelector: selector,
				ObjectSelector:    selector,
			},
		},
	}
}

var _ = Describe("DeleteConflictedDefaultTag", func() {
	ctx := context.Background()

	defer ctx.Done()

	It("should not delete tagged webhook when old webhook is deactivated", func() {
		// given
		defaultMwcObj := createMutatingWebhookWithSelector(defaultWhName, map[string]string{revLabelKey: tag.DefaultRevisionName}, deactivatedSelector)
		taggedMwcObj := createMutatingWebhookWithSelector(taggedWhName, map[string]string{tagLabelKey: tag.DefaultRevisionName, revLabelKey: tag.DefaultRevisionName}, validSelector)
		kubeclient := createFakeClient(defaultMwcObj, taggedMwcObj)
		// when
		err := DeleteConflictedDefaultTag(ctx, kubeclient)

		// then
		Expect(err).NotTo(HaveOccurred())

		var taggedMwc v1.MutatingWebhookConfiguration
		err = kubeclient.Get(ctx, types.NamespacedName{Name: taggedWhName}, &taggedMwc)
		Expect(err).NotTo(HaveOccurred())
		Expect(taggedMwcObj.Name).To(Equal(taggedMwc.Name))

		var defaultMwc v1.MutatingWebhookConfiguration
		err = kubeclient.Get(ctx, types.NamespacedName{Name: defaultWhName}, &defaultMwc)
		Expect(err).NotTo(HaveOccurred())
		Expect(defaultMwcObj.Name).To(Equal(defaultMwc.Name))
	})
	It("should delete conflicted tagged webhook if old one is not deactivated", func() {
		// given
		defaultMwcObj := createMutatingWebhookWithSelector(defaultWhName, map[string]string{revLabelKey: tag.DefaultRevisionName}, validSelector)
		taggedMwcObj := createMutatingWebhookWithSelector(taggedWhName, map[string]string{tagLabelKey: tag.DefaultRevisionName, revLabelKey: tag.DefaultRevisionName}, validSelector)
		kubeclient := createFakeClient(defaultMwcObj, taggedMwcObj)
		// when
		err := DeleteConflictedDefaultTag(ctx, kubeclient)

		// then
		Expect(err).NotTo(HaveOccurred())
		var taggedMwc v1.MutatingWebhookConfiguration
		err = kubeclient.Get(ctx, types.NamespacedName{Name: taggedWhName}, &taggedMwc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not found"))

		var defaultMwc v1.MutatingWebhookConfiguration
		err = kubeclient.Get(ctx, types.NamespacedName{Name: defaultWhName}, &defaultMwc)
		Expect(err).NotTo(HaveOccurred())
		Expect(defaultMwcObj.Name).To(Equal(defaultMwc.Name))
	})
	It("should not delete tagged webhook if there is no old default webhook", func() {
		// given
		taggedMwcObj := createMutatingWebhookWithSelector(taggedWhName, map[string]string{tagLabelKey: tag.DefaultRevisionName, revLabelKey: tag.DefaultRevisionName}, validSelector)
		kubeclient := createFakeClient(taggedMwcObj)

		// when
		err := DeleteConflictedDefaultTag(ctx, kubeclient)

		// then
		Expect(err).NotTo(HaveOccurred())
		var taggedMwc v1.MutatingWebhookConfiguration
		err = kubeclient.Get(ctx, types.NamespacedName{Name: taggedWhName}, &taggedMwc)
		Expect(err).NotTo(HaveOccurred())
		Expect(taggedMwcObj.Name).To(Equal(taggedMwc.Name))
	})
	It("should not return an error if there is no tagged webhook", func() {
		// given
		kubeclient := createFakeClient()

		// when
		err := DeleteConflictedDefaultTag(ctx, kubeclient)

		// then
		Expect(err).NotTo(HaveOccurred())
	})
})
