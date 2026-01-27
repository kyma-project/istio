package telemetry

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/e2e-framework/klient/wait"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
)

// OtelCollectorInfo contains information about a deployed OTel collector
type OtelCollectorInfo struct {
	// Name is the name of the collector pod
	Name string
	// Namespace is the namespace where collector is deployed
	Namespace string
	// ContainerName is the name of the container to check logs from
	ContainerName string
	// WorkloadSelector is the label selector in "key=value" format used to identify the workload pods
	WorkloadSelector string
}

func CreateOtelMockCollector(t *testing.T) (*OtelCollectorInfo, error) {
	t.Helper()
	rc, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to create resources client: %v", err)
		return nil, err
	}
	exposedPort := 4317
	namespace := "kyma-system"
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otel-collector-mock",
			Namespace: namespace,
			Labels: map[string]string{
				"app": "otel-collector-mock",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "otel-collector-mock",
					Image: "docker.io/istio/tcp-echo-server:1.2",
					Args:  []string{strconv.Itoa(exposedPort), "one"},
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: int32(exposedPort),
							Protocol:      corev1.ProtocolTCP,
						},
					},
				},
			},
		},
	}

	err = rc.Create(t.Context(), pod)
	if err != nil {
		t.Logf("Failed to create Otel Mock Collector container: %v", err)
		return nil, err
	}

	setup.DeclareCleanup(t, func() {
		err := rc.Delete(setup.GetCleanupContext(), pod)
		if err != nil {
			t.Logf("Failed to delete Otel Mock Collector pod: %v", err)
		} else {
			t.Logf("Otel Mock Collector pod deleted")
		}
	})

	err = wait.For(func(ctx context.Context) (done bool, err error) {
		p := &corev1.Pod{}
		err = rc.Get(t.Context(), "otel-collector-mock", namespace, p)
		if err != nil {
			t.Logf("Failed to get Otel Mock Collector pod: %v", err)
			return false, nil
		}
		if p.Status.Phase != corev1.PodRunning {
			t.Logf("Otel Mock Collector pod is not running yet. Current phase: %s", p.Status.Phase)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Logf("Otel Mock Collector pod is not running: %v", err)
		return nil, err
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "telemetry-otlp-traces",
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "otel-collector-mock",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "otlp-grpc",
					Port:       4317,
					Protocol:   "TCP",
					TargetPort: intstr.FromInt32(4317),
				},
			},
		},
	}

	err = rc.Create(t.Context(), svc)
	if err != nil {
		t.Logf("Failed to create Otel Mock Collector service: %v", err)
		return nil, err
	}
	setup.DeclareCleanup(t, func() {
		err := rc.Delete(setup.GetCleanupContext(), svc)
		if err != nil {
			t.Logf("Failed to delete Otel Mock Collector service: %v", err)
		}
	})

	return &OtelCollectorInfo{
		Name:             pod.Name,
		Namespace:        namespace,
		ContainerName:    "otel-collector-mock",
		WorkloadSelector: fmt.Sprintf("app=%s", "otel-collector-mock"),
	}, nil
}
