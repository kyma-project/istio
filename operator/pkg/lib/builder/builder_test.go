package builder_test

import (
	"errors"
	"testing"

	"github.com/brianvoe/gofakeit"
	"github.com/kyma-project/istio/operator/pkg/lib/builder"
	"github.com/stretchr/testify/require"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
)

const (
	NewNamespaceName string = "new-namespace"
)

func Test_MergeWith(t *testing.T) {
	op := istioOperator.IstioOperator{}
	gofakeit.Struct(&op)
	op.Spec.Namespace = "prevNamespace"

	fake := FakeMergeable{NewNamespaceName: NewNamespaceName}
	builder := builder.NewIstioOperatorBuilder(op)

	operator, err := builder.MergeWith(fake).Get()

	require.NoError(t, err)
	require.Equal(t, operator.Spec.Namespace, NewNamespaceName)
}

func Test_NoBaseOperator(t *testing.T) {
	fake := FakeMergeable{NewNamespaceName: NewNamespaceName}
	builder := builder.NewIstioOperatorBuilder()

	operator, err := builder.MergeWith(fake).Get()

	require.NoError(t, err)
	require.Equal(t, operator.Spec.Namespace, NewNamespaceName)
}

func Test_Error(t *testing.T) {
	fake := FakeMergeable{ThrowError: errors.New("some-error")}
	builder := builder.NewIstioOperatorBuilder()

	operator, err := builder.MergeWith(fake).Get()
	require.Empty(t,operator)
	require.Error(t,err)
}