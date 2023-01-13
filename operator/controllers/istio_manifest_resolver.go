package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/kyma-project/module-manager/operator/pkg/types"
)

var (
	ConfigFlags types.Flags
	SetFlags    types.Flags
)

const (
	istioAnnotationKey   = "owner"
	istioAnnotationValue = "istio-operator"
	istioFinalizer       = "istio-finalizer"
)

// Get returns the chart information to be processed.
func (m *ManifestResolver) Get(obj types.BaseCustomObject, _ logr.Logger) (types.InstallationSpec, error) {
	return types.InstallationSpec{
		ChartPath: m.chartPath,
		ChartFlags: types.ChartFlags{
			ConfigFlags: ConfigFlags,
			SetFlags:    SetFlags,
		},
	}, nil
}

// transform modifies the resources based on some criteria, before installation.
func transform(_ context.Context, _ types.BaseCustomObject, manifestResources *types.ManifestResources) error {
	for _, resource := range manifestResources.Items {
		annotations := resource.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string, 0)
		}
		if annotations[istioAnnotationKey] == "" {
			annotations[istioAnnotationKey] = istioAnnotationValue
			resource.SetAnnotations(annotations)
		}
	}
	return nil
}
