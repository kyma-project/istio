package istio

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/istio/operator/pkg/labels"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/retry"
)

func patchModuleResourcesWithModuleLabel(ctx context.Context, c client.Client) error {
	// we can't statically modify istio metadata easily without directly reconciling istio resources
	// this function goes through all resources created and labeled by istio installer to set additional label with module name
	// oh boy...
	s, err := k8slabels.Parse("operator.istio.io/component")
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
		{Group: "apps", Version: "v1", Kind: "DaemonSet"},
		// policy
		{Group: "policy", Version: "v1", Kind: "PodDisruptionBudget"},
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
		var obj client.Object
		for _, r := range list.Items {
			u := r.DeepCopy()
			u.SetGroupVersionKind(gvk)
			var patch client.Patch
			// Ressetkk: if the list grows, we'll have to think about some other solution
			// those resources contain templates for pods they manage.
			// Some of the istio pods (e.g. CNI) does not set operator.istio.io/component in a template,
			// So instead we additionally update the template
			switch u.GetKind() {
			case "Deployment":
				d := appsv1.Deployment{}
				if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &d); err != nil {
					return err
				}
				patch = client.StrategicMergeFrom(d.DeepCopy())
				updateObjectLabels(&d.ObjectMeta)
				updateObjectLabels(&d.Spec.Template.ObjectMeta)
				obj = &d
			case "DaemonSet":
				ds := appsv1.DaemonSet{}
				if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &ds); err != nil {
					return err
				}
				patch = client.StrategicMergeFrom(ds.DeepCopy())
				updateObjectLabels(&ds.ObjectMeta)
				updateObjectLabels(&ds.Spec.Template.ObjectMeta)
				obj = &ds
			// handle without a conversion
			default:
				// MergeFrom is used instead of StrategicMergeFrom, since Strategic cannot handle unstructured objects reliably
				// This is acceptable, as the applied labels are applied as an overlay on top of existing ones.
				patch = client.MergeFrom(u.DeepCopy())
				l := labels.SetModuleLabels(u.GetLabels())
				u.SetLabels(l)
				obj = u
			}

			if err := retry.OnError(retry.DefaultRetry, func() error {
				return c.Patch(ctx, obj, patch)
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func updateObjectLabels(obj *metav1.ObjectMeta) {
	l := labels.SetModuleLabels(obj.GetLabels())
	obj.SetLabels(l)
}
