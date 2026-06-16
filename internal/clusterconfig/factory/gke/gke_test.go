package gke_test

import (
	"testing"

	"github.com/kyma-project/istio/operator/internal/clusterconfig/factory"
	"github.com/kyma-project/istio/operator/internal/clusterconfig/factory/gke"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCNI_GetCNIValues(t *testing.T) {
	c := gke.CNI{}

	values := c.CNIValues()

	assert.Equal(t, map[string]interface{}{
		"cniBinDir": "/home/kubernetes/bin",
		"resourceQuotas": map[string]bool{
			"enabled": true,
		},
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
			f := gke.NewFactory(tt.inputs)
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
