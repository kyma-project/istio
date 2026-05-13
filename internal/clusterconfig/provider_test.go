package clusterconfig_test

import (
	"context"
	"testing"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetClusterProvider Tests

func TestGetClusterProvider_TableDriven(t *testing.T) {
	tests := []struct {
		name             string
		node             *corev1.Node
		expectedProvider string
	}{
		{
			name:             "AWS provider",
			node:             newAWSNode("test-node"),
			expectedProvider: clusterconfig.Aws,
		},
		{
			name:             "OpenStack provider",
			node:             newOpenStackNode("test-node"),
			expectedProvider: clusterconfig.Openstack,
		},
		{
			name: "Unknown provider",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
				Spec:       corev1.NodeSpec{ProviderID: "kubernetes://unknown"},
			},
			expectedProvider: clusterconfig.Other,
		},
		{
			name:             "GKE returns other",
			node:             newGKENode("test-node"),
			expectedProvider: clusterconfig.Other,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testClient := newTestClient(t, tt.node)
			provider, err := clusterconfig.GetClusterProvider(context.Background(), testClient)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedProvider, provider)
		})
	}
}

func TestGetClusterProvider_NoNodes(t *testing.T) {
	testClient := newTestClient(t)
	provider, err := clusterconfig.GetClusterProvider(context.Background(), testClient)
	require.NoError(t, err)
	assert.Equal(t, clusterconfig.Other, provider)
}
