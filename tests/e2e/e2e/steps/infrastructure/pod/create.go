package pod

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"log"
	"runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Create struct {
	Pod *corev1.Pod
}

func (p *Create) Description() string {
	var _, current, _, _ = runtime.Caller(0)
	return fmt.Sprintf("%s: name=%s, namespace=%s", current, p.Pod.Name, p.Pod.Namespace)
}

func (p *Create) Execute(ctx context.Context, k8sClient client.Client, debugLogger *log.Logger) error {
	if p.Pod == nil {
		return fmt.Errorf("pod is nil")
	}

	debugLogger.Printf("Creating Pod: %+v", *p.Pod)

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
