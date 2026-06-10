package openstack_test

import (
	"testing"

	"github.com/kyma-project/istio/operator/internal/clusterconfig/strategy/openstack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStrategy_GetLBAnnotations(t *testing.T) {
	tests := []struct {
		name        string
		isGardener  bool
		wantNeeded  bool
		wantAnnots  map[string]string
	}{
		{
			name:       "gardener returns proxy-protocol annotation",
			isGardener: true,
			wantNeeded: true,
			wantAnnots: map[string]string{
				"loadbalancer.openstack.org/proxy-protocol": "v1",
			},
		},
		{
			name:       "non-gardener returns no annotations",
			isGardener: false,
			wantNeeded: false,
			wantAnnots: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := openstack.NewStrategy(tt.isGardener)
			require.NotNil(t, s)
			require.NotNil(t, s.LB)

			annots, needed := s.GetLBAnnotations()
			assert.Equal(t, tt.wantNeeded, needed)
			assert.Equal(t, tt.wantAnnots, annots)
		})
	}
}

func TestStrategy_RequiresProxyProtocolEnvoyFilter(t *testing.T) {
	tests := []struct {
		name       string
		isGardener bool
		want       bool
	}{
		{name: "gardener requires proxy protocol envoy filter", isGardener: true, want: true},
		{name: "non-gardener still requires proxy protocol envoy filter", isGardener: false, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := openstack.NewStrategy(tt.isGardener)
			assert.Equal(t, tt.want, s.RequiresProxyProtocolEnvoyFilter())
		})
	}
}

func TestNewStrategy_NoCNI(t *testing.T) {
	s := openstack.NewStrategy(true)
	assert.Nil(t, s.CNI)
}
