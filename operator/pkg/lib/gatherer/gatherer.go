package gatherer

import (
	"context"

	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetIstioCR fetches the Istio CR from the cluster using client with supplied name and namespace
func GetIstioCR(ctx context.Context, client client.Client, name string, namespace string) (*v1alpha1.Istio, error) {
	cr := v1alpha1.Istio{}
	err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &cr)
	if err != nil {
		return nil, err
	}

	return &cr, nil
}

// GetIstioCR lists all Istio CRs on the cluster if no namespace is supplied, or from the supplied namespaces
func ListIstioCR(ctx context.Context, kubeclient client.Client, namespace ...string) (*v1alpha1.IstioList, error) {
	list := v1alpha1.IstioList{}

	if len(namespace) == 0 {
		err := kubeclient.List(ctx, &list)
		if err != nil {
			return nil, err
		}
	} else {
		for _, n := range namespace {
			namespacedList := v1alpha1.IstioList{}

			err := kubeclient.List(ctx, &namespacedList, &client.ListOptions{Namespace: n})
			if err != nil {
				return nil, err
			}

			list.Items = append(list.Items, namespacedList.Items...)
		}
	}

	return &list, nil
}
