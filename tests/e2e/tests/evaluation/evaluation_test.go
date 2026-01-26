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

		err = namespace.LabelNamespaceWithIstioInjection(t, "default")
		require.NoError(t, err)

		_, err = httpbin.NewBuilder().WithNamespace("default").DeployWithCleanup(t)
		require.NoError(t, err)

		istiodPodList, err := infrastructure.GetIstiodPods(t)
		require.NoError(t, err)
		for _, pod := range istiodPodList.Items {
			resourceassert.AssertIstioProxyResourcesForPod(t, pod, "50m", "128Mi", "1000m", "1024Mi")
		}

		igPodList, err := infrastructure.GetIngressGatewayPods(t)
		require.NoError(t, err)
		for _, pod := range igPodList.Items {
			resourceassert.AssertIstioProxyResourcesForPod(t, pod, "10m", "32Mi", "1000m", "1024Mi")
		}

		httpbinPodList, err := httpbin.GetHttpbinPods(t, "app=httpbin")
		require.NoError(t, err)
		for _, pod := range httpbinPodList.Items {
			resourceassert.AssertIstioProxyResourcesForPod(t, pod, "10m", "32Mi", "250m", "254Mi")
		}

	})
}
