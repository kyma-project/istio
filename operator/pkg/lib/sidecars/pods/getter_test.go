package pods_test

import (
	"context"
	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

func createClientSet(t *testing.T, objects ...client.Object) client.Client {
	err := v1alpha1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)
	err = v1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()

	return fakeClient
}

func TestGetPodsForCNIChange(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		c             client.Client
		expectedImage pods.SidecarImage
		wantError     bool
		wantEmpty     bool
		wantLen       int
	}{
		{
			name: "should not get any pod without istio-init container when CNI is enabled",
			c: createClientSet(t,
				fixPodWithoutInitContainer("application1", "enabled", "Running", map[string]string{}, map[string]string{}),
				fixPodWithoutInitContainer("application2", "enabled", "Terminating", map[string]string{}, map[string]string{}),
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantError:     false,
			wantEmpty:     true,
			wantLen:       0,
		},
		{
			name: "should get 2 pods with istio-init when they are in Running state when CNI is enabled",
			c: createClientSet(t,
				newSidecarPodBuilder().
					setName("application1").
					setNamespace("enabled").
					build(),
				newSidecarPodBuilder().
					setName("application2").
					setNamespace("enabled").
					build(),
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantError:     false,
			wantEmpty:     false,
			wantLen:       2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podList, err := pods.GetPodsForCNIChange(ctx, tt.c, tt.expectedImage)

			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.wantEmpty {
				require.Empty(t, podList.Items)
			} else {
				require.NotEmpty(t, podList.Items)
			}

			require.Len(t, podList.Items, tt.wantLen)
		})
	}
}

func TestGetPodsWithDifferentSidecarImage(t *testing.T) {
	ctx := context.TODO()

	expectedImage := pods.SidecarImage{
		Repository: "istio/proxyv2",
		Tag:        "1.10.0",
	}

	tests := []struct {
		name       string
		c          client.Client
		assertFunc func(t require.TestingT, val interface{})
	}{
		{
			name: "should not return pods without istio sidecar",
			c: createClientSet(t,
				fixPodWithoutSidecar("app", "custom"),
			),
			assertFunc: func(t require.TestingT, val interface{}) { require.Empty(t, val) },
		},
		{
			name: "should not return any pod when pods have correct image",
			c: createClientSet(t,
				newSidecarPodBuilder().build(),
			),
			assertFunc: func(t require.TestingT, val interface{}) { require.Empty(t, val) },
		},
		{
			name: "should return pod with different image repository",
			c: createClientSet(t,
				newSidecarPodBuilder().build(),
				newSidecarPodBuilder().
					setName("changedSidecarPod").
					setSidecarImageRepository("istio/different-proxy").
					build(),
			),
			assertFunc: func(t require.TestingT, val interface{}) {
				require.NotEmpty(t, val)
				resultPods := val.([]v1.Pod)
				require.Equal(t, "changedSidecarPod", resultPods[0].Name)
			},
		},
		{
			name: "should return pod with different image tag",
			c: createClientSet(t,
				newSidecarPodBuilder().build(),
				newSidecarPodBuilder().
					setName("changedSidecarPod").
					setSidecarImageTag("1.11.0").
					build(),
			),
			assertFunc: func(t require.TestingT, val interface{}) {
				require.NotEmpty(t, val)
				resultPods := val.([]v1.Pod)
				require.Equal(t, "changedSidecarPod", resultPods[0].Name)

			},
		},
		{
			name: "should ignore pod that has different image tag when it has not all condition status as True",
			c: createClientSet(t,
				newSidecarPodBuilder().
					setSidecarImageTag("1.12.0").
					setConditionStatus("False").
					build(),
			),
			assertFunc: func(t require.TestingT, val interface{}) { require.Empty(t, val) },
		},
		{
			name: "should ignore pod that has different image tag when it has a deletion timestamp",
			c: createClientSet(t,
				newSidecarPodBuilder().
					setSidecarImageTag("1.12.0").
					setDeletionTimestamp(time.Now()).
					build(),
			),
			assertFunc: func(t require.TestingT, val interface{}) { require.Empty(t, val) },
		},
		{
			name: "should ignore pod that has different image tag when proxy container name is not in istio annotation",
			c: createClientSet(t,
				newSidecarPodBuilder().
					setSidecarImageTag("1.12.0").
					setSidecarContainerName("custom-sidecar-proxy-container-name").
					build(),
			),
			assertFunc: func(t require.TestingT, val interface{}) { require.Empty(t, val) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podList, err := pods.GetPodsWithDifferentSidecarImage(ctx, tt.c, expectedImage)

			require.NoError(t, err)
			tt.assertFunc(t, podList.Items)
		})
	}
}
