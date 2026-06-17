package openstack_test

import (
	"testing"

	"github.com/kyma-project/istio/operator/internal/clusterconfig/factory"
	"github.com/kyma-project/istio/operator/internal/clusterconfig/factory/openstack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFactory_MakeLB(t *testing.T) {
	tests := []struct {
		name       string
		isGardener bool
		wantAnnots map[string]string
	}{
		{
			name:       "gardener returns proxy-protocol annotation",
			isGardener: true,
			wantAnnots: map[string]string{
				"loadbalancer.openstack.org/proxy-protocol": "v1",
			},
		},
		{
			name:       "non-gardener returns no annotations",
			isGardener: false,
			wantAnnots: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := openstack.NewFactory(factory.Inputs{UsesGardenOS: tt.isGardener})
			lb := f.LB()
			require.NotNil(t, lb)
			assert.Equal(t, tt.wantAnnots, lb.Annotations())
		})
	}
}

func TestFactory_MakeNeedsProxyProtocol(t *testing.T) {
	tests := []struct {
		name       string
		isGardener bool
	}{
		{name: "gardener", isGardener: true},
		{name: "non-gardener", isGardener: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := openstack.NewFactory(factory.Inputs{UsesGardenOS: tt.isGardener})
			assert.True(t, f.NeedsProxyProtocol())
		})
	}
}

func TestFactory_MakeCNI_AlwaysNil(t *testing.T) {
	f := openstack.NewFactory(factory.Inputs{UsesGardenOS: true})
	assert.Nil(t, f.CNI())
}

func TestFactory_DualStackEnabled(t *testing.T) {
	assert.True(t, openstack.NewFactory(factory.Inputs{DualStackEnabled: true}).DualStackEnabled())
	assert.False(t, openstack.NewFactory(factory.Inputs{DualStackEnabled: false}).DualStackEnabled())
}
