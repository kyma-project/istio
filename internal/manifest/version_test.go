package manifest

import (
	"fmt"
	"os"

	"github.com/coreos/go-semver/semver"
	"google.golang.org/protobuf/types/known/structpb"
	iopv1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"sigs.k8s.io/yaml"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetModuleVersion", func() {
	It("should return version from package variable", func() {
		Expect(GetModuleVersion()).To(Equal(version))
	})
})

var _ = Describe("GetIstioVersion", func() {
	merger := NewDefaultIstioMerger()

	It("should return Istio version from tag in production manifest file", func() {
		// given

		// when
		version, prerelease, err := GetIstioVersion(&merger)
		Expect(err).Should(Not(HaveOccurred()))

		iop, err := merger.GetIstioOperator(clusterconfig.Production)
		Expect(err).Should(Not(HaveOccurred()))

		prodVersion, err := semver.NewVersion(iop.Spec.Tag.GetStringValue())
		Expect(err).Should(Not(HaveOccurred()))

		// then
		Expect(version).To(Not(BeEmpty()))
		Expect(prerelease).To(Not(BeEmpty()))
		Expect(version).To(Equal(fmt.Sprintf("%d.%d.%d", prodVersion.Major, prodVersion.Minor, prodVersion.Patch)))
		Expect(prerelease).To(Equal(string(prodVersion.PreRelease)))
	})

	It("should have same version in evaluation and production manifest files", func() {
		// given

		// when
		prodIOP, err := merger.GetIstioOperator(clusterconfig.Production)
		Expect(err).Should(Not(HaveOccurred()))

		prodVersion, err := semver.NewVersion(prodIOP.Spec.Tag.GetStringValue())
		Expect(err).Should(Not(HaveOccurred()))

		evalIOP, err := merger.GetIstioOperator(clusterconfig.Evaluation)
		Expect(err).Should(Not(HaveOccurred()))

		evalVersion, err := semver.NewVersion(evalIOP.Spec.Tag.GetStringValue())
		Expect(err).Should(Not(HaveOccurred()))

		// then
		Expect(prodVersion.Equal(*evalVersion)).To(BeTrue())
	})
})

type MergerMock struct {
	tag string
}

func (m MergerMock) Merge(_ clusterconfig.ClusterSize, _ *operatorv1alpha2.Istio, _ clusterconfig.ClusterConfiguration) (string, error) {
	return "mocked istio operator merge result", nil
}

func (m MergerMock) GetIstioOperator(_ clusterconfig.ClusterSize) (iopv1alpha1.IstioOperator, error) {
	iop := iopv1alpha1.IstioOperator{}
	manifest, err := os.ReadFile("istio-operator.yaml")
	if err == nil {
		err = yaml.Unmarshal(manifest, &iop)
	}
	iop.Spec.Tag = structpb.NewStringValue(m.tag)
	return iop, err
}
