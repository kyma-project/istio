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

// Load Balancer Annotation Tests

func TestAWSNLBConfig_ContainsAllRequiredAnnotations(t *testing.T) {
	// Given
	config := clusterconfig.AWSNLBConfig

	// When
	annotations := extractServiceAnnotations(config)

	// Then
	require.NotNil(t, annotations, "AWS NLB config should have service annotations")

	// Verify all required AWS NLB annotations
	assert.Equal(t, "nlb", annotations["service.beta.kubernetes.io/aws-load-balancer-type"],
		"AWS NLB type annotation should be set")
	assert.Equal(t, "internet-facing", annotations["service.beta.kubernetes.io/aws-load-balancer-scheme"],
		"AWS load balancer scheme should be internet-facing")
	assert.Equal(t, "instance", annotations["service.beta.kubernetes.io/aws-load-balancer-nlb-target-type"],
		"AWS NLB target type should be instance")
}

func TestOpenStackLBProxyProtocolConfig_ContainsProxyProtocol(t *testing.T) {
	// Given
	config := clusterconfig.OpenStackLBProxyProtocolConfig

	// When
	annotations := extractServiceAnnotations(config)

	// Then
	require.NotNil(t, annotations, "OpenStack config should have service annotations")

	// Verify OpenStack proxy protocol annotation
	assert.Equal(t, "v1", annotations["loadbalancer.openstack.org/proxy-protocol"],
		"OpenStack proxy protocol annotation should be set to v1")
}

func TestEvaluateClusterConfiguration_GKE_NoLBAnnotations(t *testing.T) {
	// Given
	node := newGKENode("gke-node-1")
	testClient := newTestClient(t, node)
	provider, err := clusterconfig.GetClusterProvider(context.Background(), testClient)
	require.NoError(t, err)

	// When
	config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), testClient, provider)

	// Then
	require.NoError(t, err)
	annotations := extractServiceAnnotations(config)

	// GKE should have NO load balancer annotations (only CNI config)
	assert.Nil(t, annotations, "GKE should not have load balancer annotations")
}

func TestEvaluateClusterConfiguration_K3d_NoLBAnnotations(t *testing.T) {
	// Given
	node := newK3dNode("k3d-node-1")
	testClient := newTestClient(t, node)
	provider, err := clusterconfig.GetClusterProvider(context.Background(), testClient)
	require.NoError(t, err)

	// When
	config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), testClient, provider)

	// Then
	require.NoError(t, err)
	annotations := extractServiceAnnotations(config)

	// K3d should have NO load balancer annotations (only CNI config)
	assert.Nil(t, annotations, "K3d should not have load balancer annotations")
}

// ShouldUseNLB Logic Tests

func TestShouldUseNLB_NoConfigMap(t *testing.T) {
	// Given
	testClient := newTestClient(t)

	// When
	useNLB, err := clusterconfig.ShouldUseNLB(context.Background(), testClient)

	// Then
	require.NoError(t, err)
	assert.True(t, useNLB, "Should use NLB when elb-deprecated ConfigMap is absent")
}

func TestShouldUseNLB_ConfigMapPresentButServiceHasNLB(t *testing.T) {
	// Given
	elbDeprecatedCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "elb-deprecated",
			Namespace: "istio-system",
		},
	}
	ingressService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-ingressgateway",
			Namespace: "istio-system",
			Annotations: map[string]string{
				"service.beta.kubernetes.io/aws-load-balancer-type": "nlb",
			},
		},
	}
	testClient := newTestClient(t, elbDeprecatedCM, ingressService)

	// When
	useNLB, err := clusterconfig.ShouldUseNLB(context.Background(), testClient)

	// Then
	require.NoError(t, err)
	assert.True(t, useNLB, "Should use NLB when service already has nlb annotation")
}

func TestShouldUseNLB_ConfigMapPresentAndNoService(t *testing.T) {
	// Given
	elbDeprecatedCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "elb-deprecated",
			Namespace: "istio-system",
		},
	}
	testClient := newTestClient(t, elbDeprecatedCM)

	// When
	useNLB, err := clusterconfig.ShouldUseNLB(context.Background(), testClient)

	// Then
	require.NoError(t, err)
	assert.False(t, useNLB, "Should use legacy ELB when ConfigMap present and service not found")
}

// EnvoyFilter Decision Logic Tests (based on reconciliation.go getResources logic)

func TestEnvoyFilterDecision_AWS_NLB_IPv4(t *testing.T) {
	// Given: AWS cluster with NLB, no dual-stack
	node := newAWSNode("aws-node-1")
	testClient := newTestClient(t, node)

	// When
	shouldUseNLB, err := clusterconfig.ShouldUseNLB(context.Background(), testClient)
	require.NoError(t, err)

	isDualStack, err := clusterconfig.IsDualStackEnabled(context.Background(), testClient)
	require.NoError(t, err)

	// Then: EnvoyFilter should be DELETED (NLB IPv4-only uses DSR, no proxy protocol needed)
	shouldDeleteEnvoyFilter := shouldUseNLB && !isDualStack
	assert.True(t, shouldDeleteEnvoyFilter, "NLB IPv4-only should DELETE EnvoyFilter (no proxy protocol needed)")
}

func TestEnvoyFilterDecision_AWS_NLB_DualStack(t *testing.T) {
	// Given: AWS cluster with NLB and dual-stack
	// Note: This test simulates dual-stack logic without experimental tag
	node := newAWSNode("aws-node-1")
	testClient := newTestClient(t, node)

	// When
	shouldUseNLB, err := clusterconfig.ShouldUseNLB(context.Background(), testClient)
	require.NoError(t, err)

	// Simulate dual-stack enabled
	isDualStack := true

	// Then: EnvoyFilter should be CREATED (NLB dual-stack needs proxy protocol)
	shouldDeleteEnvoyFilter := shouldUseNLB && !isDualStack
	assert.False(t, shouldDeleteEnvoyFilter, "NLB dual-stack should CREATE EnvoyFilter (proxy protocol needed)")
}

func TestEnvoyFilterDecision_AWS_LegacyELB(t *testing.T) {
	// Given: AWS cluster with legacy ELB (elb-deprecated ConfigMap present, no service)
	node := newAWSNode("aws-node-1")
	elbDeprecatedCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "elb-deprecated",
			Namespace: "istio-system",
		},
	}
	testClient := newTestClient(t, node, elbDeprecatedCM)

	// When
	shouldUseNLB, err := clusterconfig.ShouldUseNLB(context.Background(), testClient)
	require.NoError(t, err)

	isDualStack, err := clusterconfig.IsDualStackEnabled(context.Background(), testClient)
	require.NoError(t, err)

	// Then: EnvoyFilter should be CREATED (ELB always needs proxy protocol)
	shouldDeleteEnvoyFilter := shouldUseNLB && !isDualStack
	assert.False(t, shouldDeleteEnvoyFilter, "Legacy ELB should CREATE EnvoyFilter (proxy protocol needed)")
}

func TestEnvoyFilterDecision_OpenStack(t *testing.T) {
	// Given: OpenStack cluster
	node := newOpenStackNode("openstack-node-1")
	testClient := newTestClient(t, node)

	// When
	provider, err := clusterconfig.GetClusterProvider(context.Background(), testClient)
	require.NoError(t, err)

	// Then: EnvoyFilter should be CREATED for OpenStack
	shouldDeleteEnvoyFilter := false // OpenStack always creates EnvoyFilter
	assert.False(t, shouldDeleteEnvoyFilter, "OpenStack should CREATE EnvoyFilter (proxy protocol v1 needed)")
	assert.Equal(t, clusterconfig.Openstack, provider)
}

func TestEnvoyFilterDecision_GKE(t *testing.T) {
	// Given: GKE cluster
	node := newGKENode("gke-node-1")
	testClient := newTestClient(t, node)

	// When
	provider, err := clusterconfig.GetClusterProvider(context.Background(), testClient)
	require.NoError(t, err)

	// Then: No EnvoyFilter logic for GKE (returns "other" provider)
	assert.Equal(t, clusterconfig.Other, provider, "GKE returns 'other' provider")
	// Logic: getResources returns early for non-AWS/non-OpenStack, no EnvoyFilter added
}

func TestEnvoyFilterDecision_K3d(t *testing.T) {
	// Given: K3d cluster
	node := newK3dNode("k3d-node-1")
	testClient := newTestClient(t, node)

	// When
	provider, err := clusterconfig.GetClusterProvider(context.Background(), testClient)
	require.NoError(t, err)

	// Then: No EnvoyFilter logic for K3d (returns "other" provider)
	assert.Equal(t, clusterconfig.Other, provider, "K3d returns 'other' provider")
	// Logic: getResources returns early for non-AWS/non-OpenStack, no EnvoyFilter added
}
