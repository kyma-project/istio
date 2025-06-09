package pod

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync/atomic"
)

type Get struct {
	PodNamespace string
	PodName      string

	Output atomic.Pointer[corev1.Pod]
}

func (p *Get) Description() string {
	var _, current, _, _ = runtime.Caller(0)
	return fmt.Sprintf("%s: name=%s, namespace=%s", current, p.PodName, p.PodNamespace)
}

func (p *Get) Args() map[string]string {
	return map[string]string{
		"Namespace":   p.PodNamespace,
		"Description": p.PodName,
	}
}

func (p *Get) Execute(ctx context.Context, k8sClient client.Client, _ *log.Logger) error {
	pod := &corev1.Pod{}

	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: p.PodNamespace, Name: p.PodName}, pod)
	if err != nil {
		return err
	}
	p.Output.Store(pod)
	return nil
}

func (p *Get) Cleanup(context.Context, client.Client) error {
	return nil
}
