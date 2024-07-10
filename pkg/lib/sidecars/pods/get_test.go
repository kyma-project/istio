package pods_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/istio/operator/internal/filter"
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

func TestPods(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pods Get Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("pods-get-suite", report)
})

var _ = Describe("GetPodsToRestart", func() {
	ctx := context.Background()
	logger := logr.Discard()

	When("Istio image changed", func() {
		expectedImage := pods.NewSidecarImage("istio", "1.10.0")
		tests := []struct {
			name       string
			c          client.Client
			predicates []filter.SidecarProxyPredicate
			limit      int
			assertFunc func(podList *v1.PodList)
		}{
			// {
			// 	name:  "Should not return pods without istio sidecar",
			// 	c:     createClientSet(helpers.FixPodWithoutSidecar("app", "custom")),
			// 	limit: 10,
			// 	assertFunc: func(podList *v1.PodList) {
			// 		Expect(podList.Items).To(BeEmpty())
			// 	},
			// },
			// {
			// 	name: "Should not return any pod when pods have correct image",
			// 	c: createClientSet(
			// 		helpers.NewSidecarPodBuilder().Build(),
			// 	),
			// 	limit: 10,
			// 	assertFunc: func(podList *v1.PodList) {
			// 		Expect(podList.Items).To(BeEmpty())
			// 	},
			// },
			// {
			// 	name: "Should return pod with different image repository",
			// 	c: createClientSet(
			// 		helpers.NewSidecarPodBuilder().Build(),
			// 		helpers.NewSidecarPodBuilder().
			// 			SetName("changedSidecarPod").
			// 			SetSidecarImageRepository("istio/different-proxy").
			// 			Build(),
			// 	),
			// 	limit: 10,
			// 	assertFunc: func(podList *v1.PodList) {
			// 		Expect(podList.Items).To(HaveLen(1))
			// 		Expect(podList.Items[0].Name).To(Equal("changedSidecarPod"))
			// 	},
			// },
			// {
			// 	name: "Should return pod with different image tag",
			// 	c: createClientSet(
			// 		helpers.NewSidecarPodBuilder().Build(),
			// 		helpers.NewSidecarPodBuilder().
			// 			SetName("changedSidecarPod").
			// 			SetSidecarImageTag("1.11.0").
			// 			Build(),
			// 	),
			// 	limit: 10,
			// 	assertFunc: func(podList *v1.PodList) {
			// 		Expect(podList.Items).To(HaveLen(1))
			// 		Expect(podList.Items[0].Name).To(Equal("changedSidecarPod"))
			// 	},
			// },
			// {
			// 	name: "Should ignore pod that has different image tag when it has not all condition status as True",
			// 	c: createClientSet(
			// 		helpers.NewSidecarPodBuilder().
			// 			SetSidecarImageTag("1.12.0").
			// 			SetConditionStatus("False").
			// 			Build(),
			// 	),
			// 	limit: 10,
			// 	assertFunc: func(podList *v1.PodList) {
			// 		Expect(podList.Items).To(BeEmpty())
			// 	},
			// },
			// {
			// 	name: "Should ignore pod that has different image tag when phase is not running",
			// 	c: createClientSet(
			// 		helpers.NewSidecarPodBuilder().
			// 			SetSidecarImageTag("1.12.0").
			// 			SetPodStatusPhase("Pending").
			// 			Build(),
			// 	),
			// 	limit: 10,
			// 	assertFunc: func(podList *v1.PodList) {
			// 		Expect(podList.Items).To(BeEmpty())
			// 	},
			// },
			// {
			// 	name: "Should ignore pod that has different image tag when it has a deletion timestamp",
			// 	c: createClientSet(
			// 		helpers.NewSidecarPodBuilder().
			// 			SetSidecarImageTag("1.12.0").
			// 			SetDeletionTimestamp(time.Now()).
			// 			Build(),
			// 	),
			// 	limit: 10,
			// 	assertFunc: func(podList *v1.PodList) {
			// 		Expect(podList.Items).To(BeEmpty())
			// 	},
			// },
			// {
			// 	name: "Should ignore pod that has different image tag when proxy container name is not in istio annotation",
			// 	c: createClientSet(
			// 		helpers.NewSidecarPodBuilder().
			// 			SetSidecarImageTag("1.12.0").
			// 			SetSidecarContainerName("custom-sidecar-proxy-container-name").
			// 			Build(),
			// 	),
			// 	limit: 10,
			// 	assertFunc: func(podList *v1.PodList) {
			// 		Expect(podList.Items).To(BeEmpty())
			// 	},
			// },
			// {
			// 	name: "Should contain only one pod when there are multiple predicates that would restart the pod",
			// 	c: createClientSet(
			// 		helpers.NewSidecarPodBuilder().
			// 			SetName("changedSidecarPod").
			// 			SetSidecarImageRepository("istio/different-proxy").
			// 			Build(),
			// 	),
			// 	limit:      10,
			// 	predicates: []filter.SidecarProxyPredicate{pods.NewRestartProxyPredicate(expectedImage, helpers.DefaultSidecarResources)},
			// 	assertFunc: func(podList *v1.PodList) {
			// 		Expect(podList.Items).To(HaveLen(1))
			// 	},
			// },
			// {
			// 	name: "Should respect limit set when getting pods to restart",
			// 	c: NewFakeClientWithLimit(
			// 		createClientSet(
			// 			helpers.NewSidecarPodBuilder().
			// 				SetName("changedSidecarPod1").
			// 				SetSidecarImageRepository("istio/different-proxy").
			// 				Build(),
			// 			helpers.NewSidecarPodBuilder().
			// 				SetName("changedSidecarPod2").
			// 				SetSidecarImageRepository("istio/different-proxy").
			// 				Build(),
			// 		), 1),
			// 	limit: 1,
			// 	assertFunc: func(podList *v1.PodList) {
			// 		Expect(podList.Items).To(HaveLen(1))
			// 		Expect(podList.Items[0].Name).To(Equal("changedSidecarPod1"))
			// 		Expect(podList.Continue).To(Equal("continue"))
			// 	},
			// },
			{
				name: "Should respect limit and use continue token to obtain rest of pods when getting pods to restart",
				c: NewFakeClientWithLimit(
					createClientSet(
						helpers.NewSidecarPodBuilder().
							SetName("changedSidecarPod1").
							SetSidecarImageRepository("istio/different-proxy").
							Build(),
						helpers.NewSidecarPodBuilder().Build(),
						helpers.NewSidecarPodBuilder().
							SetName("changedSidecarPod2").
							SetSidecarImageRepository("istio/different-proxy").
							Build(),
					), 3),
				limit: 3,
				assertFunc: func(podList *v1.PodList) {
					Expect(podList.Items).To(HaveLen(2))
					Expect(podList.Items[0].Name).To(Equal("changedSidecarPod1"))
					Expect(podList.Items[1].Name).To(Equal("changedSidecarPod2"))
					Expect(podList.Continue).To(BeEmpty())
				},
			},
		}
		for _, tt := range tests {
			FIt(tt.name, func() {
				podList, err := pods.GetPodsToRestart(ctx, tt.c, expectedImage, helpers.DefaultSidecarResources, tt.predicates, tt.limit, &logger)
				Expect(err).NotTo(HaveOccurred())
				tt.assertFunc(podList)
			})
		}
	})

	When("Sidecar Resources changed", func() {
		tests := []struct {
			name       string
			c          client.Client
			assertFunc func(podList *v1.PodList)
		}{
			{
				name: "Should not return any pod when pods have same resources",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().Build(),
				),
				assertFunc: func(podList *v1.PodList) { Expect(podList.Items).To(BeEmpty()) },
			},
			{
				name: "Should return pod with different sidecar resources",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().Build(),
					helpers.NewSidecarPodBuilder().
						SetName("changedSidecarPod").
						SetCpuRequest("400m").
						Build(),
				),
				assertFunc: func(podList *v1.PodList) {
					Expect(podList.Items).To(HaveLen(1))
					Expect(podList.Items[0].Name).To(Equal("changedSidecarPod"))
				},
			},
			{
				name: "Should ignore pod that has different resources when it has not all condition status as True",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().
						SetConditionStatus("False").
						SetCpuRequest("400m").
						Build(),
				),
				assertFunc: func(podList *v1.PodList) { Expect(podList.Items).To(BeEmpty()) },
			},
			{
				name: "Should ignore pod that has different resources when phase is not running",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().
						SetPodStatusPhase("Pending").
						SetCpuRequest("400m").
						Build(),
				),
				assertFunc: func(podList *v1.PodList) { Expect(podList.Items).To(BeEmpty()) },
			},
			{
				name: "Should ignore pod that has different resources when it has a deletion timestamp",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().
						SetDeletionTimestamp(time.Now()).
						SetCpuRequest("400m").
						Build(),
				),
				assertFunc: func(podList *v1.PodList) { Expect(podList.Items).To(BeEmpty()) },
			},
			{
				name: "Should ignore pod that with different resources when proxy container name is not in istio annotation",
				c: createClientSet(
					helpers.NewSidecarPodBuilder().
						SetSidecarContainerName("custom-sidecar-proxy-container-name").
						SetCpuRequest("400m").
						Build(),
				),
				assertFunc: func(podList *v1.PodList) { Expect(podList.Items).To(BeEmpty()) },
			},
		}
		for _, tt := range tests {
			It(tt.name, func() {
				expectedImage := pods.NewSidecarImage("istio", "1.10.0")
				podList, err := pods.GetPodsToRestart(ctx, tt.c, expectedImage, helpers.DefaultSidecarResources, []filter.SidecarProxyPredicate{}, 10, &logger)
				Expect(err).NotTo(HaveOccurred())
				tt.assertFunc(podList)
			})
		}
	})
})

var _ = Describe("GetAllInjectedPods", func() {
	ctx := context.Background()

	tests := []struct {
		name       string
		c          client.Client
		assertFunc func(podList *v1.PodList)
	}{
		{
			name:       "Should not return pods without istio sidecar",
			c:          createClientSet(helpers.FixPodWithoutSidecar("app", "custom")),
			assertFunc: func(podList *v1.PodList) { Expect(podList.Items).To(BeEmpty()) },
		},
		{
			name: "Should return pod with istio sidecar",
			c: createClientSet(
				helpers.NewSidecarPodBuilder().Build(),
			),
			assertFunc: func(podList *v1.PodList) { Expect(podList.Items).To(HaveLen(1)) },
		},
		{
			name: "Should not return pod with only istio sidecar",
			c: createClientSet(
				helpers.FixPodWithOnlySidecar("app", "custom"),
			),
			assertFunc: func(podList *v1.PodList) { Expect(podList.Items).To(HaveLen(0)) },
		},
	}
	for _, tt := range tests {
		It(tt.name, func() {
			podList, err := pods.GetAllInjectedPods(ctx, tt.c)
			Expect(err).NotTo(HaveOccurred())
			tt.assertFunc(podList)
		})
	}
})

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

type fakeClientWithLimit struct {
	client.Client
	limit                   int64
	callCount               int
	continueTokenOnNextCall bool
}

func NewFakeClientWithLimit(c client.Client, limit int64) *fakeClientWithLimit {
	return &fakeClientWithLimit{
		Client:                  c,
		limit:                   limit,
		callCount:               0,
		continueTokenOnNextCall: false,
	}
}

func (p *fakeClientWithLimit) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	p.callCount++

	foundLimitOpt := false
	continueToken := ""

	for _, opt := range opts {
		switch opt.(type) {
		case client.Limit:
			limit := opt.(client.Limit)
			if int64(limit) != p.limit {
				return errors.New("limit not set as expected")
			}
			foundLimitOpt = true
		case client.Continue:
			continueToken = string(opt.(client.Continue))
		}
	}

	if !foundLimitOpt {
		return errors.New("limit not set when listing pods")
	}

	podList, ok := list.(*v1.PodList)
	if !ok {
		return errors.New("list is not a pod list")
	}

	fmt.Printf("call number: %d\n", p.callCount)
	fmt.Printf("continue token: %s\n", continueToken)

	switch p.callCount {
	case 1:
		if podList.Continue != "" {
			return errors.New("continue token should be empty on the first call")
		}
	case 2:
		if p.continueTokenOnNextCall && continueToken != "continue" {
			return errors.New("continue token should be set correctly on the second call")
		}
	}

	err := p.Client.List(ctx, list, opts...)
	if err != nil {
		return err
	}

	if len(podList.Items) > int(p.limit) {
		if podList.Continue == "" {
			podList.Continue = "continue"
			podList.Items = podList.Items[:p.limit]
			p.continueTokenOnNextCall = true
		} else {
			podList.Items = podList.Items[p.limit:]
			podList.Continue = ""
			p.continueTokenOnNextCall = false
		}
	}

	return nil
}
