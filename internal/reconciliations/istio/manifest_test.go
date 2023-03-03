package istio

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
	"path"

	"github.com/kyma-project/istio/operator/api/v1alpha1"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

var TestTemplateData = TemplateData{
	IstioVersion:   "1.16.1",
	IstioImageBase: "distroless",
}

var _ = Describe("Manifest merge", func() {

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

	It("should return error when provided invalid path to default Istio Operator", func() {
		// given
		istioOperatorPath := "invalid/path.yaml"

		// when
		mergedIstioOperatorPath, err := merge(istioCR, istioOperatorPath, workingDir, TestTemplateData)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(mergedIstioOperatorPath).To(BeEmpty())
	})

	It("should return error when provided misconfigured default Istio Operator", func() {
		// given
		istioOperatorPath := "test/wrong-operator.yaml"

		// when
		mergedIstioOperatorPath, err := merge(istioCR, istioOperatorPath, workingDir, TestTemplateData)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(mergedIstioOperatorPath).To(BeEmpty())
	})

	It("should return merged configuration, when there is a Istio CR with valid configuration and a correct Istio Operator manifest", func() {
		// given
		istioOperatorPath := "test/test-operator.yaml"

		// when
		mergedIstioOperatorPath, err := merge(istioCR, istioOperatorPath, workingDir, TestTemplateData)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mergedIstioOperatorPath).To(Equal(path.Join(workingDir, mergedIstioOperatorFile)))

		iop := readIOP(mergedIstioOperatorPath)

		numTrustedProxies := iop.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue()

		Expect(numTrustedProxies).To(Equal(float64(4)))
		err = os.Remove(mergedIstioOperatorPath)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should return merged configuration, with IstioVersion and IstioImageBase coming from template", func() {
		// given
		istioOperatorPath := "test/template-operator.yaml"

		// when
		mergedIstioOperatorPath, err := merge(istioCR, istioOperatorPath, workingDir, TestTemplateData)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mergedIstioOperatorPath).To(Equal(path.Join(workingDir, mergedIstioOperatorFile)))

		iop := readIOP(mergedIstioOperatorPath)
		Expect(iop.Spec.Tag.GetStringValue()).To(Equal("1.16.1-distroless"))
		err = os.Remove(mergedIstioOperatorPath)
		Expect(err).ShouldNot(HaveOccurred())
	})
})

func readIOP(istioOperatorFilePath string) istioOperator.IstioOperator {
	iop := istioOperator.IstioOperator{}
	manifest, err := os.ReadFile(istioOperatorFilePath)
	Expect(err).ShouldNot(HaveOccurred())
	err = yaml.Unmarshal(manifest, &iop)
	Expect(err).ShouldNot(HaveOccurred())

	return iop
}
