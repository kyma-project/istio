package istio_resources

import (
	"context"
	_ "embed"
	"github.com/kyma-project/istio/operator/internal/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//go:embed proxy_protocol_envoy_filter.yaml
var proxyProtocolEnvoyFilter []byte

type ProxyProtocolEnvoyFilter struct {
	k8sClient client.Client
}

func NewProxyProtocolEnvoyFilter(k8sClient client.Client) ProxyProtocolEnvoyFilter {
	return ProxyProtocolEnvoyFilter{k8sClient: k8sClient}
}

func (ProxyProtocolEnvoyFilter) reconcile(ctx context.Context, k8sClient client.Client, _ metav1.OwnerReference, _ map[string]string) (controllerutil.OperationResult, error) {
	return resources.Apply(ctx, k8sClient, proxyProtocolEnvoyFilter, nil)
}

func (ProxyProtocolEnvoyFilter) Name() string {
	return "EnvoyFilter/proxy-protocol"
}
