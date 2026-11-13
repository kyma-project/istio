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

// assertMergedResources checks that the SidecarResourcesChanged evaluator in
// evals treats a pod matching expect as Continue and a differing pod as Restart.
func assertMergedResources(t *testing.T, evals []evaluation.Evaluator, expect corev1.ResourceRequirements) {
	t.Helper()
	ev := sidecarResourcesEvaluator(t, evals)
	if got := ev.Evaluate(proxyPod("matching", expect)); got != evaluation.Continue {
		t.Errorf("expected Continue for pod matching merged resources, got %v", got)
	}
	differing := resources("1m", "1Mi", "2m", "2Mi")
	if got := ev.Evaluate(proxyPod("differing", differing)); got != evaluation.Restart {
		t.Errorf("expected Restart for pod with differing resources, got %v", got)
	}
}

func TestGetEvaluators(t *testing.T) {
	mergeBase := resources("100m", "128Mi", "200m", "256Mi")
	noMutationBase := resources("100m", "128Mi", "200m", "256Mi")

	tests := []struct {
		name  string
		cfg   evaluation.Config
		check func(t *testing.T, evals []evaluation.Evaluator, err error)
	}{
		{
			name: "nil istio CR returns error",
			cfg:  evaluation.Config{IstioCR: nil},
			check: func(t *testing.T, _ []evaluation.Evaluator, err error) {
				if err == nil {
					t.Fatalf("expected error for nil istio CR, got nil")
				}
			},
		},
		{
			name: "builds the full evaluator set",
			cfg: evaluation.Config{
				IstioCR:           &istioCR.Istio{},
				LastAppliedConfig: configuration.AppliedConfig{},
				Fingerprint:       "test-fingerprint-123",
				BetaFeatures:      istiofeatures.IstioFeatures{},
				SidecarResources:  corev1.ResourceRequirements{},
			},
			check: func(t *testing.T, evals []evaluation.Evaluator, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				want := map[string]bool{
					"CNIConfigChanged":             true,
					"FingerprintChanged":           true,
					"PrometheusMergeConfigChanged": true,
					"SidecarImageChanged":          true,
					"SidecarResourcesChanged":      true,
				}
				got := make(map[string]bool)
				for _, ev := range evals {
					got[ev.Name()] = true
				}
				for name := range want {
					if !got[name] {
						t.Errorf("wanted evaluator %q not found in %v", name, got)
					}
				}
			},
		},
		{
			name: "unparseable resource quantity returns error",
			cfg: evaluation.Config{
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
			},
			check: func(t *testing.T, _ []evaluation.Evaluator, err error) {
				if err == nil {
					t.Fatalf("expected an error for unparseable resource quantity, got nil")
				}
			},
		},
		{
			name: "does not mutate the provided sidecar resources",
			cfg: evaluation.Config{
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
				SidecarResources: noMutationBase,
			},
			check: func(t *testing.T, _ []evaluation.Evaluator, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				want := resource.MustParse("200m")
				if !noMutationBase.Limits.Cpu().Equal(want) {
					t.Errorf("base was mutated: cpu limit now %v", noMutationBase.Limits.Cpu())
				}
			},
		},
		{
			name: "merge: nil components, merged equals base",
			cfg: evaluation.Config{
				IstioCR:          &istioCR.Istio{},
				BetaFeatures:     istiofeatures.IstioFeatures{},
				Fingerprint:      "x",
				SidecarResources: mergeBase,
			},
			check: func(t *testing.T, evals []evaluation.Evaluator, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				assertMergedResources(t, evals, mergeBase)
			},
		},
		{
			name: "merge: proxy resources nil, merged equals base",
			cfg: evaluation.Config{
				IstioCR:           &istioCR.Istio{},
				LastAppliedConfig: configuration.AppliedConfig{IstioSpec: istioCR.IstioSpec{Components: proxyComponents(nil)}},
				BetaFeatures:      istiofeatures.IstioFeatures{},
				Fingerprint:       "x",
				SidecarResources:  mergeBase,
			},
			check: func(t *testing.T, evals []evaluation.Evaluator, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				assertMergedResources(t, evals, mergeBase)
			},
		},
		{
			name: "merge: override only cpu limit, other fields keep base",
			cfg: evaluation.Config{
				IstioCR: &istioCR.Istio{},
				LastAppliedConfig: configuration.AppliedConfig{IstioSpec: istioCR.IstioSpec{Components: proxyComponents(&istioCR.Resources{
					Limits: &istioCR.ResourceClaims{CPU: new("250m")},
				})}},
				BetaFeatures:     istiofeatures.IstioFeatures{},
				Fingerprint:      "x",
				SidecarResources: mergeBase,
			},
			check: func(t *testing.T, evals []evaluation.Evaluator, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				assertMergedResources(t, evals, resources("100m", "128Mi", "250m", "256Mi"))
			},
		},
		{
			name: "merge: override all four fields",
			cfg: evaluation.Config{
				IstioCR: &istioCR.Istio{},
				LastAppliedConfig: configuration.AppliedConfig{IstioSpec: istioCR.IstioSpec{Components: proxyComponents(&istioCR.Resources{
					Requests: &istioCR.ResourceClaims{CPU: new("300m"), Memory: new("512Mi")},
					Limits:   &istioCR.ResourceClaims{CPU: new("400m"), Memory: new("1Gi")},
				})}},
				BetaFeatures:     istiofeatures.IstioFeatures{},
				Fingerprint:      "x",
				SidecarResources: mergeBase,
			},
			check: func(t *testing.T, evals []evaluation.Evaluator, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				assertMergedResources(t, evals, resources("300m", "512Mi", "400m", "1Gi"))
			},
		},
		{
			name: "merge: override requests memory only",
			cfg: evaluation.Config{
				IstioCR: &istioCR.Istio{},
				LastAppliedConfig: configuration.AppliedConfig{IstioSpec: istioCR.IstioSpec{Components: proxyComponents(&istioCR.Resources{
					Requests: &istioCR.ResourceClaims{Memory: new("384Mi")},
				})}},
				BetaFeatures:     istiofeatures.IstioFeatures{},
				Fingerprint:      "x",
				SidecarResources: mergeBase,
			},
			check: func(t *testing.T, evals []evaluation.Evaluator, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				assertMergedResources(t, evals, resources("100m", "384Mi", "200m", "256Mi"))
			},
		},
		{
			name: "merge: base with nil maps, requests cpu set",
			cfg: evaluation.Config{
				IstioCR: &istioCR.Istio{},
				LastAppliedConfig: configuration.AppliedConfig{IstioSpec: istioCR.IstioSpec{Components: proxyComponents(&istioCR.Resources{
					Requests: &istioCR.ResourceClaims{CPU: new("150m")},
				})}},
				BetaFeatures:     istiofeatures.IstioFeatures{},
				Fingerprint:      "x",
				SidecarResources: corev1.ResourceRequirements{},
			},
			check: func(t *testing.T, evals []evaluation.Evaluator, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				assertMergedResources(t, evals, corev1.ResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("150m")},
					Limits:   corev1.ResourceList{},
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evals, err := evaluation.GetEvaluators(tt.cfg)
			tt.check(t, evals, err)
		})
	}
}

func TestGetGatewayEvaluators(t *testing.T) {
	tests := []struct {
		name        string
		fingerprint string
	}{
		{name: "seeds a single GatewayFingerprintChanged evaluator", fingerprint: "gw-fp-1"},
		{name: "empty fingerprint still yields one evaluator", fingerprint: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evals := evaluation.GetGatewayEvaluators(tt.fingerprint)

			if len(evals) != 1 {
				t.Fatalf("GetGatewayEvaluators returned %d evaluators, want 1", len(evals))
			}
			if got := evals[0].Name(); got != "GatewayFingerprintChanged" {
				t.Errorf("evaluator name = %q, want %q", got, "GatewayFingerprintChanged")
			}
		})
	}
}
