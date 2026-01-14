package domain

import (
	"context"

	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO: remove if unused
func GetClusterDomain(ctx context.Context, c client.Client) (string, error) {
	gcm := v1.ConfigMap{}

	err := c.Get(ctx, client.ObjectKey{Name: "shoot-info", Namespace: "kube-system"}, &gcm)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return "local.kyma.dev", nil
		}

		return "", err
	}

	return gcm.Data["domain"], nil
}
