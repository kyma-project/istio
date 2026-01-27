package istioassert

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/distribution/reference"
	"github.com/masterminds/semver"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"

	"github.com/kyma-project/istio/operator/internal/istiooperator"
)

const versionCheckTimeout = 3 * time.Minute

// AssertIstiodContainerVersion asserts that the discovery container in the istiod deployment has the required version
func AssertIstiodContainerVersion(t *testing.T, c *resources.Resources) {
	t.Helper()

	merger := istiooperator.NewDefaultIstioMerger()
	istioImageVersion, err := merger.GetIstioImageVersion()
	require.NoError(t, err)

	t.Logf("Expected Istio version from manifest: %s", istioImageVersion.Tag())

	istiodDeployment := &v1.Deployment{}
	err = c.Get(t.Context(), "istiod", "istio-system", istiodDeployment)
	require.NoError(t, err)

	assertContainerVersion(t, istiodDeployment.Spec.Template.Spec.Containers, "discovery", istioImageVersion.Tag())
	t.Logf("Istiod discovery container version matches expected version: %s", istioImageVersion.Tag())
}

// AssertIngressGatewayContainerVersion asserts that the istio-proxy container in the istio-ingressgateway deployment has the required version
func AssertIngressGatewayContainerVersion(t *testing.T, c *resources.Resources) {
	t.Helper()

	merger := istiooperator.NewDefaultIstioMerger()
	istioImageVersion, err := merger.GetIstioImageVersion()
	require.NoError(t, err)

	t.Logf("Expected Istio version from manifest: %s", istioImageVersion.Tag())

	ingressDeployment := &v1.Deployment{}
	err = c.Get(t.Context(), "istio-ingressgateway", "istio-system", ingressDeployment)
	require.NoError(t, err)

	assertContainerVersion(t, ingressDeployment.Spec.Template.Spec.Containers, "istio-proxy", istioImageVersion.Tag())
	t.Logf("Ingress gateway istio-proxy container version matches expected version: %s", istioImageVersion.Tag())
}

// AssertCNINodeContainerVersion asserts that the install-cni container in the istio-cni-node daemonset has the required version
func AssertCNINodeContainerVersion(t *testing.T, c *resources.Resources) {
	t.Helper()

	merger := istiooperator.NewDefaultIstioMerger()
	istioImageVersion, err := merger.GetIstioImageVersion()
	require.NoError(t, err)

	t.Logf("Expected Istio version from manifest: %s", istioImageVersion.Tag())

	cniDaemonSet := &v1.DaemonSet{}
	err = c.Get(t.Context(), "istio-cni-node", "istio-system", cniDaemonSet)
	require.NoError(t, err)

	assertContainerVersion(t, cniDaemonSet.Spec.Template.Spec.Containers, "install-cni", istioImageVersion.Tag())
	t.Logf("CNI install-cni container version matches expected version: %s", istioImageVersion.Tag())
}

// AssertIstioProxyVersion waits for the pod to have the istio-proxy sidecar with the required version
func AssertIstioProxyVersion(t *testing.T, c *resources.Resources, labelSelector string) {
	t.Helper()

	merger := istiooperator.NewDefaultIstioMerger()
	istioImageVersion, err := merger.GetIstioImageVersion()
	require.NoError(t, err)

	t.Logf("Expected Istio version from manifest: %s", istioImageVersion.Tag())

	err = wait.For(func(ctx context.Context) (bool, error) {
		podList := &corev1.PodList{}
		err := c.List(ctx, podList, resources.WithLabelSelector(labelSelector))
		if err != nil {
			return false, err
		}
		if len(podList.Items) == 0 {
			return false, nil
		}

		pod := podList.Items[0]
		for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
			if container.Name != "istio-proxy" {
				continue
			}
			deployedVersion, err := getVersionFromImageName(container.Image)
			if err != nil {
				return false, err
			}
			return deployedVersion == istioImageVersion.Tag(), nil
		}
		return false, fmt.Errorf("istio-proxy container not found in pod %s", pod.Name)
	}, wait.WithTimeout(versionCheckTimeout), wait.WithContext(t.Context()))
	require.NoError(t, err)
	t.Logf("Istio-proxy sidecar (label selector: %s) version matches expected version: %s", labelSelector, istioImageVersion.Tag())
}

// assertContainerVersion is a helper function that asserts a specific container has the required version
func assertContainerVersion(t *testing.T, containers []corev1.Container, containerName, expectedVersion string) {
	t.Helper()

	hasExpectedVersion := false
	for _, container := range containers {
		if container.Name != containerName {
			continue
		}
		deployedVersion, err := getVersionFromImageName(container.Image)
		require.NoError(t, err)
		t.Logf("Container %s has deployed version: %s (expected: %s)", containerName, deployedVersion, expectedVersion)
		require.Equal(t, expectedVersion, deployedVersion,
			"container %s has version %s but expected %s", containerName, deployedVersion, expectedVersion)
		hasExpectedVersion = true
		break
	}
	require.True(t, hasExpectedVersion, "container %s not found", containerName)
}

// getVersionFromImageName extracts the version from a container image reference
func getVersionFromImageName(image string) (string, error) {
	noVersion := ""
	matches := reference.ReferenceRegexp.FindStringSubmatch(image)
	if len(matches) < 3 {
		return noVersion, fmt.Errorf("unable to parse container image reference: %s", image)
	}
	version, err := semver.NewVersion(matches[2])
	if err != nil {
		return noVersion, err
	}
	return version.String(), nil
}
