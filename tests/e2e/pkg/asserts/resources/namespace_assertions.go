package resourceassert

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
)

// AssertNamespaceHasAnnotation asserts that a namespace has the expected annotation
func AssertNamespaceHasAnnotation(t *testing.T, c *resources.Resources, namespaceName, annotationKey, failureMessage string) {
	t.Helper()

	ns := v1.Namespace{}
	err := c.Get(t.Context(), namespaceName, "", &ns)
	require.NoError(t, err, "Failed to get namespace %s", namespaceName)

	_, ok := ns.Annotations[annotationKey]
	require.True(t, ok, failureMessage)
}

// AssertNamespaceHasLabel asserts that a namespace has the expected label
func AssertNamespaceHasLabel(t *testing.T, c *resources.Resources, namespaceName, labelKey, failureMessage string) {
	t.Helper()

	ns := v1.Namespace{}
	err := c.Get(t.Context(), namespaceName, "", &ns)
	require.NoError(t, err, "Failed to get namespace %s", namespaceName)

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

// AssertResourceDeleted waits for a resource to be deleted within the specified timeout
func AssertResourceDeleted(t *testing.T, c *resources.Resources, obj client.Object, timeout time.Duration) {
	t.Helper()
	err := wait.For(conditions.New(c).ResourceDeleted(obj), wait.WithTimeout(timeout), wait.WithContext(t.Context()))
	require.NoError(t, err, "Resource %s was not deleted within timeout", obj.GetName())
}
