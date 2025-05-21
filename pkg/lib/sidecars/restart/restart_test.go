package restart_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/internal/tests"
	. "github.com/onsi/ginkgo/v2"
	ginkgotypes "github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/restart"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
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
	tests.GenerateGinkgoJunitReport("pods-restart-suite", report)
})

var _ = Describe("Restart Pods", func() {
	ctx := context.Background()
	logger := logr.Discard()

	It("should return warning when pod has no owner", func() {
		// given
		c := fakeClient()

		podList := v1.PodList{
			Items: []v1.Pod{
				podWithoutOwnerFixture("p1", "test-ns"),
			},
		}

		// when
		actionRestarter := restart.NewActionRestarter(c, &logger)
		warnings, err := actionRestarter.Restart(ctx, &podList, false)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).NotTo(BeEmpty())

		Expect(warnings[0].Name).To(Equal("p1"))
		Expect(warnings[0].Message).To(ContainSubstring("OwnerReferences was not found"))
	})

	It("should return warning when pod is owned by a Job", func() {
		// given
		c := fakeClient()

		podList := v1.PodList{
			Items: []v1.Pod{
				podFixture("p1", "test-ns", "Job", "owningJob"),
			},
		}

		// when
		actionRestarter := restart.NewActionRestarter(c, &logger)
		warnings, err := actionRestarter.Restart(ctx, &podList, false)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).NotTo(BeEmpty())

		Expect(warnings[0].Name).To(Equal("p1"))
		Expect(warnings[0].Message).To(ContainSubstring("owned by a Job"))
	})

	It("should rollout restart Deployment if the pod is owned by one", func() {
		// given
		c := fakeClient(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "test-ns"}})

		podList := v1.PodList{
			Items: []v1.Pod{
				podFixture("p1", "test-ns", "Deployment", "owner"),
			},
		}

		// when
		actionRestarter := restart.NewActionRestarter(c, &logger)
		warnings, err := actionRestarter.Restart(ctx, &podList, false)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).To(BeEmpty())

		obj := appsv1.Deployment{}
		err = c.Get(context.Background(), types.NamespacedName{Namespace: "test-ns", Name: "owner"}, &obj)
		Expect(err).NotTo(HaveOccurred())

		Expect(obj.Spec.Template.Annotations[restartAnnotationName]).NotTo(BeEmpty())
	})

	It("should rollout restart one Deployment if two pods are owned by one", func() {
		// given
		c := fakeClient(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "test-ns"}})

		podList := v1.PodList{
			Items: []v1.Pod{
				podFixture("p1", "test-ns", "Deployment", "owner"),
				podFixture("p2", "test-ns", "Deployment", "owner"),
			},
		}

		// when
		actionRestarter := restart.NewActionRestarter(c, &logger)
		warnings, err := actionRestarter.Restart(ctx, &podList, false)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).To(BeEmpty())

		obj := appsv1.Deployment{}
		err = c.Get(context.Background(), types.NamespacedName{Namespace: "test-ns", Name: "owner"}, &obj)
		Expect(err).NotTo(HaveOccurred())

		Expect(obj.Spec.Template.Annotations[restartAnnotationName]).NotTo(BeEmpty())
	})

	It("should rollout restart DaemonSet if the pod is owned by one", func() {
		// given
		c := fakeClient(&appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "test-ns"}})

		podList := v1.PodList{
			Items: []v1.Pod{
				podFixture("p1", "test-ns", "DaemonSet", "owner"),
			},
		}

		// when
		actionRestarter := restart.NewActionRestarter(c, &logger)
		warnings, err := actionRestarter.Restart(ctx, &podList, false)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).To(BeEmpty())

		obj := appsv1.DaemonSet{}
		err = c.Get(context.Background(), types.NamespacedName{Namespace: "test-ns", Name: "owner"}, &obj)
		Expect(err).NotTo(HaveOccurred())

		Expect(obj.Spec.Template.Annotations[restartAnnotationName]).NotTo(BeEmpty())
	})

	It("should delete a pod belonging to a ReplicaSet with no owner", func() {
		// given
		pod := podFixture("p1", "test-ns", "ReplicaSet", "owner")
		c := fakeClient(&appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "test-ns"}}, &pod)

		podList := v1.PodList{
			Items: []v1.Pod{pod},
		}

		// when
		actionRestarter := restart.NewActionRestarter(c, &logger)
		warnings, err := actionRestarter.Restart(ctx, &podList, false)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).To(BeEmpty())

		obj := v1.Pod{}
		err = c.Get(context.Background(), types.NamespacedName{Namespace: "test-ns", Name: "p1"}, &obj)

		Expect(err).To(HaveOccurred())
		Expect(k8serrors.IsNotFound(err)).To(BeTrue())
	})

	It("should delete a pod managed by a ReplicationController", func() {
		// given
		pod := podFixture("p1", "test-ns", "ReplicationController", "owner")
		c := fakeClient(&v1.ReplicationController{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "test-ns"}}, &pod)

		podList := v1.PodList{
			Items: []v1.Pod{pod},
		}

		// when
		actionRestarter := restart.NewActionRestarter(c, &logger)
		warnings, err := actionRestarter.Restart(ctx, &podList, false)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).To(BeEmpty())

		obj := v1.Pod{}
		err = c.Get(context.Background(), types.NamespacedName{Namespace: "test-ns", Name: "p1"}, &obj)

		Expect(err).To(HaveOccurred())
		Expect(k8serrors.IsNotFound(err)).To(BeTrue())
	})

	It("should rollout restart StatefulSet if the pod is owned by one", func() {
		// given
		pod := podFixture("p1", "test-ns", "StatefulSet", "owner")

		c := fakeClient(&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "test-ns"}}, &pod)

		podList := v1.PodList{
			Items: []v1.Pod{
				pod,
			},
		}

		// when
		actionRestarter := restart.NewActionRestarter(c, &logger)
		warnings, err := actionRestarter.Restart(ctx, &podList, false)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).To(BeEmpty())

		obj := appsv1.StatefulSet{}
		err = c.Get(context.Background(), types.NamespacedName{Namespace: "test-ns", Name: "owner"}, &obj)
		Expect(err).NotTo(HaveOccurred())

		Expect(obj.Spec.Template.Annotations[restartAnnotationName]).NotTo(BeEmpty())
	})

	It("should return a warning when Pod is owned by a ReplicaSet that is not found", func() {
		// given
		pod := podFixture("p1", "test-ns", "ReplicaSet", "podOwner")

		podList := v1.PodList{
			Items: []v1.Pod{
				pod,
			},
		}

		c := fakeClient(&pod)

		// when
		actionRestarter := restart.NewActionRestarter(c, &logger)
		warnings, err := actionRestarter.Restart(ctx, &podList, false)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).NotTo(BeEmpty())

		pods := v1.PodList{}
		err = c.List(context.Background(), &pods)

		Expect(err).NotTo(HaveOccurred())
		Expect(pods.Items).NotTo(BeEmpty())
	})

	It("should not delete pod when it is owned by a ReplicaSet that is found", func() {
		// given
		pod := podFixture("p1", "test-ns", "ReplicaSet", "podOwner")

		podList := v1.PodList{
			Items: []v1.Pod{
				pod,
			},
		}

		c := fakeClient(&pod, &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{Name: "name", Kind: "ReplicaSet"},
			},
			Name:      "podOwner",
			Namespace: "test-ns",
		}}, &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "test-ns",
		}})

		// when
		actionRestarter := restart.NewActionRestarter(c, &logger)
		warnings, err := actionRestarter.Restart(ctx, &podList, false)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).To(BeEmpty())

		pods := v1.PodList{}
		err = c.List(context.Background(), &pods)

		Expect(err).NotTo(HaveOccurred())
		Expect(pods.Items).NotTo(BeEmpty())
	})

	It("should do only one rollout if the StatefulSet has multiple pods", func() {
		// given
		c := fakeClient(&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "podOwner", Namespace: "test-ns"}})

		podList := v1.PodList{
			Items: []v1.Pod{
				podFixture("p1", "test-ns", "StatefulSet", "podOwner"),
				podFixture("p2", "test-ns", "StatefulSet", "podOwner"),
			},
		}

		// when
		actionRestarter := restart.NewActionRestarter(c, &logger)
		warnings, err := actionRestarter.Restart(ctx, &podList, false)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).To(BeEmpty())

		dep := appsv1.StatefulSet{}
		err = c.Get(context.Background(), types.NamespacedName{Namespace: "test-ns", Name: "podOwner"}, &dep)
		Expect(err).NotTo(HaveOccurred())
		// "StatefulSet should patch only once"
		Expect(dep.ResourceVersion).To(Equal("1000"))
	})

	It("should not rollout restart ReplicaSet owner if there are multiple ReplicaSets, with one not ready", func() {
		// given
		pod := podFixture("p1", "test-ns", "ReplicaSet", "podOwner")

		podList := v1.PodList{
			Items: []v1.Pod{
				pod,
			},
		}

		c := fakeClient(&pod,
			&appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{Name: "rsOwner", Kind: "Deployment", UID: "1234"},
					},
					Name:      "podOwner",
					Namespace: "test-ns",
				},
				Status: appsv1.ReplicaSetStatus{
					Replicas:      1,
					ReadyReplicas: 1,
				},
			},

			&appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{Name: "rsOwner", Kind: "Deployment", UID: "1234"},
					},
					Name:      "podOwner2",
					Namespace: "test-ns",
				},
				Status: appsv1.ReplicaSetStatus{
					Replicas:      1,
					ReadyReplicas: 0,
				},
			},

			&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
				Name:      "rsOwner",
				Namespace: "test-ns",
				UID:       "1234",
			}})

		// when
		actionRestarter := restart.NewActionRestarter(c, &logger)
		warnings, err := actionRestarter.Restart(ctx, &podList, false)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).To(ContainElement(
			restart.RestartWarning{
				Name:      "rsOwner",
				Namespace: "test-ns",
				Kind:      "Deployment",
				Message:   "was not restarted because there exists another not ready ReplicaSet for the same object",
			}),
		)

		deployment := appsv1.Deployment{}
		err = c.Get(context.Background(), types.NamespacedName{Name: "rsOwner", Namespace: "test-ns"}, &deployment)

		Expect(err).NotTo(HaveOccurred())
		Expect(deployment.Spec.Template.Annotations[restartAnnotationName]).To(BeEmpty())
	})

	It("should rollout restart ReplicaSet owner if the pod is owned by one that is found and has an owner", func() {
		// given
		pod := podFixture("p1", "test-ns", "ReplicaSet", "podOwner")

		podList := v1.PodList{
			Items: []v1.Pod{
				pod,
			},
		}

		c := fakeClient(&pod, &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{Name: "rsOwner", Kind: "ReplicaSet"},
			},
			Name:      "podOwner",
			Namespace: "test-ns",
		}}, &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
			Name:      "rsOwner",
			Namespace: "test-ns",
		}})

		// when
		actionRestarter := restart.NewActionRestarter(c, &logger)
		warnings, err := actionRestarter.Restart(ctx, &podList, false)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(warnings).To(BeEmpty())

		replicaSet := appsv1.ReplicaSet{}
		err = c.Get(context.Background(), types.NamespacedName{Name: "rsOwner", Namespace: "test-ns"}, &replicaSet)

		Expect(err).NotTo(HaveOccurred())
		Expect(replicaSet.Spec.Template.Annotations[restartAnnotationName]).NotTo(BeEmpty())
	})

	It("should return an error when specified for a Pod owned by a ReplicaSet that is not found", func() {
		// given
		pod := podFixture("p1", "test-ns", "ReplicaSet", "podOwner")

		podList := v1.PodList{
			Items: []v1.Pod{
				pod,
			},
		}

		c := fakeClient(&pod)
		failClient := &shouldFailClient{c, true, false}

		// when
		actionRestarter := restart.NewActionRestarter(failClient, &logger)
		warnings, err := actionRestarter.Restart(ctx, &podList, true)

		// then
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("creating pod restart action failed: intentionally failing client on client.Get"))
		Expect(warnings).To(BeEmpty())

		pods := v1.PodList{}
		err = c.List(context.Background(), &pods)

		Expect(err).NotTo(HaveOccurred())
		Expect(pods.Items).NotTo(BeEmpty())
	})

	It("should return an error when specified for a Pod owned by a ReplicaSet that did not succeed to be patched", func() {
		// given
		pod := podFixture("p1", "test-ns", "ReplicaSet", "podOwner")

		podList := v1.PodList{
			Items: []v1.Pod{
				pod,
			},
		}

		c := fakeClient(&pod, &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{Name: "rsOwner", Kind: "ReplicaSet"},
			},
			Name:      "podOwner",
			Namespace: "test-ns",
		}}, &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
			Name:      "rsOwner",
			Namespace: "test-ns",
		}})

		failClient := &shouldFailClient{c, false, true}

		// when
		actionRestarter := restart.NewActionRestarter(failClient, &logger)
		warnings, err := actionRestarter.Restart(ctx, &podList, true)

		// then
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("running pod restart action failed: intentionally failing client on client.Patch"))
		Expect(warnings).To(BeEmpty())

		pods := v1.PodList{}
		err = c.List(context.Background(), &pods)

		Expect(err).NotTo(HaveOccurred())
		Expect(pods.Items).NotTo(BeEmpty())
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

type shouldFailClient struct {
	client.Client
	FailOnGet   bool
	FailOnPatch bool
}

func (p *shouldFailClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if p.FailOnGet {
		return errors.New("intentionally failing client on client.Get")
	}
	return p.Client.Get(ctx, key, obj, opts...)
}

func (p *shouldFailClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if p.FailOnPatch {
		return errors.New("intentionally failing client on client.Patch")
	}
	return p.Client.Patch(ctx, obj, patch, opts...)
}
