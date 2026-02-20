package egressgateway

import (
	"strings"
	"testing"

	networkingv1alpha3 "istio.io/api/networking/v1alpha3"
	networkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
)

// toSubsetName converts a hostname to a valid subset name by replacing dots with hyphens.
// Subset names must be valid DNS labels (no dots allowed).
func toSubsetName(host string) string {
	return strings.ReplaceAll(host, ".", "-")
}

// SetupEgressGatewayResources creates the necessary Istio resources to route traffic through the egress gateway
func SetupEgressGatewayResources(t *testing.T, r *resources.Resources, namespace, externalHost string) error {
	t.Helper()

	// Create ServiceEntry for the external host
	if err := createServiceEntry(t, r, namespace, externalHost); err != nil {
		return err
	}

	// Create Gateway for the egress gateway
	if err := createEgressGateway(t, r, namespace, externalHost); err != nil {
		return err
	}

	// Create DestinationRule for the egress gateway
	if err := createDestinationRule(t, r, namespace, externalHost); err != nil {
		return err
	}

	// Create VirtualService to route traffic through the egress gateway
	if err := createEgressVirtualService(t, r, namespace, externalHost); err != nil {
		return err
	}

	return nil
}

func createServiceEntry(t *testing.T, r *resources.Resources, namespace, externalHost string) error {
	t.Helper()

	se := &networkingv1.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      externalHost + "-se",
			Namespace: namespace,
		},
		Spec: networkingv1alpha3.ServiceEntry{
			Hosts: []string{externalHost},
			Ports: []*networkingv1alpha3.ServicePort{
				{
					Number:   443,
					Name:     "tls",
					Protocol: "TLS",
				},
			},
			Resolution: networkingv1alpha3.ServiceEntry_DNS,
		},
	}

	t.Logf("Creating ServiceEntry %s/%s", namespace, se.Name)
	if err := r.Create(t.Context(), se); err != nil {
		t.Logf("Failed to create ServiceEntry: %v", err)
		return err
	}

	setup.DeclareCleanup(t, func() {
		if err := r.Delete(setup.GetCleanupContext(), se); err != nil {
			t.Logf("Failed to delete ServiceEntry: %v", err)
		}
	})

	return nil
}

func createEgressGateway(t *testing.T, r *resources.Resources, namespace, externalHost string) error {
	t.Helper()

	gw := &networkingv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-egressgateway",
			Namespace: namespace,
		},
		Spec: networkingv1alpha3.Gateway{
			Selector: map[string]string{
				"istio": "egressgateway",
			},
			Servers: []*networkingv1alpha3.Server{
				{
					Port: &networkingv1alpha3.Port{
						Number:   443,
						Name:     "tls",
						Protocol: "TLS",
					},
					Hosts: []string{externalHost},
					Tls: &networkingv1alpha3.ServerTLSSettings{
						Mode: networkingv1alpha3.ServerTLSSettings_PASSTHROUGH,
					},
				},
			},
		},
	}

	t.Logf("Creating Gateway %s/%s", namespace, gw.Name)
	if err := r.Create(t.Context(), gw); err != nil {
		t.Logf("Failed to create Gateway: %v", err)
		return err
	}

	setup.DeclareCleanup(t, func() {
		if err := r.Delete(setup.GetCleanupContext(), gw); err != nil {
			t.Logf("Failed to delete Gateway: %v", err)
		}
	})

	return nil
}

func createDestinationRule(t *testing.T, r *resources.Resources, namespace, externalHost string) error {
	t.Helper()

	subsetName := toSubsetName(externalHost)

	dr := &networkingv1.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "egressgateway-for-" + subsetName,
			Namespace: namespace,
		},
		Spec: networkingv1alpha3.DestinationRule{
			Host: "istio-egressgateway.istio-system.svc.cluster.local",
			Subsets: []*networkingv1alpha3.Subset{
				{
					Name: subsetName,
				},
			},
		},
	}

	t.Logf("Creating DestinationRule %s/%s", namespace, dr.Name)
	if err := r.Create(t.Context(), dr); err != nil {
		t.Logf("Failed to create DestinationRule: %v", err)
		return err
	}

	setup.DeclareCleanup(t, func() {
		if err := r.Delete(setup.GetCleanupContext(), dr); err != nil {
			t.Logf("Failed to delete DestinationRule: %v", err)
		}
	})

	return nil
}

func createEgressVirtualService(t *testing.T, r *resources.Resources, namespace, externalHost string) error {
	t.Helper()

	subsetName := toSubsetName(externalHost)

	vs := &networkingv1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "direct-" + subsetName + "-through-egress-gateway",
			Namespace: namespace,
		},
		Spec: networkingv1alpha3.VirtualService{
			Hosts:    []string{externalHost},
			Gateways: []string{"mesh", "istio-egressgateway"},
			Tls: []*networkingv1alpha3.TLSRoute{
				{
					Match: []*networkingv1alpha3.TLSMatchAttributes{
						{
							Gateways:  []string{"mesh"},
							Port:      443,
							SniHosts:  []string{externalHost},
						},
					},
					Route: []*networkingv1alpha3.RouteDestination{
						{
							Destination: &networkingv1alpha3.Destination{
								Host:   "istio-egressgateway.istio-system.svc.cluster.local",
								Subset: subsetName,
								Port: &networkingv1alpha3.PortSelector{
									Number: 443,
								},
							},
						},
					},
				},
				{
					Match: []*networkingv1alpha3.TLSMatchAttributes{
						{
							Gateways:  []string{"istio-egressgateway"},
							Port:      443,
							SniHosts:  []string{externalHost},
						},
					},
					Route: []*networkingv1alpha3.RouteDestination{
						{
							Destination: &networkingv1alpha3.Destination{
								Host: externalHost,
								Port: &networkingv1alpha3.PortSelector{
									Number: 443,
								},
							},
							Weight: 100,
						},
					},
				},
			},
		},
	}

	t.Logf("Creating VirtualService %s/%s", namespace, vs.Name)
	if err := r.Create(t.Context(), vs); err != nil {
		t.Logf("Failed to create VirtualService: %v", err)
		return err
	}

	setup.DeclareCleanup(t, func() {
		if err := r.Delete(setup.GetCleanupContext(), vs); err != nil {
			t.Logf("Failed to delete VirtualService: %v", err)
		}
	})

	return nil
}

