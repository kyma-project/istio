package ingress_gateway

import (
	"context"
	_ "embed"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:embed proxy_protocol_envoy_filter.yaml
var proxyProtocolEnvoyFilter []byte

func applyProxyProtocolEnvoyFilter(ctx context.Context, k8sClient client.Client) error {
	return applyResource(ctx, k8sClient, manifest_cm_mesh, &owner)
}
