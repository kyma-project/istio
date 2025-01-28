package sidecars_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/internal/restarter/predicates"
	"github.com/kyma-project/istio/operator/internal/tests"
	"github.com/kyma-project/istio/operator/pkg/labels"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/restart"
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

	It("should succeed without errors or warnings", func() {
		// given
		pod := getPod("test-pods", "test-namespace", "podOwner", "ReplicaSet")
		rsOwner := getReplicaSet("podOwner", "test-namespace", "rsOwner", "ReplicaSet")
		rsOwnerRS := getReplicaSet("rsOwner", "test-namespace", "base", "ReplicaSet")

		c := fakeClient(pod, rsOwner, rsOwnerRS)

		// when
		podsLister := pods.NewPods(c, &logger)
		expectedImage := predicates.NewSidecarImage("istio", "1.1.0")
		istioCR := helpers.GetIstioCR(expectedImage.Tag)
		actionRestarter := restart.NewActionRestarter(c, &logger)
		proxyRestarter := sidecars.NewProxyRestarter(c, podsLister, actionRestarter, &logger)
		warnings, hasMorePods, err := proxyRestarter.RestartProxies(ctx, expectedImage, helpers.DefaultSidecarResources, &istioCR)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).To(BeEmpty())
		Expect(hasMorePods).To(BeFalse())

		err = c.Get(ctx, client.ObjectKey{Name: rsOwnerRS.Name, Namespace: rsOwnerRS.Namespace}, rsOwnerRS)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsOwnerRS.Spec.Template.Annotations).To(HaveKey("istio-operator.kyma-project.io/restartedAt"))
	})

	It("should call restart proxies with respective predicates", func() {
		// given
		c := fakeClient()

		// when
		failClient := &shouldFailClient{c, false, true}

		podsListerMock := NewPodsMock()
		expectedImage := predicates.NewSidecarImage("istio", "1.1.0")
		istioCR := helpers.GetIstioCR(expectedImage.Tag)
		actionRestarter := restart.NewActionRestarter(failClient, &logger)
		proxyRestarter := sidecars.NewProxyRestarter(failClient, podsListerMock, actionRestarter, &logger)
		warnings, hasMorePods, err := proxyRestarter.RestartProxies(ctx, expectedImage, helpers.DefaultSidecarResources, &istioCR)

		// then
		Expect(err).ToNot(HaveOccurred())
		Expect(warnings).To(BeEmpty())
		Expect(hasMorePods).To(BeFalse())

		Expect(podsListerMock.Called).To(Equal(2))

		Expect(podsListerMock.Predicates).To(HaveLen(2))
		Expect(podsListerMock.Predicates[0]).To(HaveLen(3))
		Expect(podsListerMock.Predicates[0][0]).To(BeAssignableToTypeOf(&predicates.CompatibilityRestartPredicate{}))
		Expect(podsListerMock.Predicates[0][1]).To(BeAssignableToTypeOf(&predicates.ImageResourcesPredicate{}))
		Expect(podsListerMock.Predicates[0][2]).To(BeAssignableToTypeOf(&predicates.KymaWorkloadRestartPredicate{}))
		Expect(podsListerMock.Predicates[1]).To(HaveLen(3))
		Expect(podsListerMock.Predicates[0][0]).To(BeAssignableToTypeOf(&predicates.CompatibilityRestartPredicate{}))
		Expect(podsListerMock.Predicates[1][1]).To(BeAssignableToTypeOf(&predicates.ImageResourcesPredicate{}))
		Expect(podsListerMock.Predicates[1][2]).To(BeAssignableToTypeOf(&predicates.CustomerWorkloadRestartPredicate{}))

		Expect(podsListerMock.Limits).To(HaveLen(2))
		Expect(podsListerMock.Limits[0].PodsToRestartLimit).To(Equal(math.MaxInt))
		Expect(podsListerMock.Limits[0].PodsToListLimit).To(Equal(math.MaxInt))
		Expect(podsListerMock.Limits[1].PodsToRestartLimit).To(Equal(30))
		Expect(podsListerMock.Limits[1].PodsToListLimit).To(Equal(100))
	})

	It("should return error if compatibility predicate creation fails", func() {
		// given
		c := fakeClient()
		podsListerMock := NewPodsMock()
		expectedImage := predicates.NewSidecarImage("istio", "1.1.0")
		istioCR := helpers.GetIstioCR(expectedImage.Tag)
		istioCR.Annotations[labels.LastAppliedConfiguration] = "invalid-last-applied-configuration" // This should cause the compatibility predicate to fail
		actionRestarter := restart.NewActionRestarter(c, &logger)
		proxyRestarter := sidecars.NewProxyRestarter(c, podsListerMock, actionRestarter, &logger)

		// when
		warnings, hasMorePods, err := proxyRestarter.RestartProxies(ctx, expectedImage, helpers.DefaultSidecarResources, &istioCR)

		// then
		Expect(err).To(HaveOccurred())
		Expect(warnings).To(BeEmpty())
		Expect(hasMorePods).To(BeFalse())
		Expect(err.Error()).To(ContainSubstring("invalid character"))
	})

	It("should return error if restarting Kyma proxies fails", func() {
		// given
		c := fakeClient()
		podsListerMock := NewPodsMock()
		podsListerMock.FailOnKymaWorkload = true
		expectedImage := predicates.NewSidecarImage("istio", "1.1.0")
		istioCR := helpers.GetIstioCR(expectedImage.Tag)
		actionRestarter := restart.NewActionRestarter(c, &logger)
		proxyRestarter := sidecars.NewProxyRestarter(c, podsListerMock, actionRestarter, &logger)

		// when
		warnings, hasMorePods, err := proxyRestarter.RestartProxies(ctx, expectedImage, helpers.DefaultSidecarResources, &istioCR)

		// then
		Expect(err).To(HaveOccurred())
		Expect(warnings).To(BeEmpty())
		Expect(hasMorePods).To(BeFalse())
		Expect(err.Error()).To(ContainSubstring("intentionally failed on Kyma workload predicate"))
	})

	It("should not return error if restarting Customer proxies fails", func() {
		// given
		c := fakeClient()

		podsListerMock := NewPodsMock()
		podsListerMock.FailOnCustomerWorkload = true
		expectedImage := predicates.NewSidecarImage("istio", "1.1.0")
		istioCR := helpers.GetIstioCR(expectedImage.Tag)
		actionRestarter := restart.NewActionRestarter(c, &logger)
		proxyRestarter := sidecars.NewProxyRestarter(c, podsListerMock, actionRestarter, &logger)

		// when
		warnings, hasMorePods, err := proxyRestarter.RestartProxies(ctx, expectedImage, helpers.DefaultSidecarResources, &istioCR)

		// then
		Expect(err).ToNot(HaveOccurred())
		Expect(warnings).To(ContainElement(restart.RestartWarning{
			Name:      "n/a",
			Namespace: "n/a",
			Kind:      "n/a",
			Message:   "failed to restart Customer proxies",
		}))
		Expect(hasMorePods).To(BeFalse())
	})

	It("should return error if restarting Kyma pods have warnings", func() {
		// given
		pod := getPod("test-pod", "kyma-system", "podOwner", "ReplicaSet")
		rsOwner := getReplicaSet("podOwner", "kyma-system", "rsOwner", "ReplicaSet")
		rsOwnerRS := getReplicaSet("rsOwner", "kyma-system", "base", "ReplicaSet")
		c := fakeClient(pod, rsOwner, rsOwnerRS)

		// when
		podsLister := pods.NewPods(c, &logger)
		expectedImage := predicates.NewSidecarImage("istio", "1.1.0")
		istioCR := helpers.GetIstioCR(expectedImage.Tag)
		actionRestarter := NewActionRestartMock([]restart.RestartWarning{{Name: "test-pod", Namespace: "kyma-system", Kind: "Pod", Message: "failed to restart"}}, nil)
		proxyRestarter := sidecars.NewProxyRestarter(c, podsLister, actionRestarter, &logger)
		warnings, hasMorePods, err := proxyRestarter.RestartProxies(ctx, expectedImage, helpers.DefaultSidecarResources, &istioCR)

		// then
		Expect(err).To(HaveOccurred())
		Expect(warnings).To(BeEmpty())
		Expect(hasMorePods).To(BeFalse())

		Expect(err.Error()).To(Equal("The sidecars of the following workloads could not be restarted: kyma-system/test-pod"))
	})

	It("should not return error but a warning when it fails on restart customer proxies", func() {
		// given
		pod := getPod("test-pods", "test-namespace", "podOwner", "ReplicaSet")
		rsOwner := getReplicaSet("podOwner", "test-namespace", "rsOwner", "ReplicaSet")
		rsOwnerRS := getReplicaSet("rsOwner", "test-namespace", "base", "ReplicaSet")

		c := fakeClient(pod, rsOwner, rsOwnerRS)

		// when
		failClient := &shouldFailClient{c, false, true}

		podsLister := pods.NewPods(c, &logger)
		expectedImage := predicates.NewSidecarImage("istio", "1.1.0")
		istioCR := helpers.GetIstioCR(expectedImage.Tag)
		actionRestarter := restart.NewActionRestarter(failClient, &logger)
		proxyRestarter := sidecars.NewProxyRestarter(failClient, podsLister, actionRestarter, &logger)
		warnings, hasMorePods, err := proxyRestarter.RestartProxies(ctx, expectedImage, helpers.DefaultSidecarResources, &istioCR)

		// then
		Expect(err).ToNot(HaveOccurred())
		Expect(warnings).To(BeEmpty())
		Expect(hasMorePods).To(BeFalse())
	})
})

var _ = Describe("RestartWithPredicates", func() {
	ctx := context.Background()
	logger := logr.Discard()

	It("should succeed without errors or warnings", func() {
		// given
		pod := getPod("test-pod", "test-namespace", "podOwner", "ReplicaSet")
		rsOwner := getReplicaSet("podOwner", "test-namespace", "rsOwner", "ReplicaSet")
		rsOwnerRS := getReplicaSet("rsOwner", "test-namespace", "base", "ReplicaSet")

		c := fakeClient(pod, rsOwner, rsOwnerRS)
		preds := []predicates.SidecarProxyPredicate{
			predicates.NewImageResourcesPredicate(predicates.SidecarImage{Repository: "istio", Tag: "1.1.0"}, helpers.DefaultSidecarResources),
		}
		limits := pods.NewPodsRestartLimits(10, 10)

		// when
		podsLister := pods.NewPods(c, &logger)
		actionRestarter := restart.NewActionRestarter(c, &logger)
		proxyRestarter := sidecars.NewProxyRestarter(c, podsLister, actionRestarter, &logger)
		warnings, hasMorePods, err := proxyRestarter.RestartWithPredicates(ctx, preds, limits, true)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).To(BeEmpty())
		Expect(hasMorePods).To(BeFalse())

		err = c.Get(ctx, client.ObjectKey{Name: rsOwnerRS.Name, Namespace: rsOwnerRS.Namespace}, rsOwnerRS)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsOwnerRS.Spec.Template.Annotations).To(HaveKey("istio-operator.kyma-project.io/restartedAt"))
	})

	It("should return warning that pod not have OwnerReferences", func() {
		// given
		pod := getPod("test-pod", "test-namespace", "podOwner", "ReplicaSet")
		c := fakeClient(pod)

		preds := []predicates.SidecarProxyPredicate{
			predicates.NewImageResourcesPredicate(predicates.SidecarImage{Repository: "istio", Tag: "1.1.0"}, helpers.DefaultSidecarResources),
		}
		limits := pods.NewPodsRestartLimits(2, 2)

		// when
		podsLister := pods.NewPods(c, &logger)
		actionRestarter := restart.NewActionRestarter(c, &logger)
		proxyRestarter := sidecars.NewProxyRestarter(c, podsLister, actionRestarter, &logger)
		warnings, hasMorePods, err := proxyRestarter.RestartWithPredicates(ctx, preds, limits, true)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).ToNot(BeEmpty())
		Expect(hasMorePods).To(BeFalse())

		Expect(warnings).To(HaveLen(1))
		Expect(warnings[0].Message).To(Equal("pod sidecar could not be updated because OwnerReferences was not found."))
	})

	It("should return error if getting pods to restart fails", func() {
		// given
		preds := []predicates.SidecarProxyPredicate{
			predicates.NewImageResourcesPredicate(predicates.SidecarImage{Repository: "istio", Tag: "1.1.0"}, helpers.DefaultSidecarResources),
		}
		limits := pods.NewPodsRestartLimits(2, 2)

		// when
		c := fakeClient()
		failClient := &shouldFailClient{c, true, false}

		podsLister := pods.NewPods(failClient, &logger)
		actionRestarter := restart.NewActionRestarter(failClient, &logger)
		proxyRestarter := sidecars.NewProxyRestarter(failClient, podsLister, actionRestarter, &logger)
		warnings, hasMorePods, err := proxyRestarter.RestartWithPredicates(ctx, preds, limits, true)

		// then
		Expect(err).To(HaveOccurred())
		Expect(warnings).To(BeEmpty())
		Expect(hasMorePods).To(BeFalse())

		Expect(err.Error()).To(Equal("intentionally failing client on client.List"))
	})

	It("should return error if restarting pods fails", func() {
		// given
		pod := getPod("test-pod", "test-namespace", "podOwner", "ReplicaSet")
		rsOwner := getReplicaSet("podOwner", "test-namespace", "rsOwner", "ReplicaSet")
		rsOwnerRS := getReplicaSet("rsOwner", "test-namespace", "base", "ReplicaSet")
		c := fakeClient(pod, rsOwner, rsOwnerRS)

		preds := []predicates.SidecarProxyPredicate{
			predicates.NewImageResourcesPredicate(predicates.SidecarImage{Repository: "istio", Tag: "1.1.0"}, helpers.DefaultSidecarResources),
		}
		limits := pods.NewPodsRestartLimits(2, 2)

		// when
		failClient := &shouldFailClient{c, false, true}

		podsLister := pods.NewPods(failClient, &logger)
		actionRestarter := restart.NewActionRestarter(failClient, &logger)
		proxyRestarter := sidecars.NewProxyRestarter(failClient, podsLister, actionRestarter, &logger)
		warnings, hasMorePods, err := proxyRestarter.RestartWithPredicates(ctx, preds, limits, true)

		// then
		Expect(err).To(HaveOccurred())
		Expect(warnings).To(BeEmpty())
		Expect(hasMorePods).To(BeFalse())

		Expect(err.Error()).To(Equal("running pod restart action failed: intentionally failing client on client.Patch"))
	})

	It("should not return error and warnings if restarting pods fails with failOnError is false", func() {
		// given
		pod := getPod("test-pod", "test-namespace", "podOwner", "ReplicaSet")
		rsOwner := getReplicaSet("podOwner", "test-namespace", "rsOwner", "ReplicaSet")
		rsOwnerRS := getReplicaSet("rsOwner", "test-namespace", "base", "ReplicaSet")
		c := fakeClient(pod, rsOwner, rsOwnerRS)

		preds := []predicates.SidecarProxyPredicate{
			predicates.NewImageResourcesPredicate(predicates.SidecarImage{Repository: "istio", Tag: "1.1.0"}, helpers.DefaultSidecarResources),
		}
		limits := pods.NewPodsRestartLimits(2, 2)

		// when
		failClient := &shouldFailClient{c, false, true}

		podsLister := pods.NewPods(failClient, &logger)
		actionRestarter := restart.NewActionRestarter(failClient, &logger)
		proxyRestarter := sidecars.NewProxyRestarter(failClient, podsLister, actionRestarter, &logger)
		warnings, hasMorePods, err := proxyRestarter.RestartWithPredicates(ctx, preds, limits, false)

		// then
		Expect(err).ToNot(HaveOccurred())
		Expect(warnings).To(Equal([]restart.RestartWarning{}))
		Expect(hasMorePods).To(BeFalse())
	})
})

var _ = Describe("BuildWarningMessage", func() {
	logger := logr.Discard()

	It("should return an empty string when there are no warnings", func() {
		// given
		warnings := []restart.RestartWarning{}

		// when
		warningMessage := sidecars.BuildWarningMessage(warnings, &logger)

		// then
		Expect(warningMessage).To(BeEmpty())
	})

	It("should return a warning message with pod details when there are warnings", func() {
		// given
		warnings := []restart.RestartWarning{
			{
				Name:      "pod1",
				Namespace: "namespace1",
				Kind:      "Pod",
				Message:   "failed to restart",
			},
			{
				Name:      "pod2",
				Namespace: "namespace2",
				Kind:      "Pod",
				Message:   "failed to restart",
			},
		}

		// when
		warningMessage := sidecars.BuildWarningMessage(warnings, &logger)

		// then
		Expect(warningMessage).To(ContainSubstring("The sidecars of the following workloads could not be restarted: namespace1/pod1, namespace2/pod2"))
	})

	It("should limit the number of pods in the warning message to 5", func() {
		// given
		warnings := []restart.RestartWarning{
			{Name: "pod1", Namespace: "namespace1", Kind: "Pod", Message: "failed to restart"},
			{Name: "pod2", Namespace: "namespace2", Kind: "Pod", Message: "failed to restart"},
			{Name: "pod3", Namespace: "namespace3", Kind: "Pod", Message: "failed to restart"},
			{Name: "pod4", Namespace: "namespace4", Kind: "Pod", Message: "failed to restart"},
			{Name: "pod5", Namespace: "namespace5", Kind: "Pod", Message: "failed to restart"},
			{Name: "pod6", Namespace: "namespace6", Kind: "Pod", Message: "failed to restart"},
		}

		// when
		warningMessage := sidecars.BuildWarningMessage(warnings, &logger)

		// then
		Expect(warningMessage).To(ContainSubstring("The sidecars of the following workloads could not be restarted: namespace1/pod1, namespace2/pod2, namespace3/pod3, namespace4/pod4, namespace5/pod5 and 1 additional workload(s)"))
	})

	It("should log each warning message", func() {
		// given
		warnings := []restart.RestartWarning{
			{Name: "pod1", Namespace: "namespace1", Kind: "Pod", Message: "failed to restart"},
		}

		// when
		warningMessage := sidecars.BuildWarningMessage(warnings, &logger)

		// then
		Expect(warningMessage).To(ContainSubstring("The sidecars of the following workloads could not be restarted: namespace1/pod1"))
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

func getPod(name, namespace, ownerName, ownerKind string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: ownerName,
					Kind: ownerKind,
				},
			},
			Annotations: map[string]string{
				"sidecar.istio.io/status": "abc",
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
	}
}

func getReplicaSet(name, namespace, ownerName, ownerKind string) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: ownerName,
					Kind: ownerKind,
				},
			},
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.ReplicaSetSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"dummy": "annotation"},
				},
			},
		},
	}
}

type shouldFailClient struct {
	client.Client
	FailOnList  bool
	FailOnPatch bool
}

func (p *shouldFailClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if p.FailOnList {
		return errors.New("intentionally failing client on client.List")
	}
	return p.Client.List(ctx, list, opts...)
}

func (p *shouldFailClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if p.FailOnPatch {
		return errors.New("intentionally failing client on client.Patch")
	}
	return p.Client.Patch(ctx, obj, patch, opts...)
}

type PodsMock struct {
	Called                 int
	Predicates             map[int][]predicates.SidecarProxyPredicate
	Limits                 map[int]*pods.PodsRestartLimits
	FailOnKymaWorkload     bool
	FailOnCustomerWorkload bool
}

func NewPodsMock() *PodsMock {
	return &PodsMock{
		Called:                 0,
		Predicates:             map[int][]predicates.SidecarProxyPredicate{},
		Limits:                 map[int]*pods.PodsRestartLimits{},
		FailOnKymaWorkload:     false,
		FailOnCustomerWorkload: false,
	}
}

func (p *PodsMock) GetPodsToRestart(_ context.Context, preds []predicates.SidecarProxyPredicate, limits *pods.PodsRestartLimits) (*v1.PodList, error) {
	if p.FailOnKymaWorkload {
		_, ok := preds[len(preds)-1].(*predicates.KymaWorkloadRestartPredicate)
		if ok {
			return &v1.PodList{}, errors.New("intentionally failed on Kyma workload predicate")
		}
	}
	if p.FailOnCustomerWorkload {
		_, ok := preds[len(preds)-1].(*predicates.CustomerWorkloadRestartPredicate)
		if ok {
			return &v1.PodList{}, errors.New("intentionally failed on Customer workload predicate")
		}
	}
	p.Predicates[p.Called] = preds
	p.Limits[p.Called] = limits
	p.Called++
	return &v1.PodList{}, nil
}

func (p *PodsMock) GetAllInjectedPods(_ context.Context) (*v1.PodList, error) {
	return &v1.PodList{}, nil
}

type ActionRestartMock struct {
	warnings []restart.RestartWarning
	err      error
}

func NewActionRestartMock(warnings []restart.RestartWarning, err error) *ActionRestartMock {
	return &ActionRestartMock{
		warnings: warnings,
		err:      err,
	}
}

func (p *ActionRestartMock) RestartAction(ctx context.Context, podList *v1.PodList, failOnError bool) ([]restart.RestartWarning, error) {
	return p.warnings, p.err
}
