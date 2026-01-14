package installation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apinetworkingv1 "istio.io/api/networking/v1"
	networkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/crds"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/namespace"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
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

		httpbinPodList := &v1.PodList{}
		err = c.List(t.Context(), httpbinPodList, resources.WithLabelSelector("app=httpbin"))
		require.NoError(t, err)
		require.NotEmpty(t, httpbinPodList.Items, "httpbin pod not found")
		require.True(t, hasIstioProxy(httpbinPodList.Items[0]), "httpbin pod should have istio-proxy")

		httpbinRegularPodList := &v1.PodList{}
		err = c.List(t.Context(), httpbinRegularPodList, resources.WithLabelSelector("app=httpbin-regular-sidecar"))
		require.NoError(t, err)
		require.NotEmpty(t, httpbinRegularPodList.Items, "httpbin-regular-sidecar pod not found")
		require.True(t, hasIstioProxy(httpbinRegularPodList.Items[0]), "httpbin-regular-sidecar pod should have istio-proxy")

		istioNs := &v1.Namespace{}
		err = c.Get(t.Context(), "istio-system", "", istioNs)
		require.NoError(t, err)

		err = c.Delete(t.Context(), istioCR)
		require.NoError(t, err)

		err = wait.For(conditions.New(c).ResourceDeleted(istioCR), wait.WithTimeout(2*time.Minute))
		require.NoError(t, err)

		err = crds.AssertIstioCRDsNotPresent(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		err = wait.For(conditions.New(c).ResourceDeleted(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "istio-system",
			},
		}), wait.WithTimeout(2*time.Minute))
		require.NoError(t, err)

		err = wait.For(func(ctx context.Context) (bool, error) {
			httpbinPodList := &v1.PodList{}
			err := c.List(ctx, httpbinPodList, resources.WithLabelSelector("app=httpbin"))
			if err != nil {
				return false, err
			}
			if len(httpbinPodList.Items) == 0 {
				return false, nil
			}
			return !hasIstioProxy(httpbinPodList.Items[0]), nil
		}, wait.WithTimeout(2*time.Minute), wait.WithContext(t.Context()))
		require.NoError(t, err)

		err = wait.For(func(ctx context.Context) (bool, error) {
			httpbinRegularPodList := &v1.PodList{}
			err := c.List(ctx, httpbinRegularPodList, resources.WithLabelSelector("app=httpbin-regular-sidecar"))
			if err != nil {
				return false, err
			}
			if len(httpbinRegularPodList.Items) == 0 {
				return false, nil
			}
			return !hasIstioProxy(httpbinRegularPodList.Items[0]), nil
		}, wait.WithTimeout(2*time.Minute), wait.WithContext(t.Context()))
		require.NoError(t, err)
	})

	t.Run("Uninstallation respects the Istio resources created by the user", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrastructure.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		istioCR, err := modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		destinationRule := &networkingv1.DestinationRule{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "networking.istio.io/v1",
				Kind:       "DestinationRule",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "customer-destination-rule",
				Namespace: "default",
			},
			Spec: apinetworkingv1.DestinationRule{
				Host: "testing-svc.default.svc.cluster.local",
			},
		}
		err = c.Create(t.Context(), destinationRule)
		require.NoError(t, err)

		setup.DeclareCleanup(t, func() {
			err := c.Delete(setup.GetCleanupContext(), destinationRule)
			if err != nil && !k8serrors.IsNotFound(err) {
				t.Logf("Failed to delete destination rule: %v", err)
			}
		})

		err = c.Delete(t.Context(), istioCR)
		require.NoError(t, err)

		err = wait.For(conditions.New(c).ResourceMatch(istioCR, func(object k8s.Object) bool {
			istioCRObj := object.(*v1alpha2.Istio)
			if istioCRObj.Status.State != v1alpha2.Warning {
				return false
			}

			// Check for the correct condition
			hasCorrectCondition := false
			if istioCRObj.Status.Conditions != nil {
				for _, condition := range *istioCRObj.Status.Conditions {
					if condition.Type == string(v1alpha2.ConditionTypeReady) &&
						condition.Reason == string(v1alpha2.ConditionReasonIstioCRsDangling) &&
						condition.Status == "False" {
						hasCorrectCondition = true
						break
					}
				}
			}

			hasCorrectDescription := strings.Contains(istioCRObj.Status.Description, "There are Istio resources that block deletion")

			return hasCorrectCondition && hasCorrectDescription
		}), wait.WithTimeout(2*time.Minute))
		require.NoError(t, err)

		err = crds.AssertIstioCRDsPresent(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		istioNs := &v1.Namespace{}
		err = c.Get(t.Context(), "istio-system", "", istioNs)
		require.NoError(t, err)

		err = c.Delete(t.Context(), destinationRule)
		require.NoError(t, err)

		err = wait.For(conditions.New(c).ResourceDeleted(istioCR), wait.WithTimeout(2*time.Minute))
		require.NoError(t, err)

		err = crds.AssertIstioCRDsNotPresent(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		err = wait.For(conditions.New(c).ResourceDeleted(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "istio-system",
			},
		}), wait.WithTimeout(2*time.Minute))
		require.NoError(t, err)
	})

	t.Run("Uninstallation of Istio module if Istio was manually deleted", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrastructure.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		istioCR, err := modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		istioNs := &v1.Namespace{}
		err = c.Get(t.Context(), "istio-system", "", istioNs)
		require.NoError(t, err)

		err = crds.AssertIstioCRDsPresent(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		istioClient := istio.NewIstioClient()
		err = istioClient.Uninstall(context.Background())
		require.NoError(t, err)

		err = wait.For(conditions.New(c).ResourceDeleted(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "istio-system",
			},
		}), wait.WithTimeout(2*time.Minute))
		require.NoError(t, err)

		err = crds.AssertIstioCRDsNotPresent(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		err = c.Delete(t.Context(), istioCR)
		require.NoError(t, err)

		err = wait.For(conditions.New(c).ResourceDeleted(istioCR), wait.WithTimeout(2*time.Minute))
		require.NoError(t, err)

		err = wait.For(conditions.New(c).ResourceDeleted(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "istio-system",
			},
		}), wait.WithTimeout(2*time.Minute))
		require.NoError(t, err)

		err = crds.AssertIstioCRDsNotPresent(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)
	})

}

func hasIstioProxy(pod v1.Pod) bool {
	for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
		if container.Name == "istio-proxy" {
			return true
		}
	}
	return false
}
