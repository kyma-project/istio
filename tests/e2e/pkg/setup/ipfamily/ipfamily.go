// Package ipfamily selects Kubernetes IP family behaviour for e2e test
// fixtures and clients based on the TEST_IP_FAMILY environment variable.
//
// TEST_IP_FAMILY values: "ipv4" (default), "ipv6", "dualstack". Any other
// value panics — the intent is to fail loudly in CI rather than silently
// drift back to the default.
package ipfamily

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"

	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"
)

type Family string

const (
	IPv4Only  Family = "ipv4"
	IPv6Only  Family = "ipv6"
	DualStack Family = "dualstack"

	envVar = "TEST_IP_FAMILY"
)

// From reads TEST_IP_FAMILY and returns the selected Family. Empty defaults
// to IPv4Only; anything unrecognised panics.
func From() Family {
	v := os.Getenv(envVar)
	switch v {
	case "", string(IPv4Only):
		return IPv4Only
	case string(IPv6Only):
		return IPv6Only
	case string(DualStack):
		return DualStack
	default:
		panic(fmt.Sprintf("ipfamily: unrecognised %s=%q (want ipv4|ipv6|dualstack)", envVar, v))
	}
}

// Validate returns an error if TEST_IP_FAMILY is set to something outside
// {"", ipv4, ipv6, dualstack}. Call this from TestMain (or the first
// fixture-building helper) so a typo in a CI workflow fails before a
// Gardener shoot is provisioned rather than mid-run with a panic.
func Validate() error {
	v := os.Getenv(envVar)
	switch v {
	case "", string(IPv4Only), string(IPv6Only), string(DualStack):
		return nil
	default:
		return fmt.Errorf("ipfamily: unrecognised %s=%q (want ipv4|ipv6|dualstack)", envVar, v)
	}
}

// Policy returns the Kubernetes Service.spec.ipFamilyPolicy for this family.
func (f Family) Policy() corev1.IPFamilyPolicy {
	switch f {
	case DualStack:
		return corev1.IPFamilyPolicyPreferDualStack
	default:
		return corev1.IPFamilyPolicySingleStack
	}
}

// Families returns the ordered list to set on Service.spec.ipFamilies.
// Order matters to Kubernetes: the first entry becomes the Service's
// primary IP family (ClusterIP is allocated from that family's range, and
// pods report it first in status.podIPs). DualStack uses IPv6-first so a
// dualstack run exercises the v6 primary path; single-family modes have
// only one entry so the choice is moot.
func (f Family) Families() []corev1.IPFamily {
	switch f {
	case IPv4Only:
		return []corev1.IPFamily{corev1.IPv4Protocol}
	case IPv6Only:
		return []corev1.IPFamily{corev1.IPv6Protocol}
	case DualStack:
		return []corev1.IPFamily{corev1.IPv6Protocol, corev1.IPv4Protocol}
	}
	return nil
}

// DialNetworks returns the Go `net` network strings a test should exercise
// for this family. IPv4Only / IPv6Only return a single-element slice pinning
// that family; DualStack returns both so a test asserts the service works
// over v4 AND v6. Pass a returned value as the `network` argument to
// `net.Dialer.DialContext`, or use "tcp4"/"tcp6" with an `http.Transport`
// custom DialContext.
func (f Family) DialNetworks() []string {
	switch f {
	case IPv4Only:
		return []string{"tcp4"}
	case IPv6Only:
		return []string{"tcp6"}
	case DualStack:
		return []string{"tcp4", "tcp6"}
	}
	return nil
}

// ForEachDialNetwork runs fn once per dial network configured by
// TEST_IP_FAMILY. Each invocation lives in a t.Run(network, ...) sub-test
// and receives a pre-built http.Client whose transport is pinned to that
// TCP family via httphelper.WithNetwork; the caller-supplied `label` is
// used as the httphelper log prefix suffixed with "-<network>", so
// per-family log lines are self-labelling (e.g. `[xff-header-tcp6]`).
// The httphelper.Option slice is shared across families — put per-family
// state inside fn.
//
// This is the canonical way to exercise LB dials against dualstack shoots:
// single-family modes run one sub-test, DualStack runs both v4 and v6 and
// asserts each independently. Missing WithNetwork on a raw t.Run loop
// makes the test silently use resolver-luck on dualstack; the helper
// removes that failure mode by construction.
func ForEachDialNetwork(t *testing.T, label string, opts []httphelper.Option, fn func(t *testing.T, network string, client *http.Client)) {
	t.Helper()
	for _, network := range From().DialNetworks() {
		t.Run(network, func(t *testing.T) {
			// Prepend the invariants so caller-supplied opts can override
			// them if a suite ever legitimately needs to (functional
			// options are last-write-wins).
			all := append([]httphelper.Option{
				httphelper.WithPrefix(label + "-" + network),
				httphelper.WithNetwork(network),
			}, opts...)
			fn(t, network, httphelper.NewHTTPClient(t, all...))
		})
	}
}
