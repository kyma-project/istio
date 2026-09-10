//go:build !experimental

package clusterconfig_test

import (
	"context"
	"testing"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestIsDualStackEnabled_AlphaOptIn(t *testing.T) {
	tests := []struct {
		name       string
		alphaOptIn bool
		objects    []client.Object
		want       bool
	}{
		{
			name:       "returns false when opt-in is false even if LB is dual-stack",
			alphaOptIn: false,
			objects:    []client.Object{createKymaRuntimeConfigWithDualStack(t, true)},
			want:       false,
		},
		{
			name:       "returns true when opt-in is true and LB is dual-stack",
			alphaOptIn: true,
			objects:    []client.Object{createKymaRuntimeConfigWithDualStack(t, true)},
			want:       true,
		},
		{
			name:       "returns false when opt-in is true but LB is not dual-stack",
			alphaOptIn: true,
			objects:    []client.Object{createKymaRuntimeConfigWithDualStack(t, false)},
			want:       false,
		},
		{
			name:       "returns false when opt-in is true but kyma-provisioning-info is missing",
			alphaOptIn: true,
			objects:    nil,
			want:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := createFakeClient(t, tt.objects...)

			ds, err := clusterconfig.IsDualStackEnabled(context.Background(), c, tt.alphaOptIn)

			require.NoError(t, err)
			assert.Equal(t, tt.want, ds)
		})
	}
}
