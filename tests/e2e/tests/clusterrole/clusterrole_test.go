package clusterrole

import (
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

func TestClusterRoles_Create(t *testing.T) {
	_, err := modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
	require.NoError(t, err)

	r, err := client.ResourcesClient(t)
	require.NoError(t, err)

	l := rbacv1.ClusterRoleList{}
	err = r.List(t.Context(), &l, resources.WithLabelSelector("kyma-project.io/module=istio"))
	require.NoError(t, err)
	assert.Len(t, l.Items, 4)
}

func TestClusterRoles_Delete(t *testing.T) {
	istio, err := modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
	require.NoError(t, err)

	r, err := client.ResourcesClient(t)
	require.NoError(t, err)

	err = r.Delete(t.Context(), istio)
	require.NoError(t, err)

	l := rbacv1.ClusterRoleList{}
	err = r.List(t.Context(), &l, resources.WithLabelSelector("kyma-project.io/module=istio"))
	require.NoError(t, err)
	assert.Len(t, l.Items, 0)
}
