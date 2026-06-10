package k3d_test

import (
	"testing"

	"github.com/kyma-project/istio/operator/internal/clusterconfig/strategy/k3d"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestK3D_GetCNIValues(t *testing.T) {
	s := k3d.K3D{}

	values, needed := s.GetCNIValues()

	assert.True(t, needed)
	assert.Equal(t, map[string]interface{}{
		"cniBinDir":  "/var/lib/rancher/k3s/data/cni",
		"cniConfDir": "/var/lib/rancher/k3s/agent/etc/cni/net.d",
	}, values)
}

func TestNewStrategy(t *testing.T) {
	s := k3d.NewStrategy()

	require.NotNil(t, s)
	require.NotNil(t, s.CNI)
	assert.Nil(t, s.LB)

	values, needed := s.GetCNIValues()
	assert.True(t, needed)
	assert.NotEmpty(t, values)
}
