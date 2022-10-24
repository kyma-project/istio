package builder_test

import (
	"log"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/kyma-project/istio/lib/pkg/builder"
	kymaistiooperator "github.com/kyma-project/istio/operator/pkg/lib/kyma_istio_operator"
	"github.com/stretchr/testify/require"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
)

func Test_MergeWith(t *testing.T) {
	someKymaOperator := kymaistiooperator.KymaIstioOperator{}
	gofakeit.Struct(&someKymaOperator)

	someIstioOperator := istioOperator.IstioOperator{}
	gofakeit.Struct(&someIstioOperator)

	operatorBuilder := builder.NewIstioOperatorBuilder(log.Default(), someIstioOperator)
	operatorBuilder.MergeWith(someKymaOperator)

	out := operatorBuilder.Get()

	require.Equal(t, someKymaOperator.MeshConfig.AccessLogEncoding, out.Spec.MeshConfig.Fields["accessLogEncoding"].GetStringValue())
}
