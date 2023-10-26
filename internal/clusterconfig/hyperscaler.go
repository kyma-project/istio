package clusterconfig

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func IsHyperscalerAWS(ctx context.Context, k8sClient client.Client) (bool, error) {
	p, err := getProvider(ctx, k8sClient)
	if err != nil {
		return false, err
	}

	return strings.ToLower(p) == "aws", nil
}
