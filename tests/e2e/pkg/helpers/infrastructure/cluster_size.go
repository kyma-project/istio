package infrastructure

import (
	"errors"
	"testing"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
)

// EnsureEvaluationClusterProfile ensures the cluster is running in Evaluation profile.
// If the cluster is not in Evaluation profile or the cluster size cannot be determined, the test fails.
func EnsureEvaluationClusterProfile(t *testing.T) error {
	t.Helper()

	clusterSize := getClusterSize(t)

	if clusterSize == clusterconfig.Production {
		return errors.New("test requires Evaluation cluster profile, but cluster is Production")
	}

	if clusterSize == clusterconfig.UnknownSize {
		return errors.New("test requires Evaluation cluster profile, but cluster is Unknown")
	}

	t.Logf("Cluster profile: %s", clusterSize.String())
	return nil
}

// EnsureProductionClusterProfile ensures the cluster is running in Production profile.
// If the cluster is not in Production profile or the cluster size cannot be determined, the test fails.
func EnsureProductionClusterProfile(t *testing.T) error {
	t.Helper()

	clusterSize := getClusterSize(t)

	if clusterSize == clusterconfig.Evaluation {
		return errors.New("test requires Production cluster profile, but cluster is Evaluation")
	}

	if clusterSize == clusterconfig.UnknownSize {
		return errors.New("test requires Production cluster profile, but cluster is Unknown")
	}

	t.Logf("Cluster profile: %s", clusterSize.String())
	return nil
}

func getClusterSize(t *testing.T) clusterconfig.ClusterSize {
	t.Helper()

	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Fatalf("Failed to get resources client: %v", err)
	}

	clusterSize, err := clusterconfig.EvaluateClusterSize(t.Context(), r.GetControllerRuntimeClient())
	if err != nil {
		t.Fatalf("Failed to evaluate cluster size: %v", err)
	}

	return clusterSize
}
