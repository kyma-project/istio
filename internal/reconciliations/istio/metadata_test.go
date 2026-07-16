package istio

import (
	"context"
	"testing"

	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPatchModuleResourcesWithModuleLabel(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	c := createMetadataTestClient(t,
		&appsv1.Deployment{
			ObjectMeta: metadataObjectMeta("istio-system", "istio-ingressgateway", map[string]string{"operator.istio.io/component": "any_component"}),
		},
		&appsv1.DaemonSet{
			ObjectMeta: metadataObjectMeta("istio-system", "istio-cni-node", map[string]string{"operator.istio.io/component": "any_component"}),
		},
		&corev1.ServiceAccount{
			ObjectMeta: metadataObjectMeta("istio-system", "istiod", map[string]string{"operator.istio.io/component": "Pilot"}),
		},
		&corev1.Service{
			ObjectMeta: metadataObjectMeta("istio-system", "istiod", map[string]string{"operator.istio.io/component": "Pilot"}),
		},
		// istio.io/config=true: should receive module label
		&corev1.ConfigMap{
			ObjectMeta: metadataObjectMeta("istio-system", "istio", map[string]string{"istio.io/config": "true"}),
		},
		// additional resource: no label, but in the hardcoded list — should still receive module label
		&corev1.Secret{
			ObjectMeta: metadataObjectMeta("istio-system", "istio-ca-secret", nil),
		},
		// additional resource that also has istio.io/config=true: should not get a duplicate module label
		&corev1.ConfigMap{
			ObjectMeta: metadataObjectMeta("istio-system", "istio-gateway-status-leader", map[string]string{"istio.io/config": "true"}),
		},
		// additional resource that already has a custom label: custom label must be preserved
		&corev1.ConfigMap{
			ObjectMeta: metadataObjectMeta("istio-system", "istio-leader", map[string]string{"custom": "keep"}),
		},
		// unrelated resources must not receive module label
		&appsv1.Deployment{
			ObjectMeta: metadataObjectMeta("istio-system", "custom-deployment", nil),
		},
		&corev1.ConfigMap{
			ObjectMeta: metadataObjectMeta("istio-system", "custom-config", nil),
		},
		&corev1.Secret{
			ObjectMeta: metadataObjectMeta("istio-system", "custom-secret", nil),
		},
	)

	if err := patchModuleResourcesWithModuleLabel(ctx, c); err != nil {
		t.Fatalf("patchModuleResourcesWithModuleLabel() error = %v", err)
	}

	// operator-labeled resources
	assertMetadataLabelOnDeployment(t, ctx, c, "istio-system", "istio-ingressgateway")
	assertMetadataLabelOnDaemonSet(t, ctx, c, "istio-system", "istio-cni-node")
	assertMetadataLabelOnServiceAccount(t, ctx, c, "istio-system", "istiod")
	assertMetadataLabelOnService(t, ctx, c, "istio-system", "istiod")
	// config-labeled resources
	assertMetadataLabelOnConfigMap(t, ctx, c, "istio-system", "istio")
	//additional hardcoded resources
	assertMetadataLabelOnSecret(t, ctx, c, "istio-system", "istio-ca-secret")
	// additional resource also selected by istio.io/config- exactly 2 labels (no duplicate module label)
	assertNoDuplicateLabelsOnAdditionalWithLabel(t, ctx, c, "istio-system", "istio-gateway-status-leader")
	// additional resource with a pre-existing custom label- custom label preserved
	assertMetadataLabelOnConfigMapWithPreservedLabel(t, ctx, c, "istio-system", "istio-leader", "custom", "keep")
	assertNoMetadataLabelOnDeployment(t, ctx, c, "istio-system", "custom-deployment")
	assertNoMetadataLabelOnConfigMap(t, ctx, c, "istio-system", "custom-config")
	assertNoMetadataLabelOnSecret(t, ctx, c, "istio-system", "custom-secret")
}

func createMetadataTestClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		t.Fatalf("clientgoscheme.AddToScheme() error = %v", err)
	}
	if err := networkingv1alpha3.AddToScheme(s); err != nil {
		t.Fatalf("networkingv1alpha3.AddToScheme() error = %v", err)
	}
	return fake.NewClientBuilder().WithScheme(s).WithObjects(objects...).Build()
}

func metadataObjectMeta(namespace, name string, labels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: namespace,
		Name:      name,
		Labels:    labels,
	}
}

func assertMetadataLabelOnDeployment(t *testing.T, ctx context.Context, c client.Client, namespace, name string) {
	t.Helper()

	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &deployment); err != nil {
		t.Fatalf("get deployment %s/%s: %v", namespace, name, err)
	}

	assertModuleLabel(t, deployment.Labels)
	assertModuleLabel(t, deployment.Spec.Template.Labels)
}

func assertMetadataLabelOnDaemonSet(t *testing.T, ctx context.Context, c client.Client, namespace, name string) {
	t.Helper()

	var daemonSet appsv1.DaemonSet
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &daemonSet); err != nil {
		t.Fatalf("get daemonset %s/%s: %v", namespace, name, err)
	}

	assertModuleLabel(t, daemonSet.Labels)
	assertModuleLabel(t, daemonSet.Spec.Template.Labels)
}

func assertMetadataLabelOnConfigMap(t *testing.T, ctx context.Context, c client.Client, namespace, name string) {
	t.Helper()

	var configMap corev1.ConfigMap
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &configMap); err != nil {
		t.Fatalf("get configmap %s/%s: %v", namespace, name, err)
	}

	assertModuleLabel(t, configMap.Labels)
}

func assertMetadataLabelOnSecret(t *testing.T, ctx context.Context, c client.Client, namespace, name string) {
	t.Helper()

	var secret corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &secret); err != nil {
		t.Fatalf("get secret %s/%s: %v", namespace, name, err)
	}

	assertModuleLabel(t, secret.Labels)
}

func assertNoMetadataLabelOnDeployment(t *testing.T, ctx context.Context, c client.Client, namespace, name string) {
	t.Helper()

	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &deployment); err != nil {
		t.Fatalf("get deployment %s/%s: %v", namespace, name, err)
	}

	assertNoModuleLabel(t, deployment.Labels)
	assertNoModuleLabel(t, deployment.Spec.Template.Labels)
}

func assertNoMetadataLabelOnConfigMap(t *testing.T, ctx context.Context, c client.Client, namespace, name string) {
	t.Helper()

	var configMap corev1.ConfigMap
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &configMap); err != nil {
		t.Fatalf("get configmap %s/%s: %v", namespace, name, err)
	}

	assertNoModuleLabel(t, configMap.Labels)
}

func assertNoMetadataLabelOnSecret(t *testing.T, ctx context.Context, c client.Client, namespace, name string) {
	t.Helper()

	var secret corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &secret); err != nil {
		t.Fatalf("get secret %s/%s: %v", namespace, name, err)
	}

	assertNoModuleLabel(t, secret.Labels)
}

func assertModuleLabel(t *testing.T, labels map[string]string) {
	t.Helper()

	if got := labels["kyma-project.io/module"]; got != "istio" {
		t.Fatalf("expected kyma-project.io/module=istio, got %q (labels=%v)", got, labels)
	}
}

func assertNoModuleLabel(t *testing.T, labels map[string]string) {
	t.Helper()

	if got := labels["kyma-project.io/module"]; got != "" {
		t.Fatalf("expected kyma-project.io/module to be absent, got %q (labels=%v)", got, labels)
	}
}

func assertMetadataLabelOnServiceAccount(t *testing.T, ctx context.Context, c client.Client, namespace, name string) {
	t.Helper()

	var sa corev1.ServiceAccount
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &sa); err != nil {
		t.Fatalf("get serviceaccount %s/%s: %v", namespace, name, err)
	}

	assertModuleLabel(t, sa.Labels)
}

func assertMetadataLabelOnService(t *testing.T, ctx context.Context, c client.Client, namespace, name string) {
	t.Helper()

	var svc corev1.Service
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &svc); err != nil {
		t.Fatalf("get service %s/%s: %v", namespace, name, err)
	}

	assertModuleLabel(t, svc.Labels)
}

func assertMetadataLabelOnConfigMapWithPreservedLabel(t *testing.T, ctx context.Context, c client.Client, namespace, name, key, value string) {
	t.Helper()

	var configMap corev1.ConfigMap
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &configMap); err != nil {
		t.Fatalf("get configmap %s/%s: %v", namespace, name, err)
	}

	assertModuleLabel(t, configMap.Labels)
	if got := configMap.Labels[key]; got != value {
		t.Fatalf("expected %s=%s to be preserved, got %q (labels=%v)", key, value, got, configMap.Labels)
	}
}

func assertNoDuplicateLabelsOnAdditionalWithLabel(t *testing.T, ctx context.Context, c client.Client, namespace, name string) {
	t.Helper()

	var configMap corev1.ConfigMap
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &configMap); err != nil {
		t.Fatalf("get configmap %s/%s: %v", namespace, name, err)
	}
	//checking if it has both the config label and the module label
	assertModuleLabel(t, configMap.Labels)
	if got := configMap.Labels["istio.io/config"]; got != "true" {
		t.Fatalf("expected istio.io/config=true, got %q (labels=%v)", got, configMap.Labels)
	}

	if len(configMap.Labels) != 2 {
		t.Fatalf("expected exactly 2 labels after patching, got %d (labels=%v)", len(configMap.Labels), configMap.Labels)
	}
}
