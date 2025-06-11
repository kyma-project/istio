package pod

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync/atomic"
	"testing"
)

type Get struct {
	PodNamespace string
	PodName      string

	finished atomic.Bool
	output   *corev1.Pod
}

func (p *Get) Output() *corev1.Pod {
	if !p.finished.Load() {
		return nil
	}

	return p.output
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

func (p *Get) Execute(t *testing.T, ctx context.Context, k8sClient client.Client) error {
	pod := &corev1.Pod{}

	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: p.PodNamespace, Name: p.PodName}, pod)
	require.NoError(t, err)

	p.output = pod
	p.finished.Store(true)
	return nil
}

func (p *Get) Cleanup(context.Context, client.Client) error {
	return nil
}
