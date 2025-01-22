package sidecars_test

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/internal/restarter/predicates"
	"github.com/kyma-project/istio/operator/internal/tests"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	ginkgotypes "github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRestartProxies(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Proxy Restart Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report ginkgotypes.Report) {
	tests.GenerateGinkgoJunitReport("proxy-restart-suite", report)
})

var _ = Describe("RestartProxies", func() {
	ctx := context.Background()
	logger := logr.Discard()

	It("should succeed without warnings", func() {
		// given
		c := fakeClient(&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-namespace",
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "ReplicaSet"},
				},
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			Status: v1.PodStatus{
				Phase: "Running",
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:      "istio-proxy",
						Image:     "istio/istio-proxy:1.0.0",
						Resources: helpers.DefaultSidecarResources,
					},
				},
			},
		})

		// when
		proxyRestarter := sidecars.NewProxyRestarter()
		expectedImage := predicates.NewSidecarImage("istio", "1.1.0")
		istioCR := helpers.GetIstioCR(expectedImage.Tag)
		warnings, hasMorePods, err := proxyRestarter.RestartProxies(ctx, c, expectedImage, helpers.DefaultSidecarResources, &istioCR, &logger)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).To(BeEmpty())
		Expect(hasMorePods).To(BeFalse())
	})
})

func fakeClient(objects ...client.Object) client.Client {
	err := v1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = appsv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(objects...).
		WithIndex(&v1.Pod{}, "status.phase", helpers.FakePodStatusPhaseIndexer).
		Build()

	return fakeClient
}

// type shouldFailClient struct {
// 	client.Client
// 	FailOnGet   bool
// 	FailOnPatch bool
// }

// func (p *shouldFailClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
// 	if p.FailOnGet {
// 		return errors.New("intentionally failing client on client.Get")
// 	}
// 	return p.Client.Get(ctx, key, obj, opts...)
// }

// func (p *shouldFailClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
// 	if p.FailOnPatch {
// 		return errors.New("intentionally failing client on client.Patch")
// 	}
// 	return p.Client.Patch(ctx, obj, patch, opts...)
// }
