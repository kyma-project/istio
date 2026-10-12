package v2

import (
	"context"
	"strconv"
	"strings"

	"istio.io/istio/pkg/config/mesh"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PrometheusMergeConfigChanged is an Evaluator that detects when a pod's
// Prometheus scraping annotations no longer match the expected PrometheusMerge
// setting.
type PrometheusMergeConfigChanged struct {
	expectedPort string
	expectedPath string
	enabled      bool
}

// Evaluate returns Restart when the pod's Prometheus annotations do not match the
// expected PrometheusMerge setting, and Continue otherwise.
func (c *PrometheusMergeConfigChanged) Evaluate(obj Object) Decision {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return Continue
	}
	annotations := pod.GetAnnotations()

	matchesPath := annotations["prometheus.io/path"] == c.expectedPath
	matchesPort := annotations["prometheus.io/port"] == c.expectedPort

	// When enabling PrometheusMerge, restart if prometheusMerge annotations are missing or incorrect.
	if c.enabled {
		if !matchesPath || !matchesPort {
			return Restart
		}
		return Continue
	}

	// When disabling PrometheusMerge, restart if prometheusMerge annotations are present and correct.
	if matchesPath || matchesPort {
		return Restart
	}
	return Continue
}

// NewPrometheusMergeConfigChanged returns an Evaluator that restarts workloads
// whose Prometheus annotations do not match the given PrometheusMerge setting and
// status port.
func NewPrometheusMergeConfigChanged(enabled bool, port int32) Evaluator {
	return &PrometheusMergeConfigChanged{
		expectedPort: strconv.FormatInt(int64(port), 10),
		expectedPath: "/stats/prometheus",
		enabled:      enabled,
	}
}

// GetStatusPort returns the Istio proxy status port read from the mesh
// configuration in the istio-system/istio ConfigMap. It falls back to the
// default status port when the ConfigMap, its mesh config, or the status port
// cannot be resolved.
func GetStatusPort(ctx context.Context, c client.Client) int32 {
	const defaultPort int32 = 15020

	istioConfigMap := &corev1.ConfigMap{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: "istio-system", Name: "istio"}, istioConfigMap); err != nil {
		return defaultPort
	}

	meshConfigYAML, hasMesh := istioConfigMap.Data["mesh"]
	if !hasMesh {
		return defaultPort
	}

	// Clean up the YAML string - remove any leading indicators like "|-"
	meshConfigYAML = strings.TrimSpace(strings.TrimPrefix(meshConfigYAML, "|-"))

	meshConfig, err := mesh.ApplyMeshConfigDefaults(meshConfigYAML)
	if err != nil {
		return defaultPort
	}

	// ApplyMeshConfigDefaults always sets a non-zero status port.
	return meshConfig.GetDefaultConfig().GetStatusPort()
}
