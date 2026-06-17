package k3d_test

import (
	"testing"

	"github.com/kyma-project/istio/operator/internal/clusterconfig/factory"
	"github.com/kyma-project/istio/operator/internal/clusterconfig/factory/k3d"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestK3D_GetCNIValues(t *testing.T) {
	s := k3d.CNI{}

	values := s.CNIValues()

	assert.Equal(t, map[string]interface{}{
		"cniBinDir":  "/var/lib/rancher/k3s/data/cni",
		"cniConfDir": "/var/lib/rancher/k3s/agent/etc/cni/net.d",
	}, values)
}

func TestFactory(t *testing.T) {
	tests := []struct {
		name          string
		inputs        factory.Inputs
		wantDualStack bool
	}{
		{name: "dual stack off", inputs: factory.Inputs{DualStackEnabled: false}, wantDualStack: false},
		{name: "dual stack on", inputs: factory.Inputs{DualStackEnabled: true}, wantDualStack: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := k3d.NewFactory(tt.inputs)
			require.NotNil(t, f)

			assert.Nil(t, f.LB())
			cni := f.CNI()
			require.NotNil(t, cni)
			assert.NotEmpty(t, cni.CNIValues())
			assert.False(t, f.NeedsProxyProtocol())
			assert.Equal(t, tt.wantDualStack, f.DualStackEnabled())
		})
	}
}
