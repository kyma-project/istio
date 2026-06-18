//go:build experimental

package clusterconfig_test

import (
	"context"
	"testing"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDualStackEnabled_Experimental(t *testing.T) {
	c := createFakeClient(t, createKymaRuntimeConfigWithDualStack(t, true))

	ds, err := clusterconfig.IsDualStackEnabled(context.Background(), c)

	require.NoError(t, err)
	assert.True(t, ds)
}
