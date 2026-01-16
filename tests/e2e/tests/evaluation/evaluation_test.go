package evaluation

import (
	_ "embed"
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/namespace"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	resourceClient "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
)

func TestEvaluationProfile(t *testing.T) {
	t.Run("Installation of Istio Module with evaluation profile", func(t *testing.T) {
		err := infrastructure.EnsureEvaluationClusterProfile(t)
		require.NoError(t, err)

		istioCR, err := modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		c, err := resourceClient.ResourcesClient(t)
		require.NoError(t, err)

		err = c.Get(t.Context(), istioCR.Name, istioCR.Namespace, istioCR)
		require.NoError(t, err)

		conditions := *istioCR.Status.Conditions
		require.Equal(t, v1alpha2.Ready, istioCR.Status.State)
		require.Equal(t, string(v1alpha2.ConditionReasonReconcileSucceeded), conditions[0].Reason)
		require.Equal(t, string(v1alpha2.ConditionTypeReady), conditions[0].Type)
		require.Equal(t, metav1.ConditionTrue, conditions[0].Status)

		err = namespace.LabelNamespaceWithIstioInjection(t, "default")
		require.NoError(t, err)

		_, _, err = httpbin.DeployHttpbin(t, "default")
		require.NoError(t, err)

		// istiod
		istiodPodList, err := infrastructure.GetIstiodPods(t)
		require.NoError(t, err)

		for _, pod := range istiodPodList.Items {
			for _, container := range pod.Spec.InitContainers {
				require.Equal(t, "50m", container.Resources.Requests.Cpu().String())
				require.Equal(t, "128Mi", container.Resources.Requests.Memory().String())
				require.Equal(t, "1000m", container.Resources.Limits.Cpu().String())
				require.Equal(t, "1024Mi", container.Resources.Limits.Memory().String())
			}
		}

		// istio-ingressgateway
		igPodList, err := infrastructure.GetIngressGatewayPods(t)
		require.NoError(t, err)

		for _, pod := range igPodList.Items {
			for _, container := range pod.Spec.InitContainers {
				require.Equal(t, "10m", container.Resources.Requests.Cpu().String())
				require.Equal(t, "32Mi", container.Resources.Requests.Memory().String())
				require.Equal(t, "1000m", container.Resources.Limits.Cpu().String())
				require.Equal(t, "1024Mi", container.Resources.Limits.Memory().String())
			}
		}

		// workload
		httpbinPodList, err := infrastructure.GetHttpbinPods(t)
		require.NoError(t, err)

		containerFound := false
		for _, pod := range httpbinPodList.Items {
			for _, container := range pod.Spec.InitContainers {
				if container.Name == "istio-proxy" {
					containerFound = true
					require.Equal(t, "10m", container.Resources.Requests.Cpu().String())
					require.Equal(t, "32Mi", container.Resources.Requests.Memory().String())
					require.Equal(t, "250m", container.Resources.Limits.Cpu().String())
					require.Equal(t, "254Mi", container.Resources.Limits.Memory().String())
					break
				}
			}
		}
		require.True(t, containerFound)

	})
}
