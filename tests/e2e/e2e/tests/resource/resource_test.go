package resource_test

import (
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/setup"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/istio/operator/tests/e2e/e2e/helpers/infrastructure/yamlfile"
)

func TestCreateResource(t *testing.T) {
	k8sClient := setup.ClientFromKubeconfig(t)

	// given
	err := yamlfile.CreateObjectFromYamlFile(t, k8sClient, "pod.yaml")
	require.NoError(t, err)

	// when
	res, err := yamlfile.GetObjectFromYamlFile(t, k8sClient, "pod.yaml")
	require.NoError(t, err)

	// then
	require.NotNil(t, res, "Expected a non-nil resource after creation")
	t.Logf("Created resource: %s", res)
}
