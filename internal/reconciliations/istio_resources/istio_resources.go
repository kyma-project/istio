package istio_resources

import (
	"context"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Get returns all Istio resources required for the reconciliation specific for the given hyperscaler.
func Get(ctx context.Context, k8sClient client.Client) ([]Resource, error) {

	istioResources := []Resource{NewEnvoyFilterAllowPartialReferer(k8sClient)}
	istioResources = append(istioResources, NewGatewayKyma(k8sClient))
	istioResources = append(istioResources, NewVirtualServiceHealthz(k8sClient))
	istioResources = append(istioResources, NewPeerAuthenticationMtls(k8sClient))
	istioResources = append(istioResources, NewConfigMapControlPlane(k8sClient))
	istioResources = append(istioResources, NewConfigMapMesh(k8sClient))
	istioResources = append(istioResources, NewConfigMapPerformance(k8sClient))
	istioResources = append(istioResources, NewConfigMapService(k8sClient))
	istioResources = append(istioResources, NewConfigMapWorkload(k8sClient))

	isAws, err := clusterconfig.IsHyperscalerAWS(ctx, k8sClient)
	if err != nil {
		return nil, err
	}

	if isAws {
		istioResources = append(istioResources, NewProxyProtocolEnvoyFilter(k8sClient))
	}

	return istioResources, nil
}
