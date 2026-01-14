package infrastructure

import (
	"testing"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
)

// EnsureEvaluationClusterProfile ensures the cluster is running in Evaluation profile.
// If the cluster is not in Evaluation profile or the cluster size cannot be determined, the test fails.
func EnsureEvaluationClusterProfile(t *testing.T) {
	t.Helper()

	clusterSize := getClusterSize(t)

	if clusterSize == clusterconfig.Production {
		t.Fatalf("Test requires Evaluation cluster profile, but cluster is %s", clusterSize.String())
	}

	if clusterSize == clusterconfig.UnknownSize {
		t.Fatalf("Test requires Evaluation cluster profile, but cluster size cannot be determined")
	}

	t.Logf("Cluster profile: %s", clusterSize.String())
}

// EnsureProductionClusterProfile ensures the cluster is running in Production profile.
// If the cluster is not in Production profile or the cluster size cannot be determined, the test fails.
func EnsureProductionClusterProfile(t *testing.T) {
	t.Helper()

	clusterSize := getClusterSize(t)

	if clusterSize == clusterconfig.Evaluation {
		t.Fatalf("Test requires Production cluster profile, but cluster is %s", clusterSize.String())
	}

	if clusterSize == clusterconfig.UnknownSize {
		t.Fatalf("Test requires Production cluster profile, but cluster size cannot be determined")
	}

	t.Logf("Cluster profile: %s", clusterSize.String())
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
