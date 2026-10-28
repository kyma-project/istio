package controller

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestShouldEvaluate(t *testing.T) {
	injectFalseLabels := map[string]string{"sidecar.istio.io/inject": "false"}
	injectFalseAnnotations := map[string]string{"sidecar.istio.io/inject": "false"}
	nsUnlabeled := map[string]string{}
	nsEnabled := map[string]string{"istio-injection": "enabled"}

	makeDeployment := func(name, namespace string) *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
	}

	tests := []struct {
		name              string
		obj               *appsv1.Deployment
		nsLabels          map[string]string
		tmplLabels        map[string]string
		tmplAnnotations   map[string]string
		wantShouldEvaluate bool
	}{
		{
			name:              "ingress gateway with inject=false label+anno, ns unlabeled → in scope",
			obj:               makeDeployment("istio-ingressgateway", "istio-system"),
			nsLabels:          nsUnlabeled,
			tmplLabels:        injectFalseLabels,
			tmplAnnotations:   injectFalseAnnotations,
			wantShouldEvaluate: true,
		},
		{
			name:              "ordinary Deployment inject=false, ns unlabeled → excluded",
			obj:               makeDeployment("my-app", "default"),
			nsLabels:          nsUnlabeled,
			tmplLabels:        injectFalseLabels,
			tmplAnnotations:   injectFalseAnnotations,
			wantShouldEvaluate: false,
		},
		{
			name:              "ordinary Deployment, ns istio-injection=enabled → in scope",
			obj:               makeDeployment("my-app", "default"),
			nsLabels:          nsEnabled,
			tmplLabels:        nil,
			tmplAnnotations:   nil,
			wantShouldEvaluate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldEvaluate(tt.obj, tt.nsLabels, tt.tmplLabels, tt.tmplAnnotations)
			if got != tt.wantShouldEvaluate {
				t.Errorf("shouldEvaluate() = %v, want %v", got, tt.wantShouldEvaluate)
			}
		})
	}
}
