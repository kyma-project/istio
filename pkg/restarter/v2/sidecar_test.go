package v2

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSidecarImageChanged_Evaluate(t *testing.T) {
	tc := []struct {
		name           string
		expectedTag    string
		object         client.Object
		expectDecision Decision
	}{
		{
			name:        "tag in pod different than expected, restart",
			expectedTag: "1.31.1-distroless",
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "istio-proxy", Image: "europe-docker.pkg.dev/kyma/prod/proxyv2:1.30.0-distroless"},
						{Name: "app", Image: "foo/bar:1.0.0"},
					},
				},
			},
			expectDecision: Restart,
		},
		{
			name:        "tag matches even with different registry/name, continue",
			expectedTag: "1.31.1-distroless",
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "istio-proxy", Image: "other-registry.example.com/other/proxyv2:1.31.1-distroless"},
						{Name: "app", Image: "foo/bar:1.0.0"},
					},
				},
			},
			expectDecision: Continue,
		},
		{
			name:        "pod has custom image annotation, continue",
			expectedTag: "1.31.1-distroless",
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar",
					Annotations: map[string]string{"sidecar.istio.io/proxyImage": "custom/proxy:1.0.0"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "istio-proxy", Image: "custom/proxy:1.0.0"},
						{Name: "app", Image: "foo/bar:1.0.0"},
					},
				},
			},
			expectDecision: Continue,
		},
		{
			name:        "pod doesn't have a sidecar, continue",
			expectedTag: "1.31.1-distroless",
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "foo/bar:1.0.0"},
					},
				},
			},
			expectDecision: Continue,
		},
		{
			name:        "object not a pod, continue",
			expectedTag: "1.31.1-distroless",
			object: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
			},
			expectDecision: Continue,
		},
		{
			name:        "empty expected tag never forces restart, continue",
			expectedTag: "",
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "istio-proxy", Image: "registry/proxyv2:1.31.1-distroless"},
					},
				},
			},
			expectDecision: Continue,
		},
		{
			name:        "digest suffix stripped before tag comparison, tag matches, continue",
			expectedTag: "1.31.1-distroless",
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "istio-proxy", Image: "registry/proxyv2:1.31.1-distroless@sha256:abc123"},
					},
				},
			},
			expectDecision: Continue,
		},
		{
			name:        "registry with port, tag extracted correctly, matches, continue",
			expectedTag: "1.31.1-distroless",
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "istio-proxy", Image: "localhost:5000/ns/proxyv2:1.31.1-distroless"},
					},
				},
			},
			expectDecision: Continue,
		},
		{
			name:        "registry with port, tag differs, restart",
			expectedTag: "1.31.1-distroless",
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "istio-proxy", Image: "localhost:5000/ns/proxyv2:1.30.0-distroless"},
					},
				},
			},
			expectDecision: Restart,
		},
		{
			name:        "init container with mismatched tag, restart",
			expectedTag: "1.31.1-distroless",
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{Name: "istio-proxy", Image: "registry/proxyv2:1.30.0-distroless"},
					},
				},
			},
			expectDecision: Restart,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			eval := NewSidecarImageChangedRule(tt.expectedTag)
			d := eval.Evaluate(tt.object)
			if d != tt.expectDecision {
				t.Errorf("wrong decision: got %v, want %v", d, tt.expectDecision)
			}
		})
	}
}

func resources(cpuReq, memReq, cpuLim, memLim string) corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(cpuReq),
			corev1.ResourceMemory: resource.MustParse(memReq),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(cpuLim),
			corev1.ResourceMemory: resource.MustParse(memLim),
		},
	}
}

// limitOnly builds a ResourceRequirements with only Limits[cpu] and Requests[cpu]
// set, mirroring what ParseResourceAnnotations produces when only the cpu limit
// annotation is present (it resets both maps to empty, sets Limits[cpu], and because
// the cpu request annotation is absent, mirrors the limit into Requests[cpu]; memory
// keys remain unset).
func limitOnly(cpuLim string) corev1.ResourceRequirements {
	q := resource.MustParse(cpuLim)
	return corev1.ResourceRequirements{
		Limits:   corev1.ResourceList{corev1.ResourceCPU: q},
		Requests: corev1.ResourceList{corev1.ResourceCPU: q},
	}
}

func TestSidecarResourcesChanged_Evaluate(t *testing.T) {
	expected := resources("100m", "128Mi", "200m", "256Mi")

	tc := []struct {
		name           string
		req            corev1.ResourceRequirements
		object         client.Object
		expectDecision Decision
	}{
		{
			name: "resources in pod same as expected, continue",
			req:  expected,
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "istio-proxy", Resources: resources("100m", "128Mi", "200m", "256Mi")},
						{Name: "app"},
					},
				},
			},
			expectDecision: Continue,
		},
		{
			name: "cpu request differs, restart",
			req:  expected,
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "istio-proxy", Resources: resources("150m", "128Mi", "200m", "256Mi")},
					},
				},
			},
			expectDecision: Restart,
		},
		{
			name: "memory request differs, restart",
			req:  expected,
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "istio-proxy", Resources: resources("100m", "256Mi", "200m", "256Mi")},
					},
				},
			},
			expectDecision: Restart,
		},
		{
			name: "cpu limit differs, restart",
			req:  expected,
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "istio-proxy", Resources: resources("100m", "128Mi", "300m", "256Mi")},
					},
				},
			},
			expectDecision: Restart,
		},
		{
			name: "memory limit differs, restart",
			req:  expected,
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "istio-proxy", Resources: resources("100m", "128Mi", "200m", "512Mi")},
					},
				},
			},
			expectDecision: Restart,
		},
		{
			name: "sidecar in init containers same as expected, continue",
			req:  expected,
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{Name: "istio-proxy", Resources: resources("100m", "128Mi", "200m", "256Mi")},
					},
				},
			},
			expectDecision: Continue,
		},
		{
			name: "pod doesn't have a sidecar, continue",
			req:  expected,
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app"},
					},
				},
			},
			expectDecision: Continue,
		},
		{
			name: "object not a pod, continue",
			req:  expected,
			object: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
			},
			expectDecision: Continue,
		},
		{
			name: "annotation overrides all, pod matches annotation, continue",
			req:  expected,
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar", Annotations: map[string]string{
					"sidecar.istio.io/proxyCPU":         "111m",
					"sidecar.istio.io/proxyMemory":      "129Mi",
					"sidecar.istio.io/proxyCPULimit":    "222m",
					"sidecar.istio.io/proxyMemoryLimit": "257Mi",
				}},
				Spec: corev1.PodSpec{Containers: []corev1.Container{
					{Name: "istio-proxy", Resources: resources("111m", "129Mi", "222m", "257Mi")},
				}},
			},
			expectDecision: Continue,
		},
		{
			name: "annotation overrides cpu limit only, pod matches, continue",
			req:  expected,
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar", Annotations: map[string]string{
					"sidecar.istio.io/proxyCPULimit": "500m",
				}},
				// Istio resets then applies: only the annotated limit (and its request mirror) are set.
				Spec: corev1.PodSpec{Containers: []corev1.Container{
					{Name: "istio-proxy", Resources: limitOnly("500m")},
				}},
			},
			expectDecision: Continue,
		},
		{
			name: "annotation set but pod differs from annotation, restart",
			req:  expected,
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar", Annotations: map[string]string{
					"sidecar.istio.io/proxyCPU": "111m",
				}},
				Spec: corev1.PodSpec{Containers: []corev1.Container{
					{Name: "istio-proxy", Resources: resources("100m", "128Mi", "200m", "256Mi")},
				}},
			},
			expectDecision: Restart,
		},
		{
			name: "unparseable annotation, continue (webhook rejects pod)",
			req:  expected,
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar", Annotations: map[string]string{
					"sidecar.istio.io/proxyCPU": "not-a-quantity",
				}},
				Spec: corev1.PodSpec{Containers: []corev1.Container{
					{Name: "istio-proxy", Resources: resources("100m", "128Mi", "200m", "256Mi")},
				}},
			},
			expectDecision: Continue,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			eval := NewSidecarResourcesChangedRule(tt.req)
			d := eval.Evaluate(tt.object)
			if d != tt.expectDecision {
				t.Errorf("wrong decision: got %v, want %v", d, tt.expectDecision)
			}
		})
	}
}

func podTemplateAnnotations(annotations map[string]string) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Annotations: annotations}}
}

func TestParseResourceAnnotations(t *testing.T) {
	base := resources("100m", "128Mi", "200m", "256Mi")

	tc := []struct {
		name       string
		object     client.Object
		reqs       corev1.ResourceRequirements
		expectReqs corev1.ResourceRequirements
		expectErr  bool
	}{
		{
			name:       "no annotations, requirements unchanged",
			object:     &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			reqs:       base,
			expectReqs: base,
		},
		{
			name: "pod annotations override all fields",
			object: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"sidecar.istio.io/proxyCPU":         "300m",
				"sidecar.istio.io/proxyMemory":      "512Mi",
				"sidecar.istio.io/proxyCPULimit":    "400m",
				"sidecar.istio.io/proxyMemoryLimit": "1Gi",
			}}},
			reqs:       base,
			expectReqs: resources("300m", "512Mi", "400m", "1Gi"),
		},
		{
			name: "limits default the requests when requests are absent",
			object: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"sidecar.istio.io/proxyCPULimit":    "400m",
				"sidecar.istio.io/proxyMemoryLimit": "1Gi",
			}}},
			reqs:       base,
			expectReqs: resources("400m", "1Gi", "400m", "1Gi"),
		},
		{
			name: "deployment template annotations are honored",
			object: &appsv1.Deployment{Spec: appsv1.DeploymentSpec{
				Template: podTemplateAnnotations(map[string]string{"sidecar.istio.io/proxyCPU": "300m"}),
			}},
			reqs: base,
			expectReqs: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("300m")},
				Limits:   corev1.ResourceList{},
			},
		},
		{
			name: "daemonset template annotations are honored",
			object: &appsv1.DaemonSet{Spec: appsv1.DaemonSetSpec{
				Template: podTemplateAnnotations(map[string]string{"sidecar.istio.io/proxyMemory": "512Mi"}),
			}},
			reqs: base,
			expectReqs: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("512Mi")},
				Limits:   corev1.ResourceList{},
			},
		},
		{
			name: "statefulset template annotations are honored",
			object: &appsv1.StatefulSet{Spec: appsv1.StatefulSetSpec{
				Template: podTemplateAnnotations(map[string]string{"sidecar.istio.io/proxyCPU": "300m"}),
			}},
			reqs: base,
			expectReqs: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("300m")},
				Limits:   corev1.ResourceList{},
			},
		},
		{
			name: "replicaset template annotations are honored",
			object: &appsv1.ReplicaSet{Spec: appsv1.ReplicaSetSpec{
				Template: podTemplateAnnotations(map[string]string{"sidecar.istio.io/proxyCPU": "300m"}),
			}},
			reqs: base,
			expectReqs: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("300m")},
				Limits:   corev1.ResourceList{},
			},
		},
		{
			name: "invalid cpu request returns an error",
			object: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"sidecar.istio.io/proxyCPU": "not-a-quantity",
			}}},
			reqs:      base,
			expectErr: true,
		},
		{
			name: "invalid memory request returns an error",
			object: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"sidecar.istio.io/proxyMemory": "not-a-quantity",
			}}},
			reqs:      base,
			expectErr: true,
		},
		{
			name: "invalid cpu limit returns an error",
			object: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"sidecar.istio.io/proxyCPULimit": "not-a-quantity",
			}}},
			reqs:      base,
			expectErr: true,
		},
		{
			name: "invalid memory limit returns an error",
			object: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"sidecar.istio.io/proxyMemoryLimit": "not-a-quantity",
			}}},
			reqs:      base,
			expectErr: true,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseResourceAnnotations(tt.object, tt.reqs)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !resourcesEqual(got, tt.expectReqs) {
				t.Errorf("wrong requirements: got %v, want %v", got, tt.expectReqs)
			}
		})
	}
}

func resourcesEqual(a, b corev1.ResourceRequirements) bool {
	return a.Requests.Cpu().Equal(*b.Requests.Cpu()) &&
		a.Requests.Memory().Equal(*b.Requests.Memory()) &&
		a.Limits.Cpu().Equal(*b.Limits.Cpu()) &&
		a.Limits.Memory().Equal(*b.Limits.Memory())
}
