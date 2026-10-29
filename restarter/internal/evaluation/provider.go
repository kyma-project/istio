package evaluation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"sync"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/istiofeatures"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
	"github.com/kyma-project/istio/operator/pkg/labels"
)

// Source is the interface controllers consume to obtain a current set of evaluators.
type Source interface {
	// ProxySnapshot returns evaluators relevant for istio-proxy injected
	// workloads.
	ProxySnapshot(ctx context.Context) (evaluators []Evaluator, hash string, err error)
	// GatewaySnapshot returns evaluators and a fingerprint for Istio gateway
	// Deployments (ingress and egress).
	GatewaySnapshot(ctx context.Context) (evaluators []Evaluator, gatewayFingerprint string, err error)
}

// Provider builds and caches the evaluator slice derived from cluster state
// (Istio CR, istio-features ConfigMap). Safe for concurrent use.
type Provider struct {
	client           client.Client
	istioCRKey       types.NamespacedName
	sidecarResources corev1.ResourceRequirements

	mu       sync.RWMutex
	cacheKey string
	cached   []Evaluator
}

// NewProvider returns a Provider that reads state via c and resolves the
// Istio CR at key. sidecarResources holds the operator-supplied expected proxy
// resource requirements.
func NewProvider(c client.Client, key types.NamespacedName, sidecarResources corev1.ResourceRequirements) *Provider {
	return &Provider{
		client:           c,
		istioCRKey:       key,
		sidecarResources: sidecarResources,
	}
}

func (p *Provider) ProxySnapshot(ctx context.Context) ([]Evaluator, string, error) {
	var cr operatorv1alpha2.Istio
	if err := p.client.Get(ctx, p.istioCRKey, &cr); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("get istio CR: %w", err)
	}

	features, err := istiofeatures.Get(ctx, p.client)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, "", fmt.Errorf("get istio features: %w", err)
	}

	// GetLastAppliedConfiguration errors on an empty annotation value (invalid
	// JSON). Treat an empty annotation as "no config" — return a zero-value
	// AppliedConfig and let the fingerprint logic below produce "".
	var lastApplied configuration.AppliedConfig
	if cr.Annotations[labels.LastAppliedConfiguration] != "" {
		lastApplied, err = configuration.GetLastAppliedConfiguration(&cr)
		if err != nil {
			return nil, "", fmt.Errorf("get last applied configuration: %w", err)
		}
	}

	fp, err := proxyConfigFingerprint(lastApplied)
	if err != nil {
		return nil, "", fmt.Errorf("compute proxy fingerprint: %w", err)
	}

	effectiveResources, err := effectiveSidecarResources(lastApplied.Components, p.sidecarResources)
	if err != nil {
		return nil, "", fmt.Errorf("compute effective sidecar resources: %w", err)
	}

	// The cache key gates rebuilding the evaluator slice, so it must cover every
	// input an evaluator captures at build time: the drift fingerprint plus the
	// inputs owned by dedicated evaluators (DisableCni, PrometheusMerge, IstioTag
	// and the expected sidecar resources). A change to any of them must rebuild
	// the slice so no evaluator keeps a stale value.
	resourcesJSON, err := json.Marshal(effectiveResources)
	if err != nil {
		return nil, "", fmt.Errorf("marshal sidecar resources: %w", err)
	}
	key := fmt.Sprintf("%s|%t|%t|%s|%s",
		fp,
		features.DisableCni,
		lastApplied.Config.Telemetry.Metrics.PrometheusMerge,
		lastApplied.IstioTag,
		resourcesJSON,
	)

	p.mu.RLock()
	if key == p.cacheKey && p.cached != nil {
		cached := p.cached
		p.mu.RUnlock()
		return cached, fp, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if key == p.cacheKey && p.cached != nil {
		return p.cached, fp, nil
	}

	evaluators, err := GetEvaluators(Config{
		IstioCR:           &cr,
		LastAppliedConfig: lastApplied,
		BetaFeatures:      features,
		SidecarResources:  p.sidecarResources,
		Fingerprint:       fp,
	})
	if err != nil {
		return nil, "", fmt.Errorf("build evaluators: %w", err)
	}

	p.cacheKey = key
	p.cached = evaluators
	return evaluators, fp, nil
}

// proxyConfigFingerprint is the drift signal stamped on workloads and compared
// by the FingerprintDriftChanged evaluator. It hashes only the proxy-restart
// config fields.
func proxyConfigFingerprint(cfg configuration.AppliedConfig) (string, error) {
	// Sort inclusion regexps to ensure consistent hashing regardless of order
	inclusionRegexps := []string{}
	if m := cfg.Config.ProxyStatsMatcher; m != nil && len(m.InclusionRegexps) > 0 {
		inclusionRegexps = make([]string, len(m.InclusionRegexps))
		copy(inclusionRegexps, m.InclusionRegexps)
		slices.Sort(inclusionRegexps)
	}

	// Return empty string when no proxy field carries a value
	if !cfg.CompatibilityMode && cfg.Config.EnableDNSProxying == nil && len(inclusionRegexps) == 0 {
		return "", nil
	}

	type proxyConfig struct {
		CompatibilityMode bool
		EnableDNSProxying *bool
		InclusionRegexps  []string
	}

	pc := proxyConfig{
		CompatibilityMode: cfg.CompatibilityMode,
		EnableDNSProxying: cfg.Config.EnableDNSProxying,
		InclusionRegexps:  inclusionRegexps,
	}

	jsonData, err := json.Marshal(pc)
	if err != nil {
		return "", fmt.Errorf("marshal proxy config: %w", err)
	}

	sum := sha256.Sum256(jsonData)
	return hex.EncodeToString(sum[:]), nil
}

// GatewaySnapshot returns evaluators relevant for gateway restart.
func (p *Provider) GatewaySnapshot(ctx context.Context) ([]Evaluator, string, error) {
	var cr operatorv1alpha2.Istio
	if err := p.client.Get(ctx, p.istioCRKey, &cr); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("get istio CR: %w", err)
	}

	var lastApplied configuration.AppliedConfig
	if cr.Annotations[labels.LastAppliedConfiguration] != "" {
		var err error
		lastApplied, err = configuration.GetLastAppliedConfiguration(&cr)
		if err != nil {
			return nil, "", fmt.Errorf("get last applied configuration: %w", err)
		}
	}

	fp, err := gatewayConfigFingerprint(lastApplied)
	if err != nil {
		return nil, "", fmt.Errorf("compute gateway fingerprint: %w", err)
	}

	return GetGatewayEvaluators(fp), fp, nil
}

func gatewayConfigFingerprint(cfg configuration.AppliedConfig) (string, error) {
	numTrustedProxies := cfg.Config.NumTrustedProxies
	forwardClientCertDetails := cfg.Config.ForwardClientCertDetails
	trustDomain := cfg.Config.TrustDomain
	proxyStatsMatcher := cfg.Config.ProxyStatsMatcher

	if numTrustedProxies == nil && forwardClientCertDetails == nil &&
		trustDomain == nil && proxyStatsMatcher == nil {
		return "", nil
	}

	type gatewayConfig struct {
		NumTrustedProxies        *int
		ForwardClientCertDetails *operatorv1alpha2.XFCCStrategy
		TrustDomain              *string
		InclusionRegexps         []string
	}

	// Sort inclusion regexps to ensure consistent hashing regardless of order
	inclusionRegexps := []string{}
	if proxyStatsMatcher != nil && len(proxyStatsMatcher.InclusionRegexps) > 0 {
		inclusionRegexps = make([]string, len(proxyStatsMatcher.InclusionRegexps))
		copy(inclusionRegexps, proxyStatsMatcher.InclusionRegexps)
		slices.Sort(inclusionRegexps)
	}

	gc := gatewayConfig{
		NumTrustedProxies:        numTrustedProxies,
		ForwardClientCertDetails: forwardClientCertDetails,
		TrustDomain:              trustDomain,
		InclusionRegexps:         inclusionRegexps,
	}

	// Marshal to JSON and hash
	jsonData, err := json.Marshal(gc)
	if err != nil {
		return "", fmt.Errorf("marshal gateway config: %w", err)
	}

	sum := sha256.Sum256(jsonData)
	return hex.EncodeToString(sum[:]), nil
}
