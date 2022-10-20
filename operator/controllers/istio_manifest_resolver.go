package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/module-manager/operator/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
func (m *ManifestResolver) Get(obj types.BaseCustomObject, logger logr.Logger) (types.InstallationSpec, error) {
	istioOperator, valid := obj.(*operatorv1alpha1.Istio)
	if !valid {
		return types.InstallationSpec{},
			fmt.Errorf("invalid type conversion for %s", client.ObjectKeyFromObject(obj))
	}
	return types.InstallationSpec{
		ChartPath:   m.chartPath,
		ReleaseName: istioOperator.Spec.ReleaseName,
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
