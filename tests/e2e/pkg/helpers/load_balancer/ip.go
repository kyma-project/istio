package load_balancer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup/ipfamily"
)

// dnsWaitTimeout bounds how long we wait for Route 53 to publish records
// for a single address family. DualStack pays this up to twice in the
// worst case (ip4 then ip6, sequentially).
const dnsWaitTimeout = 3 * time.Minute

// GetLoadBalancerAddress returns the istio-ingressgateway's public
// "host:port" with the hostname preserved. Callers dial this string with an
// http.Client whose transport picks the IP family; DNS resolution happens
// there. Returning the resolved IP would strip SNI, break cert validation,
// and pin us to whichever family the resolver happened to return first.
func GetLoadBalancerAddress(t *testing.T, c client.Client) (string, error) {
	t.Helper()
	ctx := t.Context()
	istioIngressGatewayNamespaceName := types.NamespacedName{
		Name:      "istio-ingressgateway",
		Namespace: "istio-system",
	}

	var ingressHost string
	var ingressPort int32

	runsOnGardener, err := runsOnGardener(ctx, c)
	if err != nil {
		return "", err
	}

	if runsOnGardener {
		var svc corev1.Service

		// Wait for the LoadBalancer to be provisioned (up to 5 minutes)
		err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
			if err := c.Get(ctx, istioIngressGatewayNamespaceName, &svc); err != nil {
				if k8serrors.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}

			if len(svc.Status.LoadBalancer.Ingress) == 0 {
				return false, nil
			}
			ingress := svc.Status.LoadBalancer.Ingress[0]
			return ingress.Hostname != "" || ingress.IP != "", nil
		})

		if err != nil {
			return "", fmt.Errorf("failed to wait for LoadBalancer to be provisioned: %w", err)
		}

		ingress := svc.Status.LoadBalancer.Ingress[0]
		// Prefer the hostname (AWS NLB, most managed LBs). Fall back to IP
		// only when the LB is IP-based (bare-metal, some cloud providers).
		if ingress.Hostname != "" {
			ingressHost = ingress.Hostname
		} else {
			ingressHost = ingress.IP
		}

		for _, port := range svc.Spec.Ports {
			if port.Name == "http2" {
				ingressPort = port.Port
			}
		}

		// AWS NLBs surface their hostname in Service status before Route 53
		// has necessarily published records for every requested family. On a
		// dualstack shoot we have seen AAAA appear before A by ~30-90s;
		// tests that dial tcp4 in that window hit NXDOMAIN and burn their
		// per-request timeout on retries instead of failing loudly. Wait
		// here until each family we intend to exercise resolves, or return
		// a precise error naming the missing family. Skip the wait when the
		// LB is IP-based — LookupIP on a raw literal in the wrong family
		// returns an empty slice and would burn the full timeout.
		if net.ParseIP(ingressHost) == nil {
			if err := waitForDNS(t, ingressHost, ipfamily.From().DialNetworks()); err != nil {
				return "", err
			}
			// DNS records exist and are stable, but AWS NLB target-group
			// health checks lag behind by another 30-90s: dialling now can
			// still hit `dial tcpX <ip>:port: i/o timeout` because the
			// registered ENIs have no healthy backend yet. Probe TCP once
			// per family to gate on actual reachability, not resolver
			// state.
			if err := waitForTCPReady(t, ingressHost, ingressPort, ipfamily.From().DialNetworks()); err != nil {
				return "", err
			}
		}
	} else {
		// In case we are not running on Gardener we assume that it's a k3d cluster that has 127.0.0.1 as default address
		ingressHost = "localhost"
		ingressPort = 80
	}

	return fmt.Sprintf("%s:%d", ingressHost, ingressPort), nil
}

// waitForDNS polls the host resolver until every dial network in `networks`
// (values from ipfamily.DialNetworks, e.g. "tcp4"/"tcp6") returns a STABLE
// non-empty address set. Stability means two consecutive polls returned the
// same set of addresses (order-insensitive). This closes the window between
// "AWS Service.status has a hostname" and "Route 53 has published records
// for both families" on dualstack shoots, and also protects against
// partial propagation: an AWS NLB in a 3-AZ shoot registers up to 3 A
// records per family, and Route 53 can publish them staggered — dialling
// after the first record appears may hit an ENI whose data-plane is not
// yet ready. Requiring stability (matching consecutive results) avoids
// dialling into the transient middle state. On timeout the error names
// the family that never stabilised, so callers see e.g.
// `hostname X: no ip4 addresses` instead of a generic per-request NXDOMAIN
// retry loop. Parent-context cancellation is surfaced unwrapped so callers
// can errors.Is-check it.
func waitForDNS(t *testing.T, host string, networks []string) error {
	t.Helper()
	// Resolver "ip4" / "ip6" mirror the socket-family filter Go's dialer
	// applies for "tcp4" / "tcp6".
	ipNetworkFor := map[string]string{"tcp4": "ip4", "tcp6": "ip6"}
	for _, n := range networks {
		ipNet, ok := ipNetworkFor[n]
		if !ok {
			return fmt.Errorf("waitForDNS: unsupported network %q (want tcp4 or tcp6)", n)
		}
		if err := waitFamilyDNS(t, ipNet, host); err != nil {
			return err
		}
	}
	return nil
}

// waitFamilyDNS is the per-family half of waitForDNS: one resolver poll
// loop for one address family, returning nil once the returned set has
// been stable across two consecutive polls or a wrapped error naming the
// family on timeout.
func waitFamilyDNS(t *testing.T, ipNet, host string) error {
	t.Helper()
	ctx := t.Context()
	var previous []string
	lastErr := fmt.Errorf("no lookup attempted")
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, dnsWaitTimeout, true, func(ctx context.Context) (bool, error) {
		addrs, err := net.DefaultResolver.LookupIP(ctx, ipNet, host)
		switch {
		case err != nil:
			lastErr = err
			previous = nil
		case len(addrs) == 0:
			lastErr = fmt.Errorf("no %s addresses", ipNet)
			previous = nil
		default:
			current := normaliseAddrs(addrs)
			if previous != nil && slicesEqual(previous, current) {
				return true, nil
			}
			// First successful poll or set changed — record and wait
			// another interval to confirm stability.
			lastErr = fmt.Errorf("%s address set not yet stable: %v", ipNet, current)
			previous = current
		}
		t.Logf("waitForDNS %s %q: %v", ipNet, host, lastErr)
		return false, nil
	})
	if err == nil {
		return nil
	}
	// Preserve parent-context cancellation / deadline as-is so callers can
	// errors.Is against context.Canceled or context.DeadlineExceeded. Only
	// wrap when the poll's own timeout fired.
	if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(err, ctxErr) {
		return err
	}
	return fmt.Errorf("hostname %q: no stable %s addresses after %s: %w", host, ipNet, dnsWaitTimeout, lastErr)
}

// waitForTCPReady dials each requested network to `host:port` with a short
// per-attempt timeout until the connection succeeds (a completed TCP
// handshake) or `dnsWaitTimeout` elapses. It closes the successful
// connection immediately — the only signal we want is "the LB target group
// has at least one healthy backend on this family". This layer is
// necessary in addition to waitForDNS because AWS NLB registers Route 53
// records before its target-group health checks pass; dialling in that
// window returns `dial tcpX <ip>:port: i/o timeout`. Parent-context
// cancellation is surfaced unwrapped so callers can errors.Is-check it.
func waitForTCPReady(t *testing.T, host string, port int32, networks []string) error {
	t.Helper()
	ctx := t.Context()
	addr := net.JoinHostPort(host, strconv.Itoa(int(port)))
	for _, network := range networks {
		if network != "tcp4" && network != "tcp6" {
			return fmt.Errorf("waitForTCPReady: unsupported network %q (want tcp4 or tcp6)", network)
		}
		lastErr := fmt.Errorf("no dial attempted")
		attempt := 0
		err := wait.PollUntilContextTimeout(ctx, 5*time.Second, dnsWaitTimeout, true, func(ctx context.Context) (bool, error) {
			attempt++
			d := net.Dialer{Timeout: 5 * time.Second}
			conn, err := d.DialContext(ctx, network, addr)
			if err != nil {
				lastErr = err
				t.Logf("waitForTCPReady: %s dial to %q attempt %d failed: %v", network, addr, attempt, err)
				return false, nil
			}
			_ = conn.Close()
			t.Logf("waitForTCPReady: %s dial to %q attempt %d succeeded", network, addr, attempt)
			return true, nil
		})
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(err, ctxErr) {
				return err
			}
			return fmt.Errorf("host %q port %d: no %s TCP reachability after %s: %w", host, port, network, dnsWaitTimeout, lastErr)
		}
	}
	return nil
}

// normaliseAddrs converts a slice of net.IP into a sorted slice of strings so
// two lookups returning the same set in different order compare equal.
func normaliseAddrs(addrs []net.IP) []string {
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, a.String())
	}
	sort.Strings(out)
	return out
}

// slicesEqual reports whether two sorted []string slices are identical.
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func runsOnGardener(ctx context.Context, k8sClient client.Client) (bool, error) {
	cmShootInfo := corev1.ConfigMap{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "kube-system", Name: "shoot-info"}, &cmShootInfo)

	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
