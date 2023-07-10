package ingressgateway

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/pkg/lib/common"
)

const (
	deploymentNS   string = "istio-system"
	deploymentName string = "istio-ingressgateway"
)

func RestartDeployment(ctx context.Context, k8sClient client.Client) error {
	deployment := appsv1.Deployment{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: deploymentNS, Name: deploymentName}, &deployment)
	if err != nil {
		return err
	}
	deployment.Spec.Template.Annotations = common.AddRestartAnnotation(deployment.Spec.Template.Annotations)
	return k8sClient.Update(ctx, &deployment)
}

func NeedsRestart(ctx context.Context, client client.Client, istioCR operatorv1alpha1.Istio) (bool, error) {
	numTrustedProxies, err := GetNumTrustedProxyFromIstioCM(ctx, client)
	if err != nil {
		return false, err
	}

	isNewNotNil := (istioCR.Spec.Config.NumTrustedProxies != nil)
	isOldNotNil := (numTrustedProxies != nil)
	if isNewNotNil && isOldNotNil && *istioCR.Spec.Config.NumTrustedProxies != *numTrustedProxies {
		return true, nil
	}
	if isNewNotNil != isOldNotNil {
		return true, nil
	}

	return false, nil
}
