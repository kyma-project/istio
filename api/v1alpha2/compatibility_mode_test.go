package v1alpha2_test

import (
	. "github.com/onsi/ginkgo/v2" //nolint:revive // Ginkgo tests are generally written without a direct package reference
	. "github.com/onsi/gomega"    //nolint:revive // Gomega asserts are generally written without a direct package reference
	operatorv1alpha1 "istio.io/istio/operator/pkg/apis"
	"istio.io/istio/operator/pkg/values"
	"istio.io/istio/pkg/config/mesh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	istiov1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
)

var _ = Describe("Compatibility Mode", func() {
	Context("Istio Pilot", func() {
		It("should set compatibility variables on Istio Pilot when compatibility mode is on", func() {
			// given
			iop := operatorv1alpha1.IstioOperator{
				Spec: operatorv1alpha1.IstioOperatorSpec{},
			}
			istioCR := istiov1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: istiov1alpha2.IstioSpec{
					CompatibilityMode: true,
				},
			}

			// when
			out, err := istioCR.MergeInto(iop)

			// then
			Expect(err).ShouldNot(HaveOccurred())

			existingEnvs := map[string]string{}
			for _, v := range out.Spec.Components.Pilot.Kubernetes.Env {
				existingEnvs[v.Name] = v.Value
			}

			for k, v := range istiov1alpha2.PilotCompatibilityEnvVars {
				Expect(existingEnvs[k]).To(Equal(v))
			}

		})

		It("should not set compatibility variables on Istio Pilot when compatibility mode is off", func() {
			// given
			iop := operatorv1alpha1.IstioOperator{
				Spec: operatorv1alpha1.IstioOperatorSpec{},
			}
			istioCR := istiov1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: istiov1alpha2.IstioSpec{
					CompatibilityMode: false,
					Components:        &istiov1alpha2.Components{Pilot: &istiov1alpha2.IstioComponent{}},
				},
			}

			// when
			out, err := istioCR.MergeInto(iop)

			// then
			Expect(err).ShouldNot(HaveOccurred())

			variableCounter := 0
			for _, value := range out.Spec.Components.Pilot.Kubernetes.Env {
				if v, ok := istiov1alpha2.PilotCompatibilityEnvVars[value.Name]; ok && value.Value == v {
					variableCounter++
				}
			}

			Expect(variableCounter).To(Equal(0))
		})

		It("should not set compatibility variables on Istio Pilot when compatibility mode is is not configured in IstioCR", func() {
			// given
			iop := operatorv1alpha1.IstioOperator{
				Spec: operatorv1alpha1.IstioOperatorSpec{},
			}
			istioCR := istiov1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: istiov1alpha2.IstioSpec{
					Components: &istiov1alpha2.Components{Pilot: &istiov1alpha2.IstioComponent{}},
				},
			}

			// when
			out, err := istioCR.MergeInto(iop)

			// then
			Expect(err).ShouldNot(HaveOccurred())

			variableCounter := 0
			for _, value := range out.Spec.Components.Pilot.Kubernetes.Env {
				if v, ok := istiov1alpha2.PilotCompatibilityEnvVars[value.Name]; ok && value.Value == v {
					variableCounter++
				}
			}

			Expect(variableCounter).To(Equal(0))
		})
	})
	Context("MeshConfig ProxyMetadata", func() {
		It("should set compatibility variables in proxyMetadata when no meshConfig is defined", func() {
			// given
			iop := operatorv1alpha1.IstioOperator{
				Spec: operatorv1alpha1.IstioOperatorSpec{},
			}
			istioCR := istiov1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: istiov1alpha2.IstioSpec{
					CompatibilityMode: true,
				},
			}

			// when
			out, err := istioCR.MergeInto(iop)

			// then
			Expect(err).ShouldNot(HaveOccurred())

			for fieldName, value := range istiov1alpha2.ProxyMetaDataCompatibility {
				field, exist := getProxyMetadataField(out, fieldName)
				Expect(exist).To(BeTrue())
				Expect(field.(string)).To(Equal(value))
			}
		})

		It("should set compatibility variables in proxyMetadata without overwriting existing variables", func() {
			// given
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

			istioCR := istiov1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: istiov1alpha2.IstioSpec{
					CompatibilityMode: true,
				},
			}

			// when
			out, err := istioCR.MergeInto(iop)

			// then
			Expect(err).ShouldNot(HaveOccurred())

			for fieldName, value := range istiov1alpha2.ProxyMetaDataCompatibility {
				field, exist := getProxyMetadataField(out, fieldName)
				Expect(exist).To(BeTrue())
				Expect(field.(string)).To(Equal(value))
			}
		})

		It("should not set compatibility variables when compatibility mode is off", func() {
			// given
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

			istioCR := istiov1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: istiov1alpha2.IstioSpec{
					CompatibilityMode: false,
				},
			}

			// when
			out, err := istioCR.MergeInto(iop)

			// then
			Expect(err).ShouldNot(HaveOccurred())

			for fieldName := range istiov1alpha2.ProxyMetaDataCompatibility {
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
