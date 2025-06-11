package pod

import (
	"context"
	"fmt"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/executor"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

type Create struct {
	Pod *corev1.Pod
}

func (p *Create) Description() string {
	return fmt.Sprintf("%s: name=%s, namespace=%s", "Create Pod", p.Pod.Name, p.Pod.Namespace)
}

func (p *Create) Execute(t *testing.T, ctx context.Context, k8sClient client.Client) error {
	if p.Pod == nil {
		return fmt.Errorf("pod is nil")
	}

	executor.Debugf(t, "Creating Pod: %+v", *p.Pod)

	err := k8sClient.Create(ctx, p.Pod)
	if err != nil {
		return fmt.Errorf("failed to create pod: %w", err)
	}

	// Send the created pod to the output channel
	return nil
}

func (p *Create) Cleanup(ctx context.Context, k8sClient client.Client) error {
	if p.Pod == nil {
		return fmt.Errorf("pod is nil")
	}
	err := k8sClient.Delete(ctx, p.Pod)
	if err != nil {
		return fmt.Errorf("failed to delete pod: %w", err)
	}
	return nil
}
