package factory_test

import (
	"testing"

	"github.com/kyma-project/istio/operator/internal/clusterconfig/factory"
	"github.com/stretchr/testify/assert"
)

func TestDefaultFactory(t *testing.T) {
	tests := []struct {
		name           string
		inputs         factory.Inputs
		wantDualStack  bool
		wantNeedsProxy bool
	}{
		{
			name:           "dual stack disabled",
			inputs:         factory.Inputs{DualStackEnabled: false, UsesGardenOS: false},
			wantDualStack:  false,
			wantNeedsProxy: false,
		},
		{
			name:           "dual stack enabled",
			inputs:         factory.Inputs{DualStackEnabled: true, UsesGardenOS: false},
			wantDualStack:  true,
			wantNeedsProxy: false,
		},
		{
			name:           "garden OS does not affect default factory",
			inputs:         factory.Inputs{DualStackEnabled: false, UsesGardenOS: true},
			wantDualStack:  false,
			wantNeedsProxy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := factory.DefaultFactory(tt.inputs)

			assert.Nil(t, f.LB())
			assert.Nil(t, f.CNI())
			assert.Equal(t, tt.wantNeedsProxy, f.NeedsProxyProtocol())
			assert.Equal(t, tt.wantDualStack, f.DualStackEnabled())
		})
	}
}
