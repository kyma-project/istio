package resourceassert

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AssertNamespaceHasAnnotation asserts that a namespace has the expected annotation
func AssertNamespaceHasAnnotation(t *testing.T, ns v1.Namespace, annotationKey, failureMessage string) {
	t.Helper()
	_, ok := ns.Annotations[annotationKey]
	require.True(t, ok, failureMessage)
}

// AssertNamespaceHasLabel asserts that a namespace has the expected label
func AssertNamespaceHasLabel(t *testing.T, ns v1.Namespace, labelKey, failureMessage string) {
	t.Helper()
	_, ok := ns.Labels[labelKey]
	require.True(t, ok, failureMessage)
}

// AssertObjectHasLabelWithValue asserts that a Kubernetes object has the expected label with the expected value
func AssertObjectHasLabelWithValue(t *testing.T, obj metav1.Object, labelKey, expectedValue string) {
	t.Helper()
	value, ok := obj.GetLabels()[labelKey]
	require.True(t, ok, "Missing label %s", labelKey)
	require.Equal(t, expectedValue, value, "Label %s has unexpected value", labelKey)
}

