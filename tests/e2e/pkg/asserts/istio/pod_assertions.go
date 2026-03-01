package istioassert

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
)

const defaultTimeout = 30 * time.Second

// AssertIstioProxyPresent waits for the pod to have the istio-proxy sidecar
func AssertIstioProxyPresent(t *testing.T, c *resources.Resources, labelSelector string) {
	t.Helper()

	err := wait.For(func(ctx context.Context) (bool, error) {
		podList := &v1.PodList{}
		err := c.List(ctx, podList, resources.WithLabelSelector(labelSelector))
		if err != nil {
			return false, err
		}
		if len(podList.Items) == 0 {
			return false, nil
		}
		return hasIstioProxy(podList.Items[0]), nil
	}, wait.WithTimeout(defaultTimeout), wait.WithContext(t.Context()))
	require.NoError(t, err)
}

// AssertIstioProxyAbsent waits for the pod to not have the istio-proxy sidecar
func AssertIstioProxyAbsent(t *testing.T, c *resources.Resources, labelSelector string) {
	t.Helper()

	err := wait.For(func(ctx context.Context) (bool, error) {
		podList := &v1.PodList{}
		err := c.List(ctx, podList, resources.WithLabelSelector(labelSelector))
		if err != nil {
			return false, err
		}
		if len(podList.Items) == 0 {
			return false, nil
		}
		return !hasIstioProxy(podList.Items[0]), nil
	}, wait.WithTimeout(defaultTimeout), wait.WithContext(t.Context()))
	require.NoError(t, err)
}

// hasIstioProxy checks if a pod has the istio-proxy container
func hasIstioProxy(pod v1.Pod) bool {
	for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
		if container.Name == "istio-proxy" {
			return true
		}
	}
	return false
}

// AssertIstioProxyPresentInNamespace waits for the pod in a specific namespace to have the istio-proxy sidecar
func AssertIstioProxyPresentInNamespace(t *testing.T, c *resources.Resources, namespace, labelSelector string) {
	t.Helper()

	err := wait.For(func(ctx context.Context) (bool, error) {
		podList := &v1.PodList{}
		err := c.List(ctx, podList, resources.WithLabelSelector(labelSelector))
		if err != nil {
			return false, err
		}
		for _, pod := range podList.Items {
			if pod.Namespace == namespace && hasIstioProxy(pod) {
				return true, nil
			}
		}
		return false, nil
	}, wait.WithTimeout(defaultTimeout), wait.WithContext(t.Context()))
	require.NoError(t, err, "Pod with label %s in namespace %s should have istio-proxy sidecar", labelSelector, namespace)
}

