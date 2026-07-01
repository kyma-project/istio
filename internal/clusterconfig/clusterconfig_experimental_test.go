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

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestIsDualStackEnabled_Experimental(t *testing.T) {
	c := createFakeClient(t, createKymaRuntimeConfigWithDualStack(t, true))

	ds, err := clusterconfig.IsDualStackEnabled(context.Background(), c)

	require.NoError(t, err)
	assert.True(t, ds)
}

var awsNLBDualStackConfig = clusterconfig.ClusterConfiguration(map[string]interface{}{
	"spec": map[string]interface{}{
		"values": map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"serviceAnnotations": map[string]string{
						"service.beta.kubernetes.io/aws-load-balancer-scheme":          "internet-facing",
						"service.beta.kubernetes.io/aws-load-balancer-type":            "nlb",
						"service.beta.kubernetes.io/aws-load-balancer-nlb-target-type": "instance",
						"service.beta.kubernetes.io/aws-load-balancer-proxy-protocol":  "*",
					},
				},
			},
		},
	},
})

func TestDiscoverClusterProvider_Experimental(t *testing.T) {
	tests := []struct {
		name    string
		objects []client.Object
		want    clusterconfig.ClusterConfiguration
	}{
		{
			name: "AWS uses NLB when dualstack is enabled and ingressgateway is nil",
			objects: []client.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{Name: "aws-123"},
					Spec:       corev1.NodeSpec{ProviderID: "aws://asdasdads"},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "kyma-provisioning-info", Namespace: "kyma-system"},
					Data: map[string]string{
						"details": `networkDetails:
    dualStackIPEnabled: true`,
					},
				},
			},
			want: awsNLBDualStackConfig,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := createFakeClient(t, tt.objects...)

			str, err := clusterconfig.BuildFactory(context.Background(), c)
			require.NoError(t, err)
			got := clusterconfig.ClusterConfigurationFromFactory(str)

			assert.Equal(t, tt.want, got)
		})
	}
}
