package clusterrole

import (
	"testing"
	"time"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

var managedClusterRoles = []string{
	"kyma-istio-resources-edit",
	"kyma-istio-resources-view",
}

func TestClusterRoles_Create(t *testing.T) {
	_, err := modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
	require.NoError(t, err)

	r, err := client.ResourcesClient(t)
	require.NoError(t, err)

	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		for _, name := range managedClusterRoles {
			role := rbacv1.ClusterRole{}
			assert.NoError(collect, r.Get(t.Context(), name, "", &role))
		}
	}, 1*time.Minute, time.Second*10, "kyma-istio-resources not created")
}

func TestClusterRoles_Delete(t *testing.T) {
	istio, err := modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
	require.NoError(t, err)

	r, err := client.ResourcesClient(t)
	require.NoError(t, err)

	err = r.Delete(t.Context(), istio)
	require.NoError(t, err)

	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		for _, name := range managedClusterRoles {
			role := rbacv1.ClusterRole{}
			assert.True(collect, errors.IsNotFound(r.Get(t.Context(), name, "", &role)))
		}
	}, 1*time.Minute, time.Second*10, "kyma-istio-resources-edit still present")
}
