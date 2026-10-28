package evaluation_test

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/istiofeatures"
	"github.com/kyma-project/istio/operator/pkg/labels"
	"github.com/kyma-project/istio/operator/restarter/internal/evaluation"
)

const (
	testIstioCRName      = "default"
	testIstioCRNamespace = "kyma-system"
)

// annotationHash returns the SHA-256 hex hash of s, or "" when s is empty.
// Mirrors the logic of configFingerprint in provider.go so tests can compute
// expected fingerprint values without importing the unexported helper.
func annotationHash(s string) string {
	if s == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := operatorv1alpha2.AddToScheme(scheme); err != nil {
		t.Fatalf("add operatorv1alpha2 to scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 to scheme: %v", err)
	}
	return scheme
}

func newIstioCR(annotations map[string]string) *operatorv1alpha2.Istio {
	return &operatorv1alpha2.Istio{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testIstioCRName,
			Namespace:   testIstioCRNamespace,
			Annotations: annotations,
		},
	}
}

func newFeaturesConfigMap(resourceVersion string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            istiofeatures.ConfigMapName,
			Namespace:       istiofeatures.ConfigMapNamespace,
			ResourceVersion: resourceVersion,
		},
		Data: map[string]string{
			"features": `{"disableCni": false}`,
		},
	}
}

func testSidecarResources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2000m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}
}

func newProvider(t *testing.T, objs ...client.Object) *evaluation.Provider {
	t.Helper()
	scheme := newTestScheme(t)
	builder := fake.NewClientBuilder().WithScheme(scheme)
	if len(objs) > 0 {
		builder = builder.WithObjects(objs...)
	}
	c := builder.Build()
	return evaluation.NewProvider(
		c,
		types.NamespacedName{Namespace: testIstioCRNamespace, Name: testIstioCRName},
		testSidecarResources(),
	)
}

func slicePtr(evs []evaluation.Evaluator) uintptr {
	return reflect.ValueOf(evs).Pointer()
}

func TestProvider_Snapshot_MissingCR(t *testing.T) {
	// Only the ConfigMap exists; no Istio CR.
	p := newProvider(t, newFeaturesConfigMap("1"))

	got, fingerprint, err := p.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("want nil slice when CR is missing, got %v", got)
	}
	if fingerprint != "" {
		t.Errorf("want empty fingerprint when CR is missing, got %q", fingerprint)
	}
}

func TestProvider_Snapshot_MissingFeaturesConfigMap(t *testing.T) {
	// CR exists, but the istio-features ConfigMap does not — treated as optional.
	p := newProvider(t, newIstioCR(nil))

	got, _, err := p.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("unexpected error when istio-features ConfigMap is missing: %v", err)
	}
	if len(got) == 0 {
		t.Errorf("expected non-empty evaluators even when istio-features ConfigMap is missing")
	}
}

func TestProvider_Snapshot_CacheHit(t *testing.T) {
	p := newProvider(t, newIstioCR(nil), newFeaturesConfigMap("1"))

	first, firstFP, err := p.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("first snapshot: %v", err)
	}
	if len(first) == 0 {
		t.Fatalf("expected non-empty evaluators on first call")
	}

	second, secondFP, err := p.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("second snapshot: %v", err)
	}
	// Cache hit: same slice header (pointer) means Provider did not rebuild.
	if slicePtr(first) != slicePtr(second) {
		t.Errorf("expected cached slice to be returned verbatim; got different slice pointers")
	}
	if len(first) != len(second) {
		t.Errorf("cache hit: length changed between calls (%d -> %d)", len(first), len(second))
	}
	// No lastAppliedConfiguration annotation => fingerprint is "" on both calls.
	if firstFP != "" {
		t.Errorf("expected empty fingerprint when annotation is absent, got %q", firstFP)
	}
	if firstFP != secondFP {
		t.Errorf("cache hit: fingerprint must be stable between calls, got %q -> %q", firstFP, secondFP)
	}
}

// TestProvider_Snapshot_FingerprintEqualsAnnotationHash verifies that the
// returned fingerprint is exactly the SHA-256 hash of the
// lastAppliedConfiguration annotation value.
func TestProvider_Snapshot_FingerprintEqualsAnnotationHash(t *testing.T) {
	const annotationValue = `{"config":{"numTrustedProxies":1}}`
	p := newProvider(t,
		newIstioCR(map[string]string{labels.LastAppliedConfiguration: annotationValue}),
		newFeaturesConfigMap("1"),
	)

	_, fp, err := p.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	want := annotationHash(annotationValue)
	if fp != want {
		t.Errorf("fingerprint = %q, want %q (sha256 of annotation)", fp, want)
	}
}

// TestProvider_Snapshot_EmptyAnnotationYieldsEmptyFingerprint verifies that a
// CR with an explicitly empty lastAppliedConfiguration annotation produces a
// "" fingerprint, not a hash of the empty string.
func TestProvider_Snapshot_EmptyAnnotationYieldsEmptyFingerprint(t *testing.T) {
	p := newProvider(t,
		newIstioCR(map[string]string{labels.LastAppliedConfiguration: ""}),
		newFeaturesConfigMap("1"),
	)

	_, fp, err := p.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if fp != "" {
		t.Errorf("expected empty fingerprint for empty annotation, got %q", fp)
	}
}

// TestProvider_Snapshot_FingerprintStableWhenOnlySpecChanges verifies that
// changing the Istio CR spec (without touching the annotation) does NOT change
// the returned fingerprint, because the fingerprint is derived solely from the
// lastAppliedConfiguration annotation.
func TestProvider_Snapshot_FingerprintStableWhenOnlySpecChanges(t *testing.T) {
	const annotationValue = `{"config":{"numTrustedProxies":1}}`
	scheme := newTestScheme(t)
	cr := newIstioCR(map[string]string{labels.LastAppliedConfiguration: annotationValue})
	cm := newFeaturesConfigMap("1")
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr, cm).Build()
	p := evaluation.NewProvider(
		c,
		types.NamespacedName{Namespace: testIstioCRNamespace, Name: testIstioCRName},
		testSidecarResources(),
	)

	_, firstFP, err := p.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("first snapshot: %v", err)
	}

	// Mutate the Istio CR spec on the server; leave the annotation unchanged.
	var live operatorv1alpha2.Istio
	if err := c.Get(t.Context(), types.NamespacedName{Namespace: testIstioCRNamespace, Name: testIstioCRName}, &live); err != nil {
		t.Fatalf("get live CR: %v", err)
	}
	live.Spec.CompatibilityMode = true
	if err := c.Update(t.Context(), &live); err != nil {
		t.Fatalf("update CR: %v", err)
	}

	_, secondFP, err := p.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("second snapshot: %v", err)
	}

	want := annotationHash(annotationValue)
	if firstFP != want {
		t.Errorf("first fingerprint = %q, want %q", firstFP, want)
	}
	if secondFP != want {
		t.Errorf("second fingerprint = %q, want %q (fingerprint must not change when only spec changes)", secondFP, want)
	}
}

// TestProvider_Snapshot_FingerprintStableWhenOnlyFeaturesConfigMapChanges verifies
// that updating the istio-features ConfigMap (without touching the annotation)
// does NOT change the returned fingerprint.
func TestProvider_Snapshot_FingerprintStableWhenOnlyFeaturesConfigMapChanges(t *testing.T) {
	const annotationValue = `{"config":{"numTrustedProxies":1}}`
	scheme := newTestScheme(t)
	cr := newIstioCR(map[string]string{labels.LastAppliedConfiguration: annotationValue})
	cm := newFeaturesConfigMap("1")
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr, cm).Build()
	p := evaluation.NewProvider(
		c,
		types.NamespacedName{Namespace: testIstioCRNamespace, Name: testIstioCRName},
		testSidecarResources(),
	)

	_, firstFP, err := p.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("first snapshot: %v", err)
	}

	// Update the ConfigMap to bump its ResourceVersion; annotation unchanged.
	var liveCM corev1.ConfigMap
	if err := c.Get(t.Context(), types.NamespacedName{Namespace: istiofeatures.ConfigMapNamespace, Name: istiofeatures.ConfigMapName}, &liveCM); err != nil {
		t.Fatalf("get live CM: %v", err)
	}
	if liveCM.Data == nil {
		liveCM.Data = map[string]string{}
	}
	liveCM.Data["features"] = `{"disableCni": true}`
	if err := c.Update(t.Context(), &liveCM); err != nil {
		t.Fatalf("update CM: %v", err)
	}

	_, secondFP, err := p.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("second snapshot: %v", err)
	}

	want := annotationHash(annotationValue)
	if firstFP != want {
		t.Errorf("first fingerprint = %q, want %q", firstFP, want)
	}
	if secondFP != want {
		t.Errorf("second fingerprint = %q, want %q (fingerprint must not change when only ConfigMap changes)", secondFP, want)
	}
}

func TestProvider_Snapshot_RebuildOnSpecChange(t *testing.T) {
	scheme := newTestScheme(t)
	cr := newIstioCR(nil)
	cm := newFeaturesConfigMap("1")
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr, cm).Build()
	p := evaluation.NewProvider(
		c,
		types.NamespacedName{Namespace: testIstioCRNamespace, Name: testIstioCRName},
		testSidecarResources(),
	)

	first, _, err := p.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("first snapshot: %v", err)
	}
	if len(first) == 0 {
		t.Fatalf("expected non-empty evaluators")
	}

	// Mutate the Istio CR spec on the server.
	var live operatorv1alpha2.Istio
	if err := c.Get(t.Context(), types.NamespacedName{Namespace: testIstioCRNamespace, Name: testIstioCRName}, &live); err != nil {
		t.Fatalf("get live CR: %v", err)
	}
	live.Spec.CompatibilityMode = true
	if err := c.Update(t.Context(), &live); err != nil {
		t.Fatalf("update CR: %v", err)
	}

	second, _, err := p.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("second snapshot: %v", err)
	}
	// Evaluator slice must be rebuilt after a spec change even though the
	// fingerprint (annotation-derived) stays the same.
	if slicePtr(first) == slicePtr(second) {
		t.Errorf("expected rebuild after spec change; got cached slice")
	}
	if len(second) == 0 {
		t.Errorf("expected non-empty evaluators after rebuild")
	}
}

func TestProvider_Snapshot_RebuildOnFeaturesConfigMapResourceVersionChange(t *testing.T) {
	scheme := newTestScheme(t)
	cr := newIstioCR(nil)
	cm := newFeaturesConfigMap("1")
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr, cm).Build()
	p := evaluation.NewProvider(
		c,
		types.NamespacedName{Namespace: testIstioCRNamespace, Name: testIstioCRName},
		testSidecarResources(),
	)

	first, _, err := p.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("first snapshot: %v", err)
	}
	if len(first) == 0 {
		t.Fatalf("expected non-empty evaluators")
	}

	// Update the ConfigMap to bump its ResourceVersion.
	var liveCM corev1.ConfigMap
	if err := c.Get(t.Context(), types.NamespacedName{Namespace: istiofeatures.ConfigMapNamespace, Name: istiofeatures.ConfigMapName}, &liveCM); err != nil {
		t.Fatalf("get live CM: %v", err)
	}
	if liveCM.Data == nil {
		liveCM.Data = map[string]string{}
	}
	liveCM.Data["features"] = `{"disableCni": true}`
	if err := c.Update(t.Context(), &liveCM); err != nil {
		t.Fatalf("update CM: %v", err)
	}

	second, _, err := p.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("second snapshot: %v", err)
	}
	// Evaluator slice must be rebuilt after a ConfigMap ResourceVersion change
	// even though the fingerprint (annotation-derived) stays the same.
	if slicePtr(first) == slicePtr(second) {
		t.Errorf("expected rebuild after ConfigMap resourceVersion change; got cached slice")
	}
	if len(second) == 0 {
		t.Errorf("expected non-empty evaluators after rebuild")
	}
}

func TestProvider_Snapshot_RebuildOnLastAppliedAnnotationChange(t *testing.T) {
	scheme := newTestScheme(t)
	cr := newIstioCR(map[string]string{
		labels.LastAppliedConfiguration: `{"config":{"numTrustedProxies":1}}`,
	})
	cm := newFeaturesConfigMap("1")
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr, cm).Build()
	p := evaluation.NewProvider(
		c,
		types.NamespacedName{Namespace: testIstioCRNamespace, Name: testIstioCRName},
		testSidecarResources(),
	)

	first, firstFP, err := p.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("first snapshot: %v", err)
	}
	if len(first) == 0 {
		t.Fatalf("expected non-empty evaluators")
	}

	// Change only the lastAppliedConfiguration annotation; spec unchanged.
	var live operatorv1alpha2.Istio
	if err := c.Get(t.Context(), types.NamespacedName{Namespace: testIstioCRNamespace, Name: testIstioCRName}, &live); err != nil {
		t.Fatalf("get live CR: %v", err)
	}
	if live.Annotations == nil {
		live.Annotations = map[string]string{}
	}
	const newAnnotation = `{"config":{"numTrustedProxies":2}}`
	live.Annotations[labels.LastAppliedConfiguration] = newAnnotation
	if err := c.Update(t.Context(), &live); err != nil {
		t.Fatalf("update CR annotations: %v", err)
	}

	second, secondFP, err := p.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("second snapshot: %v", err)
	}
	if slicePtr(first) == slicePtr(second) {
		t.Errorf("expected rebuild after lastAppliedConfiguration annotation change; got cached slice")
	}
	if len(second) == 0 {
		t.Errorf("expected non-empty evaluators after rebuild")
	}
	// Fingerprint must reflect the new annotation value.
	wantFirst := annotationHash(`{"config":{"numTrustedProxies":1}}`)
	wantSecond := annotationHash(newAnnotation)
	if firstFP != wantFirst {
		t.Errorf("first fingerprint = %q, want %q", firstFP, wantFirst)
	}
	if secondFP != wantSecond {
		t.Errorf("second fingerprint = %q, want %q", secondFP, wantSecond)
	}
	if firstFP == secondFP {
		t.Errorf("expected fingerprint to change after lastAppliedConfiguration change, stayed %q", firstFP)
	}
}

func TestProvider_Snapshot_SidecarResourcesUsed(t *testing.T) {
	res := testSidecarResources()
	const annotationValue = `{"config":{"numTrustedProxies":1}}`
	p := newProvider(t,
		newIstioCR(map[string]string{labels.LastAppliedConfiguration: annotationValue}),
		newFeaturesConfigMap("1"),
	)

	evals, fp, err := p.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(evals) == 0 {
		t.Fatalf("expected non-empty evaluators")
	}
	wantFP := annotationHash(annotationValue)
	if fp != wantFP {
		t.Errorf("fingerprint = %q, want %q", fp, wantFP)
	}

	// Second call must return the same fingerprint (stable).
	_, fp2, err := p.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("second snapshot: %v", err)
	}
	if fp != fp2 {
		t.Errorf("fingerprint not stable across two calls: %q -> %q", fp, fp2)
	}

	// A pod whose istio-proxy container resources exactly match the expected
	// values should produce Continue from the SidecarResourcesChanged evaluator.
	matchingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "matching", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:      "istio-proxy",
					Resources: res,
				},
			},
		},
	}

	var resourcesEval evaluation.Evaluator
	for _, e := range evals {
		if e.Name() == "SidecarResourcesChanged" {
			resourcesEval = e
			break
		}
	}
	if resourcesEval == nil {
		t.Fatal("SidecarResourcesChanged evaluator not present in snapshot")
	}

	if got := resourcesEval.Evaluate(matchingPod); got != evaluation.Continue {
		t.Errorf("SidecarResourcesChanged: expected Continue for matching pod, got %v", got)
	}

	// A pod whose istio-proxy container has different resources should produce Restart.
	differingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "differing", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "istio-proxy",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("50m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
					},
				},
			},
		},
	}

	if got := resourcesEval.Evaluate(differingPod); got != evaluation.Restart {
		t.Errorf("SidecarResourcesChanged: expected Restart for differing pod, got %v", got)
	}
}
