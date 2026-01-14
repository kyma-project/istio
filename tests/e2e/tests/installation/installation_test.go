package installation

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/inf.v0"
	v3 "istio.io/client-go/pkg/apis/security/v1"
	v2 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/config"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/crds"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
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
		httpbinPodList := &v1.PodList{}
		err = c.List(t.Context(), httpbinPodList, resources.WithLabelSelector("app=httpbin"))
		require.NoError(t, err)

		for _, pod := range httpbinPodList.Items {
			proxy := pod.Spec.InitContainers[1]
			require.Equal(t, "istio-proxy", proxy.Name)

			err = assertResources(resourceStruct{
				Cpu:    *proxy.Resources.Requests.Cpu(),
				Memory: *proxy.Resources.Requests.Memory(),
			}, "10m", "192Mi")
			require.NoError(t, err)

			err = assertResources(resourceStruct{
				Cpu:    *proxy.Resources.Limits.Cpu(),
				Memory: *proxy.Resources.Limits.Memory(),
			}, "1000m", "1024Mi")
			require.NoError(t, err)
		}

		// istio-ingressgateway
		ingressPodList := &v1.PodList{}
		err = c.List(t.Context(), ingressPodList, resources.WithLabelSelector("app=istio-ingressgateway"))
		require.NoError(t, err)

		for _, pod := range ingressPodList.Items {
			proxy := pod.Spec.Containers[0]
			require.Equal(t, "istio-proxy", proxy.Name)
			err = assertResources(resourceStruct{
				Cpu:    *proxy.Resources.Requests.Cpu(),
				Memory: *proxy.Resources.Requests.Memory(),
			}, "100m", "128Mi")
			require.NoError(t, err)

			err = assertResources(resourceStruct{
				Cpu:    *proxy.Resources.Limits.Cpu(),
				Memory: *proxy.Resources.Limits.Memory(),
			}, "2000m", "1024Mi")
			require.NoError(t, err)
		}

		// istiod
		istiodPodList := &v1.PodList{}
		err = c.List(t.Context(), istiodPodList, resources.WithLabelSelector("app=istiod"))
		require.NoError(t, err)
		for _, pod := range istiodPodList.Items {
			istiod := pod.Spec.Containers[0]
			require.Equal(t, "discovery", istiod.Name)
			err = assertResources(resourceStruct{
				Cpu:    *istiod.Resources.Requests.Cpu(),
				Memory: *istiod.Resources.Requests.Memory(),
			}, "100m", "512Mi")
			require.NoError(t, err)

			err = assertResources(resourceStruct{
				Cpu:    *istiod.Resources.Limits.Cpu(),
				Memory: *istiod.Resources.Limits.Memory(),
			}, "4000m", "2048Mi")
			require.NoError(t, err)
		}
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
		_, ok := istioNs.Annotations["istios.operator.kyma-project.io/managed-by-disclaimer"]
		require.True(t, ok, "istio-system namespace is not labeled with istios.operator.kyma-project.io/managed-by-disclaimer")
		_, ok = istioNs.Labels["namespaces.warden.kyma-project.io/validate"]
		require.True(t, ok, "istio-system namespace is not labeled with namespaces.warden.kyma-project.io/validate=true")

		// istiod is ready
		istiodDeployment := &v2.Deployment{}
		err = c.Get(t.Context(), "istiod", "istio-system", istiodDeployment)
		require.NoError(t, err)
		err = wait.For(conditions.New(c).DeploymentConditionMatch(istiodDeployment, v2.DeploymentAvailable, v1.ConditionTrue), wait.WithContext(t.Context()))

		// istio-ingressgateway is ready
		ingressDeployment := &v2.Deployment{}
		err = c.Get(t.Context(), "istio-ingressgateway", "istio-system", ingressDeployment)
		require.NoError(t, err)
		err = wait.For(conditions.New(c).DeploymentConditionMatch(ingressDeployment, v2.DeploymentAvailable, v1.ConditionTrue), wait.WithContext(t.Context()))
		require.NoError(t, err)

		// istio-egressgateway is ready
		egressDeployment := &v2.Deployment{}
		err = c.Get(t.Context(), "istio-egressgateway", "istio-system", egressDeployment)
		require.NoError(t, err)
		err = wait.For(conditions.New(c).DeploymentConditionMatch(egressDeployment, v2.DeploymentAvailable, v1.ConditionTrue), wait.WithContext(t.Context()))
		require.NoError(t, err)

		// istio-cni-node is ready
		cniDaemonSet := &v2.DaemonSet{}
		err = c.Get(t.Context(), "istio-cni-node", "istio-system", cniDaemonSet)
		require.NoError(t, err)
		err = wait.For(conditions.New(c).DaemonSetReady(cniDaemonSet), wait.WithContext(t.Context()))
		require.NoError(t, err)

		// ensure pilot limits and requests
		istiodPodList := &v1.PodList{}
		err = c.List(t.Context(), istiodPodList, resources.WithLabelSelector("app=istiod"))
		require.NoError(t, err)
		for _, pod := range istiodPodList.Items {
			istiod := pod.Spec.Containers[0]
			require.Equal(t, "discovery", istiod.Name)
			err = assertResources(resourceStruct{
				Cpu:    *istiod.Resources.Requests.Cpu(),
				Memory: *istiod.Resources.Requests.Memory(),
			}, "15m", "200Mi")
			require.NoError(t, err)

			err = assertResources(resourceStruct{
				Cpu:    *istiod.Resources.Limits.Cpu(),
				Memory: *istiod.Resources.Limits.Memory(),
			}, "1200m", "1200Mi")
			require.NoError(t, err)
		}

		// ensure ingressgateway limits and requests
		ingressPodList := &v1.PodList{}
		err = c.List(t.Context(), ingressPodList, resources.WithLabelSelector("app=istio-ingressgateway"))
		require.NoError(t, err)
		for _, pod := range ingressPodList.Items {
			ingress := pod.Spec.Containers[0]
			require.Equal(t, "istio-proxy", ingress.Name)
			err = assertResources(resourceStruct{
				Cpu:    *ingress.Resources.Requests.Cpu(),
				Memory: *ingress.Resources.Requests.Memory(),
			}, "80m", "200Mi")
			require.NoError(t, err)

			err = assertResources(resourceStruct{
				Cpu:    *ingress.Resources.Limits.Cpu(),
				Memory: *ingress.Resources.Limits.Memory(),
			}, "1500m", "1200Mi")
			require.NoError(t, err)
		}

		// ensure egressgateway limits and requests
		egressPodList := &v1.PodList{}
		err = c.List(t.Context(), egressPodList, resources.WithLabelSelector("app=istio-egressgateway"))
		require.NoError(t, err)
		for _, pod := range egressPodList.Items {
			egress := pod.Spec.Containers[0]
			require.Equal(t, "istio-proxy", egress.Name)
			err = assertResources(resourceStruct{
				Cpu:    *egress.Resources.Requests.Cpu(),
				Memory: *egress.Resources.Requests.Memory(),
			}, "70m", "190Mi")
			require.NoError(t, err)

			err = assertResources(resourceStruct{
				Cpu:    *egress.Resources.Limits.Cpu(),
				Memory: *egress.Resources.Limits.Memory(),
			}, "1400m", "1100Mi")
			require.NoError(t, err)
		}
	})

	t.Run("Managed Istio resources are present", func(t *testing.T) {
		cfg := config.Get()

		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		pa := v3.PeerAuthentication{}
		err = c.Get(t.Context(), "default", "istio-system", &pa)
		require.NoError(t, err)

		v, ok := pa.Labels["app.kubernetes.io/version"]
		require.True(t, ok, "Missing app.kubernetes.io/version label on PeerAuthentication")
		require.Equal(t, cfg.OperatorVersion, v)
	})

	t.Run("Installation of Istio module with Istio CR in different namespace", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		// Create Istio CR in 'default' namespace (not kyma-system)
		// Use ApplyAndCleanupWithoutReadinessCheck since we expect it to be in Error state
		istioCR, err := modulehelpers.NewIstioCRBuilder().
			WithNamespace("default").
			ApplyAndCleanupWithoutReadinessCheck(t)
		require.NoError(t, err)

		// Wait for the error state since it's in wrong namespace
		err = wait.For(conditions.New(c).ResourceMatch(istioCR, func(object k8s.Object) bool {
			istio := object.(*v1alpha2.Istio)

			ensureConditions := func() bool {
				for _, condition := range *istio.Status.Conditions {
					if condition.Type == string(v1alpha2.ConditionTypeReady) &&
						condition.Reason == string(v1alpha2.ConditionReasonReconcileFailed) &&
						condition.Status == "False" {
						return true
					}
				}
				return false
			}

			if istio.Status.State == v1alpha2.Error &&
				strings.Contains(istio.Status.Description, "Stopped Istio CR reconciliation: istio CR is not in kyma-system namespace") &&
				strings.Contains(istio.Status.Description, "Will not reconcile automatically") &&
				ensureConditions() {
				return true
			}

			return false
		}))
		require.NoError(t, err)
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

		// Wait for second CR to show warning
		err = wait.For(conditions.New(c).ResourceMatch(secondIstioCR, func(object k8s.Object) bool {
			istio := object.(*v1alpha2.Istio)
			ensureConditions := func() bool {
				for _, condition := range *istio.Status.Conditions {
					if condition.Type == string(v1alpha2.ConditionTypeReady) &&
						condition.Reason == string(v1alpha2.ConditionReasonOlderCRExists) &&
						condition.Status == "False" {
						return true
					}
				}
				return false
			}

			if istio.Status.State == v1alpha2.Warning &&
				strings.Contains(istio.Status.Description, "Stopped Istio CR reconciliation: only Istio CR default in kyma-system reconciles the module") &&
				strings.Contains(istio.Status.Description, "Will not reconcile automatically") &&
				ensureConditions() {
				return true
			}

			return false
		}))
		require.NoError(t, err)
	})

	t.Run("Istio module resources are reconciled, when they are deleted manually", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		// Create Istio CR
		ib := modulehelpers.NewIstioCRBuilder()
		_, err = ib.ApplyAndCleanup(t)
		require.NoError(t, err)

		err = c.Delete(t.Context(), &v2.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "istiod",
				Namespace: "istio-system",
			},
		})
		require.NoError(t, err)

		err = c.Delete(t.Context(), &v2.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "istio-ingressgateway",
				Namespace: "istio-system",
			},
		})
		require.NoError(t, err)

		err = c.Delete(t.Context(), &v2.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "istio-cni-node",
				Namespace: "istio-system",
			},
		})
		require.NoError(t, err)

		err = c.Delete(t.Context(), &v3.PeerAuthentication{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default",
				Namespace: "istio-system",
			},
		})
		require.NoError(t, err)

		err = ib.WithAnnotation("trigger-restart", "true").Update(t)
		require.NoError(t, err)

		// check if the resources are recreated by the operator
		istiodDeployment := &v2.Deployment{}
		err = c.Get(t.Context(), "istiod", "istio-system", istiodDeployment)
		require.NoError(t, err)

		ingressDeployment := &v2.Deployment{}
		err = c.Get(t.Context(), "istio-ingressgateway", "istio-system", ingressDeployment)
		require.NoError(t, err)

		cniDaemonSet := &v2.DaemonSet{}
		err = c.Get(t.Context(), "istio-cni-node", "istio-system", cniDaemonSet)
		require.NoError(t, err)

		pa := v3.PeerAuthentication{}
		err = c.Get(t.Context(), "default", "istio-system", &pa)
		require.NoError(t, err)

	})

}

type resourceStruct struct {
	Cpu    resource.Quantity
	Memory resource.Quantity
}

func assertResources(actualResources resourceStruct, expectedCpu, expectedMemory string) error {

	cpuMilli, err := strconv.Atoi(strings.TrimSuffix(expectedCpu, "m"))
	if err != nil {
		return err
	}

	memMilli, err := strconv.Atoi(strings.TrimSuffix(expectedMemory, "Mi"))
	if err != nil {
		return err
	}

	if resource.NewDecimalQuantity(*inf.NewDec(int64(cpuMilli), inf.Scale(resource.Milli)), resource.DecimalSI).Equal(actualResources.Cpu) {
		return fmt.Errorf("cpu wasn't expected; expected=%v got=%v", resource.NewScaledQuantity(int64(cpuMilli), resource.Milli), actualResources.Cpu)
	}

	if resource.NewDecimalQuantity(*inf.NewDec(int64(memMilli), inf.Scale(resource.Milli)), resource.DecimalSI).Equal(actualResources.Memory) {
		return fmt.Errorf("memory wasn't expected; expected=%v got=%v", resource.NewScaledQuantity(int64(memMilli), resource.Milli), actualResources.Memory)
	}

	return nil
}
