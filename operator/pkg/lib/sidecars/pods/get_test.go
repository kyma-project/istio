package pods_test

import (
	"context"
	"fmt"
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
	enabledNamespace := fixNamespaceWith("enabled", map[string]string{"istio-injection": "enabled"})
	disabledNamespace := fixNamespaceWith("disabled", map[string]string{"istio-injection": "disabled"})

	tests := []struct {
		name          string
		c             client.Client
		expectedImage pods.SidecarImage
		isCNIEnabled  bool
		wantEmpty     bool
		wantLen       int
	}{
		{
			name: "should not get any pod without istio-init container when CNI is enabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").
					setInitContainer("istio-validation").setPodStatusPhase("Running").build(),
				newSidecarPodBuilder().setName("application2").setNamespace("enabled").
					setInitContainer("istio-validation").setPodStatusPhase("Terminating").build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			isCNIEnabled:  true,
			wantEmpty:     true,
			wantLen:       0,
		},
		{
			name: "should not get pods in system namespaces when CNI is enabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("kube-system").build(),
				newSidecarPodBuilder().setName("application2").setNamespace("kube-public").build(),
				newSidecarPodBuilder().setName("application3").setNamespace("istio-system").build(),
				newSidecarPodBuilder().setName("application4").setNamespace("enabled").build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			isCNIEnabled:  true,
			wantEmpty:     false,
			wantLen:       1,
		},
		{
			name: "should get 2 pods with istio-init when they are in Running state when CNI is enabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").build(),
				newSidecarPodBuilder().setName("application2").setNamespace("enabled").build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			isCNIEnabled:  true,
			wantEmpty:     false,
			wantLen:       2,
		},
		{
			name: "should not get pod with istio-init in Terminating state",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").build(),
				newSidecarPodBuilder().setName("application2").setNamespace("enabled").
					setPodStatusPhase("Terminating").build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			isCNIEnabled:  true,
			wantEmpty:     false,
			wantLen:       1,
		},
		{
			name: "should not get pod with istio-validation container when CNI is enabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").build(),
				newSidecarPodBuilder().setName("application2").setNamespace("enabled").
					setInitContainer("istio-validation").build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			isCNIEnabled:  true,
			wantEmpty:     false,
			wantLen:       1,
		},
		{
			name: "should get 2 pods with istio-validation container when CNI is disabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").
					setInitContainer("istio-validation").build(),
				newSidecarPodBuilder().setName("application2").setNamespace("enabled").
					setInitContainer("istio-validation").build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			isCNIEnabled:  false,
			wantEmpty:     false,
			wantLen:       2,
		},
		{
			name: "should not get any pod with istio-validation container in disabled namespace when CNI is disabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("disabled").
					setInitContainer("istio-validation").build(),
				disabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			isCNIEnabled:  false,
			wantEmpty:     true,
			wantLen:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podList, err := pods.GetPodsForCNIChange(ctx, tt.c, tt.isCNIEnabled)
			require.NoError(t, err)

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
			name: "should ignore pod that has different image tag when phase is not running",
			c: createClientSet(t,
				newSidecarPodBuilder().
					setSidecarImageTag("1.12.0").
					setPodStatusPhase("Pending").
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

func TestGetPodsWithoutSidecar(t *testing.T) {
	ctx := context.Background()
	enabledNamespace := fixNamespaceWith("enabled", map[string]string{"istio-injection": "enabled"})
	disabledNamespace := fixNamespaceWith("disabled", map[string]string{"istio-injection": "disabled"})
	noLabelNamespace := fixNamespaceWith("nolabel", map[string]string{"testns": "true"})

	type sidecarTest struct {
		name          string
		c             client.Client
		expectedImage pods.SidecarImage
		wantLen       int
	}

	sidecarInjectionEnabledTests := []sidecarTest{
		{
			name: "should get pod without Istio sidecar and with proper namespace label",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").
					disableSidecar().build(),
				newSidecarPodBuilder().setName("application2").setNamespace("enabled").
					disableSidecar().setPodStatusPhase("Terminating").build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       1,
		},
		{
			name: "should not get pods without Istio sidecar in system namespaces",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").
					disableSidecar().build(),
				newSidecarPodBuilder().setName("application2").setNamespace("kube-system").
					disableSidecar().build(),
				newSidecarPodBuilder().setName("application3").setNamespace("kube-public").
					disableSidecar().build(),
				newSidecarPodBuilder().setName("application4").setNamespace("istio-system").
					disableSidecar().build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       1,
		},
		{
			name: "should not get pod with Istio sidecar",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("nolabel").build(),
				noLabelNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should get pod without Istio sidecar in namespace labeled istio-injection=enabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").
					disableSidecar().build(),
				newSidecarPodBuilder().setName("application2").setNamespace("enabled").build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       1,
		},
		{
			name: "should not get pod without Istio sidecar in namespace labeled istio-injection=disabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("disabled").
					disableSidecar().build(),
				newSidecarPodBuilder().setName("application2").setNamespace("disabled").build(),
				disabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should not get pod with Istio sidecar and annotated sidecar.istio.io/inject=true in namespace labeled istio-injection=enabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").
					setPodAnnotations(map[string]string{"sidecar.istio.io/inject": "true"}).build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should get pod without Istio sidecar and annotated sidecar.istio.io/inject=true in namespace labeled istio-injection=enabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").
					disableSidecar().setPodAnnotations(map[string]string{"sidecar.istio.io/inject": "true"}).build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       1,
		},
		{
			name: "should not get pod without Istio sidecar and annotated sidecar.istio.io/inject=true in namespace labeled istio-injection=disabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("disabled").
					disableSidecar().setPodAnnotations(map[string]string{"sidecar.istio.io/inject": "true"}).build(),
				disabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should not get pod without Istio sidecar and annotated sidecar.istio.io/inject=false in namespace labeled istio-injection=enabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").
					disableSidecar().setPodAnnotations(map[string]string{"sidecar.istio.io/inject": "false"}).build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should not get pod without Istio sidecar and annotated sidecar.istio.io/inject=false in namespace labeled istio-injection=disabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("disabled").
					disableSidecar().setPodAnnotations(map[string]string{"sidecar.istio.io/inject": "false"}).build(),
				disabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should get pod without Istio sidecar in namespace without label",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("nolabel").
					disableSidecar().build(),
				noLabelNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       1,
		},
		{
			name: "should not get pod in HostNetwork",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("nolabel").setPodHostNetwork().
					disableSidecar().build(),
				newSidecarPodBuilder().setName("application2").setNamespace("nolabel").setPodHostNetwork().
					disableSidecar().setPodAnnotations(map[string]string{"sidecar.istio.io/inject": "true"}).build(),
				noLabelNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should get pod without Istio sidecar when annotated sidecar.istio.io/inject=true in namespace without label",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("nolabel").
					disableSidecar().setPodAnnotations(map[string]string{"sidecar.istio.io/inject": "true"}).build(),
				noLabelNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       1,
		},
		{
			name: "should not get pod without Istio sidecar when annotated sidecar.istio.io/inject=false in namespace without label",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("nolabel").
					disableSidecar().setPodAnnotations(map[string]string{"sidecar.istio.io/inject": "false"}).build(),
				noLabelNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should get pod without Istio sidecar and labeled sidecar.istio.io/inject=true (or not labeled) in namespace without label",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("nolabel").
					disableSidecar().setPodLabels(map[string]string{"sidecar.istio.io/inject": "true"}).build(),
				newSidecarPodBuilder().setName("application2").setNamespace("nolabel").
					disableSidecar().build(),
				noLabelNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       2,
		},
		{
			name: "should not get pod without Istio sidecar and labeled sidecar.istio.io/inject=false in namespace without label",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("nolabel").
					disableSidecar().setPodLabels(map[string]string{"sidecar.istio.io/inject": "false"}).build(),
				noLabelNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should not get pod without Istio sidecar and annotated sidecar.istio.io/inject=true and labeled sidecar.istio.io/inject=false in namespace labeled istio-injection=enabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").
					disableSidecar().setPodLabels(map[string]string{"sidecar.istio.io/inject": "false"}).
					setPodAnnotations(map[string]string{"sidecar.istio.io/inject": "true"}).build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should get pod without Istio sidecar and annotated sidecar.istio.io/inject=false and labeled sidecar.istio.io/inject=true in namespace labeled istio-injection=enabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").
					disableSidecar().setPodAnnotations(map[string]string{"sidecar.istio.io/inject": "false"}).
					setPodLabels(map[string]string{"sidecar.istio.io/inject": "true"}).build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       1,
		},
		{
			name: "should not get pod without Istio sidecar with label sidecar.istio.io/inject=true (or false) in namespace labeled istio-injection=enabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("disabled").disableSidecar().
					setPodLabels(map[string]string{"sidecar.istio.io/inject": "true"}).build(),
				newSidecarPodBuilder().setName("application2").setNamespace("disabled").disableSidecar().
					setPodLabels(map[string]string{"sidecar.istio.io/inject": "false"}).build(),
				disabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
	}

	for _, tt := range sidecarInjectionEnabledTests {
		t.Run(fmt.Sprintf("sidecar injection enabled by default %s", tt.name), func(t *testing.T) {
			podList, err := pods.GetPodsWithoutSidecar(ctx, tt.c, true)
			require.NoError(t, err)
			require.Len(t, podList.Items, tt.wantLen)
		})
	}

	sidecarInjectionDisabledTests := []sidecarTest{
		{
			name: "should get pod with proper namespace label",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").
					disableSidecar().build(),
				newSidecarPodBuilder().setName("application2").setNamespace("enabled").
					disableSidecar().setPodStatusPhase("Terminating").build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       1,
		},
		{
			name: "should not get pod with Istio sidecar",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should not get pod without Istio sidecar in namespace labeled istio-injection=disabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("disabled").
					disableSidecar().build(),
				newSidecarPodBuilder().setName("application2").setNamespace("disabled").build(),
				disabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should not get pod with Istio sidecar and annotated sidecar.istio.io/inject=true in namespace labeled istio-injection=enabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").
					setPodAnnotations(map[string]string{"sidecar.istio.io/inject": "true"}).build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should get pod without Istio sidecar and annotated sidecar.istio.io/inject=true in namespace labeled istio-injection=enabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").disableSidecar().
					setPodAnnotations(map[string]string{"sidecar.istio.io/inject": "true"}).build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       1,
		},
		{
			name: "should not get pod without Istio sidecar and annotated sidecar.istio.io/inject=true in namespace labeled istio-injection=disabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("disabled").disableSidecar().
					setPodAnnotations(map[string]string{"sidecar.istio.io/inject": "true"}).build(),
				disabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should not get pod without Istio sidecar and annotated sidecar.istio.io/inject=false in namespace labeled istio-injection=enabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").disableSidecar().
					setPodAnnotations(map[string]string{"sidecar.istio.io/inject": "false"}).build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should not get pod without Istio sidecar when not annotated in namespace without label",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("nolabel").
					disableSidecar().build(),
				noLabelNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should not get pod in HostNetwork",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("nolabel").disableSidecar().
					setPodAnnotations(map[string]string{"sidecar.istio.io/inject": "true"}).setPodHostNetwork().build(),
				newSidecarPodBuilder().setName("application2").setNamespace("nolabel").
					disableSidecar().setPodHostNetwork().build(),
				noLabelNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should not get pod without Istio sidecar and annotated sidecar.istio.io/inject=true in namespace without label",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("nolabel").disableSidecar().
					setPodAnnotations(map[string]string{"sidecar.istio.io/inject": "true"}).build(),
				noLabelNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should get pod without Istio sidecar and labeled sidecar.istio.io/inject=true in namespace without label",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("nolabel").disableSidecar().
					setPodLabels(map[string]string{"sidecar.istio.io/inject": "true"}).build(),
				noLabelNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       1,
		},
		{
			name: "should not get pod without Istio sidecar and annotated sidecar.istio.io/inject=true and labeled sidecar.istio.io/inject=false in namespace labeled istio-injection=enabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").disableSidecar().
					setPodLabels(map[string]string{"sidecar.istio.io/inject": "false"}).
					setPodAnnotations(map[string]string{"sidecar.istio.io/inject": "true"}).build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       0,
		},
		{
			name: "should get pod without Istio sidecar and annotated sidecar.istio.io/inject=false and labeled sidecar.istio.io/inject=true in namespace labeled istio-injection=enabled",
			c: createClientSet(t,
				newSidecarPodBuilder().setName("application1").setNamespace("enabled").disableSidecar().
					setPodLabels(map[string]string{"sidecar.istio.io/inject": "true"}).
					setPodAnnotations(map[string]string{"sidecar.istio.io/inject": "false"}).build(),
				enabledNamespace,
			),
			expectedImage: pods.SidecarImage{Repository: "istio/proxyv2", Tag: "1.10.0"},
			wantLen:       1,
		},
	}

	for _, tt := range sidecarInjectionDisabledTests {
		t.Run(fmt.Sprintf("sidecar injection disabled by default %s", tt.name), func(t *testing.T) {
			podList, err := pods.GetPodsWithoutSidecar(ctx, tt.c, false)
			require.NoError(t, err)
			require.Len(t, podList.Items, tt.wantLen)
		})
	}
}
