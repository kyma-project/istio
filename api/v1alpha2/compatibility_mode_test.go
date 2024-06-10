package v1alpha2

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	operatorv1alpha1 "istio.io/api/operator/v1alpha1"
	iopv1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Compatibility Mode", func() {
	Context("Istio Pilot", func() {
		It("should set compatibility variables on Istio Pilot when compatibility mode is on", func() {
			//given
			iop := iopv1alpha1.IstioOperator{
				Spec: &operatorv1alpha1.IstioOperatorSpec{},
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
			for _, v := range out.Spec.Components.Pilot.K8S.GetEnv() {
				existingEnvs[v.Name] = v.Value
			}

			for k, v := range pilotCompatibilityEnvVars {
				Expect(existingEnvs[k]).To(Equal(v))
			}

		})

		It("should set compatibility variables on Istio Pilot when compatibility mode is on despite disable external name alias annotation set to false", func() {
			//given
			iop := iopv1alpha1.IstioOperator{
				Spec: &operatorv1alpha1.IstioOperatorSpec{},
			}
			istioCR := Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"istio-operator.kyma-project.io/disable-external-name-alias": "false",
					},
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
			for _, v := range out.Spec.Components.Pilot.K8S.GetEnv() {
				existingEnvs[v.Name] = v.Value
			}

			for k, v := range pilotCompatibilityEnvVars {
				Expect(existingEnvs[k]).To(Equal(v))
			}
		})

		It("should not set compatibility variables on Istio Pilot when compatibility mode is off", func() {
			//given
			iop := iopv1alpha1.IstioOperator{
				Spec: &operatorv1alpha1.IstioOperatorSpec{},
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

			variableCounter := 0
			for _, value := range out.Spec.Components.Pilot.K8S.GetEnv() {
				if v, ok := pilotCompatibilityEnvVars[value.Name]; ok && value.Value == v {
					variableCounter++
				}
			}

			Expect(variableCounter).To(Equal(0))
		})

		It("should not set compatibility variables on Istio Pilot when compatibility mode is is not configured in IstioCR", func() {
			//given
			iop := iopv1alpha1.IstioOperator{
				Spec: &operatorv1alpha1.IstioOperatorSpec{},
			}
			istioCR := Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: IstioSpec{},
			}

			// when
			out, err := istioCR.MergeInto(iop)

			//then
			Expect(err).ShouldNot(HaveOccurred())

			variableCounter := 0
			for _, value := range out.Spec.Components.Pilot.K8S.GetEnv() {
				if v, ok := pilotCompatibilityEnvVars[value.Name]; ok && value.Value == v {
					variableCounter++
				}
			}

			Expect(variableCounter).To(Equal(0))
		})
	})
})
