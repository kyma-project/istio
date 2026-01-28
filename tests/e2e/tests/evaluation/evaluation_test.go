package evaluation

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	istioassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/istio"
	resourceassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/resources"
	resourceClient "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/namespace"
)

const defaultNamespace = "default"

func TestEvaluationProfile(t *testing.T) {
	t.Run("Installation of Istio Module with evaluation profile", func(t *testing.T) {
		err := infrastructure.EnsureEvaluationClusterProfile(t)
		require.NoError(t, err)

		istioCR, err := modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		c, err := resourceClient.ResourcesClient(t)
		require.NoError(t, err)

		istioassert.AssertIstioStatus(t, c, istioCR,
			istioassert.WithExpectedState(v1alpha2.Ready),
			istioassert.WithExpectedCondition(v1alpha2.ConditionTypeReady, "True", v1alpha2.ConditionReasonReconcileSucceeded),
		)

		err = namespace.LabelNamespaceWithIstioInjection(t, defaultNamespace)
		require.NoError(t, err)

		httpbinDeployment, err := httpbin.NewBuilder().WithNamespace(defaultNamespace).DeployWithCleanup(t)
		require.NoError(t, err)

		istioassert.AssertIstiodPodResources(t, c, "50m", "128Mi", "1000m", "1024Mi")
		istioassert.AssertIngressGatewayPodResources(t, c, "10m", "32Mi", "1000m", "1024Mi")

		httpbinPodList, err := httpbin.GetHttpbinPods(t, httpbinDeployment.WorkloadSelector)
		require.NoError(t, err)
		for _, pod := range httpbinPodList.Items {
			resourceassert.AssertIstioProxyResourcesForPod(t, pod, "10m", "32Mi", "250m", "254Mi")
		}

	})
}
