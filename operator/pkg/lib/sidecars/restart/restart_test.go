package restart_test

import (
	"context"
	"github.com/go-logr/logr"
	"testing"

	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/restart"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const restartAnnotationName = "istio-operator.kyma-project.io/restartedAt"

func TestRestart(t *testing.T) {
	ctx := context.TODO()
	logger := logr.Discard()

	t.Run("should return warning when pod has no owner", func(t *testing.T) {
		// given
		c := fakeClient(t)

		podList := v1.PodList{
			Items: []v1.Pod{
				podWithoutOwnerFixture("p1", "test-ns"),
			},
		}

		// when
		warnings, err := restart.Restart(ctx, c, podList, &logger)

		// then
		require.NoError(t, err)
		require.NotEmpty(t, warnings)

		require.Equal(t, "p1", warnings[0].Name)
		require.Contains(t, warnings[0].Message, "OwnerReferences was not found")
	})

	t.Run("should return warning when pod is owned by a Job", func(t *testing.T) {
		// given
		c := fakeClient(t)

		podList := v1.PodList{
			Items: []v1.Pod{
				podFixture("p1", "test-ns", "Job", "owningJob"),
			},
		}

		// when
		warnings, err := restart.Restart(ctx, c, podList, &logger)

		// then
		require.NoError(t, err)
		require.NotEmpty(t, warnings)

		require.Equal(t, "p1", warnings[0].Name)
		require.Contains(t, warnings[0].Message, "owned by a Job")
	})

	t.Run("should return warning when pod is owned by a Job", func(t *testing.T) {
		// given
		c := fakeClient(t)

		podList := v1.PodList{
			Items: []v1.Pod{
				podFixture("p1", "test-ns", "Job", "owningJob"),
			},
		}

		// when
		warnings, err := restart.Restart(ctx, c, podList, &logger)

		// then
		require.NoError(t, err)
		require.NotEmpty(t, warnings)

		require.Equal(t, "p1", warnings[0].Name)
		require.Contains(t, warnings[0].Message, "owned by a Job")
	})

	t.Run("should rollout restart Deployment if the pod is owned by one", func(t *testing.T) {
		// given
		c := fakeClient(t, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "test-ns"}})

		podList := v1.PodList{
			Items: []v1.Pod{
				podFixture("p1", "test-ns", "Deployment", "owner"),
			},
		}

		// when
		warnings, err := restart.Restart(ctx, c, podList, &logger)

		// then
		require.NoError(t, err)
		require.Empty(t, warnings)

		obj := appsv1.Deployment{}
		err = c.Get(context.TODO(), types.NamespacedName{Namespace: "test-ns", Name: "owner"}, &obj)
		require.NoError(t, err)

		require.NotEmpty(t, obj.Annotations[restartAnnotationName])
	})

	t.Run("should rollout restart one Deployment if two pods are owned by one", func(t *testing.T) {
		// given
		c := fakeClient(t, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "test-ns"}})

		podList := v1.PodList{
			Items: []v1.Pod{
				podFixture("p1", "test-ns", "Deployment", "owner"),
				podFixture("p2", "test-ns", "Deployment", "owner"),
			},
		}

		// when
		warnings, err := restart.Restart(ctx, c, podList, &logger)

		// then
		require.NoError(t, err)
		require.Empty(t, warnings)

		obj := appsv1.Deployment{}
		err = c.Get(context.TODO(), types.NamespacedName{Namespace: "test-ns", Name: "owner"}, &obj)
		require.NoError(t, err)

		require.NotEmpty(t, obj.Annotations[restartAnnotationName])
	})

	t.Run("should rollout restart DaemonSet if the pod is owned by one", func(t *testing.T) {
		// given
		c := fakeClient(t, &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "test-ns"}})

		podList := v1.PodList{
			Items: []v1.Pod{
				podFixture("p1", "test-ns", "DaemonSet", "owner"),
			},
		}

		// when
		warnings, err := restart.Restart(ctx, c, podList, &logger)

		// then
		require.NoError(t, err)
		require.Empty(t, warnings)

		obj := appsv1.DaemonSet{}
		err = c.Get(context.TODO(), types.NamespacedName{Namespace: "test-ns", Name: "owner"}, &obj)
		require.NoError(t, err)

		require.NotEmpty(t, obj.Annotations[restartAnnotationName])
	})

	t.Run("should delete a pod belonging to a ReplicaSet with no owner", func(t *testing.T) {
		// given
		pod := podFixture("p1", "test-ns", "ReplicaSet", "owner")
		c := fakeClient(t, &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "test-ns"}}, &pod)

		podList := v1.PodList{
			Items: []v1.Pod{pod},
		}

		// when
		warnings, err := restart.Restart(ctx, c, podList, &logger)

		// then
		require.NoError(t, err)
		require.Empty(t, warnings)

		obj := v1.Pod{}
		err = c.Get(context.TODO(), types.NamespacedName{Namespace: "test-ns", Name: "p1"}, &obj)

		require.Error(t, err)
		require.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("should delete a pod managed by a ReplicationController", func(t *testing.T) {
		// given
		pod := podFixture("p1", "test-ns", "ReplicationController", "owner")
		c := fakeClient(t, &v1.ReplicationController{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "test-ns"}}, &pod)

		podList := v1.PodList{
			Items: []v1.Pod{pod},
		}

		// when
		warnings, err := restart.Restart(ctx, c, podList, &logger)

		// then
		require.NoError(t, err)
		require.Empty(t, warnings)

		obj := v1.Pod{}
		err = c.Get(context.TODO(), types.NamespacedName{Namespace: "test-ns", Name: "p1"}, &obj)

		require.Error(t, err)
		require.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("should rollout restart StatefulSet if the pod is owned by one", func(t *testing.T) {
		// given
		pod := podFixture("p1", "test-ns", "StatefulSet", "owner")

		c := fakeClient(t, &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "test-ns"}}, &pod)

		podList := v1.PodList{
			Items: []v1.Pod{
				pod,
			},
		}

		// when
		warnings, err := restart.Restart(ctx, c, podList, &logger)

		// then
		require.NoError(t, err)
		require.Empty(t, warnings)

		obj := appsv1.StatefulSet{}
		err = c.Get(context.TODO(), types.NamespacedName{Namespace: "test-ns", Name: "owner"}, &obj)
		require.NoError(t, err)

		require.NotEmpty(t, obj.Annotations[restartAnnotationName])
	})

	t.Run("should return a warning when Pod is owned by a ReplicaSet that is not found", func(t *testing.T) {
		// given
		pod := podFixture("p1", "test-ns", "ReplicaSet", "podOwner")

		podList := v1.PodList{
			Items: []v1.Pod{
				pod,
			},
		}

		c := fakeClient(t, &pod)

		// when
		warnings, err := restart.Restart(ctx, c, podList, &logger)

		// then
		require.NoError(t, err)
		require.NotEmpty(t, warnings)

		pods := v1.PodList{}
		err = c.List(context.TODO(), &pods)

		require.NoError(t, err)
		require.NotEmpty(t, pods.Items)
	})

	t.Run("should not delete pod when it is owned by a ReplicaSet that is found", func(t *testing.T) {
		// given
		pod := podFixture("p1", "test-ns", "ReplicaSet", "podOwner")

		podList := v1.PodList{
			Items: []v1.Pod{
				pod,
			},
		}

		c := fakeClient(t, &pod, &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
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
		warnings, err := restart.Restart(ctx, c, podList, &logger)

		// then
		require.NoError(t, err)
		require.Empty(t, warnings)

		pods := v1.PodList{}
		err = c.List(context.TODO(), &pods)

		require.NoError(t, err)
		require.NotEmpty(t, pods.Items)
	})

	t.Run("should do only one rollout if the StatefulSet has multiple pods", func(t *testing.T) {
		// given
		c := fakeClient(t, &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "podOwner", Namespace: "test-ns"}})

		podList := v1.PodList{
			Items: []v1.Pod{
				podFixture("p1", "test-ns", "StatefulSet", "podOwner"),
				podFixture("p2", "test-ns", "StatefulSet", "podOwner"),
			},
		}

		// when
		warnings, err := restart.Restart(ctx, c, podList, &logger)

		// then
		require.NoError(t, err)
		require.Empty(t, warnings)

		dep := appsv1.StatefulSet{}
		err = c.Get(context.TODO(), types.NamespacedName{Namespace: "test-ns", Name: "podOwner"}, &dep)
		require.NoError(t, err)
		require.Equal(t, "1000", dep.ResourceVersion, "StatefulSet should patch only once")
	})

	t.Run("should rollout restart ReplicaSet owner if the pod is owned by one that is found and has an owner", func(t *testing.T) {
		// given
		pod := podFixture("p1", "test-ns", "ReplicaSet", "podOwner")

		podList := v1.PodList{
			Items: []v1.Pod{
				pod,
			},
		}

		c := fakeClient(t, &pod, &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
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
		warnings, err := restart.Restart(ctx, c, podList, &logger)

		// then
		require.NoError(t, err)
		require.Empty(t, warnings)

		replicaSet := appsv1.ReplicaSet{}
		err = c.Get(context.TODO(), types.NamespacedName{Name: "rsOwner", Namespace: "test-ns"}, &replicaSet)

		require.NoError(t, err)
		require.NotEmpty(t, replicaSet.Annotations[restartAnnotationName])
	})
}

func fakeClient(t *testing.T, objects ...client.Object) client.Client {
	err := v1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)
	err = appsv1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()

	return fakeClient
}
