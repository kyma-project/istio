package restarter

import (
	"context"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/describederrors"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
	"github.com/kyma-project/istio/operator/internal/status"
	"github.com/kyma-project/istio/operator/pkg/lib/annotations"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/retry"
	appsv1 "k8s.io/api/apps/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NetworkPolicy struct {
	client        client.Client
	statusHandler status.Status
}

func NewForNetworkPolicy(client client.Client, statusHandler status.Status) *NetworkPolicy {
	return &NetworkPolicy{
		client:        client,
		statusHandler: statusHandler,
	}
}

func (np *NetworkPolicy) Restart(ctx context.Context, istioCR *v1alpha2.Istio) (describederrors.DescribedError, bool) {
	lastAppliedConfig, err := configuration.GetLastAppliedConfiguration(istioCR)
	if err != nil {
		return describederrors.NewDescribedError(err, "Could not get last applied configuration"), false
	}

	if lastAppliedConfig.EnableModuleNetworkPolicies != istioCR.Spec.EnableModuleNetworkPolicies {
		return restartControlPlaneComponents(ctx, np.client)
	}
	return nil, false
}

func restartControlPlaneComponents(ctx context.Context, client client.Client) (describederrors.DescribedError, bool) {
	err := restartIngressGateway(ctx, client)
	if err != nil {
		return describederrors.NewDescribedError(err, "Failed to restart Ingress Gateway"), false
	}
	err = restartIstiod(ctx, client)
	if err != nil {
		return describederrors.NewDescribedError(err, "Failed to restart Istiod"), false
	}
	err = restartCNI(ctx, client)
	if err != nil {
		return describederrors.NewDescribedError(err, "Failed to restart CNI"), false
	}
	return nil, true
}

const (
	cniNamespace     = "istio-system"
	cniDaemonSetName = "istio-cni-node"
)

func restartCNI(ctx context.Context, c client.Client) error {
	ctrl.Log.Info("Restarting CNI")
	var daemonSet appsv1.DaemonSet
	err := c.Get(context.Background(), types.NamespacedName{Namespace: cniNamespace, Name: cniDaemonSetName}, &daemonSet)

	patch := client.StrategicMergeFrom((&daemonSet).DeepCopy())
	daemonSet.Spec.Template.Annotations = annotations.AddRestartAnnotation(daemonSet.Spec.Template.Annotations)

	err = retry.OnError(retry.DefaultRetry, func() error {
		err = c.Patch(ctx, &daemonSet, patch)
		if err != nil {
			ctrl.Log.Info("Retrying CNI restart")
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	ctrl.Log.Info("CNI restart finished")

	return nil
}

const (
	istiodNamespace      = "istio-system"
	istiodDeploymentName = "istiod"
)

func restartIstiod(ctx context.Context, c client.Client) error {
	ctrl.Log.Info("Restarting istiod")

	deployment := appsv1.Deployment{}
	err := c.Get(ctx, types.NamespacedName{Namespace: istiodNamespace, Name: istiodDeploymentName}, &deployment)
	if err != nil {
		// If ingress gateway deployment is missing, we should not fail, as it may have not yet been created
		// In that case, the upcoming creation of the deployment will do the same thing as we would require from the restart
		if k8sErrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	patch := client.StrategicMergeFrom((&deployment).DeepCopy())
	deployment.Spec.Template.Annotations = annotations.AddRestartAnnotation(deployment.Spec.Template.Annotations)

	err = retry.OnError(retry.DefaultRetry, func() error {
		err = c.Patch(ctx, &deployment, patch)
		if err != nil {
			ctrl.Log.Info("Retrying istiod restart")
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	ctrl.Log.Info("istiod restart finished")

	return nil
}
