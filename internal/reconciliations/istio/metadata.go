package istio

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/istio/operator/pkg/labels"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/retry"
)

const (
	operatorComponentSelector = "operator.istio.io/component"
	istioConfigSelector       = "istio.io/config=true"
)

func patchModuleResourcesWithModuleLabel(ctx context.Context, c client.Client) error {
	// we can't statically modify istio metadata easily without directly reconciling istio resources
	// this function goes through all resources created and labeled by istio installer to set additional label with module name
	// oh boy...
	operatorSelector, err := k8slabels.Parse(operatorComponentSelector)
	if err != nil {
		return err
	}
	configSelector, err := k8slabels.Parse(istioConfigSelector)
	if err != nil {
		return err
	}
	// additional resources that should be labeled regardless of their labels. might need to be extended
	additionalResources := []client.Object{
		//gateway status leader deprecated since Istio v1.27 https://github.com/istio/istio/pull/55715
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "istio-gateway-status-leader", Namespace: "istio-system"}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "istio-ip-autoallocate", Namespace: "istio-system"}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "istio-leader", Namespace: "istio-system"}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "istio-namespace-controller-election", Namespace: "istio-system"}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "istio-ca-secret", Namespace: "istio-system"}},
	}

	kinds := []schema.GroupVersionKind{
		// core
		{Group: "", Version: "v1", Kind: "Pod"},
		{Group: "", Version: "v1", Kind: "Secret"},
		{Group: "", Version: "v1", Kind: "ConfigMap"},
		{Group: "", Version: "v1", Kind: "ServiceAccount"},
		{Group: "", Version: "v1", Kind: "Service"},
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
		apiErr := c.List(ctx, &list, &client.ListOptions{LabelSelector: operatorSelector})
		if client.IgnoreNotFound(apiErr) != nil {
			return apiErr
		}
		//concatenate with list for different selector. so far only needed for ConfigMap but this works for every kind
		configList := unstructured.UnstructuredList{}
		configList.SetGroupVersionKind(gvk)
		apiErr = c.List(ctx, &configList, &client.ListOptions{LabelSelector: configSelector})
		if client.IgnoreNotFound(apiErr) != nil {
			return apiErr
		}
		seen := make(map[string]bool, len(list.Items))

		for _, item := range list.Items {
			key := item.GetNamespace() + "/" + item.GetName()
			seen[key] = true
		}

		for _, item := range configList.Items {
			key := item.GetNamespace() + "/" + item.GetName()
			if seen[key] {
				continue
			}
			list.Items = append(list.Items, item)
			seen[key] = true
		}

		for _, r := range list.Items {
			var (
				obj   client.Object
				patch client.Patch
			)
			u := r.DeepCopy()
			u.SetGroupVersionKind(gvk)

			// Ressetkk: if the list grows, we'll have to think about some other solution
			// those resources contain templates for pods they manage.
			// Some of the istio pods (e.g. CNI) does not set operator.istio.io/component in a template,
			// So instead we additionally update the template
			switch u.GetKind() {
			case "Deployment":
				d := appsv1.Deployment{}
				if convertErr := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &d); convertErr != nil {
					return convertErr
				}
				patch = client.StrategicMergeFrom(d.DeepCopy())
				updateObjectLabels(&d.ObjectMeta)
				updateObjectLabels(&d.Spec.Template.ObjectMeta)
				obj = &d
			case "DaemonSet":
				ds := appsv1.DaemonSet{}
				if convertErr := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &ds); convertErr != nil {
					return convertErr
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

			if retryErr := retry.OnError(retry.DefaultRetry, func() error {
				return c.Patch(ctx, obj, patch)
			}); retryErr != nil {
				return retryErr
			}
		}
	}
	for _, r := range additionalResources {
		if err := patchAdditionalResourceWithModuleLabel(ctx, c, r); err != nil {
			return err
		}
	}
	return nil
}

// fetches the resource from the cluster and patches it with the module label
func patchAdditionalResourceWithModuleLabel(ctx context.Context, c client.Client, obj client.Object) error {
	var fetched client.Object

	switch obj.(type) {
	case *corev1.ConfigMap:
		fetched = &corev1.ConfigMap{}
	case *corev1.Secret:
		fetched = &corev1.Secret{}
	default:
		return nil
	}
	apiErr := c.Get(ctx, client.ObjectKeyFromObject(obj), fetched)
	if apiErr != nil {
		if client.IgnoreNotFound(apiErr) == nil {
			return nil
		}
		return apiErr
	}
	//type assert back to client.Object because MergeFrom requires it
	patch := client.MergeFrom(fetched.DeepCopyObject().(client.Object))
	fetched.SetLabels(labels.SetModuleLabels(fetched.GetLabels()))

	if retryErr := retry.OnError(retry.DefaultRetry, func() error {
		return c.Patch(ctx, fetched, patch)
	}); retryErr != nil {
		return retryErr
	}
	return nil
}

func updateObjectLabels(obj *metav1.ObjectMeta) {
	l := labels.SetModuleLabels(obj.GetLabels())
	obj.SetLabels(l)
}
