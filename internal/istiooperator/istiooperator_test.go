package istiooperator_test

import (
	meshv1alpha1 "istio.io/api/mesh/v1alpha1"
	"istio.io/istio/operator/pkg/values"
	"istio.io/istio/pkg/util/protomarshal"
	"os"
	"path"
	"testing"

	"github.com/kyma-project/istio/operator/internal/istiooperator"
	"github.com/kyma-project/istio/operator/internal/tests"
	"github.com/onsi/ginkgo/v2/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	iopv1alpha1 "istio.io/istio/operator/pkg/apis"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestManifest(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Manifest Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("istiooperator-suite", report)
})

var _ = Describe("Merge", func() {
	numTrustedProxies := 4
	istioCR := &v1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
		Name:      "istio-test",
		Namespace: "namespace",
	},
		Spec: v1alpha2.IstioSpec{
			Config: v1alpha2.Config{
				NumTrustedProxies: &numTrustedProxies,
			},
		},
	}

	DescribeTable("Merge for different cluster sizes", func(clusterSize clusterconfig.ClusterSize, shouldError bool, igwMinReplicas int) {
		// given
		sut := istiooperator.NewDefaultIstioMerger()

		// when
		mergedIstioOperatorPath, err := sut.Merge(clusterSize, istioCR, clusterconfig.ClusterConfiguration{})

		// then
		if shouldError {
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(Equal("unsupported cluster size"))
			Expect(mergedIstioOperatorPath).To(BeEmpty())
		} else {
			Expect(err).ShouldNot(HaveOccurred())
			Expect(mergedIstioOperatorPath).To(Equal(path.Join("/tmp", istiooperator.MergedIstioOperatorFile)))

			iop := readIOP(mergedIstioOperatorPath)

			var meshConfigTyped meshv1alpha1.MeshConfig

			err = protomarshal.Unmarshal(iop.Spec.MeshConfig, &meshConfigTyped)
			Expect(err).ShouldNot(HaveOccurred())

			numTrustedProxies := meshConfigTyped.DefaultConfig.GetGatewayTopology().GetNumTrustedProxies()
			Expect(numTrustedProxies).To(Equal(numTrustedProxies))

			Expect(*iop.Spec.Components.IngressGateways[0].Kubernetes.HpaSpec.MinReplicas).To(Equal(int32(igwMinReplicas)))

			err = os.Remove(mergedIstioOperatorPath)
			Expect(err).ShouldNot(HaveOccurred())
		}
	},
		Entry("should return error when provided Unknown cluster size", clusterconfig.UnknownSize, true, 0),
		Entry("should return merged configuration for Evaluation cluster size", clusterconfig.Evaluation, false, 1),
		Entry("should return merged configuration for Production cluster size", clusterconfig.Production, false, 3),
	)

	It("should return merged configuration when overrides are provided", func() {
		// given
		newCniBinDirPath := "overriden/path"

		clusterConfig := map[string]interface{}{
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

		sut := istiooperator.NewDefaultIstioMerger()

		// when
		mergedIstioOperatorPath, err := sut.Merge(clusterconfig.Production, istioCR, clusterConfig)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mergedIstioOperatorPath).To(Equal(path.Join("/tmp", istiooperator.MergedIstioOperatorFile)))

		iop := readIOP(mergedIstioOperatorPath)

		var typedMeshConfig meshv1alpha1.MeshConfig

		err = protomarshal.Unmarshal(iop.Spec.MeshConfig, &typedMeshConfig)
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies := typedMeshConfig.DefaultConfig.GetGatewayTopology().GetNumTrustedProxies()

		Expect(numTrustedProxies).To(Equal(uint32(4)))

		baseEnabled := iop.Spec.Components.Base.Enabled.GetValueOrTrue()
		Expect(baseEnabled).To(BeFalse())

		valuesMap, err := values.MapFromObject(iop.Spec.Values)
		Expect(err).ShouldNot(HaveOccurred())

		Expect(valuesMap["cni"]).NotTo(BeNil())
		Expect(valuesMap["cni"].(map[string]interface{})["cniBinDir"]).To(Equal(newCniBinDirPath))

		err = os.Remove(mergedIstioOperatorPath)
		Expect(err).ShouldNot(HaveOccurred())
	})
})

var _ = Describe("NewIstioImageVersionFromTag", func() {
	It("should return IstioImageVersion for a correct semantic version", func() {
		// when
		version, err := istiooperator.NewIstioImageVersionFromTag("1.12.3-blah")

		// then
		Expect(err).Should(Not(HaveOccurred()))
		Expect(version.Version()).Should(Equal("1.12.3"))
		Expect(version.Flavor()).Should(Equal("blah"))
		Expect(version.Tag()).Should(Equal("1.12.3-blah"))
	})

	It("should return error for an incorrect semantic version", func() {
		// when
		version, err := istiooperator.NewIstioImageVersionFromTag("1.2.99.3")

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).Should(ContainSubstring("invalid syntax"))
		Expect(version.Empty()).Should(BeTrue())
	})
})

var _ = Describe("GetIstioImageVersion", func() {
	It("should return Istio version and verify production and evaluation istio operator files have same hub and tag", func() {
		// given
		merger := istiooperator.NewDefaultIstioMerger()

		// when
		imageVersion, err := merger.GetIstioImageVersion()
		Expect(err).Should(Not(HaveOccurred()))

		ioProduction := readIOP("../../internal/istiooperator/istio-operator.yaml")
		Expect(err).Should(Not(HaveOccurred()))

		ioEvaluation := readIOP("../../internal/istiooperator/istio-operator-light.yaml")
		Expect(err).Should(Not(HaveOccurred()))

		// then
		Expect(imageVersion.Tag()).To(Equal(ioProduction.Spec.Tag.(string)))
		Expect(ioProduction.Spec.Hub).To(Equal(ioEvaluation.Spec.Hub))
		Expect(ioProduction.Spec.Tag.(string)).To(Equal(ioEvaluation.Spec.Tag.(string)))
	})
})

func readIOP(iopv1alpha1FilePath string) iopv1alpha1.IstioOperator {
	iop := iopv1alpha1.IstioOperator{}

	istioOpertor, err := os.ReadFile(iopv1alpha1FilePath)
	Expect(err).ShouldNot(HaveOccurred())

	err = yaml.Unmarshal(istioOpertor, &iop)
	Expect(err).ShouldNot(HaveOccurred())

	return iop
}
