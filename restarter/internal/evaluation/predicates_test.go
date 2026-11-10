package evaluation_test

import (
	"testing"

	istioCR "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/istiofeatures"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
	"github.com/kyma-project/istio/operator/restarter/internal/evaluation"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func proxyComponents(res *istioCR.Resources) *istioCR.Components {
	return &istioCR.Components{
		Proxy: &istioCR.ProxyComponent{
			K8S: &istioCR.ProxyK8sConfig{Resources: res},
		},
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

func proxyPod(name string, res corev1.ResourceRequirements) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:      "istio-proxy",
					Resources: res,
				},
			},
		},
	}
}

func sidecarResourcesEvaluator(t *testing.T, evals []evaluation.Evaluator) evaluation.Evaluator {
	t.Helper()
	for _, e := range evals {
		if e.Name() == "SidecarResourcesChanged" {
			return e
		}
	}
	t.Fatal("SidecarResourcesChanged evaluator not present")
	return nil
}

func TestGetEvaluators_SidecarResourcesMerge(t *testing.T) {
	base := resources("100m", "128Mi", "200m", "256Mi")

	tc := []struct {
		name       string
		components *istioCR.Components
		base       corev1.ResourceRequirements
		expect     corev1.ResourceRequirements
	}{
		{
			name:       "nil components, merged equals base",
			components: nil,
			base:       base,
			expect:     base,
		},
		{
			name:       "proxy resources nil, merged equals base",
			components: proxyComponents(nil),
			base:       base,
			expect:     base,
		},
		{
			name: "override only cpu limit, other fields keep base",
			components: proxyComponents(&istioCR.Resources{
				Limits: &istioCR.ResourceClaims{CPU: new("250m")},
			}),
			base:   base,
			expect: resources("100m", "128Mi", "250m", "256Mi"),
		},
		{
			name: "override all four fields",
			components: proxyComponents(&istioCR.Resources{
				Requests: &istioCR.ResourceClaims{CPU: new("300m"), Memory: new("512Mi")},
				Limits:   &istioCR.ResourceClaims{CPU: new("400m"), Memory: new("1Gi")},
			}),
			base:   base,
			expect: resources("300m", "512Mi", "400m", "1Gi"),
		},
		{
			name: "override requests memory only",
			components: proxyComponents(&istioCR.Resources{
				Requests: &istioCR.ResourceClaims{Memory: new("384Mi")},
			}),
			base:   base,
			expect: resources("100m", "384Mi", "200m", "256Mi"),
		},
		{
			name: "base with nil maps, requests cpu set",
			components: proxyComponents(&istioCR.Resources{
				Requests: &istioCR.ResourceClaims{CPU: new("150m")},
			}),
			base: corev1.ResourceRequirements{},
			expect: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("150m")},
				Limits:   corev1.ResourceList{},
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			cfg := evaluation.Config{
				IstioCR:           &istioCR.Istio{},
				LastAppliedConfig: configuration.AppliedConfig{IstioSpec: istioCR.IstioSpec{Components: tt.components}},
				BetaFeatures:      istiofeatures.IstioFeatures{},
				Fingerprint:       "x",
				SidecarResources:  tt.base,
			}

			evals, err := evaluation.GetEvaluators(cfg)
			if err != nil {
				t.Fatalf("GetEvaluators: unexpected error: %v", err)
			}

			ev := sidecarResourcesEvaluator(t, evals)

			if got := ev.Evaluate(proxyPod("matching", tt.expect)); got != evaluation.Continue {
				t.Errorf("expected Continue for pod matching merged resources, got %v", got)
			}

			differing := resources("1m", "1Mi", "2m", "2Mi")
			if got := ev.Evaluate(proxyPod("differing", differing)); got != evaluation.Restart {
				t.Errorf("expected Restart for pod with differing resources, got %v", got)
			}
		})
	}
}

func TestGetEvaluators_UnparseableResourceError(t *testing.T) {
	cfg := evaluation.Config{
		IstioCR: &istioCR.Istio{},
		LastAppliedConfig: configuration.AppliedConfig{
			IstioSpec: istioCR.IstioSpec{
				Components: proxyComponents(&istioCR.Resources{
					Limits: &istioCR.ResourceClaims{CPU: new("not-a-quantity")},
				}),
			},
		},
		BetaFeatures:     istiofeatures.IstioFeatures{},
		Fingerprint:      "x",
		SidecarResources: resources("100m", "128Mi", "200m", "256Mi"),
	}

	if _, err := evaluation.GetEvaluators(cfg); err == nil {
		t.Fatalf("expected an error for unparseable resource quantity, got nil")
	}
}

func TestGetEvaluators_DoesNotMutateSidecarResources(t *testing.T) {
	base := resources("100m", "128Mi", "200m", "256Mi")
	cfg := evaluation.Config{
		IstioCR: &istioCR.Istio{},
		LastAppliedConfig: configuration.AppliedConfig{
			IstioSpec: istioCR.IstioSpec{
				Components: proxyComponents(&istioCR.Resources{
					Limits: &istioCR.ResourceClaims{CPU: new("999m")},
				}),
			},
		},
		BetaFeatures:     istiofeatures.IstioFeatures{},
		Fingerprint:      "x",
		SidecarResources: base,
	}

	if _, err := evaluation.GetEvaluators(cfg); err != nil {
		t.Fatalf("GetEvaluators: unexpected error: %v", err)
	}

	want := resource.MustParse("200m")
	if !base.Limits.Cpu().Equal(want) {
		t.Errorf("base was mutated: cpu limit now %v", base.Limits.Cpu())
	}
}

func TestGetEvaluators(t *testing.T) {
	tests := []struct {
		name        string
		fingerprint string
		wantNames   map[string]bool
	}{
		{
			name:        "builds the full evaluator set",
			fingerprint: "test-fingerprint-123",
			wantNames: map[string]bool{
				"CNIConfigChanged":             true,
				"FingerprintChanged":           true,
				"PrometheusMergeConfigChanged": true,
				"SidecarImageChanged":          true,
				"SidecarResourcesChanged":      true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := evaluation.Config{
				IstioCR:           &istioCR.Istio{},
				LastAppliedConfig: configuration.AppliedConfig{},
				Fingerprint:       tt.fingerprint,
				BetaFeatures:      istiofeatures.IstioFeatures{},
				SidecarResources:  corev1.ResourceRequirements{},
			}

			evaluators, err := evaluation.GetEvaluators(cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			evaluatorNames := make(map[string]bool)
			for _, ev := range evaluators {
				evaluatorNames[ev.Name()] = true
			}

			for wantName := range tt.wantNames {
				if !evaluatorNames[wantName] {
					t.Errorf("wanted evaluator %q not found in %v", wantName, evaluatorNames)
				}
			}
		})
	}
}
