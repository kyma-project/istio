package pods_test

import (
	"context"
	"github.com/kyma-project/istio/operator/internal/filter"
	"testing"
	"time"

	"github.com/kyma-project/istio/operator/internal/tests"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/test/helpers"
)

func createClientSet(objects ...client.Object) client.Client {
	err := v1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = v1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(objects...).
		WithIndex(&v1.Pod{}, "status.phase", helpers.FakePodStatusPhaseIndexer).
		Build()

	return fakeClient
}

func TestGetPods(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pods Get Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("pods-get-suite", report)
})

var _ = Describe("Get Pods", func() {

	ctx := context.Background()
	logger := logr.Discard()

	When("Istio image changed", func() {

		expectedImage := pods.SidecarImage{
			Repository: "istio/proxyv2",
			Tag:        "1.10.0",
		}

		tests := []struct {
			name       string
			c          client.Client
			assertFunc func(val interface{})
		}{
			{
				name:       "should not return pods without istio sidecar",
				c:          createClientSet(helpers.FixPodWithoutSidecar("app", "custom")),
				assertFunc: func(val interface{}) { Expect(val).To(BeEmpty()) },
			},
			{
				name: "should not return any pod when pods have correct image",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().Build(),
				),
				assertFunc: func(val interface{}) { Expect(val).To(BeEmpty()) },
			},
			{
				name: "should return pod with different image repository",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().Build(),
					helpers.NewSidecarPodBuilder().
						SetName("changedSidecarPod").
						SetSidecarImageRepository("istio/different-proxy").
						Build(),
				),
				assertFunc: func(val interface{}) {
					Expect(val).NotTo(BeEmpty())
					resultPods := val.([]v1.Pod)
					Expect(resultPods[0].Name).To(Equal("changedSidecarPod"))
				},
			},
			{
				name: "should return pod with different image tag",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().Build(),
					helpers.NewSidecarPodBuilder().
						SetName("changedSidecarPod").
						SetSidecarImageTag("1.11.0").
						Build(),
				),
				assertFunc: func(val interface{}) {
					Expect(val).NotTo(BeEmpty())
					resultPods := val.([]v1.Pod)
					Expect(resultPods[0].Name).To(Equal("changedSidecarPod"))

				},
			},
			{
				name: "should ignore pod that has different image tag when it has not all condition status as True",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().
						SetSidecarImageTag("1.12.0").
						SetConditionStatus("False").
						Build(),
				),
				assertFunc: func(val interface{}) { Expect(val).To(BeEmpty()) },
			},
			{
				name: "should ignore pod that has different image tag when phase is not running",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().
						SetSidecarImageTag("1.12.0").
						SetPodStatusPhase("Pending").
						Build(),
				),
				assertFunc: func(val interface{}) { Expect(val).To(BeEmpty()) },
			},
			{
				name: "should ignore pod that has different image tag when it has a deletion timestamp",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().
						SetSidecarImageTag("1.12.0").
						SetDeletionTimestamp(time.Now()).
						Build(),
				),
				assertFunc: func(val interface{}) { Expect(val).To(BeEmpty()) },
			},
			{
				name: "should ignore pod that has different image tag when proxy container name is not in istio annotation",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().
						SetSidecarImageTag("1.12.0").
						SetSidecarContainerName("custom-sidecar-proxy-container-name").
						Build(),
				),
				assertFunc: func(val interface{}) { Expect(val).To(BeEmpty()) },
			},
		}

		for _, tt := range tests {
			It(tt.name, func() {
				podList, err := pods.GetPodsToRestart(ctx, tt.c, expectedImage, helpers.DefaultSidecarResources, []filter.SidecarProxyPredicate{}, &logger)

				Expect(err).NotTo(HaveOccurred())
				tt.assertFunc(podList.Items)
			})
		}
	})

	When("Sidecar Resources changed", func() {

		tests := []struct {
			name       string
			c          client.Client
			assertFunc func(val interface{})
		}{
			{
				name: "should not return any pod when pods have same resources",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().Build(),
				),
				assertFunc: func(val interface{}) { Expect(val).To(BeEmpty()) },
			},
			{
				name: "should return pod with different sidecar resources",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().Build(),
					helpers.NewSidecarPodBuilder().
						SetName("changedSidecarPod").
						SetCpuRequest("400m").
						Build(),
				),
				assertFunc: func(val interface{}) {
					Expect(val).NotTo(BeEmpty())
					resultPods := val.([]v1.Pod)
					Expect(resultPods[0].Name).To(Equal("changedSidecarPod"))
				},
			},
			{
				name: "should ignore pod that has different resources when it has not all condition status as True",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().
						SetConditionStatus("False").
						SetCpuRequest("400m").
						Build(),
				),
				assertFunc: func(val interface{}) { Expect(val).To(BeEmpty()) },
			},
			{
				name: "should ignore pod that has different resources when phase is not running",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().
						SetPodStatusPhase("Pending").
						SetCpuRequest("400m").
						Build(),
				),
				assertFunc: func(val interface{}) { Expect(val).To(BeEmpty()) },
			},
			{
				name: "should ignore pod that has different resources when it has a deletion timestamp",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().
						SetDeletionTimestamp(time.Now()).
						SetCpuRequest("400m").
						Build(),
				),
				assertFunc: func(val interface{}) { Expect(val).To(BeEmpty()) },
			},
			{
				name: "should ignore pod that with different resources when proxy container name is not in istio annotation",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().
						SetSidecarContainerName("custom-sidecar-proxy-container-name").
						SetCpuRequest("400m").
						Build(),
				),
				assertFunc: func(val interface{}) { Expect(val).To(BeEmpty()) },
			},
		}

		for _, tt := range tests {
			It(tt.name, func() {
				expectedImage := pods.SidecarImage{
					Repository: "istio/proxyv2",
					Tag:        "1.10.0",
				}

				podList, err := pods.GetPodsToRestart(ctx, tt.c, expectedImage, helpers.DefaultSidecarResources, []filter.SidecarProxyPredicate{}, &logger)

				Expect(err).NotTo(HaveOccurred())
				tt.assertFunc(podList.Items)
			})
		}
	})
})

var _ = Describe("GetAllInjectedPods", func() {

	ctx := context.Background()

	tests := []struct {
		name       string
		c          client.Client
		assertFunc func(val interface{})
	}{
		{
			name:       "should not return pods without istio sidecar",
			c:          createClientSet(helpers.FixPodWithoutSidecar("app", "custom")),
			assertFunc: func(val interface{}) { Expect(val).To(BeEmpty()) },
		},
		{
			name: "should return pod with istio sidecar",
			c: createClientSet(
				helpers.NewSidecarPodBuilder().Build(),
			),
			assertFunc: func(val interface{}) { Expect(val).To(HaveLen(1)) },
		},
		{
			name: "should not return pod with only istio sidecar",
			c: createClientSet(
				helpers.FixPodWithOnlySidecar("app", "custom"),
			),
			assertFunc: func(val interface{}) { Expect(val).To(HaveLen(0)) },
		},
	}

	for _, tt := range tests {
		It(tt.name, func() {
			podList, err := pods.GetAllInjectedPods(ctx, tt.c)

			Expect(err).NotTo(HaveOccurred())
			tt.assertFunc(podList.Items)
		})
	}
})
