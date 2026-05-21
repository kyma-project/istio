//go:build experimental

package clusterconfig_test

import (
	"context"
	"testing"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func createKymaProvisioningInfoCM(dualStackEnabled bool) *corev1.ConfigMap {
	networkDetails := map[string]interface{}{
		"dualStackIPEnabled": dualStackEnabled,
	}
	details := map[string]interface{}{
		"networkDetails": networkDetails,
	}
	detailsBytes, _ := yaml.Marshal(details)

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kyma-provisioning-info",
			Namespace: "kyma-system",
		},
		Data: map[string]string{
			"details": string(detailsBytes),
		},
	}
}

func TestIsDualStackEnabled_WithExperimentalTag(t *testing.T) {
	// Given
	cm := createKymaProvisioningInfoCM(true)
	client := newTestClient(t, cm)

	// When
	enabled, err := clusterconfig.IsDualStackEnabled(context.Background(), client)

	// Then
	require.NoError(t, err)
	assert.True(t, enabled)
}
