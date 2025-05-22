package pod

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Create struct {
	Context   context.Context
	Pod       *corev1.Pod
	K8SClient client.Client
}

func (p *Create) Name() string {
	return "CreatePod"
}

func (p *Create) Args() map[string]string {
	return map[string]string{
		"Pod": fmt.Sprintf("%+v", p.Pod),
	}
}

func (p *Create) Execute() error {
	if p.Pod == nil {
		return fmt.Errorf("pod is nil")
	}
	err := p.K8SClient.Create(p.Context, p.Pod)
	if err != nil {
		return err
	}
	return nil
}

func (p *Create) AssertSuccess() error {
	get := &Get{
		Context:      p.Context,
		PodNamespace: p.Pod.Namespace,
		PodName:      p.Pod.Name,
		K8SClient:    p.K8SClient,
	}

	err := get.Execute()
	if err != nil {
		return err
	}
	if get.Pod == nil {
		return fmt.Errorf("pod is nil")
	}

	return nil
}

func (p *Create) Output() interface{} {
	return nil
}
