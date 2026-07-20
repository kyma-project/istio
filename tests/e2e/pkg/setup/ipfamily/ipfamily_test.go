package ipfamily_test

import (
	"slices"
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup/ipfamily"
)

func TestFromEnv(t *testing.T) {
	cases := []struct {
		env      string
		want     ipfamily.Family
		policy   corev1.IPFamilyPolicy
		families []corev1.IPFamily
		networks []string
	}{
		{"", ipfamily.IPv4Only, corev1.IPFamilyPolicySingleStack, []corev1.IPFamily{corev1.IPv4Protocol}, []string{"tcp4"}},
		{"ipv4", ipfamily.IPv4Only, corev1.IPFamilyPolicySingleStack, []corev1.IPFamily{corev1.IPv4Protocol}, []string{"tcp4"}},
		{"ipv6", ipfamily.IPv6Only, corev1.IPFamilyPolicySingleStack, []corev1.IPFamily{corev1.IPv6Protocol}, []string{"tcp6"}},
		{"dualstack", ipfamily.DualStack, corev1.IPFamilyPolicyPreferDualStack, []corev1.IPFamily{corev1.IPv6Protocol, corev1.IPv4Protocol}, []string{"tcp4", "tcp6"}},
	}

	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			t.Setenv("TEST_IP_FAMILY", tc.env)
			f := ipfamily.From()
			if f != tc.want {
				t.Fatalf("From() = %q, want %q", f, tc.want)
			}
			if got := f.Policy(); got != tc.policy {
				t.Errorf("Policy() = %q, want %q", got, tc.policy)
			}
			if got := f.Families(); !slices.Equal(got, tc.families) {
				t.Errorf("Families() = %v, want %v", got, tc.families)
			}
			if got := f.DialNetworks(); !slices.Equal(got, tc.networks) {
				t.Errorf("DialNetworks() = %v, want %v", got, tc.networks)
			}
		})
	}
}

func TestFromInvalidPanics(t *testing.T) {
	t.Setenv("TEST_IP_FAMILY", "bogus")
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on invalid TEST_IP_FAMILY")
		}
	}()
	_ = ipfamily.From()
}
