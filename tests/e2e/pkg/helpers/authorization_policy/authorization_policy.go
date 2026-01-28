package authorization_policy

import (
	"strings"
	"testing"

	apisecurityv1 "istio.io/api/security/v1"
	apiv1beta1 "istio.io/api/type/v1beta1"
	securityv1 "istio.io/client-go/pkg/apis/security/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
)

// CreateExtAuthzPolicy creates an AuthorizationPolicy with CUSTOM action for external authorization.
// The policy will match workloads with the specified labelSelector (e.g., "app=httpbin") and apply to the specified operation path.
func CreateExtAuthzPolicy(t *testing.T, name, namespace, labelSelector, provider, operation string) error {
	t.Helper()
	t.Logf("creating authorization policy %s/%s", namespace, name)

	c, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return err
	}

	// Parse label selector from "key=value" format
	matchLabels := make(map[string]string)
	if labelSelector != "" {
		parts := strings.SplitN(labelSelector, "=", 2)
		if len(parts) == 2 {
			matchLabels[parts[0]] = parts[1]
		}
	}

	ap := &securityv1.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: apisecurityv1.AuthorizationPolicy{
			Selector: &apiv1beta1.WorkloadSelector{
				MatchLabels: matchLabels,
			},
			Action: apisecurityv1.AuthorizationPolicy_CUSTOM,
			ActionDetail: &apisecurityv1.AuthorizationPolicy_Provider{
				Provider: &apisecurityv1.AuthorizationPolicy_ExtensionProvider{
					Name: provider,
				},
			},
			Rules: []*apisecurityv1.Rule{
				{
					To: []*apisecurityv1.Rule_To{
						{
							Operation: &apisecurityv1.Operation{
								Paths: []string{operation},
							},
						},
					},
				},
			},
		},
	}

	err = c.Create(t.Context(), ap)
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			t.Logf("Failed to create authorization policy: %v", err)
			return err
		}
		t.Logf("authorization policy %s/%s already exists", namespace, name)
	} else {
		t.Logf("authorization policy %s/%s created", namespace, name)
	}

	setup.DeclareCleanup(t, func() {
		t.Logf("Cleaning up authorization policy %s in namespace %s", ap.GetName(), ap.GetNamespace())
		err := c.Delete(setup.GetCleanupContext(), ap)
		if err != nil {
			t.Logf("Failed to delete resource %s: %v", ap.GetName(), err)
			return
		}
		t.Logf("authorization policy %s/%s deleted", namespace, name)
	})

	return nil
}
