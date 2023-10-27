package clusterconfig

import (
	"context"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func IsHyperscalerAWS(ctx context.Context, k8sClient client.Client) (bool, error) {
	p, err := getProvider(ctx, k8sClient)
	if err != nil && k8serrors.IsNotFound(err) {
		ctrl.Log.Info("shoot-info not found to get provider. Assuming that the cluster is not running on Gardener.")
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return strings.ToLower(p) == "aws", nil
}
