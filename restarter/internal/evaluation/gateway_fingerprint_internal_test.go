package evaluation

import (
	"testing"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
)

func TestGatewayConfigFingerprint(t *testing.T) {
	t.Run("all fields nil", func(t *testing.T) {
		cfg := configuration.AppliedConfig{}
		hash, err := gatewayConfigFingerprint(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hash != "" {
			t.Errorf("expected empty hash, got %q", hash)
		}
	})

	t.Run("only NumTrustedProxies set", func(t *testing.T) {
		cfg := configuration.AppliedConfig{
			IstioSpec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: new(1),
				},
			},
		}
		hash, err := gatewayConfigFingerprint(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hash == "" {
			t.Errorf("expected non-empty hash, got empty")
		}
	})

	t.Run("TrustDomain difference", func(t *testing.T) {
		cfg1 := configuration.AppliedConfig{
			IstioSpec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					TrustDomain: new("cluster.local"),
				},
			},
		}
		cfg2 := configuration.AppliedConfig{
			IstioSpec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					TrustDomain: new("different.local"),
				},
			},
		}
		hash1, err1 := gatewayConfigFingerprint(cfg1)
		hash2, err2 := gatewayConfigFingerprint(cfg2)
		if err1 != nil || err2 != nil {
			t.Fatalf("unexpected errors: %v, %v", err1, err2)
		}
		if hash1 == hash2 {
			t.Errorf("expected different hashes for different TrustDomains, got same: %q", hash1)
		}
	})

	t.Run("ForwardClientCertDetails difference", func(t *testing.T) {
		cfg1 := configuration.AppliedConfig{
			IstioSpec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					ForwardClientCertDetails: new(operatorv1alpha2.XFCCStrategy("APPEND_FORWARD")),
				},
			},
		}
		cfg2 := configuration.AppliedConfig{
			IstioSpec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					ForwardClientCertDetails: new(operatorv1alpha2.XFCCStrategy("SANITIZE_SET")),
				},
			},
		}
		hash1, err1 := gatewayConfigFingerprint(cfg1)
		hash2, err2 := gatewayConfigFingerprint(cfg2)
		if err1 != nil || err2 != nil {
			t.Fatalf("unexpected errors: %v, %v", err1, err2)
		}
		if hash1 == hash2 {
			t.Errorf("expected different hashes for different ForwardClientCertDetails, got same: %q", hash1)
		}
	})

	t.Run("ProxyStatsMatcher with regexps in different order yields same hash", func(t *testing.T) {
		cfg1 := configuration.AppliedConfig{
			IstioSpec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					ProxyStatsMatcher: &operatorv1alpha2.ProxyStatsMatcher{
						InclusionRegexps: []string{"aaa", "bbb", "ccc"},
					},
				},
			},
		}
		cfg2 := configuration.AppliedConfig{
			IstioSpec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					ProxyStatsMatcher: &operatorv1alpha2.ProxyStatsMatcher{
						InclusionRegexps: []string{"ccc", "aaa", "bbb"},
					},
				},
			},
		}
		hash1, err1 := gatewayConfigFingerprint(cfg1)
		hash2, err2 := gatewayConfigFingerprint(cfg2)
		if err1 != nil || err2 != nil {
			t.Fatalf("unexpected errors: %v, %v", err1, err2)
		}
		if hash1 != hash2 {
			t.Errorf("expected same hash for same regexps in different order, got %q vs %q", hash1, hash2)
		}
	})

	t.Run("non-gateway field (CompatibilityMode) does not affect hash", func(t *testing.T) {
		cfg1 := configuration.AppliedConfig{
			IstioSpec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: new(1),
				},
				CompatibilityMode: true,
			},
		}
		cfg2 := configuration.AppliedConfig{
			IstioSpec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: new(1),
				},
				CompatibilityMode: false,
			},
		}
		hash1, err1 := gatewayConfigFingerprint(cfg1)
		hash2, err2 := gatewayConfigFingerprint(cfg2)
		if err1 != nil || err2 != nil {
			t.Fatalf("unexpected errors: %v, %v", err1, err2)
		}
		if hash1 != hash2 {
			t.Errorf("expected same hash regardless of CompatibilityMode, got %q vs %q", hash1, hash2)
		}
	})
}

func intPtr(v int) *int {
	return &v
}

func strPtr(v string) *string {
	return &v
}
