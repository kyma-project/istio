package infrastructure

import (
	"fmt"
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

func GetHttpbinPods(t *testing.T) (*v1.PodList, error) {
	t.Helper()
	c, err := client.ResourcesClient(t)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources client: %w", err)
	}

	podList := &v1.PodList{}
	err = c.List(t.Context(), podList, resources.WithLabelSelector("app=httpbin"))
	if err != nil {
		return nil, fmt.Errorf("failed to list istiod pods: %w", err)
	}

	return podList, nil
}
