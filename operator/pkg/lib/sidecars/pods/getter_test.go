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
				fixPodWithSidecar("application1", "enabled", "istio/proxyv2", "1.10.0"),
				fixPodWithSidecar("application2", "enabled", "istio/proxyv2", "1.10.0"),
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

	tests := []struct {
		name          string
		c             client.Client
		expectedImage pods.SidecarImage
		wantError     bool
		assertFunc    func(t require.TestingT, val interface{})
	}{
		{
			name: "should not return pods without istio sidecar",
			c: createClientSet(t,
				fixPodWithoutSidecar("app", "custom"),
			),
			expectedImage: pods.SidecarImage{
				Repository: "istio/proxyv2",
				Tag:        "1.10.0",
			},
			wantError:  false,
			assertFunc: func(t require.TestingT, val interface{}) { require.Empty(t, val) },
		},
		{
			name: "should not return any pod when pods have correct image",
			c: createClientSet(t,
				fixPodWithSidecar("app", "custom", "istio/proxyv2", "1.10.0"),
			),
			expectedImage: pods.SidecarImage{
				Repository: "istio/proxyv2",
				Tag:        "1.10.0",
			},
			wantError:  false,
			assertFunc: func(t require.TestingT, val interface{}) { require.Empty(t, val) },
		},
		{
			name: "should return pod with different image repository",
			c: createClientSet(t,
				fixPodWithSidecar("app", "custom", "istio/proxyv2", "1.10.0"),
				fixPodWithSidecar("changedSidecarPod", "custom", "istio/different-proxy", "1.10.0"),
			),
			expectedImage: pods.SidecarImage{
				Repository: "istio/proxyv2",
				Tag:        "1.10.0",
			},
			wantError: false,
			assertFunc: func(t require.TestingT, val interface{}) {
				require.NotEmpty(t, val)
				resultPods := val.([]v1.Pod)
				require.Equal(t, "changedSidecarPod", resultPods[0].Name)

			},
		},
		{
			name: "should return pod with different image tag",
			c: createClientSet(t,
				fixPodWithSidecar("app", "custom", "istio/proxyv2", "1.10.0"),
				fixPodWithSidecar("changedSidecarPod", "custom", "istio/proxyv2", "1.11.0"),
			),
			expectedImage: pods.SidecarImage{
				Repository: "istio/proxyv2",
				Tag:        "1.10.0",
			},
			wantError: false,
			assertFunc: func(t require.TestingT, val interface{}) {
				require.NotEmpty(t, val)
				resultPods := val.([]v1.Pod)
				require.Equal(t, "changedSidecarPod", resultPods[0].Name)

			},
		},
		{
			name: "should ignore pod that has different image tag when it has not all condition status as True",
			c: createClientSet(t,
				fixPodWithSidecarAndConditionStatus("app", "custom", "istio/proxyv2", "1.12.0", "False"),
			),
			expectedImage: pods.SidecarImage{
				Repository: "istio/proxyv2",
				Tag:        "1.10.0",
			},
			wantError:  false,
			assertFunc: func(t require.TestingT, val interface{}) { require.Empty(t, val) },
		},
		{
			name: "should ignore pod that has different image tag when it has a deletion timestamp",
			c: createClientSet(t,
				fixPodWithSidecarWithDeletionTimestamp("app", "custom", "istio/proxyv2", "1.12.0"),
			),
			expectedImage: pods.SidecarImage{
				Repository: "istio/proxyv2",
				Tag:        "1.10.0",
			},
			wantError:  false,
			assertFunc: func(t require.TestingT, val interface{}) { require.Empty(t, val) },
		},
		{
			name: "should ignore pod that has different image tag when proxy container name is not in istio annotation",
			c: createClientSet(t,
				fixPodWithSidecarWithSidecarContainerName("app", "custom", "istio/proxyv2", "1.12.0", "custom-sidecar-proxy-name"),
			),
			expectedImage: pods.SidecarImage{
				Repository: "istio/proxyv2",
				Tag:        "1.10.0",
			},
			wantError:  false,
			assertFunc: func(t require.TestingT, val interface{}) { require.Empty(t, val) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podList, err := pods.GetPodsWithDifferentSidecarImage(ctx, tt.c, tt.expectedImage)

			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.assertFunc(t, podList.Items)
		})
	}
}
