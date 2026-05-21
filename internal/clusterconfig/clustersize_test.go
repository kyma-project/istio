package clusterconfig_test

import (
	"context"
	"testing"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EvaluateClusterSize Tests

func TestEvaluateClusterSize_BelowCPUThreshold(t *testing.T) {
	// Given
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-1",
		},
		Status: corev1.NodeStatus{
			Capacity: map[corev1.ResourceName]resource.Quantity{
				"cpu":    *resource.NewQuantity(clusterconfig.ProductionClusterCPUThreshold-1, resource.DecimalSI),
				"memory": *resource.NewScaledQuantity(32, resource.Giga),
			},
		},
	}
	testClient := newTestClient(t, node)

	// When
	size, err := clusterconfig.EvaluateClusterSize(context.Background(), testClient)

	// Then
	require.NoError(t, err)
	assert.Equal(t, clusterconfig.Evaluation, size)
}

func TestEvaluateClusterSize_BelowMemoryThreshold(t *testing.T) {
	// Given
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-1",
		},
		Status: corev1.NodeStatus{
			Capacity: map[corev1.ResourceName]resource.Quantity{
				"cpu":    *resource.NewMilliQuantity(12000, resource.DecimalSI),
				"memory": *resource.NewScaledQuantity(clusterconfig.ProductionClusterMemoryThresholdGi-1, resource.Giga),
			},
		},
	}
	testClient := newTestClient(t, node)

	// When
	size, err := clusterconfig.EvaluateClusterSize(context.Background(), testClient)

	// Then
	require.NoError(t, err)
	assert.Equal(t, clusterconfig.Evaluation, size)
}

func TestEvaluateClusterSize_Production(t *testing.T) {
	// Given
	node1 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-1",
		},
		Status: corev1.NodeStatus{
			Capacity: map[corev1.ResourceName]resource.Quantity{
				"cpu":    *resource.NewQuantity(clusterconfig.ProductionClusterCPUThreshold, resource.DecimalSI),
				"memory": *resource.NewScaledQuantity(clusterconfig.ProductionClusterMemoryThresholdGi, resource.Giga),
			},
		},
	}
	node2 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-2",
		},
		Status: corev1.NodeStatus{
			Capacity: map[corev1.ResourceName]resource.Quantity{
				"cpu":    *resource.NewQuantity(clusterconfig.ProductionClusterCPUThreshold, resource.DecimalSI),
				"memory": *resource.NewScaledQuantity(clusterconfig.ProductionClusterMemoryThresholdGi, resource.Giga),
			},
		},
	}
	testClient := newTestClient(t, node1, node2)

	// When
	size, err := clusterconfig.EvaluateClusterSize(context.Background(), testClient)

	// Then
	require.NoError(t, err)
	assert.Equal(t, clusterconfig.Production, size)
}
