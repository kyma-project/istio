package destination_rule

import (
	"testing"

	apinetworkingv1 "istio.io/api/networking/v1"
	networkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
)

// CreateDestinationRule creates a destination rule and registers cleanup
func CreateDestinationRule(t *testing.T, name, namespace, host string) (*networkingv1.DestinationRule, error) {
	t.Helper()
	t.Logf("creating destination rule %s/%s", namespace, name)

	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return nil, err
	}

	destinationRule := &networkingv1.DestinationRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.istio.io/v1",
			Kind:       "DestinationRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: apinetworkingv1.DestinationRule{
			Host: host,
		},
	}

	t.Logf("applying destination rule: %+v", destinationRule)

	err = r.Create(t.Context(), destinationRule)
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			t.Logf("Failed to create destination rule: %v", err)
			return nil, err
		}
		t.Logf("destination rule %s/%s already exists", namespace, name)
	} else {
		t.Logf("destination rule %s/%s created", namespace, name)
	}

	setup.DeclareCleanup(t, func() {
		err := r.Delete(setup.GetCleanupContext(), destinationRule)
		if err != nil && !k8serrors.IsNotFound(err) {
			t.Logf("Failed to delete destination rule: %v", err)
		} else {
			t.Logf("destination rule %s/%s deleted", namespace, name)
		}
	})

	return destinationRule, nil
}
