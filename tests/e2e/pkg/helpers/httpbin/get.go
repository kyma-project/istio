package httpbin

import (
	"fmt"
	"testing"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
)

func GetHttpbinPods(t *testing.T, labelSelector string) (*v1.PodList, error) {
	t.Helper()
	c, err := client.ResourcesClient(t)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources client: %w", err)
	}

	podList := &v1.PodList{}
	err = c.List(t.Context(), podList, resources.WithLabelSelector(labelSelector))
	if err != nil {
		return nil, fmt.Errorf("failed to list httpbin pods with selector %s: %w", labelSelector, err)
	}

	return podList, nil
}
