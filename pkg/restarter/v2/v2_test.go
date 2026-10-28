package v2

import (
	"testing"

	istioCR "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/istiofeatures"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func proxyComponents(res *istioCR.Resources) *istioCR.Components {
	return &istioCR.Components{
		Proxy: &istioCR.ProxyComponent{
			K8S: &istioCR.ProxyK8sConfig{Resources: res},
		},
	}
}

func TestEffectiveSidecarResources(t *testing.T) {
	base := resources("100m", "128Mi", "200m", "256Mi")

	tc := []struct {
		name       string
		components *istioCR.Components
		base       corev1.ResourceRequirements
		expect     corev1.ResourceRequirements
		expectErr  bool
	}{
		{
			name:       "nil components, base unchanged",
			components: nil,
			base:       base,
			expect:     base,
		},
		{
			name:       "proxy resources nil, base unchanged",
			components: proxyComponents(nil),
			base:       base,
			expect:     base,
		},
		{
			name: "last applied overrides only cpu limit, other fields keep base",
			components: proxyComponents(&istioCR.Resources{
				Limits: &istioCR.ResourceClaims{CPU: ptr.To("250m")},
			}),
			base:   base,
			expect: resources("100m", "128Mi", "250m", "256Mi"),
		},
		{
			name: "last applied overrides all four fields",
			components: proxyComponents(&istioCR.Resources{
				Requests: &istioCR.ResourceClaims{CPU: ptr.To("300m"), Memory: ptr.To("512Mi")},
				Limits:   &istioCR.ResourceClaims{CPU: ptr.To("400m"), Memory: ptr.To("1Gi")},
			}),
			base:   base,
			expect: resources("300m", "512Mi", "400m", "1Gi"),
		},
		{
			name: "last applied overrides requests memory only",
			components: proxyComponents(&istioCR.Resources{
				Requests: &istioCR.ResourceClaims{Memory: ptr.To("384Mi")},
			}),
			base:   base,
			expect: resources("100m", "384Mi", "200m", "256Mi"),
		},
		{
			name: "base with nil maps, last applied sets cpu request",
			components: proxyComponents(&istioCR.Resources{
				Requests: &istioCR.ResourceClaims{CPU: ptr.To("150m")},
			}),
			base: corev1.ResourceRequirements{},
			expect: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("150m")},
				Limits:   corev1.ResourceList{},
			},
		},
		{
			name: "unparseable value returns error",
			components: proxyComponents(&istioCR.Resources{
				Limits: &istioCR.ResourceClaims{CPU: ptr.To("not-a-quantity")},
			}),
			base:      base,
			expectErr: true,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			got, err := effectiveSidecarResources(tt.components, tt.base)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !resourcesEqual(got, tt.expect) {
				t.Errorf("wrong resources: got %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestEffectiveSidecarResources_DoesNotMutateBase(t *testing.T) {
	base := resources("100m", "128Mi", "200m", "256Mi")
	components := proxyComponents(&istioCR.Resources{
		Limits: &istioCR.ResourceClaims{CPU: ptr.To("999m")},
	})

	if _, err := effectiveSidecarResources(components, base); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := resource.MustParse("200m")
	if !base.Limits.Cpu().Equal(want) {
		t.Errorf("base was mutated: cpu limit now %v", base.Limits.Cpu())
	}
}

func TestGetEvaluators(t *testing.T) {
	tests := []struct {
		name                  string
		useFingerprintDrift   bool
		fingerprint           string
		wantNames             map[string]bool
		wantExcludedNames     map[string]bool
	}{
		{
			name:                "per-field drift mode",
			useFingerprintDrift: false,
			fingerprint:         "",
			wantNames: map[string]bool{
				"CNIConfigChanged":                      true,
				"PrometheusMergeConfigChanged":          true,
				"SidecarImageChanged":                   true,
				"SidecarResourcesChanged":               true,
				"CompatibilityModeChanged":              true,
				"DNSProxyingChanged":                    true,
				"NumTrustedProxiesConfigChanged":        true,
				"ForwardClientCertDetailsConfigChanged": true,
				"TrustDomainConfigChanged":              true,
				"ProxyStatsMatcherConfigChanged":        true,
			},
			wantExcludedNames: map[string]bool{
				"FingerprintDriftChanged": true,
			},
		},
		{
			name:                "fingerprint-drift mode",
			useFingerprintDrift: true,
			fingerprint:         "test-fingerprint-123",
			wantNames: map[string]bool{
				"CNIConfigChanged":            true,
				"PrometheusMergeConfigChanged": true,
				"SidecarImageChanged":          true,
				"SidecarResourcesChanged":      true,
				"FingerprintDriftChanged":      true,
			},
			wantExcludedNames: map[string]bool{
				"CompatibilityModeChanged":              true,
				"DNSProxyingChanged":                    true,
				"NumTrustedProxiesConfigChanged":        true,
				"ForwardClientCertDetailsConfigChanged": true,
				"TrustDomainConfigChanged":              true,
				"ProxyStatsMatcherConfigChanged":        true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				IstioCR:             &istioCR.Istio{},
				LastAppliedConfig:   configuration.AppliedConfig{},
				UseFingerprintDrift: tt.useFingerprintDrift,
				Fingerprint:         tt.fingerprint,
				BetaFeatures:        istiofeatures.IstioFeatures{},
				SidecarResources:    corev1.ResourceRequirements{},
			}

			evaluators, err := GetEvaluators(cfg)
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

			for excludedName := range tt.wantExcludedNames {
				if evaluatorNames[excludedName] {
					t.Errorf("excluded evaluator %q should not be present in %v", excludedName, evaluatorNames)
				}
			}
		})
	}
}
