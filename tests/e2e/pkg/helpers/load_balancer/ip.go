package load_balancer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
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
func GetLoadBalancerAddress(ctx context.Context, c client.Client) (string, error) {
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
			if err := waitForDNS(ctx, ingressHost, ipfamily.From().DialNetworks()); err != nil {
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

// GetLoadBalancerIP is retained for backwards compatibility with callers
// that predate the dualstack work; it returns the same hostname-preserving
// value as GetLoadBalancerAddress.
func GetLoadBalancerIP(ctx context.Context, c client.Client) (string, error) {
	return GetLoadBalancerAddress(ctx, c)
}

// waitForDNS polls the host resolver until every dial network in `networks`
// (values from ipfamily.DialNetworks, e.g. "tcp4"/"tcp6") returns at least
// one address. This closes the window between "AWS Service.status has a
// hostname" and "Route 53 has published records for both families" on
// dualstack shoots. On timeout the error names the family that never
// resolved, so callers see e.g. `hostname X: no ip4 addresses` instead of
// a generic per-request NXDOMAIN retry loop. Parent-context cancellation
// is surfaced unwrapped so callers can errors.Is-check it.
func waitForDNS(ctx context.Context, host string, networks []string) error {
	// Resolver "ip4" / "ip6" mirror the socket-family filter Go's dialer
	// applies for "tcp4" / "tcp6".
	ipNetworkFor := map[string]string{"tcp4": "ip4", "tcp6": "ip6"}

	for _, n := range networks {
		ipNet, ok := ipNetworkFor[n]
		if !ok {
			// Unknown network — trust the caller and skip.
			continue
		}
		lastErr := fmt.Errorf("no lookup attempted")
		attempt := 0
		err := wait.PollUntilContextTimeout(ctx, 5*time.Second, dnsWaitTimeout, true, func(ctx context.Context) (bool, error) {
			attempt++
			addrs, err := net.DefaultResolver.LookupIP(ctx, ipNet, host)
			if err != nil {
				lastErr = err
				log.Printf("waitForDNS: %s lookup for %q attempt %d failed: %v", ipNet, host, attempt, err)
				return false, nil
			}
			if len(addrs) == 0 {
				lastErr = fmt.Errorf("no %s addresses", ipNet)
				log.Printf("waitForDNS: %s lookup for %q attempt %d returned no addresses", ipNet, host, attempt)
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			// Preserve parent-context cancellation / deadline as-is so
			// callers can errors.Is against context.Canceled or
			// context.DeadlineExceeded. Only wrap when the poll's own
			// timeout fired.
			if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(err, ctxErr) {
				return err
			}
			return fmt.Errorf("hostname %q: no %s addresses after %s: %w", host, ipNet, dnsWaitTimeout, lastErr)
		}
	}
	return nil
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
