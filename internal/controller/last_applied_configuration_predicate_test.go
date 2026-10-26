package controller

import (
	"testing"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/pkg/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func istioWithAnnotations(annotations map[string]string) *operatorv1alpha2.Istio {
	return &operatorv1alpha2.Istio{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
		},
	}
}

func TestLastAppliedConfigurationChangedPredicate_Update(t *testing.T) {
	tests := []struct {
		name string
		old  client.Object
		new  client.Object
		want bool
	}{
		{
			name: "newly set when annotation was absent",
			old:  istioWithAnnotations(nil),
			new:  istioWithAnnotations(map[string]string{labels.LastAppliedConfiguration: "config-a"}),
			want: true,
		},
		{
			name: "value changed",
			old:  istioWithAnnotations(map[string]string{labels.LastAppliedConfiguration: "config-a"}),
			new:  istioWithAnnotations(map[string]string{labels.LastAppliedConfiguration: "config-b"}),
			want: true,
		},
		{
			name: "value unchanged",
			old:  istioWithAnnotations(map[string]string{labels.LastAppliedConfiguration: "config-a"}),
			new:  istioWithAnnotations(map[string]string{labels.LastAppliedConfiguration: "config-a"}),
			want: false,
		},
		{
			name: "unrelated annotation changed, lastAppliedConfiguration unchanged",
			old: istioWithAnnotations(map[string]string{
				labels.LastAppliedConfiguration:      "config-a",
				"operator.kyma-project.io/istio-tag": "1.0.0",
			}),
			new: istioWithAnnotations(map[string]string{
				labels.LastAppliedConfiguration:      "config-a",
				"operator.kyma-project.io/istio-tag": "2.0.0",
			}),
			want: false,
		},
		{
			name: "nil old object",
			old:  nil,
			new:  istioWithAnnotations(map[string]string{labels.LastAppliedConfiguration: "config-a"}),
			want: false,
		},
		{
			name: "nil new object",
			old:  istioWithAnnotations(map[string]string{labels.LastAppliedConfiguration: "config-a"}),
			new:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := LastAppliedConfigurationChangedPredicate{}
			got := p.Update(event.TypedUpdateEvent[client.Object]{
				ObjectOld: tt.old,
				ObjectNew: tt.new,
			})
			if got != tt.want {
				t.Errorf("Update() = %v, want %v", got, tt.want)
			}
		})
	}
}
