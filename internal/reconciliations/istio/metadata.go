package istio

import (
	"context"
	"github.com/kyma-project/istio/operator/pkg/labels"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func updateResourcesMetadataForSelector(ctx context.Context, c client.Client) error {
	s, err := k8slabels.Parse("install.operator.istio.io/owning-resource")
	if err != nil {
		return err
	}
	kinds := []schema.GroupVersionKind{
		// core
		{Group: "", Version: "v1", Kind: "Pod"},
		{Group: "", Version: "v1", Kind: "Secret"},
		{Group: "", Version: "v1", Kind: "ConfigMap"},
		{Group: "", Version: "v1", Kind: "ServiceAccount"},
		// apiextensions
		{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"},
		// apps
		{Group: "apps", Version: "v1", Kind: "Deployment"},
		{Group: "apps", Version: "v1", Kind: "ReplicaSet"},
		{Group: "apps", Version: "v1", Kind: "DaemonSet"},
		// autoscaling
		{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"},
		// rbac
		{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"},
		{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"},
		{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"},
		{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"},
		// admission
		{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "ValidatingWebhookConfiguration"},
		{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "MutatingWebhookConfiguration"},
		// istio
		{Group: "networking.istio.io", Version: "v1alpha3", Kind: "EnvoyFilter"},
	}

	for _, gvk := range kinds {
		list := unstructured.UnstructuredList{}
		list.SetGroupVersionKind(gvk)
		err := c.List(ctx, &list, &client.ListOptions{LabelSelector: s})
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		for _, r := range list.Items {
			u := r.DeepCopy()
			l := labels.SetModuleLabels(u.GetLabels())
			u.SetLabels(l)
			if err := c.Update(ctx, u); err != nil {
				return err
			}
		}
	}
	return nil
}
