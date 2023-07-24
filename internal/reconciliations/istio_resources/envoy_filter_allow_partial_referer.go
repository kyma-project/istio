package istio_resources

import (
	"context"
	_ "embed"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

//go:embed envoy_filter_allow_partial_referer.yaml
var manifest []byte

type EnvoyFilterAllowPartialReferer struct {
}

func NewEnvoyFilterAllowPartialReferer() EnvoyFilterAllowPartialReferer {
	return EnvoyFilterAllowPartialReferer{}
}

func (EnvoyFilterAllowPartialReferer) apply(ctx context.Context, k8sClient client.Client) (controllerutil.OperationResult, error) {
	var filter unstructured.Unstructured
	err := yaml.Unmarshal(manifest, &filter)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	spec := filter.Object["spec"]

	return controllerutil.CreateOrUpdate(ctx, k8sClient, &filter, func() error {
		filter.Object["spec"] = spec
		return nil
	})
}

func (EnvoyFilterAllowPartialReferer) Name() string {
	return "partial referer envoy filter"
}

func (EnvoyFilterAllowPartialReferer) RequiresProxyRestart(p v1.Pod) bool {
	return pods.HasIstioSidecarStatusAnnotation(p) &&
		pods.IsPodReady(p)
	// TODO add correct filter
}

func (EnvoyFilterAllowPartialReferer) RequiresIngressGatewayRestart(p v1.Pod) bool {
	// TODO add correct filter
	return !p.CreationTimestamp.IsZero()
}
