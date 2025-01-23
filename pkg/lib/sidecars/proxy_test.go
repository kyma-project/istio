package sidecars_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/kyma-project/istio/operator/internal/restarter/predicates"
	"github.com/kyma-project/istio/operator/internal/tests"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
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

func NewStdoutLogger() logr.Logger {
	l := &stdoutlogger{
		Formatter: funcr.NewFormatter(funcr.Options{}),
	}
	return logr.New(l)
}

type stdoutlogger struct {
	funcr.Formatter
	logMsgType bool
}

func (l stdoutlogger) WithName(name string) logr.LogSink {
	l.Formatter.AddName(name)
	return &l
}

func (l stdoutlogger) WithValues(kvList ...interface{}) logr.LogSink {
	l.Formatter.AddValues(kvList)
	return &l
}

func (l stdoutlogger) WithCallDepth(depth int) logr.LogSink {
	l.Formatter.AddCallDepth(depth)
	return &l
}

func (l stdoutlogger) Info(level int, msg string, kvList ...interface{}) {
	prefix, args := l.FormatInfo(level, msg, kvList)
	l.write("INFO", prefix, args)
}

func (l stdoutlogger) Error(err error, msg string, kvList ...interface{}) {
	prefix, args := l.FormatError(err, msg, kvList)
	l.write("ERROR", prefix, args)
}

func (l stdoutlogger) write(msgType, prefix, args string) {
	var parts []string
	if l.logMsgType {
		parts = append(parts, msgType)
	}
	if prefix != "" {
		parts = append(parts, prefix)
	}
	parts = append(parts, args)
	fmt.Println(strings.Join(parts, ": "))
}

// WithLogMsgType returns a copy of the logger with new settings for
// logging the message type. It returns the original logger if the
// underlying LogSink is not a stdoutlogger.
func WithLogMsgType(log logr.Logger, logMsgType bool) logr.Logger {
	if l, ok := log.GetSink().(*stdoutlogger); ok {
		clone := *l
		clone.logMsgType = logMsgType
		log = log.WithSink(&clone)
	}
	return log
}

// Assert conformance to the interfaces.
var _ logr.LogSink = &stdoutlogger{}
var _ logr.CallDepthLogSink = &stdoutlogger{}

func TestRestartProxies(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Proxy Restart Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report ginkgotypes.Report) {
	tests.GenerateGinkgoJunitReport("proxy-restart-suite", report)
})

var _ = Describe("RestartProxies", func() {
	ctx := context.Background()
	//logger := logr.Discard()
	logger := NewStdoutLogger()

	It("should succeed without errors or warnings", func() {
		// given
		pod := getPod("test-pods", "test-namespace", "podOwner", "ReplicaSet")
		rsOwner := getReplicaSet("podOwner", "test-namespace", "rsOwner", "ReplicaSet")
		rsOwnerRS := getReplicaSet("rsOwner", "test-namespace", "base", "ReplicaSet")

		c := fakeClient(pod, rsOwner, rsOwnerRS)

		// when
		proxyRestarter := sidecars.NewProxyRestarter()
		expectedImage := predicates.NewSidecarImage("istio", "1.1.0")
		istioCR := helpers.GetIstioCR(expectedImage.Tag)
		warnings, hasMorePods, err := proxyRestarter.RestartProxies(ctx, c, expectedImage, helpers.DefaultSidecarResources, &istioCR, &logger)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).To(BeEmpty())
		Expect(hasMorePods).To(BeFalse())

		err = c.Get(ctx, client.ObjectKey{Name: rsOwnerRS.Name, Namespace: rsOwnerRS.Namespace}, rsOwnerRS)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsOwnerRS.Spec.Template.Annotations).To(HaveKey("istio-operator.kyma-project.io/restartedAt"))
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

var _ = Describe("RestartWithPredicates", func() {
	ctx := context.Background()
	logger := NewStdoutLogger()

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
		proxyRestarter := sidecars.NewProxyRestarter()
		warnings, hasMorePods, err := proxyRestarter.RestartWithPredicates(ctx, c, preds, limits, &logger)

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
		proxyRestarter := sidecars.NewProxyRestarter()
		warnings, hasMorePods, err := proxyRestarter.RestartWithPredicates(ctx, c, preds, limits, &logger)

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

		proxyRestarter := sidecars.NewProxyRestarter()
		warnings, hasMorePods, err := proxyRestarter.RestartWithPredicates(ctx, failClient, preds, limits, &logger)

		// then
		Expect(err).To(HaveOccurred())
		Expect(warnings).To(BeNil())
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

		proxyRestarter := sidecars.NewProxyRestarter()
		warnings, hasMorePods, err := proxyRestarter.RestartWithPredicates(ctx, failClient, preds, limits, &logger)

		// then
		Expect(err).To(HaveOccurred())
		Expect(warnings).To(BeNil())
		Expect(hasMorePods).To(BeFalse())

		Expect(err.Error()).To(Equal("running pod restart action failed: intentionally failing client on client.Patch"))
	})
})

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
