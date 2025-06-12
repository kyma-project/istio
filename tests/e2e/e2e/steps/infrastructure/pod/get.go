package pod

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

type Get struct {
	PodNamespace string
	PodName      string

	output *corev1.Pod
}

func (p *Get) Output() *corev1.Pod {
	return p.output
}

func (p *Get) Description() string {
	return fmt.Sprintf("%s: name=%s, namespace=%s", "Get Pod", p.PodName, p.PodNamespace)
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
	return nil
}

func (p *Get) Cleanup(*testing.T, context.Context, client.Client) error {
	return nil
}
