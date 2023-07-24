package istio_resources

import (
	"context"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

type SampleResource struct{}

func NewSampleResource() *SampleResource {
	return &SampleResource{}
}

func (s *SampleResource) Apply(ctx context.Context, k8sClient client.Client) (changed bool, err error) {
	manifest, err := os.ReadFile("sample.yaml")
	if err != nil {
		return false, err
	}
	var filter unstructured.Unstructured
	err = yaml.Unmarshal(manifest, &filter)
	if err != nil {
		return false, err
	}

	spec := filter.Object["spec"]

	result, err := controllerutil.CreateOrUpdate(ctx, k8sClient, &filter, func() error { filter.Object["spec"] = spec; return nil })
	if err != nil {
		return false, err
	}

	if result == controllerutil.OperationResultNone {
		return false, nil
	}

	return true, nil
}

func (s *SampleResource) RestartPredicate(_ context.Context, _ client.Client) (igRestart bool, proxyRestart bool) {
	//TODO: fill in actual predicate
	return true, true
}
