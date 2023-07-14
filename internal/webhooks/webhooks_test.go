package webhooks

import (
	"context"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	"github.com/stretchr/testify/require"
	"istio.io/istio/istioctl/pkg/tag"
	v1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	revLabelKey   = "istio.io/rev"
	defaultWhName = "istio-sidecar-injector"
	taggedWhName  = "istio-revision-tag-default"
)

var validSelector = &metav1.LabelSelector{
	MatchExpressions: []metav1.LabelSelectorRequirement{{
		Key:      "istio-injection",
		Operator: "DoesNotExist",
	}},
}

var deactivatedSelector = &metav1.LabelSelector{
	MatchLabels: deactivatedLabel,
}

func createFakeClient(objects ...client.Object) client.Client {
	operatorv1alpha1.AddToScheme(scheme.Scheme)
	corev1.AddToScheme(scheme.Scheme)
	appsv1.AddToScheme(scheme.Scheme)
	networkingv1alpha3.AddToScheme(scheme.Scheme)

	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()
}

func createMutatingWebhookWithSelector(whConfName string, labels map[string]string, selector *metav1.LabelSelector) *v1.MutatingWebhookConfiguration {
	return &v1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:   whConfName,
			Labels: labels,
		},
		Webhooks: []v1.MutatingWebhook{
			{
				Name:              "object.sidecar-injector.istio.io",
				NamespaceSelector: selector,
				ObjectSelector:    selector,
			},
		},
	}
}

func Test_DeleteConflictedDefaultTag(t *testing.T) {
	ctx := context.Background()

	defer ctx.Done()

	t.Run("should not delete tagged webhook when old webhook is deactivated", func(t *testing.T) {
		// given
		defaultMwcObj := createMutatingWebhookWithSelector(defaultWhName, map[string]string{revLabelKey: tag.DefaultRevisionName}, deactivatedSelector)
		taggedMwcObj := createMutatingWebhookWithSelector(taggedWhName, map[string]string{tag.IstioTagLabel: tag.DefaultRevisionName, revLabelKey: tag.DefaultRevisionName}, validSelector)
		kubeclient := createFakeClient(defaultMwcObj, taggedMwcObj)
		// when
		err := DeleteConflictedDefaultTag(ctx, kubeclient)

		// then
		require.NoError(t, err)

		var taggedMwc v1.MutatingWebhookConfiguration
		err = kubeclient.Get(ctx, types.NamespacedName{Name: taggedWhName}, &taggedMwc)
		require.NoError(t, err)
		require.Equal(t, taggedMwcObj.Name, taggedMwc.Name)

		var defaultMwc v1.MutatingWebhookConfiguration
		err = kubeclient.Get(ctx, types.NamespacedName{Name: defaultWhName}, &defaultMwc)
		require.NoError(t, err)
		require.Equal(t, defaultMwcObj.Name, defaultMwc.Name)
	})
	t.Run("should delete conflicted tagged webhook if old one is not deactivated", func(t *testing.T) {
		// given
		defaultMwcObj := createMutatingWebhookWithSelector(defaultWhName, map[string]string{revLabelKey: tag.DefaultRevisionName}, validSelector)
		taggedMwcObj := createMutatingWebhookWithSelector(taggedWhName, map[string]string{tag.IstioTagLabel: tag.DefaultRevisionName, revLabelKey: tag.DefaultRevisionName}, validSelector)
		kubeclient := createFakeClient(defaultMwcObj, taggedMwcObj)
		// when
		err := DeleteConflictedDefaultTag(ctx, kubeclient)

		// then
		require.NoError(t, err)
		var taggedMwc v1.MutatingWebhookConfiguration
		err = kubeclient.Get(ctx, types.NamespacedName{Name: taggedWhName}, &taggedMwc)
		require.ErrorContains(t, err, "not found")

		var defaultMwc v1.MutatingWebhookConfiguration
		err = kubeclient.Get(ctx, types.NamespacedName{Name: defaultWhName}, &defaultMwc)
		require.NoError(t, err)
		require.Equal(t, defaultMwcObj.Name, defaultMwc.Name)
	})
	t.Run("should not delete tagged webhook if there is no old default webhook", func(t *testing.T) {
		// given
		taggedMwcObj := createMutatingWebhookWithSelector(taggedWhName, map[string]string{tag.IstioTagLabel: tag.DefaultRevisionName, revLabelKey: tag.DefaultRevisionName}, validSelector)
		kubeclient := createFakeClient(taggedMwcObj)

		// when
		err := DeleteConflictedDefaultTag(ctx, kubeclient)

		// then
		require.NoError(t, err)
		var taggedMwc v1.MutatingWebhookConfiguration
		err = kubeclient.Get(ctx, types.NamespacedName{Name: taggedWhName}, &taggedMwc)
		require.NoError(t, err)
		require.Equal(t, taggedMwcObj.Name, taggedMwc.Name)
	})
	t.Run("should not return an error if there is no tagged webhook", func(t *testing.T) {
		// given
		kubeclient := createFakeClient()

		// when
		err := DeleteConflictedDefaultTag(ctx, kubeclient)

		// then
		require.NoError(t, err)
	})
}
