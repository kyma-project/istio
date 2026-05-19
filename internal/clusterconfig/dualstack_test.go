package clusterconfig_test

import (
	"context"
	"testing"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// IsDualStackEnabled Tests (without experimental tag)

func TestIsDualStackEnabled_WithoutExperimentalTag(t *testing.T) {
	// Given
	testClient := newTestClient(t)

	// When
	enabled, err := clusterconfig.IsDualStackEnabled(context.Background(), testClient)

	// Then
	require.NoError(t, err)
	assert.False(t, enabled)
}
