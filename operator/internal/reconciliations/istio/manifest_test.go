package istio

import (
	"os"
	"path"
	"testing"

	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

var TestTemplateData TemplateData = TemplateData{
	IstioVersion:   "1.16.1",
	IstioImageBase: "distroless",
}

func Test_merge(t *testing.T) {
	numTrustedProxies := 4
	istioCR := &v1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
		Name:      "istio-test",
		Namespace: "namespace",
	},
		Spec: v1alpha1.IstioSpec{
			Config: v1alpha1.Config{
				NumTrustedProxies: &numTrustedProxies,
			},
		},
	}
	workingDir := "test"

	t.Run("should return error when provided invalid path to default Istio Operator", func(t *testing.T) {
		// given
		istioOperatorPath := "invalid/path.yaml"

		// when
		mergedIstioOperatorPath, err := merge(istioCR, istioOperatorPath, workingDir, TestTemplateData)

		// then
		require.Error(t, err)
		require.Equal(t, "", mergedIstioOperatorPath)
	})

	t.Run("should return error when provided misconfigured default Istio Operator", func(t *testing.T) {
		// given
		istioOperatorPath := "test/wrong-operator.yaml"

		// when
		mergedIstioOperatorPath, err := merge(istioCR, istioOperatorPath, workingDir, TestTemplateData)

		// then
		require.Error(t, err)
		require.Equal(t, "", mergedIstioOperatorPath)
	})

	t.Run("should return merged configuration, when there is a Istio CR with valid configuration and a correct Istio Operator manifest", func(t *testing.T) {
		// given
		istioOperatorPath := "test/test-operator.yaml"

		// when
		mergedIstioOperatorPath, err := merge(istioCR, istioOperatorPath, workingDir, TestTemplateData)

		// then
		require.NoError(t, err)
		require.Equal(t, path.Join(workingDir, mergedIstioOperatorFile), mergedIstioOperatorPath)
		iop := readIOP(t, mergedIstioOperatorPath)
		require.Equal(t, float64(4), iop.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue())
		err = os.Remove(mergedIstioOperatorPath)
		require.NoError(t, err)
	})

	t.Run("should return merged configuration, with IstioVersion and IstioImageBase coming from template", func(t *testing.T) {
		// given
		istioOperatorPath := "test/template-operator.yaml"

		// when
		mergedIstioOperatorPath, err := merge(istioCR, istioOperatorPath, workingDir, TestTemplateData)

		// then
		require.NoError(t, err)
		require.Equal(t, path.Join(workingDir, mergedIstioOperatorFile), mergedIstioOperatorPath)

		iop := readIOP(t, mergedIstioOperatorPath)
		require.Equal(t, "1.16.1-distroless", iop.Spec.Tag.GetStringValue())
		err = os.Remove(mergedIstioOperatorPath)
		require.NoError(t, err)
	})
}

func readIOP(t *testing.T, istioOperatorFilePath string) istioOperator.IstioOperator {
	iop := istioOperator.IstioOperator{}
	manifest, err := os.ReadFile(istioOperatorFilePath)
	require.NoError(t, err)
	err = yaml.Unmarshal(manifest, &iop)
	require.NoError(t, err)

	return iop
}
