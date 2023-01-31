package pods_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/test/helpers"
)

func createClientSet(t *testing.T, objects ...client.Object) client.Client {
	err := v1alpha1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)
	err = v1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(objects...).
		// TODO: WithIndex is not supported in current version of controller runtime, should be readded later on
		// WithIndex(&v1.Pod{}, "status.phase", helpers.FakePodStatusPhaseIndexer).
		Build()

	return fakeClient
}

func TestGetPodsForCNIChange(t *testing.T) {
	ctx := context.Background()
	logger := logr.Discard()

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
			c: createClientSet(t,
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
			c: createClientSet(t,
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
			c: createClientSet(t,
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
			c: createClientSet(t,
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
			c: createClientSet(t,
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
			c: createClientSet(t,
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
			c: createClientSet(t,
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
		t.Run(tt.name, func(t *testing.T) {
			podList, err := pods.GetPodsForCNIChange(ctx, tt.c, tt.isCNIEnabled, &logger)
			require.NoError(t, err)

			if tt.wantEmpty {
				require.Empty(t, podList.Items)
			} else {
				require.NotEmpty(t, podList.Items)
			}

			for _, pod := range podList.Items {
				require.Contains(t, tt.expectedPodNames, pod.Name)
			}

			require.Len(t, podList.Items, tt.wantLen)
		})
	}
}

func TestGetPodsWithDifferentSidecarImage(t *testing.T) {
	ctx := context.TODO()
	logger := logr.Discard()

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
				helpers.FixPodWithoutSidecar("app", "custom"),
			),
			assertFunc: func(t require.TestingT, val interface{}) { require.Empty(t, val) },
		},
		{
			name: "should not return any pod when pods have correct image",
			c: createClientSet(t,
				helpers.NewSidecarPodBuilder().Build(),
			),
			assertFunc: func(t require.TestingT, val interface{}) { require.Empty(t, val) },
		},
		{
			name: "should return pod with different image repository",
			c: createClientSet(t,
				helpers.NewSidecarPodBuilder().Build(),
				helpers.NewSidecarPodBuilder().
					SetName("changedSidecarPod").
					SetSidecarImageRepository("istio/different-proxy").
					Build(),
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
				helpers.NewSidecarPodBuilder().Build(),
				helpers.NewSidecarPodBuilder().
					SetName("changedSidecarPod").
					SetSidecarImageTag("1.11.0").
					Build(),
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
				helpers.NewSidecarPodBuilder().
					SetSidecarImageTag("1.12.0").
					SetConditionStatus("False").
					Build(),
			),
			assertFunc: func(t require.TestingT, val interface{}) { require.Empty(t, val) },
		},
		{
			name: "should ignore pod that has different image tag when phase is not running",
			c: createClientSet(t,
				helpers.NewSidecarPodBuilder().
					SetSidecarImageTag("1.12.0").
					SetPodStatusPhase("Pending").
					Build(),
			),
			assertFunc: func(t require.TestingT, val interface{}) { require.Empty(t, val) },
		},
		{
			name: "should ignore pod that has different image tag when it has a deletion timestamp",
			c: createClientSet(t,
				helpers.NewSidecarPodBuilder().
					SetSidecarImageTag("1.12.0").
					SetDeletionTimestamp(time.Now()).
					Build(),
			),
			assertFunc: func(t require.TestingT, val interface{}) { require.Empty(t, val) },
		},
		{
			name: "should ignore pod that has different image tag when proxy container name is not in istio annotation",
			c: createClientSet(t,
				helpers.NewSidecarPodBuilder().
					SetSidecarImageTag("1.12.0").
					SetSidecarContainerName("custom-sidecar-proxy-container-name").
					Build(),
			),
			assertFunc: func(t require.TestingT, val interface{}) { require.Empty(t, val) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podList, err := pods.GetPodsWithDifferentSidecarImage(ctx, tt.c, expectedImage, &logger)

			require.NoError(t, err)
			tt.assertFunc(t, podList.Items)
		})
	}
}
