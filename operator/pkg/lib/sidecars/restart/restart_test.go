package restart_test

import (
	"context"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/restart"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
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
}

func fakeClient(t *testing.T, objects ...client.Object) client.Client {
	err := v1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()

	return fakeClient
}
