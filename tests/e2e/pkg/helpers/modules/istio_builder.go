package modules

import (
	"context"
	"encoding/json"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/extauth"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
)

// IstioCRBuilder provides a fluent API for building Istio custom resources.
//
// Usage Examples:
//
//	// Basic usage with default values
//	err := modules.NewIstioCRBuilder().ApplyAndCleanup(t)
//
//	// Custom configuration
//	err := modules.NewIstioCRBuilder().
//		WithName("my-istio").
//		WithNamespace("istio-system").
//		WithPilotResources("100m", "128Mi", "1000m", "1Gi").
//		WithPilotHPA(1, 5).
//		WithIngressGatewayResources("100m", "128Mi", "2000m", "1Gi").
//		ApplyAndCleanup(t)
//
//	// Advanced component configuration
//	pilot := modules.NewPilotComponent()
//	pilot.K8s.Resources = modules.NewResources("200m", "256Mi", "2000m", "2Gi")
//
//	err := modules.NewIstioCRBuilder().
//		WithPilot(pilot).
//		WithEgressGateway(modules.NewEgressGatewayComponent(true)).
//		ApplyAndCleanup(t)
//
//	// Build without applying (for testing or manual apply)
//	istioCR := modules.NewIstioCRBuilder().
//		WithName("test-istio").
//		Build()
//
//	// Update existing CR
//	builder := modules.NewIstioCRBuilder().WithName("my-istio")
//	builder.ApplyAndCleanup(t)
//	// ... later
//	builder.WithCompatibilityMode(true).Update(t)
type IstioCRBuilder struct {
	istio *v1alpha2.Istio
}

// NewIstioCRBuilder creates a new builder with default values
func NewIstioCRBuilder() *IstioCRBuilder {
	return &IstioCRBuilder{
		istio: &v1alpha2.Istio{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "operator.kyma-project.io/v1alpha2",
				Kind:       "Istio",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default",
				Namespace: "kyma-system",
				Labels: map[string]string{
					"app.kubernetes.io/name": "default",
				},
			},
			Spec: v1alpha2.IstioSpec{},
		},
	}
}

// WithName sets the name of the Istio CR
func (b *IstioCRBuilder) WithName(name string) *IstioCRBuilder {
	b.istio.ObjectMeta.Name = name
	return b
}

// WithNamespace sets the namespace of the Istio CR
func (b *IstioCRBuilder) WithNamespace(namespace string) *IstioCRBuilder {
	b.istio.ObjectMeta.Namespace = namespace
	return b
}

// WithLabel adds a label to the Istio CR
func (b *IstioCRBuilder) WithLabel(key, value string) *IstioCRBuilder {
	if b.istio.ObjectMeta.Labels == nil {
		b.istio.ObjectMeta.Labels = make(map[string]string)
	}
	b.istio.ObjectMeta.Labels[key] = value
	return b
}

// WithAnnotation adds an annotation to the Istio CR
func (b *IstioCRBuilder) WithAnnotation(key, value string) *IstioCRBuilder {
	if b.istio.ObjectMeta.Annotations == nil {
		b.istio.ObjectMeta.Annotations = make(map[string]string)
	}
	b.istio.ObjectMeta.Annotations[key] = value
	return b
}

// WithNumTrustedProxies sets the number of trusted proxies
func (b *IstioCRBuilder) WithNumTrustedProxies(num int) *IstioCRBuilder {
	b.istio.Spec.Config.NumTrustedProxies = &num
	return b
}

// WithForwardClientCertDetails sets the X-Forwarded-Client-Cert header strategy
func (b *IstioCRBuilder) WithForwardClientCertDetails(strategy v1alpha2.XFCCStrategy) *IstioCRBuilder {
	b.istio.Spec.Config.ForwardClientCertDetails = &strategy
	return b
}

// WithAuthorizer adds an authorizer to the configuration
func (b *IstioCRBuilder) WithAuthorizer(authorizer *v1alpha2.Authorizer) *IstioCRBuilder {
	b.istio.Spec.Config.Authorizers = append(b.istio.Spec.Config.Authorizers, authorizer)
	return b
}

// WithGatewayExternalTrafficPolicy sets the external traffic policy for the Istio Ingress Gateway
func (b *IstioCRBuilder) WithGatewayExternalTrafficPolicy(policy string) *IstioCRBuilder {
	b.istio.Spec.Config.GatewayExternalTrafficPolicy = &policy
	return b
}

// WithTelemetry sets the telemetry configuration
func (b *IstioCRBuilder) WithTelemetry(telemetry v1alpha2.Telemetry) *IstioCRBuilder {
	b.istio.Spec.Config.Telemetry = telemetry
	return b
}

// WithTrustDomain sets the trust domain
func (b *IstioCRBuilder) WithTrustDomain(trustDomain string) *IstioCRBuilder {
	b.istio.Spec.Config.TrustDomain = &trustDomain
	return b
}

// WithCompatibilityMode enables or disables compatibility mode
func (b *IstioCRBuilder) WithCompatibilityMode(enabled bool) *IstioCRBuilder {
	b.istio.Spec.CompatibilityMode = enabled
	return b
}

// WithEnableModuleNetworkPolicies enables or disables module-managed NetworkPolicies
func (b *IstioCRBuilder) WithEnableModuleNetworkPolicies(enabled bool) *IstioCRBuilder {
	b.istio.Spec.NetworkPoliciesEnabled= enabled
	return b
}

// WithExperimental sets the experimental configuration
func (b *IstioCRBuilder) WithExperimental(experimental *v1alpha2.Experimental) *IstioCRBuilder {
	b.istio.Spec.Experimental = experimental
	return b
}

// WithPilot configures the Istiod component
func (b *IstioCRBuilder) WithPilot(pilot *v1alpha2.IstioComponent) *IstioCRBuilder {
	if b.istio.Spec.Components == nil {
		b.istio.Spec.Components = &v1alpha2.Components{}
	}
	b.istio.Spec.Components.Pilot = pilot
	return b
}

// WithIngressGateway configures the Istio Ingress Gateway component
func (b *IstioCRBuilder) WithIngressGateway(ingressGateway *v1alpha2.IstioComponent) *IstioCRBuilder {
	if b.istio.Spec.Components == nil {
		b.istio.Spec.Components = &v1alpha2.Components{}
	}
	b.istio.Spec.Components.IngressGateway = ingressGateway
	return b
}

// WithEgressGateway configures the Istio Egress Gateway component
func (b *IstioCRBuilder) WithEgressGateway(egressGateway *v1alpha2.EgressGateway) *IstioCRBuilder {
	if b.istio.Spec.Components == nil {
		b.istio.Spec.Components = &v1alpha2.Components{}
	}
	b.istio.Spec.Components.EgressGateway = egressGateway
	return b
}

// WithCNI configures the Istio CNI DaemonSet component
func (b *IstioCRBuilder) WithCNI(cni *v1alpha2.CniComponent) *IstioCRBuilder {
	if b.istio.Spec.Components == nil {
		b.istio.Spec.Components = &v1alpha2.Components{}
	}
	b.istio.Spec.Components.Cni = cni
	return b
}

// WithProxy configures the Istio sidecar proxy component
func (b *IstioCRBuilder) WithProxy(proxy *v1alpha2.ProxyComponent) *IstioCRBuilder {
	if b.istio.Spec.Components == nil {
		b.istio.Spec.Components = &v1alpha2.Components{}
	}
	b.istio.Spec.Components.Proxy = proxy
	return b
}

// Build returns the constructed Istio CR
func (b *IstioCRBuilder) Build() *v1alpha2.Istio {
	return b.istio
}

// ApplyAndCleanup applies the Istio CR and registers a cleanup function
func (b *IstioCRBuilder) ApplyAndCleanup(t *testing.T) (*v1alpha2.Istio, error) {
	t.Helper()

	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return nil, err
	}

	icr := b.Build()

	// Log the Istio CR before applying
	logIstioCR(t, icr)

	err = r.Create(t.Context(), icr)
	if err != nil {
		t.Logf("Failed to create Istio custom resource: %v", err)
		return nil, err
	}

	// Register cleanup before waiting for readiness
	registerIstioCRCleanup(t, icr)

	err = waitForIstioCRReadiness(t, r, icr)
	if err != nil {
		t.Logf("Istio custom resource is not ready: %v", err)
		return nil, err
	}

	t.Log("Istio custom resource applied successfully with cleanup registered")
	return icr, nil
}

// ApplyAndCleanupWithoutReadinessCheck applies the Istio CR and registers a cleanup function
// but does NOT wait for the CR to become ready. Use this when you expect the CR to be in
// Error, Warning, or other non-Ready states.
func (b *IstioCRBuilder) ApplyAndCleanupWithoutReadinessCheck(t *testing.T) (*v1alpha2.Istio, error) {
	t.Helper()

	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return nil, err
	}

	icr := b.Build()

	// Log the Istio CR before applying
	logIstioCR(t, icr)

	err = r.Create(t.Context(), icr)
	if err != nil {
		t.Logf("Failed to create Istio custom resource: %v", err)
		return nil, err
	}

	// Register cleanup
	registerIstioCRCleanup(t, icr)

	t.Log("Istio custom resource applied successfully with cleanup registered (without readiness check)")
	return icr, nil
}

// Update updates an existing Istio CR in the cluster
func (b *IstioCRBuilder) Update(t *testing.T) error {
	t.Helper()
	t.Log("Updating Istio custom resource")

	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return err
	}

	// Build the desired Istio CR with updates
	desiredIcr := b.Build()

	// First, get the existing Istio CR from the cluster to obtain the resourceVersion
	existingIcr := &v1alpha2.Istio{}
	err = r.Get(t.Context(), desiredIcr.Name, desiredIcr.Namespace, existingIcr)
	if err != nil {
		t.Logf("Failed to get existing Istio custom resource: %v", err)
		return err
	}

	// Copy the desired spec to the existing CR (preserving resourceVersion and other metadata)
	existingIcr.Spec = desiredIcr.Spec

	// Copy any annotations or labels that were set
	if desiredIcr.Annotations != nil {
		if existingIcr.Annotations == nil {
			existingIcr.Annotations = make(map[string]string)
		}
		for k, v := range desiredIcr.Annotations {
			existingIcr.Annotations[k] = v
		}
	}
	if desiredIcr.Labels != nil {
		if existingIcr.Labels == nil {
			existingIcr.Labels = make(map[string]string)
		}
		for k, v := range desiredIcr.Labels {
			existingIcr.Labels[k] = v
		}
	}

	// Log the Istio CR before updating
	logIstioCR(t, existingIcr)

	err = r.Update(t.Context(), existingIcr)
	if err != nil {
		t.Logf("Failed to update Istio custom resource: %v", err)
		return err
	}

	err = waitForIstioCRReadiness(t, r, existingIcr)
	if err != nil {
		t.Logf("Istio custom resource is not ready after update: %v", err)
		return err
	}

	t.Log("Istio custom resource updated successfully")
	return nil
}

// Delete deletes the Istio CR from the cluster
func (b *IstioCRBuilder) Delete(t *testing.T, ctx context.Context) error {
	t.Helper()
	t.Log("Deleting Istio custom resource")

	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return err
	}

	icr := b.Build()

	err = r.Delete(ctx, icr)
	if err != nil {
		t.Logf("Failed to delete Istio custom resource: %v", err)
		return err
	}

	err = waitForIstioCRDeletion(t, r, icr)
	if err != nil {
		t.Logf("Failed to wait for Istio custom resource deletion: %v", err)
		return err
	}

	t.Log("Istio custom resource deleted successfully")
	return nil
}

// Helper builder functions for common component configurations

// NewPilotComponent creates a basic Pilot/Istiod component configuration
func NewPilotComponent() *v1alpha2.IstioComponent {
	return &v1alpha2.IstioComponent{
		K8s: &v1alpha2.KubernetesResourcesConfig{},
	}
}

// NewIngressGatewayComponent creates a basic Ingress Gateway component configuration
func NewIngressGatewayComponent() *v1alpha2.IstioComponent {
	return &v1alpha2.IstioComponent{
		K8s: &v1alpha2.KubernetesResourcesConfig{},
	}
}

// NewEgressGatewayComponent creates a basic Egress Gateway component configuration
func NewEgressGatewayComponent(enabled bool) *v1alpha2.EgressGateway {
	return &v1alpha2.EgressGateway{
		Enabled: &enabled,
		K8s:     &v1alpha2.KubernetesResourcesConfig{},
	}
}

// NewCNIComponent creates a basic CNI component configuration
func NewCNIComponent() *v1alpha2.CniComponent {
	return &v1alpha2.CniComponent{
		K8S: &v1alpha2.CniK8sConfig{},
	}
}

// NewProxyComponent creates a basic Proxy component configuration
func NewProxyComponent() *v1alpha2.ProxyComponent {
	return &v1alpha2.ProxyComponent{
		K8S: &v1alpha2.ProxyK8sConfig{},
	}
}

// Helper functions for resource configurations

// NewResources creates a Resources configuration with the specified CPU and memory
func NewResources(cpuRequests, memoryRequests, cpuLimits, memoryLimits string) *v1alpha2.Resources {
	resources := &v1alpha2.Resources{}

	if cpuRequests != "" || memoryRequests != "" {
		resources.Requests = &v1alpha2.ResourceClaims{}
		if cpuRequests != "" {
			resources.Requests.CPU = &cpuRequests
		}
		if memoryRequests != "" {
			resources.Requests.Memory = &memoryRequests
		}
	}

	if cpuLimits != "" || memoryLimits != "" {
		resources.Limits = &v1alpha2.ResourceClaims{}
		if cpuLimits != "" {
			resources.Limits.CPU = &cpuLimits
		}
		if memoryLimits != "" {
			resources.Limits.Memory = &memoryLimits
		}
	}

	return resources
}

// NewHPASpec creates an HPA configuration
func NewHPASpec(minReplicas, maxReplicas int32) *v1alpha2.HPASpec {
	return &v1alpha2.HPASpec{
		MinReplicas: &minReplicas,
		MaxReplicas: &maxReplicas,
	}
}

// NewStrategy creates a rolling update strategy
func NewStrategy(maxSurge, maxUnavailable string) *v1alpha2.Strategy {
	maxSurgeIntOrString := intstr.FromString(maxSurge)
	maxUnavailableIntOrString := intstr.FromString(maxUnavailable)

	return &v1alpha2.Strategy{
		RollingUpdate: &v1alpha2.RollingUpdate{
			MaxSurge:       &maxSurgeIntOrString,
			MaxUnavailable: &maxUnavailableIntOrString,
		},
	}
}

// NewCNIAffinity creates an affinity configuration for CNI
func NewCNIAffinity() *corev1.Affinity {
	return &corev1.Affinity{
		// Add default affinity configuration as needed
	}
}

// Component builder methods to add to the fluent API

// WithPilotResources is a convenience method to set pilot resources
func (b *IstioCRBuilder) WithPilotResources(cpuRequests, memoryRequests, cpuLimits, memoryLimits string) *IstioCRBuilder {
	pilot := NewPilotComponent()
	pilot.K8s.Resources = NewResources(cpuRequests, memoryRequests, cpuLimits, memoryLimits)
	return b.WithPilot(pilot)
}

// WithIngressGatewayResources is a convenience method to set ingress gateway resources
func (b *IstioCRBuilder) WithIngressGatewayResources(cpuRequests, memoryRequests, cpuLimits, memoryLimits string) *IstioCRBuilder {
	ingressGateway := NewIngressGatewayComponent()
	ingressGateway.K8s.Resources = NewResources(cpuRequests, memoryRequests, cpuLimits, memoryLimits)
	return b.WithIngressGateway(ingressGateway)
}

// WithEgressGatewayResources is a convenience method to enable egress gateway and set resources
func (b *IstioCRBuilder) WithEgressGatewayResources(cpuRequests, memoryRequests, cpuLimits, memoryLimits string) *IstioCRBuilder {
	enabled := true
	egressGateway := &v1alpha2.EgressGateway{
		Enabled: &enabled,
		K8s: &v1alpha2.KubernetesResourcesConfig{
			Resources: NewResources(cpuRequests, memoryRequests, cpuLimits, memoryLimits),
		},
	}
	return b.WithEgressGateway(egressGateway)
}

// WithPilotHPA is a convenience method to set pilot HPA
func (b *IstioCRBuilder) WithPilotHPA(minReplicas, maxReplicas int32) *IstioCRBuilder {
	if b.istio.Spec.Components == nil {
		b.istio.Spec.Components = &v1alpha2.Components{}
	}
	if b.istio.Spec.Components.Pilot == nil {
		b.istio.Spec.Components.Pilot = NewPilotComponent()
	}
	b.istio.Spec.Components.Pilot.K8s.HPASpec = NewHPASpec(minReplicas, maxReplicas)
	return b
}

// WithIngressGatewayHPA is a convenience method to set ingress gateway HPA
func (b *IstioCRBuilder) WithIngressGatewayHPA(minReplicas, maxReplicas int32) *IstioCRBuilder {
	if b.istio.Spec.Components == nil {
		b.istio.Spec.Components = &v1alpha2.Components{}
	}
	if b.istio.Spec.Components.IngressGateway == nil {
		b.istio.Spec.Components.IngressGateway = NewIngressGatewayComponent()
	}
	b.istio.Spec.Components.IngressGateway.K8s.HPASpec = NewHPASpec(minReplicas, maxReplicas)
	return b
}

// WithProxyResources is a convenience method to set proxy resources
func (b *IstioCRBuilder) WithProxyResources(cpuRequests, memoryRequests, cpuLimits, memoryLimits string) *IstioCRBuilder {
	proxy := NewProxyComponent()
	proxy.K8S.Resources = NewResources(cpuRequests, memoryRequests, cpuLimits, memoryLimits)
	return b.WithProxy(proxy)
}

// WithCNIResources is a convenience method to set CNI resources
func (b *IstioCRBuilder) WithCNIResources(cpuRequests, memoryRequests, cpuLimits, memoryLimits string) *IstioCRBuilder {
	cni := NewCNIComponent()
	cni.K8S.Resources = NewResources(cpuRequests, memoryRequests, cpuLimits, memoryLimits)
	return b.WithCNI(cni)
}

// WithEnableAmbient is a convenience method to enable or disable ambient mode
func (b *IstioCRBuilder) WithEnableAmbient(enabled bool) *IstioCRBuilder {
	if b.istio.Spec.Experimental == nil {
		b.istio.Spec.Experimental = &v1alpha2.Experimental{}
	}
	b.istio.Spec.Experimental.EnableAmbient = &enabled
	return b
}

// logIstioCR logs the Istio CR as JSON for debugging purposes
func logIstioCR(t *testing.T, icr *v1alpha2.Istio) {
	t.Helper()
	jsonData, err := json.MarshalIndent(icr, "", "  ")
	if err != nil {
		t.Logf("Failed to marshal Istio CR to JSON: %v", err)
		return
	}
	t.Logf("Istio CR to be applied:\n%s", string(jsonData))
}

// registerIstioCRCleanup registers a cleanup function for the Istio CR
func registerIstioCRCleanup(t *testing.T, istioCR *v1alpha2.Istio) {
	t.Helper()
	setup.DeclareCleanup(t, func() {
		t.Log("Cleaning up Istio after the tests")
		err := teardownIstioCR(t, istioCR)
		if err != nil {
			t.Logf("Failed to clean up Istio custom resource: %v", err)
		} else {
			t.Log("Istio custom resource cleaned up successfully")
		}
	})
}

// NewAuthorizerFromExtAuth creates an Authorizer configuration from an external auth deployment.
func NewAuthorizerFromExtAuth(extAuth *extauth.DeploymentInfo) *v1alpha2.Authorizer {
	return &v1alpha2.Authorizer{
		Name:    extAuth.Name,
		Port:    uint32(extAuth.HttpPort),
		Service: extAuth.Host,
		Headers: &v1alpha2.Headers{
			InCheck: &v1alpha2.InCheck{
				Include: []string{"X-Ext-Authz"},
				Add: map[string]string{
					"X-Add-In-Check": "value",
				},
			},
		},
	}
}
