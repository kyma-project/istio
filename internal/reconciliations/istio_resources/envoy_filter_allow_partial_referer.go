package istio_resources

import (
	"context"
	_ "embed"
	"time"

	"github.com/kyma-project/istio/operator/internal/filter"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

//go:embed envoy_filter_allow_partial_referer.yaml
var manifest_ef_allow_partial_referer []byte

const EnvoyFilterAnnotation = "istios.operator.kyma-project.io/updatedAt"

type EnvoyFilterAllowPartialReferer struct {
	k8sClient client.Client
}

type EnvoyFilterEvaluator struct {
	envoyUpdateTime time.Time
}

func NewEnvoyFilterAllowPartialReferer(k8sClient client.Client) EnvoyFilterAllowPartialReferer {
	return EnvoyFilterAllowPartialReferer{k8sClient: k8sClient}
}

func (e EnvoyFilterAllowPartialReferer) NewProxyRestartEvaluator(ctx context.Context) (filter.ProxyRestartEvaluator, error) {
	return newEnvoyFilterEvaluator(ctx, e.k8sClient)
}

func (e EnvoyFilterAllowPartialReferer) NewIngressGatewayEvaluator(ctx context.Context) (filter.IngressGatewayRestartEvaluator, error) {
	return newEnvoyFilterEvaluator(ctx, e.k8sClient)
}

func newEnvoyFilterEvaluator(ctx context.Context, k8sClient client.Client) (EnvoyFilterEvaluator, error) {
	updateTime, err := getUpdateTime(ctx, k8sClient)
	if err != nil {
		return EnvoyFilterEvaluator{}, err
	}
	return EnvoyFilterEvaluator{envoyUpdateTime: updateTime}, nil
}

func (e EnvoyFilterAllowPartialReferer) apply(ctx context.Context, k8sClient client.Client) (controllerutil.OperationResult, error) {
	var envoyFilter unstructured.Unstructured
	err := yaml.Unmarshal(manifest_ef_allow_partial_referer, &envoyFilter)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	spec := envoyFilter.Object["spec"]

	result, err := controllerutil.CreateOrUpdate(ctx, k8sClient, &envoyFilter, func() error {
		envoyFilter.Object["spec"] = spec
		return nil
	})
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	var efaFound, daFound bool
	annotations := envoyFilter.GetAnnotations()
	if annotations != nil {
		_, efaFound = annotations[EnvoyFilterAnnotation]
		_, daFound = annotations[istio.DisclaimerKey]
	}

	if result != controllerutil.OperationResultNone || !efaFound {
		err := annotateWithTimestamp(ctx, envoyFilter, k8sClient)
		if err != nil {
			return controllerutil.OperationResultNone, err
		}
	}
	if !daFound {
		err := annotateWithDisclaimer(ctx, envoyFilter, k8sClient)
		if err != nil {
			return controllerutil.OperationResultNone, err
		}
	}

	return result, nil
}

func (EnvoyFilterAllowPartialReferer) Name() string {
	return "partial referer envoy filter"
}

func (e EnvoyFilterEvaluator) RequiresProxyRestart(p v1.Pod) bool {
	return pods.HasIstioSidecarStatusAnnotation(p) &&
		pods.IsPodReady(p) && podIsOlder(p, e.envoyUpdateTime)
}

func (e EnvoyFilterEvaluator) RequiresIngressGatewayRestart(p v1.Pod) bool {
	return podIsOlder(p, e.envoyUpdateTime)
}

// Checks whether the pod CreationTimestamp is older than EnvoyFilterAnnotation
// If EnvoyFilterAnnotation returns false
func podIsOlder(pod v1.Pod, envoyTime time.Time) bool {
	return !pod.CreationTimestamp.IsZero() && !envoyTime.IsZero() && pod.CreationTimestamp.Compare(envoyTime) <= 0
}

func annotateWithTimestamp(ctx context.Context, filter unstructured.Unstructured, k8sClient client.Client) error {
	annotations := filter.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[EnvoyFilterAnnotation] = time.Now().Format(time.RFC3339)
	filter.SetAnnotations(annotations)

	err := k8sClient.Update(ctx, &filter)
	return err
}

func getUpdateTime(ctx context.Context, k8sClient client.Client) (time.Time, error) {
	var object unstructured.Unstructured
	err := yaml.Unmarshal(manifest_ef_allow_partial_referer, &object)
	if err != nil {
		return time.Time{}, err
	}

	err = k8sClient.Get(ctx, types.NamespacedName{Namespace: object.GetNamespace(), Name: object.GetName()}, &object)
	if err != nil {
		return time.Time{}, err
	}

	annotations := object.GetAnnotations()
	annot, ok := annotations[EnvoyFilterAnnotation]
	if ok {
		return time.Parse(time.RFC3339, annot)
	} else {
		return time.Time{}, nil
	}
}
