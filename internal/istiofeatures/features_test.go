package istiofeatures_test

import (
	"context"
	"testing"

	"github.com/kyma-project/istio/operator/internal/istiofeatures"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newFakeClient(t *testing.T, objects ...runtime.Object) *fake.ClientBuilder {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add corev1 to scheme: %v", err)
	}
	b := fake.NewClientBuilder().WithScheme(scheme)
	for _, o := range objects {
		b = b.WithRuntimeObjects(o)
	}
	return b
}

func TestGetIstioFeatures_ConfigMapNotFound(t *testing.T) {
	client := newFakeClient(t).Build()

	result, err := istiofeatures.Get(context.Background(), client)

	if err == nil {
		t.Error("expected an error when ConfigMap does not exist, got nil")
	}
	if result != (istiofeatures.IstioFeatures{}) {
		t.Errorf("expected zero-value IstioFeatures, got %+v", result)
	}
}

func TestGetIstioFeatures_ConfigMapExists(t *testing.T) {
	tests := []struct {
		name    string
		cmData  map[string]string
		want    istiofeatures.IstioFeatures
		wantErr bool
	}{
		{
			name:    "features key missing returns default IstioFeatures",
			cmData:  map[string]string{},
			want:    istiofeatures.IstioFeatures{},
			wantErr: false,
		},
		{
			name:    "empty JSON object returns default IstioFeatures",
			cmData:  map[string]string{"features": `{}`},
			want:    istiofeatures.IstioFeatures{DisableCni: false},
			wantErr: false,
		},
		{
			name:    "disableCni set to true",
			cmData:  map[string]string{"features": `{"disableCni": true}`},
			want:    istiofeatures.IstioFeatures{DisableCni: true},
			wantErr: false,
		},
		{
			name:    "disableCni set to false",
			cmData:  map[string]string{"features": `{"disableCni": false}`},
			want:    istiofeatures.IstioFeatures{DisableCni: false},
			wantErr: false,
		},
		{
			name:    "malformed JSON returns error",
			cmData:  map[string]string{"features": `{not valid json`},
			want:    istiofeatures.IstioFeatures{},
			wantErr: true,
		},
		{
			name:    "enableControlPlaneVPA set to true",
			cmData:  map[string]string{"features": `{"enableControlPlaneVPA": true}`},
			want:    istiofeatures.IstioFeatures{EnableControlPlaneVPA: true},
			wantErr: false,
		},
		{
			name:    "enableControlPlaneVPA set to false",
			cmData:  map[string]string{"features": `{"enableControlPlaneVPA": false}`},
			want:    istiofeatures.IstioFeatures{EnableControlPlaneVPA: false},
			wantErr: false,
		},
		{
			name:    "both features set",
			cmData:  map[string]string{"features": `{"disableCni": true, "enableControlPlaneVPA": true}`},
			want:    istiofeatures.IstioFeatures{DisableCni: true, EnableControlPlaneVPA: true},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &corev1.ConfigMap{
				ObjectMeta: v1.ObjectMeta{
					Name:      "istio-features",
					Namespace: "kyma-system",
				},
				Data: tt.cmData,
			}
			client := newFakeClient(t, cm).Build()

			result, err := istiofeatures.Get(context.Background(), client)

			if tt.wantErr && err == nil {
				t.Error("expected an error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.want {
				t.Errorf("expected %+v, got %+v", tt.want, result)
			}
		})
	}
}
