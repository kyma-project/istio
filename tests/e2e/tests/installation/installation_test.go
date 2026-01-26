package installation

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	resourceassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/resources"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/config"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	istioassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/istio"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/crds"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	istiohelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/istio"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/namespace"
)

func TestInstallation(t *testing.T) {
	t.Run("Installation of Istio module with default values", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		err = namespace.LabelNamespaceWithIstioInjection(t, "default")
		require.NoError(t, err)

		_, err = httpbin.NewBuilder().WithNamespace("default").DeployWithCleanup(t)
		require.NoError(t, err)

		// user workload
		httpbinPodList, err := httpbin.GetHttpbinPods(t, "app=httpbin")
		require.NoError(t, err)

		for _, pod := range httpbinPodList.Items {
			resourceassert.AssertIstioProxyResourcesForPod(t, pod, "10m", "192Mi", "1000m", "1024Mi")
		}

		// istio-ingressgateway
		ingressPodList, err := infrahelpers.GetIngressGatewayPods(t)
		require.NoError(t, err)

		for _, pod := range ingressPodList.Items {
			resourceassert.AssertIstioProxyResourcesForPod(t, pod, "100m", "128Mi", "2000m", "1024Mi")
		}

		istioassert.AssertIstiodPodResources(t, c, "100m", "512Mi", "4000m", "2048Mi")
	})

	t.Run("Installation of Istio module with custom values", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().
			WithPilotResources("15m", "200Mi", "1200m", "1200Mi").
			WithIngressGatewayResources("80m", "200Mi", "1500m", "1200Mi").
			WithEgressGatewayResources("70m", "190Mi", "1400m", "1100Mi").
			ApplyAndCleanup(t)
		require.NoError(t, err)

		err = crds.AssertIstioCRDsPresent(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		istioNs := v1.Namespace{}
		err = c.Get(t.Context(), "istio-system", "", &istioNs)
		require.NoError(t, err)

		resourceassert.AssertNamespaceHasAnnotation(
			t,
			istioNs,
			"istios.operator.kyma-project.io/managed-by-disclaimer",
			"istio-system namespace is not labeled with istios.operator.kyma-project.io/managed-by-disclaimer",
		)
		resourceassert.AssertNamespaceHasLabel(
			t,
			istioNs,
			"namespaces.warden.kyma-project.io/validate",
			"istio-system namespace is not labeled with namespaces.warden.kyma-project.io/validate=true",
		)

		istioassert.AssertIstiodReady(t, c)
		istioassert.AssertIngressGatewayReady(t, c)
		istioassert.AssertEgressGatewayReady(t, c)
		istioassert.AssertCNINodeReady(t, c)

		istioassert.AssertIstiodPodResources(t, c, "15m", "200Mi", "1200m", "1200Mi")
		istioassert.AssertIngressGatewayPodResources(t, c, "80m", "200Mi", "1500m", "1200Mi")
		istioassert.AssertEgressGatewayPodResources(t, c, "70m", "190Mi", "1400m", "1100Mi")
	})

	t.Run("Managed Istio resources are present", func(t *testing.T) {
		cfg := config.Get()

		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		pa := istioassert.AssertDefaultPeerAuthenticationExists(t, c)
		resourceassert.AssertObjectHasLabelWithValue(t, pa, "app.kubernetes.io/version", cfg.OperatorVersion)
	})

	t.Run("Installation of Istio module with Istio CR in different namespace", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		// Create Istio CR in default namespace (not kyma-system)
		// Use ApplyAndCleanupWithoutReadinessCheck since we expect it to be in Error state
		istioCR, err := modulehelpers.NewIstioCRBuilder().
			WithNamespace("default").
			ApplyAndCleanupWithoutReadinessCheck(t)
		require.NoError(t, err)

		// Wait for the error state since it's in wrong namespace
		istioassert.AssertErrorStatus(t, c, istioCR,
			istioassert.WithExpectedCondition(
				v1alpha2.ConditionTypeReady,
				metav1.ConditionFalse,
				v1alpha2.ConditionReasonReconcileFailed,
			),
			istioassert.WithExpectedDescriptionContaining(
				"Stopped Istio CR reconciliation: istio CR is not in kyma-system namespace",
				"Will not reconcile automatically",
			),
		)
	})

	t.Run("Installation of Istio module with a second Istio CR in kyma-system namespace", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		// Create first Istio CR with default name
		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		// Create second Istio CR in same namespace
		secondIstioCR, err := modulehelpers.NewIstioCRBuilder().
			WithName("second-istio-cr").
			ApplyAndCleanupWithoutReadinessCheck(t)
		require.NoError(t, err)

		// Wait for second CR to show warning
		istioassert.AssertWarningStatus(t, c, secondIstioCR,
			istioassert.WithExpectedCondition(
				v1alpha2.ConditionTypeReady,
				metav1.ConditionFalse,
				v1alpha2.ConditionReasonOlderCRExists,
			),
			istioassert.WithExpectedDescriptionContaining(
				"Stopped Istio CR reconciliation: only Istio CR default in kyma-system reconciles the module",
				"Will not reconcile automatically",
			),
		)
	})

	t.Run("Istio module resources are reconciled, when they are deleted manually", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		// Create Istio CR
		ib := modulehelpers.NewIstioCRBuilder()
		_, err = ib.ApplyAndCleanup(t)
		require.NoError(t, err)

		istiohelpers.DeleteIstiod(t, c)
		istiohelpers.DeleteIngressGateway(t, c)
		istiohelpers.DeleteCNINode(t, c)
		istiohelpers.DeleteDefaultPeerAuthentication(t, c)

		err = ib.TriggerReconciliation(t)
		require.NoError(t, err)

		// check if the resources are recreated by the operator
		istioassert.AssertIstiodReady(t, c)
		istioassert.AssertIngressGatewayReady(t, c)
		istioassert.AssertCNINodeReady(t, c)
		istioassert.AssertDefaultPeerAuthenticationExists(t, c)

	})

}
