package istio_resources

import "sigs.k8s.io/controller-runtime/pkg/client"

func Get(k8sClient client.Client) []Resource {

	istioResources := []Resource{}
	istioResources = append(istioResources, NewGatewayKyma(k8sClient))
	istioResources = append(istioResources, NewVirtualServiceHealthz(k8sClient))
	istioResources = append(istioResources, NewPeerAuthenticationMtls(k8sClient))
	istioResources = append(istioResources, NewConfigMapControlPlane(k8sClient))
	istioResources = append(istioResources, NewConfigMapMesh(k8sClient))
	istioResources = append(istioResources, NewConfigMapPerformance(k8sClient))
	istioResources = append(istioResources, NewConfigMapService(k8sClient))
	istioResources = append(istioResources, NewConfigMapWorkload(k8sClient))
	return istioResources
}
