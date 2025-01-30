package remove_test

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/internal/tests"
	. "github.com/onsi/ginkgo/v2"
	ginkgotypes "github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"

	sidecarRemover "github.com/kyma-project/istio/operator/pkg/lib/sidecars/remove"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/test/helpers"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const restartAnnotationName = "istio-operator.kyma-project.io/restartedAt"

func TestRestartPods(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pods Restart Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report ginkgotypes.Report) {
	tests.GenerateGinkgoJunitReport("remove-sidecar-suite", report)
})

var _ = Describe("Remove Sidecar", func() {
	ctx := context.Background()
	logger := logr.Discard()

	It("should rollout restart Deployment if the pod has sidecar", func() {
		// given
		c := fakeClient(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "test-ns"}})
		err := c.Create(ctx, helpers.NewSidecarPodBuilder().
			SetOwnerReference(metav1.OwnerReference{Kind: "Deployment", Name: "owner"}).
			SetNamespace("test-ns").
			Build())

		Expect(err).NotTo(HaveOccurred())

		// when
		warnings, err := sidecarRemover.RemoveSidecars(ctx, c, &logger)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).To(BeEmpty())

		obj := appsv1.Deployment{}
		err = c.Get(context.Background(), types.NamespacedName{Namespace: "test-ns", Name: "owner"}, &obj)
		Expect(err).NotTo(HaveOccurred())

		Expect(obj.Spec.Template.Annotations[restartAnnotationName]).NotTo(BeEmpty())
	})

	It("should not rollout restart Deployment if the pod doesn't have sidecar", func() {
		// given
		c := fakeClient(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "test-ns"}})
		err := c.Create(ctx, helpers.NewSidecarPodBuilder().DisableSidecar().
			SetOwnerReference(metav1.OwnerReference{Kind: "Deployment", Name: "owner"}).
			SetNamespace("test-ns").
			Build())

		Expect(err).NotTo(HaveOccurred())

		// when
		warnings, err := sidecarRemover.RemoveSidecars(ctx, c, &logger)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).To(BeEmpty())

		obj := appsv1.Deployment{}
		err = c.Get(context.Background(), types.NamespacedName{Namespace: "test-ns", Name: "owner"}, &obj)
		Expect(err).NotTo(HaveOccurred())

		Expect(obj.Spec.Template.Annotations[restartAnnotationName]).To(BeEmpty())
	})
})

func fakeClient(objects ...client.Object) client.Client {
	err := v1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = appsv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()

	return fakeClient
}
