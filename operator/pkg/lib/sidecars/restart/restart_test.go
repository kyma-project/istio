package restart_test

import (
	"context"
	"testing"

	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/restart"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRestart(t *testing.T) {
	ctx := context.TODO()

	t.Run("should return warning when pod has no owner", func(t *testing.T) {
		// given
		c := fakeClient(t)

		podList := v1.PodList{
			Items: []v1.Pod{
				podWithoutOwnerFixture("p1", "test-ns"),
			},
		}

		// when
		warnings, err := restart.Restart(ctx, c, podList)

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
		warnings, err := restart.Restart(ctx, c, podList)

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
		warnings, err := restart.Restart(ctx, c, podList)

		// then
		require.NoError(t, err)
		require.NotEmpty(t, warnings)

		require.Equal(t, "p1", warnings[0].Name)
		require.Contains(t, warnings[0].Message, "owned by a Job")
	})

	t.Run("should return warning when pod is owned by a Deployment and restart timed out", func(t *testing.T) {
		// given
		c := fakeClient(t)

		podList := v1.PodList{
			Items: []v1.Pod{
				podFixture("p1", "test-ns", "Deployment", "owner"),
			},
		}

		// when
		warnings, err := restart.Restart(ctx, c, podList)

		// then
		require.NoError(t, err)
		require.NotEmpty(t, warnings)

		require.Equal(t, "owner", warnings[0].Name)
		require.Contains(t, warnings[0].Message, "could not be rolled out")
	})

	t.Run("should return warning when pod is owned by a DaemonSet and restart timed out", func(t *testing.T) {
		// given
		c := fakeClient(t)

		podList := v1.PodList{
			Items: []v1.Pod{
				podFixture("p1", "test-ns", "DaemonSet", "owner"),
			},
		}

		// when
		warnings, err := restart.Restart(ctx, c, podList)

		// then
		require.NoError(t, err)
		require.NotEmpty(t, warnings)

		require.Equal(t, "owner", warnings[0].Name)
		require.Contains(t, warnings[0].Message, "could not be rolled out")
	})

	t.Run("should return warning when pod is owned by a ReplicaSet and restart timed out", func(t *testing.T) {
		// given
		rsName := "podOwner"
		namespace := "test-ns"

		c := fakeClient(t, replicaSetFixture(rsName, namespace, "rsOwner", "Deployment"))

		podList := v1.PodList{
			Items: []v1.Pod{
				podFixture("p1", namespace, "ReplicaSet", rsName),
			},
		}

		// when
		warnings, err := restart.Restart(ctx, c, podList)

		// then
		require.NoError(t, err)
		require.NotEmpty(t, warnings)

		require.Equal(t, "rsOwner", warnings[0].Name)
		require.Contains(t, warnings[0].Message, "could not be rolled out")
	})

	t.Run("should return warning when pod is owned by a StatefulSet and restart timed out", func(t *testing.T) {
		// given
		c := fakeClient(t)

		podList := v1.PodList{
			Items: []v1.Pod{
				podFixture("p1", "test-ns", "StatefulSet", "owner"),
			},
		}

		// when
		warnings, err := restart.Restart(ctx, c, podList)

		// then
		require.NoError(t, err)
		require.NotEmpty(t, warnings)

		require.Equal(t, "owner", warnings[0].Name)
		require.Contains(t, warnings[0].Message, "could not be rolled out")
	})

	t.Run("should delete pod when it's is owned by a ReplicaSet that is not found", func(t *testing.T) {
		// given
		c := fakeClient(t)

		podList := v1.PodList{
			Items: []v1.Pod{
				podFixture("p1", "test-ns", "ReplicaSet", "podOwner"),
			},
		}

		// when
		restart.Restart(ctx, c, podList)

		// then
		// TODO
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
