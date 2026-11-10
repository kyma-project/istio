package evaluation_test

import (
	"testing"

	"github.com/kyma-project/istio/operator/pkg/labels"
)

func gatewayFP(t *testing.T, annotationValue string) string {
	t.Helper()
	p := newProvider(t,
		newIstioCR(map[string]string{labels.LastAppliedConfiguration: annotationValue}),
		newFeaturesConfigMap("1"),
	)
	_, fp, err := p.GatewaySnapshot(t.Context())
	if err != nil {
		t.Fatalf("GatewaySnapshot(%q): unexpected error: %v", annotationValue, err)
	}
	return fp
}

func TestProvider_GatewaySnapshot_Fingerprint(t *testing.T) {
	t.Run("empty annotation yields empty fingerprint", func(t *testing.T) {
		if fp := gatewayFP(t, ""); fp != "" {
			t.Errorf("expected empty fingerprint, got %q", fp)
		}
	})

	t.Run("gateway field set yields non-empty fingerprint", func(t *testing.T) {
		if fp := gatewayFP(t, `{"config":{"numTrustedProxies":1}}`); fp == "" {
			t.Errorf("expected non-empty fingerprint, got empty")
		}
	})

	t.Run("TrustDomain difference changes fingerprint", func(t *testing.T) {
		fp1 := gatewayFP(t, `{"config":{"trustDomain":"cluster.local"}}`)
		fp2 := gatewayFP(t, `{"config":{"trustDomain":"different.local"}}`)
		if fp1 == fp2 {
			t.Errorf("expected different fingerprints for different TrustDomains, got same: %q", fp1)
		}
	})

	t.Run("ForwardClientCertDetails difference changes fingerprint", func(t *testing.T) {
		fp1 := gatewayFP(t, `{"config":{"forwardClientCertDetails":"APPEND_FORWARD"}}`)
		fp2 := gatewayFP(t, `{"config":{"forwardClientCertDetails":"SANITIZE_SET"}}`)
		if fp1 == fp2 {
			t.Errorf("expected different fingerprints for different ForwardClientCertDetails, got same: %q", fp1)
		}
	})

	t.Run("ProxyStatsMatcher regexps in different order yield same fingerprint", func(t *testing.T) {
		fp1 := gatewayFP(t, `{"config":{"proxyStatsMatcher":{"inclusionRegexps":["aaa","bbb","ccc"]}}}`)
		fp2 := gatewayFP(t, `{"config":{"proxyStatsMatcher":{"inclusionRegexps":["ccc","aaa","bbb"]}}}`)
		if fp1 != fp2 {
			t.Errorf("expected same fingerprint for same regexps in different order, got %q vs %q", fp1, fp2)
		}
	})

	t.Run("CompatibilityMode does not affect gateway fingerprint", func(t *testing.T) {
		fp1 := gatewayFP(t, `{"config":{"numTrustedProxies":1},"compatibilityMode":true}`)
		fp2 := gatewayFP(t, `{"config":{"numTrustedProxies":1},"compatibilityMode":false}`)
		if fp1 != fp2 {
			t.Errorf("expected same fingerprint regardless of CompatibilityMode, got %q vs %q", fp1, fp2)
		}
	})
}
