package e2e_test

import (
	"context"
	"github.com/cucumber/godog/colors"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/infrastructure/pod"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestExecutor_Execute(t *testing.T) {
	k8sClient := fake.NewFakeClient()
	type fields struct {
		Steps        []e2e.Step
		LogOutputs   []*log.Logger
		TraceOutputs []*log.Logger
	}
	type args struct {
		t *testing.T
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "TestExecutor_Execute_GetPod_Success",
			fields: fields{
				Steps: []e2e.Step{
					&pod.Create{
						Context: context.Background(),
						Pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-pod",
								Namespace: "default",
							},
						},
						K8SClient: k8sClient,
					},
					&pod.Get{
						Context:      context.Background(),
						PodName:      "test-pod",
						PodNamespace: "default",
						K8SClient:    k8sClient,
					},
				},
				LogOutputs: []*log.Logger{
					log.New(os.Stdout,
						colors.Green("[LOG] "),
						0,
					),
				},
				TraceOutputs: []*log.Logger{
					log.New(colors.Colored(os.Stdout),
						colors.Yellow("[TRACE] "),
						0,
					),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := e2e.NewExecutor(t, tt.fields.Steps, tt.fields.LogOutputs, tt.fields.TraceOutputs)
			if err := e.Execute(); (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
