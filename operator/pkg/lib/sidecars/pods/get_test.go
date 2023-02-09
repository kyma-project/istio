package pods_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
	"time"

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
		// TODO: WithIndex is not supported in current version of controller runtime, should be readded later on
		// WithIndex(&v1.Pod{}, "status.phase", helpers.FakePodStatusPhaseIndexer).
		Build()

	return fakeClient
}

func TestGetPods(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pods Get Suite")
}

var _ = Describe("Get Pods", func() {

	ctx := context.Background()
	logger := logr.Discard()

	When("CNI configuration changed", func() {

		enabledNamespace := helpers.FixNamespaceWith("enabled", map[string]string{"istio-injection": "enabled"})
		disabledNamespace := helpers.FixNamespaceWith("disabled", map[string]string{"istio-injection": "disabled"})

		tests := []struct {
			name             string
			c                client.Client
			expectedPodNames []string
			isCNIEnabled     bool
			wantEmpty        bool
			wantLen          int
		}{
			{
				name: "should not get any pod without istio-init container when CNI is enabled",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().SetName("application1").SetNamespace("enabled").
						SetInitContainer("istio-validation").SetPodStatusPhase("Running").Build(),
					helpers.NewSidecarPodBuilder().SetName("application2").SetNamespace("enabled").
						SetInitContainer("istio-validation").SetPodStatusPhase("Terminating").Build(),
					enabledNamespace,
				),
				expectedPodNames: []string{},
				isCNIEnabled:     true,
				wantEmpty:        true,
				wantLen:          0,
			},
			{
				name: "should not get pods in system namespaces when CNI is enabled",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().SetName("application1").SetNamespace("kube-system").Build(),
					helpers.NewSidecarPodBuilder().SetName("application2").SetNamespace("kube-public").Build(),
					helpers.NewSidecarPodBuilder().SetName("application3").SetNamespace("istio-system").Build(),
					helpers.NewSidecarPodBuilder().SetName("application4").SetNamespace("enabled").Build(),
					enabledNamespace,
				),
				expectedPodNames: []string{"application4"},
				isCNIEnabled:     true,
				wantEmpty:        false,
				wantLen:          1,
			},
			{
				name: "should get 2 pods with istio-init when they are in Running state when CNI is enabled",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().SetName("application1").SetNamespace("enabled").Build(),
					helpers.NewSidecarPodBuilder().SetName("application2").SetNamespace("enabled").Build(),
					enabledNamespace,
				),
				expectedPodNames: []string{"application1", "application2"},
				isCNIEnabled:     true,
				wantEmpty:        false,
				wantLen:          2,
			},
			{
				name: "should not get pod with istio-init in Terminating state",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().SetName("application1").SetNamespace("enabled").Build(),
					helpers.NewSidecarPodBuilder().SetName("application2").SetNamespace("enabled").
						SetPodStatusPhase("Terminating").Build(),
					enabledNamespace,
				),
				expectedPodNames: []string{"application1"},

				isCNIEnabled: true,
				wantEmpty:    false,
				wantLen:      1,
			},
			{
				name: "should not get pod with istio-validation container when CNI is enabled",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().SetName("application1").SetNamespace("enabled").Build(),
					helpers.NewSidecarPodBuilder().SetName("application2").SetNamespace("enabled").
						SetInitContainer("istio-validation").Build(),
					enabledNamespace,
				),
				expectedPodNames: []string{"application1"},
				isCNIEnabled:     true,
				wantEmpty:        false,
				wantLen:          1,
			},
			{
				name: "should get 2 pods with istio-validation container when CNI is disabled",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().SetName("application1").SetNamespace("enabled").
						SetInitContainer("istio-validation").Build(),
					helpers.NewSidecarPodBuilder().SetName("application2").SetNamespace("enabled").
						SetInitContainer("istio-validation").Build(),
					enabledNamespace,
				),
				expectedPodNames: []string{"application1", "application2"},
				isCNIEnabled:     false,
				wantEmpty:        false,
				wantLen:          2,
			},
			{
				name: "should not get any pod with istio-validation container in disabled namespace when CNI is disabled",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().SetName("application1").SetNamespace("disabled").
						SetInitContainer("istio-validation").Build(),
					disabledNamespace,
				),
				expectedPodNames: []string{},
				isCNIEnabled:     false,
				wantEmpty:        true,
				wantLen:          0,
			},
		}

		for _, tt := range tests {
			It(tt.name, func() {
				podList, err := pods.GetPodsForCNIChange(ctx, tt.c, tt.isCNIEnabled, &logger)
				Expect(err).NotTo(HaveOccurred())

				if tt.wantEmpty {
					Expect(podList.Items).To(BeEmpty())
				} else {
					Expect(podList.Items).NotTo(BeEmpty())
				}

				for _, pod := range podList.Items {
					Expect(tt.expectedPodNames).To(ContainElement(pod.Name))
				}

				Expect(podList.Items).To(HaveLen(tt.wantLen))
			})
		}
	})

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
				podList, err := pods.GetPodsWithDifferentSidecarImage(ctx, tt.c, expectedImage, &logger)

				Expect(err).NotTo(HaveOccurred())
				tt.assertFunc(podList.Items)
			})
		}
	})
})
