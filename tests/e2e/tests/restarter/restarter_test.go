package restarter

import (
	"context"
	"testing"
	"time"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
)

// testing ingressgateway restarts
func TestRestarter_IngressGateway(t *testing.T) {
	t.Run("NumTrustedProxies changed, gateway restarted", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().
			ApplyAndCleanup(t)
		require.NoError(t, err)
		dep := appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "istio-ingressgateway", Namespace: "istio-system"}}
		require.NoError(t, wait.For(conditions.New(c).DeploymentAvailable(dep.GetName(), dep.GetNamespace())))
		require.NoError(t, c.Get(t.Context(), dep.GetName(), dep.GetNamespace(), &dep))
		oldGeneration := dep.Status.ObservedGeneration
		oldRestartedAt := dep.Spec.Template.ObjectMeta.GetAnnotations()["istio-operator.kyma-project.io/restartedAt"]

		ingressGatewayPods := &corev1.PodList{}
		err = c.List(t.Context(), ingressGatewayPods,
			resources.WithLabelSelector("app=istio-ingressgateway"),
			resources.WithFieldSelector("metadata.namespace=istio-system"),
		)
		require.NoError(t, err)

		var podUIDs []types.UID
		for _, pod := range ingressGatewayPods.Items {
			podUIDs = append(podUIDs, pod.GetUID())
		}

		// Update IstioCR in kyma-system with numTrustedProxies=1
		err = modulehelpers.NewIstioCRBuilder().
			WithNumTrustedProxies(1).
			Update(t)
		require.NoError(t, err)

		assert.NoError(t, wait.For(conditions.New(c).DeploymentAvailable(dep.GetName(), dep.GetNamespace()), wait.WithTimeout(time.Minute*5)))
		assert.NoError(t, wait.For(conditions.New(c).ResourceMatch(&dep, func(obj k8s.Object) bool {
			observed := obj.(*appsv1.Deployment)
			if observed.Status.ObservedGeneration != oldGeneration &&
				observed.Spec.Template.GetAnnotations()["istio-operator.kyma-project.io/restartedAt"] != oldRestartedAt {
				return true
			}
			return false
		})))
		assert.NoError(t, wait.For(func(ctx context.Context) (bool, error) {
			newPods := &corev1.PodList{}
			err = c.List(ctx, newPods,
				resources.WithLabelSelector("app=istio-ingressgateway"),
				resources.WithFieldSelector("metadata.namespace=istio-system"))
			if err != nil {
				return false, err
			}
			for _, pod := range newPods.Items {
				for _, oldUID := range podUIDs {
					if pod.GetUID() == oldUID {
						return false, nil
					}
				}
			}
			return true, nil
		}))
	})
	t.Run("TrustDomain changed, gateway restarted", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().
			ApplyAndCleanup(t)
		require.NoError(t, err)
		dep := appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "istio-ingressgateway", Namespace: "istio-system"}}
		require.NoError(t, wait.For(conditions.New(c).DeploymentAvailable(dep.GetName(), dep.GetNamespace())))
		require.NoError(t, c.Get(t.Context(), dep.GetName(), dep.GetNamespace(), &dep))
		oldGeneration := dep.Status.ObservedGeneration
		oldRestartedAt := dep.Spec.Template.ObjectMeta.GetAnnotations()["istio-operator.kyma-project.io/restartedAt"]

		ingressGatewayPods := &corev1.PodList{}
		err = c.List(t.Context(), ingressGatewayPods,
			resources.WithLabelSelector("app=istio-ingressgateway"),
			resources.WithFieldSelector("metadata.namespace=istio-system"),
		)
		require.NoError(t, err)

		var podUIDs []types.UID
		for _, pod := range ingressGatewayPods.Items {
			podUIDs = append(podUIDs, pod.GetUID())
		}

		// Update IstioCR in kyma-system with numTrustedProxies=1
		err = modulehelpers.NewIstioCRBuilder().
			WithTrustDomain("trust.com").
			Update(t)
		require.NoError(t, err)

		assert.NoError(t, wait.For(conditions.New(c).DeploymentAvailable(dep.GetName(), dep.GetNamespace()), wait.WithTimeout(time.Minute*5)))
		assert.NoError(t, wait.For(conditions.New(c).ResourceMatch(&dep, func(obj k8s.Object) bool {
			observed := obj.(*appsv1.Deployment)
			if observed.Status.ObservedGeneration != oldGeneration &&
				observed.Spec.Template.GetAnnotations()["istio-operator.kyma-project.io/restartedAt"] != oldRestartedAt {
				return true
			}
			return false
		})))
		assert.NoError(t, wait.For(func(ctx context.Context) (bool, error) {
			newPods := &corev1.PodList{}
			err = c.List(ctx, newPods,
				resources.WithLabelSelector("app=istio-ingressgateway"),
				resources.WithFieldSelector("metadata.namespace=istio-system"))
			if err != nil {
				return false, err
			}
			for _, pod := range newPods.Items {
				for _, oldUID := range podUIDs {
					if pod.GetUID() == oldUID {
						return false, nil
					}
				}
			}
			return true, nil
		}))
	})
	t.Run("ForwardClientCertDetails changed, gateway restarted", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().
			ApplyAndCleanup(t)
		require.NoError(t, err)
		dep := appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "istio-ingressgateway", Namespace: "istio-system"}}
		require.NoError(t, wait.For(conditions.New(c).DeploymentAvailable(dep.GetName(), dep.GetNamespace())))
		require.NoError(t, c.Get(t.Context(), dep.GetName(), dep.GetNamespace(), &dep))
		oldGeneration := dep.Status.ObservedGeneration
		oldRestartedAt := dep.Spec.Template.ObjectMeta.GetAnnotations()["istio-operator.kyma-project.io/restartedAt"]

		ingressGatewayPods := &corev1.PodList{}
		err = c.List(t.Context(), ingressGatewayPods,
			resources.WithLabelSelector("app=istio-ingressgateway"),
			resources.WithFieldSelector("metadata.namespace=istio-system"),
		)
		require.NoError(t, err)

		var podUIDs []types.UID
		for _, pod := range ingressGatewayPods.Items {
			podUIDs = append(podUIDs, pod.GetUID())
		}

		// Update IstioCR in kyma-system with numTrustedProxies=1
		err = modulehelpers.NewIstioCRBuilder().
			WithForwardClientCertDetails(operatorv1alpha2.Sanitize).
			Update(t)
		require.NoError(t, err)

		assert.NoError(t, wait.For(conditions.New(c).DeploymentAvailable(dep.GetName(), dep.GetNamespace()), wait.WithTimeout(time.Minute*5)))
		assert.NoError(t, wait.For(conditions.New(c).ResourceMatch(&dep, func(obj k8s.Object) bool {
			observed := obj.(*appsv1.Deployment)
			if observed.Status.ObservedGeneration != oldGeneration &&
				observed.Spec.Template.GetAnnotations()["istio-operator.kyma-project.io/restartedAt"] != oldRestartedAt {
				return true
			}
			return false
		})))
		assert.NoError(t, wait.For(func(ctx context.Context) (bool, error) {
			newPods := &corev1.PodList{}
			err = c.List(ctx, newPods,
				resources.WithLabelSelector("app=istio-ingressgateway"),
				resources.WithFieldSelector("metadata.namespace=istio-system"))
			if err != nil {
				return false, err
			}
			for _, pod := range newPods.Items {
				for _, oldUID := range podUIDs {
					if pod.GetUID() == oldUID {
						return false, nil
					}
				}
			}
			return true, nil
		}))
	})
}
