package v1alpha1

import (
	"github.com/kyma-project/istio/operator/internal/tests"
	"github.com/onsi/ginkgo/v2/types"
	operatorv1alpha1 "istio.io/api/operator/v1alpha1"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"istio.io/istio/pkg/config/mesh"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"google.golang.org/protobuf/types/known/structpb"
	"istio.io/api/mesh/v1alpha1"
	"istio.io/istio/pkg/util/protomarshal"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Merge Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("merge-api-suite", report)
})

var _ = Describe("Merge", func() {

	It("Should update numTrustedProxies on IstioOperator from 1 to 5", func() {
		// given
		m := mesh.DefaultMeshConfig()
		m.DefaultConfig.GatewayTopology = &v1alpha1.Topology{NumTrustedProxies: 1}
		meshConfig := convert(m)

		iop := istioOperator.IstioOperator{
			Spec: &operatorv1alpha1.IstioOperatorSpec{
				MeshConfig: meshConfig,
			},
		}

		numProxies := 5
		istioCR := Istio{Spec: IstioSpec{Config: Config{NumTrustedProxies: &numProxies}}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies := out.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue()
		Expect(numTrustedProxies).To(Equal(float64(5)))
	})

	It("Should set numTrustedProxies on IstioOperator to 5 when no GatewayTopology is configured", func() {
		// given
		m := mesh.DefaultMeshConfig()
		meshConfig := convert(m)

		iop := istioOperator.IstioOperator{
			Spec: &operatorv1alpha1.IstioOperatorSpec{
				MeshConfig: meshConfig,
			},
		}

		numProxies := 5

		istioCR := Istio{Spec: IstioSpec{Config: Config{NumTrustedProxies: &numProxies}}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies := out.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue()
		Expect(numTrustedProxies).To(Equal(float64(numProxies)))
	})

	It("Should set numTrustedProxies on IstioOperator to 5 when IstioOperator has nil spec", func() {
		// given
		iop := istioOperator.IstioOperator{
			Spec: nil,
		}

		numProxies := 5

		istioCR := Istio{Spec: IstioSpec{Config: Config{NumTrustedProxies: &numProxies}}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies := out.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue()
		Expect(numTrustedProxies).To(Equal(float64(numProxies)))
	})

	It("Should set numTrustedProxies on IstioOperator to 5 when IstioOperator has nil mesh config", func() {
		// given
		iop := istioOperator.IstioOperator{
			Spec: &operatorv1alpha1.IstioOperatorSpec{
				MeshConfig: nil,
			},
		}

		numProxies := 5

		istioCR := Istio{Spec: IstioSpec{Config: Config{NumTrustedProxies: &numProxies}}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies := out.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue()
		Expect(numTrustedProxies).To(Equal(float64(numProxies)))
	})

	It("Should change nothing if config is empty", func() {
		// given
		m := mesh.DefaultMeshConfig()
		m.DefaultConfig.GatewayTopology = &v1alpha1.Topology{NumTrustedProxies: 1}
		meshConfig := convert(m)

		iop := istioOperator.IstioOperator{
			Spec: &operatorv1alpha1.IstioOperatorSpec{
				MeshConfig: meshConfig,
			},
		}

		istioCR := Istio{Spec: IstioSpec{}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies := out.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue()
		Expect(numTrustedProxies).To(Equal(float64(1)))
	})
	It("Should set numTrustedProxies on IstioOperator to 5 when there is no defaultConfig in meshConfig", func() {
		// given
		m := &v1alpha1.MeshConfig{
			EnableTracing: true,
		}
		meshConfig := convert(m)

		iop := istioOperator.IstioOperator{
			Spec: &operatorv1alpha1.IstioOperatorSpec{
				MeshConfig: meshConfig,
			},
		}
		numProxies := 5

		istioCR := Istio{Spec: IstioSpec{Config: Config{NumTrustedProxies: &numProxies}}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies := out.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue()
		Expect(numTrustedProxies).To(Equal(float64(5)))
	})
})

func convert(a *v1alpha1.MeshConfig) *structpb.Struct {

	mMap, err := protomarshal.ToJSONMap(a)
	Expect(err).ShouldNot(HaveOccurred())

	mStruct, err := structpb.NewStruct(mMap)
	Expect(err).ShouldNot(HaveOccurred())

	return mStruct
}
