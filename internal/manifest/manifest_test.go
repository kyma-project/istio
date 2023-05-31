package manifest

import (
	"github.com/kyma-project/istio/operator/internal/tests"
	"github.com/onsi/ginkgo/v2/types"
	"os"
	"path"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

var TestTemplateData = TemplateData{
	IstioVersion:   "1.16.1",
	IstioImageBase: "distroless",
}

func TestManifest(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Manifest Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("manifest-suite", report)
})

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
		sut := IstioMerger{
			istioOperatorFilePath: "invalid/path.yaml",
			workingDir:            workingDir,
		}

		// when
		mergedIstioOperatorPath, err := sut.Merge(istioCR, TestTemplateData, clusterconfig.ClusterConfiguration{})

		// then
		Expect(err).Should(HaveOccurred())
		Expect(mergedIstioOperatorPath).To(BeEmpty())
	})

	It("should return error when provided misconfigured default Istio Operator", func() {
		// given
		sut := IstioMerger{
			istioOperatorFilePath: "test/wrong-operator.yaml",
			workingDir:            workingDir,
		}

		// when
		mergedIstioOperatorPath, err := sut.Merge(istioCR, TestTemplateData, clusterconfig.ClusterConfiguration{})

		// then
		Expect(err).Should(HaveOccurred())
		Expect(mergedIstioOperatorPath).To(BeEmpty())
	})

	It("should return merged configuration, when there is a Istio CR with valid configuration and a correct Istio Operator manifest", func() {
		// given
		sut := IstioMerger{
			istioOperatorFilePath: "test/test-operator.yaml",
			workingDir:            workingDir,
		}

		// when
		mergedIstioOperatorPath, err := sut.Merge(istioCR, TestTemplateData, clusterconfig.ClusterConfiguration{})

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
		sut := IstioMerger{
			istioOperatorFilePath: "test/template-operator.yaml",
			workingDir:            workingDir,
		}

		// when
		mergedIstioOperatorPath, err := sut.Merge(istioCR, TestTemplateData, clusterconfig.ClusterConfiguration{})

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mergedIstioOperatorPath).To(Equal(path.Join(workingDir, mergedIstioOperatorFile)))

		iop := readIOP(mergedIstioOperatorPath)
		Expect(iop.Spec.Tag.GetStringValue()).To(Equal("1.16.1-distroless"))
		err = os.Remove(mergedIstioOperatorPath)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should return merged configuration with overrides when provided", func() {
		// given
		newCniBinDirPath := "overriden/path"

		clusterconfig := map[string]interface{}{
			"spec": map[string]interface{}{
				"components": map[string]interface{}{
					"base": map[string]bool{
						"enabled": false,
					},
				},
				"values": map[string]interface{}{
					"cni": map[string]string{
						"cniBinDir": newCniBinDirPath,
					},
				},
			},
		}

		sut := IstioMerger{
			istioOperatorFilePath: "test/test-operator.yaml",
			workingDir:            workingDir,
		}

		// when
		mergedIstioOperatorPath, err := sut.Merge(istioCR, TestTemplateData, clusterconfig)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mergedIstioOperatorPath).To(Equal(path.Join(workingDir, mergedIstioOperatorFile)))

		iop := readIOP(mergedIstioOperatorPath)

		numTrustedProxies := iop.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue()

		Expect(numTrustedProxies).To(Equal(float64(4)))

		baseEnabled := iop.Spec.Components.Base.Enabled.Value
		Expect(baseEnabled).To(BeFalse())

		Expect(iop.Spec.Values.Fields["cni"]).NotTo(BeNil())
		Expect(iop.Spec.Values.Fields["cni"].GetStructValue().Fields["cniBinDir"]).NotTo(BeNil())
		cniBinDir := iop.Spec.Values.Fields["cni"].GetStructValue().Fields["cniBinDir"].GetStringValue()
		Expect(cniBinDir).To(Equal(newCniBinDirPath))

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
