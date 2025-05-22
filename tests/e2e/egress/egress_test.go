package egress_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"istio.io/api/networking/v1alpha3"
	"istio.io/api/telemetry/v1alpha1"
	istionetworkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	istiotelemetryv1 "istio.io/client-go/pkg/apis/telemetry/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
)

var (
	defaultWaitBackoff = wait.Backoff{
		Cap:      10 * time.Minute,
		Duration: time.Second,
		Steps:    30,
		Factor:   1.5,
	}
)

// TestE2EEgressConnectivity tests the connectivity of istio installed through istio-module.
// This test expects that istio-module is installed and access to cluster is set up via KUBECONFIG env.
func TestE2EEgressConnectivity(t *testing.T) {
	// initialization
	ctx := t.Context()
	cfg := config.GetConfigOrDie()
	c, err := initK8sClient(cfg)
	if err != nil {
		t.Fatalf("failed to init cluster client: %v", err)
	}
	setupIstio(ctx, c, t)

	// initialize testcases
	// note: test might fail randomly from random downtime to httpbin.org with error Connection reset by peer.
	// This is a flake, and we need to think how to resolve that eventually.
	tc := []struct {
		name                 string
		cmd                  []string
		expectError          bool
		applyNetworkPolicy   bool
		applyEgressConfig    bool
		enableIstioInjection bool
	}{
		{
			name:                 "connection to httpbin service is OK when egress is deployed",
			cmd:                  []string{"curl", "-sSL", "-m", "10", "https://httpbin.org/headers"},
			enableIstioInjection: true,
			applyEgressConfig:    true,
			expectError:          false,
		},
		{
			name:               "connection to httpbin service is refused when NetworkPolicy is applied",
			cmd:                []string{"curl", "-sSL", "-m", "10", "https://httpbin.org/headers"},
			applyNetworkPolicy: true,
			// sidecar init fails when NP is applied. When uncommented, the test will pass despite confirming manually
			// that connection is refused with NP
			enableIstioInjection: true,
			expectError:          true,
		},
		{
			name:                 "connection to httpbin service is OK when NetworkPolicy is applied and egress is configured",
			cmd:                  []string{"curl", "-sSL", "-m", "10", "https://httpbin.org/headers"},
			enableIstioInjection: true,
			applyEgressConfig:    true,
			applyNetworkPolicy:   true,
			expectError:          false,
		},
		{
			name:                 "connection to kyma-project is refused when NetworkPolicy is applied and egress is configured",
			cmd:                  []string{"curl", "-sSL", "-m", "10", "https://kyma-project.io"},
			enableIstioInjection: true,
			applyEgressConfig:    true,
			applyNetworkPolicy:   true,
			expectError:          true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			ns := createTestNamespace(ctx, c, tt.enableIstioInjection, t)
			if tt.applyEgressConfig {
				err := applyEgressConfig(ctx, c, ns)
				if err != nil {
					t.Errorf("failed to create egress resources: %v", err)
				}
			}
			if tt.applyNetworkPolicy {
				applyNetworPolicy(ctx, c, t, ns)
			}

			out, err := runCurlInPod(ctx, cfg, ns, tt.cmd)
			if err != nil && !tt.expectError {
				t.Errorf("got an error but shouldn't have: %v", err)
			}
			if err == nil && tt.expectError {
				t.Error("didn't get an error but expected one")
			}

			t.Log(string(out))
		})
	}
}

func setupIstio(ctx context.Context, c client.Client, t *testing.T) {
	// this function is a helper
	t.Helper()
	// configure cluster before testcases
	// init istio CR and wait for installation
	istioCR := operatorv1alpha2.Istio{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-operator",
			Namespace: "kyma-system",
		},
		Spec: operatorv1alpha2.IstioSpec{
			Components: &operatorv1alpha2.Components{
				EgressGateway: &operatorv1alpha2.EgressGateway{
					Enabled: ptr.To(true),
				},
			},
		},
	}
	t.Cleanup(func() {
		_ = c.Delete(ctx, &istioCR)
		_ = wait.ExponentialBackoffWithContext(ctx, defaultWaitBackoff, func(ctx context.Context) (bool, error) {
			ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "istio-system"}}
			err := c.Get(ctx, client.ObjectKeyFromObject(&ns), &ns)
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		})
	})

	if err := c.Create(ctx, &istioCR); err != nil {
		t.Fatalf("failed to create Istio CR: %v", err)
	}
	// wait for resource as function
	err := wait.ExponentialBackoffWithContext(ctx, defaultWaitBackoff, func(ctx context.Context) (bool, error) {
		err := c.Get(ctx, client.ObjectKeyFromObject(&istioCR), &istioCR)
		if err != nil {
			return false, err
		}
		state := string(istioCR.Status.State)
		if state == "Ready" {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	logsCR := istiotelemetryv1.Telemetry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mesh-default",
			Namespace: "istio-system",
		},
		Spec: v1alpha1.Telemetry{
			AccessLogging: []*v1alpha1.AccessLogging{
				{
					Providers: []*v1alpha1.ProviderRef{
						{
							Name: "envoy",
						},
					},
				},
			},
		},
	}
	t.Cleanup(func() {
		_ = c.Delete(ctx, &logsCR)
	})
	if err := c.Create(ctx, &logsCR); err != nil {
		t.Fatalf("failed to create istio telemetry resource: %v", err)
	}
}
func createTestNamespace(ctx context.Context, c client.Client, istioInjection bool, t *testing.T) string {
	t.Helper()
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "e2e-egress-",
			Labels: func() map[string]string {
				if istioInjection {
					return map[string]string{"istio-injection": "enabled"}
				}
				return nil
			}(),
		},
	}
	t.Cleanup(func() {
		_ = c.Delete(ctx, &ns)
	})
	if err := c.Create(ctx, &ns); err != nil {
		t.Errorf("failed to create namespace: %v", err)
	}
	return ns.Name
}
func runCurlInPod(ctx context.Context, cfg *rest.Config, namespace string, command []string) ([]byte, error) {
	c, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	run := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "curl-",
			Namespace:    namespace,
			Labels:       map[string]string{"test-workload": "true"},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:    "curl",
					Image:   "curlimages/curl",
					Command: command,
				},
			},
		},
	}
	run, err = c.CoreV1().Pods(namespace).Create(ctx, run, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	var exitCode int32
	err = wait.ExponentialBackoffWithContext(ctx, defaultWaitBackoff, func(ctx context.Context) (bool, error) {
		p, err := c.CoreV1().Pods(namespace).Get(ctx, run.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if len(p.Status.ContainerStatuses) == 0 {
			return false, nil
		}
		for _, s := range p.Status.ContainerStatuses {
			if s.Name == "curl" && s.State.Terminated != nil {
				exitCode = s.State.Terminated.ExitCode
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	out, err := c.CoreV1().Pods(namespace).GetLogs(run.Name, &corev1.PodLogOptions{Container: "curl"}).DoRaw(ctx)
	if err != nil {
		return nil, err
	}
	if exitCode != 0 {
		return out, fmt.Errorf("non-zero exit code: %v", exitCode)
	}

	return out, nil
}
func applyNetworPolicy(ctx context.Context, c client.Client, t *testing.T, namespace string) {
	t.Helper()
	np := networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "e2e-network-policy-",
			Namespace:    namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							IPBlock: &networkingv1.IPBlock{
								CIDR: "169.254.20.10/32",
							},
						},
						{
							IPBlock: &networkingv1.IPBlock{
								// this can cause the test to fail if the Service IP range changes in different clusters.
								// Proceed with caution.
								CIDR: "100.104.0.0/13",
							},
						},
						{
							IPBlock: &networkingv1.IPBlock{
								CIDR: "fd30:1319:f1e:230b::1/128",
							},
						},
						{
							NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{
								"kubernetes.io/metadata.name": "kube-system",
							}},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: ptr.To(corev1.ProtocolUDP),
							Port:     ptr.To(intstr.FromInt32(53)),
						},
					},
				},
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{
								"kubernetes.io/metadata.name": "istio-system",
							}},
						},
					},
				},
			}},
	}
	if err := c.Create(ctx, &np); err != nil {
		t.Errorf("failed to create NetworkPolicy: %v", err)
	}
}
func applyEgressConfig(ctx context.Context, c client.Client, namespace string) error {
	// TODO (@Ressetkk): set up fixture to define different types of configuration (HTTP, TLS, TLS Origination)
	sEntry := &istionetworkingv1.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "e2e-service-entry-",
			Namespace:    namespace,
		},
		Spec: v1alpha3.ServiceEntry{
			Hosts: []string{"httpbin.org"},
			Ports: []*v1alpha3.ServicePort{
				{
					Name:     "tls",
					Number:   443,
					Protocol: "TLS",
				},
			},
			Resolution: v1alpha3.ServiceEntry_DNS,
		},
	}
	if err := c.Create(ctx, sEntry); err != nil {
		return err
	}
	gateway := &istionetworkingv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "egress-gateway-",
			Namespace:    namespace,
		},
		Spec: v1alpha3.Gateway{
			Selector: map[string]string{"istio": "egressgateway"},
			Servers: []*v1alpha3.Server{
				{
					Port: &v1alpha3.Port{
						Number:   443,
						Protocol: "TLS",
						Name:     "tls",
					},
					Hosts: []string{"httpbin.org"},
					Tls:   &v1alpha3.ServerTLSSettings{Mode: v1alpha3.ServerTLSSettings_PASSTHROUGH},
				},
			},
		},
	}
	if err := c.Create(ctx, gateway); err != nil {
		return err
	}
	dr := &istionetworkingv1.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "egress-dr-",
			Namespace:    namespace,
		},
		Spec: v1alpha3.DestinationRule{
			Host: "istio-egressgateway.istio-system.svc.cluster.local",
			Subsets: []*v1alpha3.Subset{
				{
					Name: "kyma-project",
				},
			},
		},
	}
	if err := c.Create(ctx, dr); err != nil {
		return err
	}
	vs := &istionetworkingv1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "egress-vs-",
			Namespace:    namespace,
		},
		Spec: v1alpha3.VirtualService{
			Hosts:    []string{"httpbin.org"},
			Gateways: []string{gateway.Name, "mesh"},
			Tls: []*v1alpha3.TLSRoute{
				{
					Match: []*v1alpha3.TLSMatchAttributes{
						{
							Gateways: []string{"mesh"},
							Port:     443,
							SniHosts: []string{"httpbin.org"},
						},
					},
					Route: []*v1alpha3.RouteDestination{
						{
							Destination: &v1alpha3.Destination{
								Host:   "istio-egressgateway.istio-system.svc.cluster.local",
								Subset: "kyma-project",
								Port:   &v1alpha3.PortSelector{Number: 443},
							},
						},
					},
				},
				{
					Match: []*v1alpha3.TLSMatchAttributes{
						{
							Gateways: []string{gateway.Name},
							Port:     443,
							SniHosts: []string{"httpbin.org"},
						},
					},
					Route: []*v1alpha3.RouteDestination{
						{
							Destination: &v1alpha3.Destination{
								Host: "httpbin.org",
								Port: &v1alpha3.PortSelector{Number: 443},
							},
							Weight: 100,
						},
					},
				},
			},
		},
	}
	if err := c.Create(ctx, vs); err != nil {
		return err
	}
	return nil
}
func initK8sClient(config *rest.Config) (client.Client, error) {
	// (@Ressetkk): it would be better to remove usage of controller-runtime client.Client
	// 	and generate ClientSet for our API using client-gen, as we need to sometimes use non-CRUD operations, which
	//	 controller-runtime does not implement.
	c, err := client.New(config, client.Options{})
	if err != nil {
		return nil, err
	}

	_ = operatorv1alpha2.AddToScheme(c.Scheme())
	_ = istionetworkingv1.AddToScheme(c.Scheme())
	_ = istiotelemetryv1.AddToScheme(c.Scheme())

	return c, nil
}
