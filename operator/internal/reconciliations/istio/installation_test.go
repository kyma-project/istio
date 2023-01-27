package istio

import (
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("IstioInstallation", func() {
	Context("configurationChanged", func() {
		It("should return true if \"operator.kyma-project.io/lastAppliedConfiguration\" annotation is not present on CR", func() {
			// given
			cr := operatorv1alpha1.Istio{}

			// when
			changed, err := configurationChanged(cr)

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(changed).To(BeTrue())
		})

		It("should return true if lastAppliedConfiguration has different number of numTrustedProxies than CR", func() {
			// given
			newNumTrustedProxies := 2
			cr := operatorv1alpha1.Istio{ObjectMeta: v1.ObjectMeta{
				Annotations: map[string]string{
					LastAppliedConfiguration: `{"config":{"numTrustedProxies":1}}`,
				},
			},
				Spec: operatorv1alpha1.IstioSpec{
					Config: operatorv1alpha1.Config{
						NumTrustedProxies: &newNumTrustedProxies,
					},
				},
			}

			// when
			changed, err := configurationChanged(cr)

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(changed).To(BeTrue())
		})

		It("should return true if lastAppliedConfiguration has \"nil\" number of numTrustedProxies and CR doesn'y", func() {
			// given
			newNumTrustedProxies := 2
			cr := operatorv1alpha1.Istio{ObjectMeta: v1.ObjectMeta{
				Annotations: map[string]string{
					LastAppliedConfiguration: `{"config":{}}`,
				},
			},
				Spec: operatorv1alpha1.IstioSpec{
					Config: operatorv1alpha1.Config{
						NumTrustedProxies: &newNumTrustedProxies,
					},
				},
			}

			// when
			changed, err := configurationChanged(cr)

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(changed).To(BeTrue())
		})

		It("should return true if lastAppliedConfiguration has any number of numTrustedProxies and CR has \"nil\"", func() {
			// given
			cr := operatorv1alpha1.Istio{ObjectMeta: v1.ObjectMeta{
				Annotations: map[string]string{
					LastAppliedConfiguration: `{"config":{"numTrustedProxies":1}}`,
				},
			},
				Spec: operatorv1alpha1.IstioSpec{
					Config: operatorv1alpha1.Config{
						NumTrustedProxies: nil,
					},
				},
			}

			// when
			changed, err := configurationChanged(cr)

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(changed).To(BeTrue())
		})

		It("should return false if both configurations have \"nil\" numTrustedProxies", func() {
			// given
			cr := operatorv1alpha1.Istio{ObjectMeta: v1.ObjectMeta{
				Annotations: map[string]string{
					LastAppliedConfiguration: `{"config":{}}`,
				},
			},
				Spec: operatorv1alpha1.IstioSpec{
					Config: operatorv1alpha1.Config{
						NumTrustedProxies: nil,
					},
				},
			}

			// when
			changed, err := configurationChanged(cr)

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(changed).To(BeFalse())
		})

		It("should return false if lastAppliedConfiguration has the same number of numTrustedProxies as CR", func() {
			// given
			newNumTrustedProxies := 1
			cr := operatorv1alpha1.Istio{ObjectMeta: v1.ObjectMeta{
				Annotations: map[string]string{
					LastAppliedConfiguration: `{"config":{"numTrustedProxies":1}}`,
				},
			},
				Spec: operatorv1alpha1.IstioSpec{
					Config: operatorv1alpha1.Config{
						NumTrustedProxies: &newNumTrustedProxies,
					},
				},
			}

			// when
			changed, err := configurationChanged(cr)

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(changed).To(BeFalse())
		})
	})
})
