package istio_test

import (
	"fmt"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	mockIstioTag             string = "1.16.1-distroless"
	lastAppliedConfiguration string = "operator.kyma-project.io/lastAppliedConfiguration"
)

var _ = Describe("CR configuration", func() {
	Context("UpdateLastAppliedConfiguration", func() {
		It("should update CR with IstioVersion and spec of CR", func() {
			// given
			cr := operatorv1alpha1.Istio{}

			// when
			updatedCR, err := istio.UpdateLastAppliedConfiguration(cr, mockIstioTag)

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(updatedCR.Annotations).To(Not(BeEmpty()))

			Expect(updatedCR.Annotations[lastAppliedConfiguration]).To(Equal(fmt.Sprintf(`{"config":{},"IstioTag":"%s"}`, mockIstioTag)))
		})
	})

	Context("ConfigurationChanged", func() {
		Context("Istio version doesn't change", func() {
			It("should return Create if \"operator.kyma-project.io/lastAppliedConfiguration\" annotation is not present on CR", func() {
				// given
				cr := operatorv1alpha1.Istio{}

				// when
				changed, err := istio.EvaluateIstioCRChanges(cr, mockIstioTag)

				// then
				Expect(err).ShouldNot(HaveOccurred())
				Expect(changed).To(Equal(istio.Create))
			})

			It("should return ConfigurationUpdate if lastAppliedConfiguration has different number of numTrustedProxies than CR", func() {
				// given
				newNumTrustedProxies := 2
				cr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						lastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":1},"IstioTag":"%s"}`, mockIstioTag),
					},
				},
					Spec: operatorv1alpha1.IstioSpec{
						Config: operatorv1alpha1.Config{
							NumTrustedProxies: &newNumTrustedProxies,
						},
					},
				}

				// when
				changed, err := istio.EvaluateIstioCRChanges(cr, mockIstioTag)

				// then
				Expect(err).ShouldNot(HaveOccurred())
				Expect(changed).To(Equal(istio.ConfigurationUpdate))
			})

			It("should return ConfigurationUpdate if lastAppliedConfiguration has \"nil\" number of numTrustedProxies and CR doesn'y", func() {
				// given
				newNumTrustedProxies := 2
				cr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						lastAppliedConfiguration: fmt.Sprintf(`{"config":{},"IstioTag":"%s"}`, mockIstioTag),
					},
				},
					Spec: operatorv1alpha1.IstioSpec{
						Config: operatorv1alpha1.Config{
							NumTrustedProxies: &newNumTrustedProxies,
						},
					},
				}

				// when
				changed, err := istio.EvaluateIstioCRChanges(cr, mockIstioTag)

				// then
				Expect(err).ShouldNot(HaveOccurred())
				Expect(changed).To(Equal(istio.ConfigurationUpdate))
			})

			It("should return ConfigurationUpdate if lastAppliedConfiguration has any number of numTrustedProxies and CR has \"nil\"", func() {
				// given
				cr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						lastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":1},"IstioTag":"%s"}`, mockIstioTag),
					},
				},
					Spec: operatorv1alpha1.IstioSpec{
						Config: operatorv1alpha1.Config{
							NumTrustedProxies: nil,
						},
					},
				}

				// when
				changed, err := istio.EvaluateIstioCRChanges(cr, mockIstioTag)

				// then
				Expect(err).ShouldNot(HaveOccurred())
				Expect(changed).To(Equal(istio.ConfigurationUpdate))
			})

			It("should return NoChange if both configurations have \"nil\" numTrustedProxies", func() {
				// given
				cr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						lastAppliedConfiguration: fmt.Sprintf(`{"config":{},"IstioTag":"%s"}`, mockIstioTag),
					},
				},
					Spec: operatorv1alpha1.IstioSpec{
						Config: operatorv1alpha1.Config{
							NumTrustedProxies: nil,
						},
					},
				}

				// when
				changed, err := istio.EvaluateIstioCRChanges(cr, mockIstioTag)

				// then
				Expect(err).ShouldNot(HaveOccurred())
				Expect(changed).To(Equal(istio.NoChange))
			})

			It("should return NoChange if lastAppliedConfiguration has the same number of numTrustedProxies as CR", func() {
				// given
				newNumTrustedProxies := 1
				cr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						lastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":1},"IstioTag":"%s"}`, mockIstioTag),
					},
				},
					Spec: operatorv1alpha1.IstioSpec{
						Config: operatorv1alpha1.Config{
							NumTrustedProxies: &newNumTrustedProxies,
						},
					},
				}

				// when
				changed, err := istio.EvaluateIstioCRChanges(cr, mockIstioTag)

				// then
				Expect(err).ShouldNot(HaveOccurred())
				Expect(changed).To(Equal(istio.NoChange))
			})
		})
		Context("Istio version changes", func() {
			It("should return VersionUpdate if IstioVersion in annotation is different than in CR and configuration didn't change", func() {
				// given
				cr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						lastAppliedConfiguration: fmt.Sprintf(`{"config":{},"IstioTag":"%s"}`, mockIstioTag),
					},
				}}

				// when
				changed, err := istio.EvaluateIstioCRChanges(cr, "1.16.3-distroless")

				// then
				Expect(err).ShouldNot(HaveOccurred())
				Expect(changed).To(Equal(istio.VersionUpdate))
			})

			It("should return VersionUpdate if IstioVersion in annotation is different than in CR and configuration changed", func() {
				// given
				cr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						lastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":1},"IstioTag":"%s"}`, mockIstioTag),
					},
				}}

				// when
				changed, err := istio.EvaluateIstioCRChanges(cr, "1.16.3-distroless")

				// then
				Expect(err).ShouldNot(HaveOccurred())
				Expect(changed).To(Equal(istio.VersionUpdate | istio.ConfigurationUpdate))
			})
		})
	})
})
