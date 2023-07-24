package istio_resources

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type resource interface {
	Apply(ctx context.Context, client client.Client) (changed bool, err error)
	RestartPredicate(ctx context.Context, client client.Client) (igRestart bool, proxyRestart bool)
}

type IstioResources struct {
	resourceList []resource
}

func (i *IstioResources) AddResourceToList(r resource) {
	i.resourceList = append(i.resourceList, r)
}

func (i *IstioResources) ApplyResources(ctx context.Context, client client.Client) (igRestart bool, proxyRestart bool, err error) {
	for _, r := range i.resourceList {
		needsIgRestart, needsProxyRestart := r.RestartPredicate(ctx, client)
		changed, err := r.Apply(ctx, client)
		if err != nil {
			return false, false, err
		}
		if changed == true {
			igRestart = igRestart || needsIgRestart
			proxyRestart = proxyRestart || needsProxyRestart
		}
	}

	return igRestart, proxyRestart, nil
}
