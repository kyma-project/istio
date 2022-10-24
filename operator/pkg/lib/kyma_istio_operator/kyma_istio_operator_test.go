package kymaistiooperator

import (
	"testing"

	"github.com/brianvoe/gofakeit/v6"

	"github.com/stretchr/testify/require"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
)

func Test_Merge(t *testing.T) {
	someKymaOperator := KymaIstioOperator{}
	err := gofakeit.Struct(&someKymaOperator)
	if err != nil {
		t.Fatal(err)
	}

	someKymaOperator.MeshConfig.DefaultConfig.GatewayTopology.NumTrustedProxies=5
	someKymaOperator.MeshConfig.AccessLogEncoding=""

	someIstioOperator := istioOperator.IstioOperator{}
	err = gofakeit.Struct(&someIstioOperator)
	if err != nil {
		t.Fatal(err)
	}

	prev_access_log_encoding := someIstioOperator.Spec.MeshConfig.Fields["accessLogEncoding"].GetStringValue()

	out, err := someKymaOperator.Merge(someIstioOperator)

	require.NoError(t, err)

	require.Equal(t, prev_access_log_encoding, out.Spec.MeshConfig.Fields["accessLogEncoding"].GetStringValue())
	require.Equal(t, someKymaOperator.MeshConfig.AccessLogFile, out.Spec.MeshConfig.Fields["accessLogFile"].GetStringValue())
	require.NotNil(t, out.Spec.MeshConfig.Fields["defaultConfig"])
	require.Equal(t, someKymaOperator.MeshConfig.DefaultConfig.GatewayTopology.NumTrustedProxies, 
		uint(out.Spec.MeshConfig.Fields["defaultConfig"].GetStructValue().
	Fields["gatewayTopology"].GetStructValue().
	Fields["numTrustedProxies"].GetNumberValue()))
}
