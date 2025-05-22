package pod

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Get struct {
	Context      context.Context
	PodNamespace string
	PodName      string
	K8SClient    client.Client

	Pod *corev1.Pod
}

func (p *Get) Name() string {
	return "Get"
}

func (p *Get) Args() map[string]string {
	return map[string]string{
		"Namespace": p.PodNamespace,
		"Name":      p.PodName,
	}
}

func (p *Get) Execute() error {
	pod := &corev1.Pod{}
	err := p.K8SClient.Get(p.Context, types.NamespacedName{Namespace: p.PodNamespace, Name: p.PodName}, pod)
	if err != nil {
		return err
	}
	p.Pod = pod
	return nil
}

func (p *Get) AssertSuccess() error {
	if p.Pod == nil {
		return fmt.Errorf("pod is nil")
	}
	return nil
}

func (p *Get) Output() interface{} {
	return p.Pod
}
