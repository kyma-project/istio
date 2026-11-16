package evaluation

import (
	"strconv"

	corev1 "k8s.io/api/core/v1"
)

// PrometheusMergeConfigChanged is an Evaluator that detects when a pod's
// Prometheus scraping annotations no longer match the expected PrometheusMerge
// setting.
type PrometheusMergeConfigChanged struct {
	expectedPort string
	expectedPath string
	enabled      bool
}

func (c *PrometheusMergeConfigChanged) Name() string { return "PrometheusMergeConfigChanged" }

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
