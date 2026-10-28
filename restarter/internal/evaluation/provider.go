package evaluation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	// Snapshot returns the current evaluators and the hash of the config
	// state they were derived from. The fingerprint is derived solely from the
	// Istio CR's lastAppliedConfiguration annotation.
	Snapshot(ctx context.Context) (evaluators []Evaluator, hash string, err error)
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

// Snapshot returns the current evaluator slice and a fingerprint derived
// solely from the Istio CR's lastAppliedConfiguration annotation, rebuilding
// the slice only when the underlying inputs have changed. A missing Istio CR
// yields (nil, "", nil). A missing or empty lastAppliedConfiguration
// annotation yields "" as the fingerprint.
func (p *Provider) Snapshot(ctx context.Context) ([]Evaluator, string, error) {
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

	sidecarResources := p.sidecarResources

	var featuresRV string
	var featuresCM corev1.ConfigMap
	if err := p.client.Get(ctx, types.NamespacedName{
		Name:      istiofeatures.ConfigMapName,
		Namespace: istiofeatures.ConfigMapNamespace,
	}, &featuresCM); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, "", fmt.Errorf("get istio-features configmap: %w", err)
		}
	} else {
		featuresRV = featuresCM.ResourceVersion
	}

	key, err := buildCacheKey(cr, cr.Annotations[labels.LastAppliedConfiguration], featuresRV, sidecarResources)
	if err != nil {
		return nil, "", fmt.Errorf("build cache key: %w", err)
	}

	fp := configFingerprint(cr.Annotations[labels.LastAppliedConfiguration])

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
		SidecarResources:  sidecarResources,
		Fingerprint:       fp,
	})
	if err != nil {
		return nil, "", fmt.Errorf("build evaluators: %w", err)
	}

	p.cacheKey = key
	p.cached = evaluators
	return evaluators, fp, nil
}

func buildCacheKey(cr operatorv1alpha2.Istio, lastAppliedAnnotation, featuresRV string, resources corev1.ResourceRequirements) (string, error) {
	specJSON, err := json.Marshal(cr.Spec)
	if err != nil {
		return "", fmt.Errorf("marshal istio spec: %w", err)
	}
	resourcesJSON, err := json.Marshal(resources)
	if err != nil {
		return "", fmt.Errorf("marshal sidecar resources: %w", err)
	}

	h := sha256.New()
	h.Write(specJSON)
	h.Write([]byte{0})
	h.Write([]byte(lastAppliedAnnotation))
	h.Write([]byte{0})
	h.Write([]byte(featuresRV))
	h.Write([]byte{0})
	h.Write(resourcesJSON)

	return hex.EncodeToString(h.Sum(nil)), nil
}

func configFingerprint(lastAppliedAnnotation string) string {
	if lastAppliedAnnotation == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(lastAppliedAnnotation))
	return hex.EncodeToString(sum[:])
}
