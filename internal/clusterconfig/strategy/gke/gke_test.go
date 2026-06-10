package gke_test

import (
	"testing"

	"github.com/kyma-project/istio/operator/internal/clusterconfig/strategy/gke"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCNI_GetCNIValues(t *testing.T) {
	c := gke.CNI{}

	values := c.GetCNIValues()

	assert.Equal(t, map[string]interface{}{
		"cniBinDir": "/home/kubernetes/bin",
		"resourceQuotas": map[string]bool{
			"enabled": true,
		},
	}, values)
}

func TestNewStrategy(t *testing.T) {
	s := gke.NewStrategy()

	require.NotNil(t, s)
	require.NotNil(t, s.CNI)
	assert.Nil(t, s.LB)

	values := s.GetCNIValues()
	assert.NotEmpty(t, values)
}
