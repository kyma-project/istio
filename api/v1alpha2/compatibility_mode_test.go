package v1alpha2

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	operatorv1alpha1 "istio.io/istio/operator/pkg/apis"
	"istio.io/istio/operator/pkg/values"
	"istio.io/istio/pkg/config/mesh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Compatibility Mode", func() {
	Context("Istio Pilot", func() {
		It("should set compatibility variables on Istio Pilot when compatibility mode is on", func() {
			//given
			iop := operatorv1alpha1.IstioOperator{
				Spec: operatorv1alpha1.IstioOperatorSpec{},
			}
			istioCR := Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: IstioSpec{
					CompatibilityMode: true,
				},
			}

			// when
			out, err := istioCR.MergeInto(iop)

			//then
			Expect(err).ShouldNot(HaveOccurred())

			existingEnvs := map[string]string{}
			for _, v := range out.Spec.Components.Pilot.Kubernetes.Env {
				existingEnvs[v.Name] = v.Value
			}

			for k, v := range pilotCompatibilityEnvVars {
				Expect(existingEnvs[k]).To(Equal(v))
			}

		})

		It("should not set compatibility variables on Istio Pilot when compatibility mode is off", func() {
			//given
			iop := operatorv1alpha1.IstioOperator{
				Spec: operatorv1alpha1.IstioOperatorSpec{},
			}
			istioCR := Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: IstioSpec{
					CompatibilityMode: false,
					Components:        &Components{Pilot: &IstioComponent{}},
				},
			}

			// when
			out, err := istioCR.MergeInto(iop)

			//then
			Expect(err).ShouldNot(HaveOccurred())

			variableCounter := 0
			for _, value := range out.Spec.Components.Pilot.Kubernetes.Env {
				if v, ok := pilotCompatibilityEnvVars[value.Name]; ok && value.Value == v {
					variableCounter++
				}
			}

			Expect(variableCounter).To(Equal(0))
		})

		It("should not set compatibility variables on Istio Pilot when compatibility mode is is not configured in IstioCR", func() {
			//given
			iop := operatorv1alpha1.IstioOperator{
				Spec: operatorv1alpha1.IstioOperatorSpec{},
			}
			istioCR := Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: IstioSpec{
					Components: &Components{Pilot: &IstioComponent{}},
				},
			}

			// when
			out, err := istioCR.MergeInto(iop)

			//then
			Expect(err).ShouldNot(HaveOccurred())

			variableCounter := 0
			for _, value := range out.Spec.Components.Pilot.Kubernetes.Env {
				if v, ok := pilotCompatibilityEnvVars[value.Name]; ok && value.Value == v {
					variableCounter++
				}
			}

			Expect(variableCounter).To(Equal(0))
		})
	})
	Context("MeshConfig ProxyMetadata", func() {
		It("should set compatibility variables in proxyMetadata when no meshConfig is defined", func() {
			//given
			iop := operatorv1alpha1.IstioOperator{
				Spec: operatorv1alpha1.IstioOperatorSpec{},
			}
			istioCR := Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: IstioSpec{
					CompatibilityMode: true,
				},
			}

			// when
			out, err := istioCR.MergeInto(iop)

			//then
			Expect(err).ShouldNot(HaveOccurred())

			for fieldName, value := range ProxyMetaDataCompatibility {
				field, exist := getProxyMetadataField(out, fieldName)
				Expect(exist).To(BeTrue())
				Expect(field.(string)).To(Equal(value))
			}
		})

		It("should set compatibility variables in proxyMetadata without overwriting existing variables", func() {
			//given
			m := mesh.DefaultMeshConfig()
			m.DefaultConfig.ProxyMetadata = map[string]string{
				"BOOTSTRAP_XDS_AGENT": "true",
			}

			meshConfig := convert(m)

			iop := operatorv1alpha1.IstioOperator{
				Spec: operatorv1alpha1.IstioOperatorSpec{
					MeshConfig: meshConfig,
				},
			}

			istioCR := Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: IstioSpec{
					CompatibilityMode: true,
				},
			}

			// when
			out, err := istioCR.MergeInto(iop)

			//then
			Expect(err).ShouldNot(HaveOccurred())

			for fieldName, value := range ProxyMetaDataCompatibility {
				field, exist := getProxyMetadataField(out, fieldName)
				Expect(exist).To(BeTrue())
				Expect(field.(string)).To(Equal(value))
			}
		})

		It("should not set compatibility variables when compatibility mode is off", func() {
			//given
			m := mesh.DefaultMeshConfig()
			m.DefaultConfig.ProxyMetadata = map[string]string{
				"BOOTSTRAP_XDS_AGENT": "true",
			}

			meshConfig := convert(m)

			iop := operatorv1alpha1.IstioOperator{
				Spec: operatorv1alpha1.IstioOperatorSpec{
					MeshConfig: meshConfig,
				},
			}

			istioCR := Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: IstioSpec{
					CompatibilityMode: false,
				},
			}

			// when
			out, err := istioCR.MergeInto(iop)

			//then
			Expect(err).ShouldNot(HaveOccurred())

			for fieldName, _ := range ProxyMetaDataCompatibility {
				_, exist := getProxyMetadataField(out, fieldName)
				Expect(exist).To(BeFalse())
			}
		})
	})
})

func getProxyMetadataField(iop operatorv1alpha1.IstioOperator, fieldName string) (any, bool) {
	mapMeshConfig, err := values.MapFromObject(iop.Spec.MeshConfig)
	Expect(err).ShouldNot(HaveOccurred())
	return mapMeshConfig.GetPath("defaultConfig.proxyMetadata." + fieldName)
}
