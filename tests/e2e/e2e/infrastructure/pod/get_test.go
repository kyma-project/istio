package pod

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestGetPod_Execute(t *testing.T) {
	type fields struct {
		Context      context.Context
		PodNamespace string
		PodName      string
		K8SClient    client.Client
	}
	tests := []struct {
		name        string
		fields      fields
		wantErr     bool
		wantSuccess bool
	}{
		{
			name: "TestGetPod_Execute_Failure",
			fields: fields{
				Context:      context.TODO(),
				PodNamespace: "default",
				PodName:      "test-pod",
				K8SClient:    fake.NewFakeClient(),
			},
			wantErr:     true,
			wantSuccess: false,
		},
		{
			name: "TestGetPod_Execute_Success",
			fields: fields{
				Context:      context.TODO(),
				PodNamespace: "default",
				PodName:      "test-pod",
				K8SClient: fake.NewFakeClient(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
				}),
			},
			wantErr:     false,
			wantSuccess: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Get{
				Context:      tt.fields.Context,
				PodNamespace: tt.fields.PodNamespace,
				PodName:      tt.fields.PodName,
				K8SClient:    tt.fields.K8SClient,
			}
			if err := p.Execute(); (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := p.AssertSuccess(); (err == nil) != tt.wantSuccess {
				t.Errorf("AssertSuccess() error = %v, wantSuccess %v", err, tt.wantSuccess)
			}
		})
	}
}
