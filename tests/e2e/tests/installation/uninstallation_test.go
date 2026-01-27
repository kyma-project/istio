package installation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	istioassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/istio"
	resourceassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/resources"
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

		err = namespace.LabelNamespaceWithIstioInjection(t, defaultNamespace)
		require.NoError(t, err)

		httpbinDeployment, err := httpbin.NewBuilder().WithNamespace(defaultNamespace).DeployWithCleanup(t)
		require.NoError(t, err)

		httpbinRegularSidecarDeployment, err := httpbin.NewBuilder().WithName("httpbin-regular-sidecar").WithNamespace(defaultNamespace).WithRegularSidecar().DeployWithCleanup(t)
		require.NoError(t, err)

		istioassert.AssertIstioProxyPresent(t, c, httpbinDeployment.WorkloadSelector)
		istioassert.AssertIstioProxyPresent(t, c, httpbinRegularSidecarDeployment.WorkloadSelector)

		err = istioassert.AssertIstioNamespaceExists(t, c)
		require.NoError(t, err)

		err = c.Delete(t.Context(), istioCR)
		require.NoError(t, err)

		resourceassert.AssertResourceDeleted(t, c, istioCR, 1*time.Minute)

		err = crds.AssertIstioCRDsNotPresent(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		err = istioassert.AssertIstioNamespaceDeleted(t, c, 2*time.Minute)
		require.NoError(t, err)

		istioassert.AssertIstioProxyAbsent(t, c, httpbinDeployment.WorkloadSelector)
		istioassert.AssertIstioProxyAbsent(t, c, httpbinRegularSidecarDeployment.WorkloadSelector)
	})

	t.Run("Uninstallation respects the Istio resources created by the user", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrastructure.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		istioCR, err := modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		destinationRule, err := destination_rule.CreateDestinationRule(t, "customer-destination-rule", defaultNamespace, "testing-svc."+defaultNamespace+".svc.cluster.local")
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

		resourceassert.AssertResourceDeleted(t, c, istioCR, 2*time.Minute)

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

		err = istioassert.AssertIstioNamespaceDeleted(t, c, 1*time.Minute)
		require.NoError(t, err)

		err = crds.AssertIstioCRDsNotPresent(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		err = c.Delete(t.Context(), istioCR)
		require.NoError(t, err)

		resourceassert.AssertResourceDeleted(t, c, istioCR, 1*time.Minute)

		err = istioassert.AssertIstioNamespaceDeleted(t, c, 2*time.Minute)
		require.NoError(t, err)

		err = crds.AssertIstioCRDsNotPresent(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)
	})

}
