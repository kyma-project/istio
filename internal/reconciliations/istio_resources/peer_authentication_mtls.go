package istio_resources

import (
	"context"
	_ "embed"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//go:embed peer_authentication_mtls.yaml
var manifest_pa_mtls []byte

type PeerAuthenticationMtls struct {
	k8sClient client.Client
}

func NewPeerAuthenticationMtls(k8sClient client.Client) PeerAuthenticationMtls {
	return PeerAuthenticationMtls{k8sClient: k8sClient}
}

func (PeerAuthenticationMtls) apply(ctx context.Context, k8sClient client.Client, _ metav1.OwnerReference, _ map[string]string) (controllerutil.OperationResult, error) {
	return applyResource(ctx, k8sClient, manifest_pa_mtls, nil)
}

func (PeerAuthenticationMtls) Name() string {
	return "PeerAuthentication/default"
}
