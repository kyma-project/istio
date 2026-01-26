package installation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	httpbinassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/httpbin"
	istioassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/istio"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/crds"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/destination_rule"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/namespace"
)

func TestUninstall(t *testing.T) {
	t.Run("Uninstallation of Istio module", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrastructure.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		istioCR, err := modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		err = namespace.LabelNamespaceWithIstioInjection(t, "default")
		require.NoError(t, err)

		_, err = httpbin.NewBuilder().WithNamespace("default").DeployWithCleanup(t)
		require.NoError(t, err)

		_, err = httpbin.NewBuilder().WithName("httpbin-regular-sidecar").WithNamespace("default").WithRegularSidecar().DeployWithCleanup(t)
		require.NoError(t, err)

		httpbinassert.AssertIstioProxyPresent(t, c, "app=httpbin")
		httpbinassert.AssertIstioProxyPresent(t, c, "app=httpbin-regular-sidecar")

		err = istioassert.AssertIstioNamespaceExists(t, c)
		require.NoError(t, err)

		err = c.Delete(t.Context(), istioCR)
		require.NoError(t, err)

		err = wait.For(conditions.New(c).ResourceDeleted(istioCR), wait.WithTimeout(1 * time.Minute))
		require.NoError(t, err)

		err = crds.AssertIstioCRDsNotPresent(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		err = istioassert.AssertIstioNamespaceDeleted(t, c, 2*time.Minute)
		require.NoError(t, err)

		httpbinassert.AssertIstioProxyAbsent(t, c, "app=httpbin")
		httpbinassert.AssertIstioProxyAbsent(t, c, "app=httpbin-regular-sidecar")
	})

	t.Run("Uninstallation respects the Istio resources created by the user", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrastructure.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		istioCR, err := modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		destinationRule, err := destination_rule.CreateDestinationRule(t, "customer-destination-rule", "default", "testing-svc.default.svc.cluster.local")
		require.NoError(t, err)

		err = c.Delete(t.Context(), istioCR)
		require.NoError(t, err)

		istioassert.AssertWarningStatus(t, c, istioCR,
			istioassert.WithExpectedCondition(
				v1alpha2.ConditionTypeReady,
				metav1.ConditionFalse,
				v1alpha2.ConditionReasonIstioCRsDangling,
			),
			istioassert.WithExpectedDescriptionContaining(
				"There are Istio resources that block deletion",
			),
			istioassert.WithTimeout(2*time.Minute),
		)

		err = crds.AssertIstioCRDsPresent(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		err = istioassert.AssertIstioNamespaceExists(t, c)
		require.NoError(t, err)

		err = c.Delete(t.Context(), destinationRule)
		require.NoError(t, err)

		err = wait.For(conditions.New(c).ResourceDeleted(istioCR), wait.WithTimeout(2*time.Minute))
		require.NoError(t, err)

		err = crds.AssertIstioCRDsNotPresent(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		err = istioassert.AssertIstioNamespaceDeleted(t, c, 2*time.Minute)
		require.NoError(t, err)
	})

	t.Run("Uninstallation of Istio module if Istio was manually deleted", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrastructure.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		istioCR, err := modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		err = istioassert.AssertIstioNamespaceExists(t, c)
		require.NoError(t, err)

		err = crds.AssertIstioCRDsPresent(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		istioClient := istio.NewIstioClient()
		err = istioClient.Uninstall(t.Context())
		require.NoError(t, err)

		err = istioassert.AssertIstioNamespaceDeleted(t, c, 1 * time.Minute)
		require.NoError(t, err)

		err = crds.AssertIstioCRDsNotPresent(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		err = c.Delete(t.Context(), istioCR)
		require.NoError(t, err)

		err = wait.For(conditions.New(c).ResourceDeleted(istioCR), wait.WithTimeout(1 * time.Minute))
		require.NoError(t, err)

		err = istioassert.AssertIstioNamespaceDeleted(t, c, 2*time.Minute)
		require.NoError(t, err)

		err = crds.AssertIstioCRDsNotPresent(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)
	})

}
