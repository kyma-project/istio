package ns

import (
	"context"
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/e2e/setup"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

type CreateOpts func(object k8s.Object)

func WithLabels(labels map[string]string) CreateOpts {
	return func(object k8s.Object) {
		object.SetLabels(labels)
	}
}

func CreateNamespace(ctx context.Context, t *testing.T, name string, cfg *envconf.Config, opts ...CreateOpts) error {
	r, err := resources.New(cfg.Client().RESTConfig())
	if err != nil {
		return err
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}

	for _, opt := range opts {
		opt(ns)
	}

	setup.DeclareCleanup(t, func() {
		t.Log("Deleting namespace: ", name)
		require.NoError(t, DeleteNamespace(ctx, name, cfg))
	})
	t.Log("Creating namespace: ", name)

	return r.Create(ctx, ns)
}

func DeleteNamespace(ctx context.Context, name string, cfg *envconf.Config) error {
	r, err := resources.New(cfg.Client().RESTConfig())
	if err != nil {
		return err
	}
	return r.Delete(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}})
}
