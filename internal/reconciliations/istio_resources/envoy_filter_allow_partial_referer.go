package istio_resources

import (
	"context"
	_ "embed"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
	"time"
)

//go:embed envoy_filter_allow_partial_referer.yaml
var manifest []byte

const EnvoyFilterAnnotation = "istios.operator.kyma-project.io/updatedAt"

type EnvoyFilterAllowPartialReferer struct {
	ctx             context.Context
	k8sClient       client.Client
	envoyUpdateTime time.Time
}

func NewEnvoyFilterAllowPartialReferer(ctx context.Context, k8sClient client.Client) EnvoyFilterAllowPartialReferer {
	return EnvoyFilterAllowPartialReferer{ctx: ctx, k8sClient: k8sClient}
}

func (EnvoyFilterAllowPartialReferer) apply(ctx context.Context, k8sClient client.Client) (controllerutil.OperationResult, error) {
	var filter unstructured.Unstructured
	err := yaml.Unmarshal(manifest, &filter)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	spec := filter.Object["spec"]

	result, err := controllerutil.CreateOrUpdate(ctx, k8sClient, &filter, func() error {
		filter.Object["spec"] = spec
		return nil
	})
	if err != nil {
		return controllerutil.OperationResultNone, err
	}
	var ok bool
	if filter.GetAnnotations() != nil {
		_, ok = filter.GetAnnotations()[EnvoyFilterAnnotation]
	}

	if result != controllerutil.OperationResultNone || filter.GetAnnotations() == nil || !ok {
		err := annotateWithTimestamp(ctx, filter, k8sClient)
		if err != nil {
			return controllerutil.OperationResultNone, err
		}
	}

	return result, nil
}

func (*EnvoyFilterAllowPartialReferer) Name() string {
	return "partial referer envoy filter"
}

func (e *EnvoyFilterAllowPartialReferer) RequiresProxyRestart(p v1.Pod) (bool, error) {
	if e.envoyUpdateTime.IsZero() {
		updateTime, err := getUpdateTime(e.ctx, e.k8sClient)
		if err != nil {
			return false, err
		}
		e.envoyUpdateTime = updateTime
	}

	return podIsOlder(p, e.envoyUpdateTime) && pods.HasIstioSidecarStatusAnnotation(p) &&
		pods.IsPodReady(p), nil
}

func (e *EnvoyFilterAllowPartialReferer) RequiresIngressGatewayRestart(p v1.Pod) (bool, error) {
	if e.envoyUpdateTime.IsZero() {
		updateTime, err := getUpdateTime(e.ctx, e.k8sClient)
		if err != nil {
			return false, err
		}
		e.envoyUpdateTime = updateTime
	}

	return podIsOlder(p, e.envoyUpdateTime), nil
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
	err := yaml.Unmarshal(manifest, &object)
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
