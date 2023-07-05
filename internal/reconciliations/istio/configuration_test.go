package istio_test

import (
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/util/intstr"

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

			It("should return an error if lastAppliedConfiguration cannot be unmarshalled", func() {
				// given
				newNumTrustedProxies := 2
				cr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						lastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":"a"},"IstioTag":"%s"}`, mockIstioTag),
					},
				},
					Spec: operatorv1alpha1.IstioSpec{
						Config: operatorv1alpha1.Config{
							NumTrustedProxies: &newNumTrustedProxies,
						},
					},
				}

				// when
				_, err := istio.EvaluateIstioCRChanges(cr, mockIstioTag)

				// then
				Expect(err).Should(HaveOccurred())
				Expect(err.Description()).To(Equal("Could not unmarshal last applied configuration: json: cannot unmarshal string into Go struct field Config.config.numTrustedProxies of type int"))
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
		Context("Istio component configuration changes", func() {
			DescribeTable("Component configuration table", func(a, b operatorv1alpha1.Istio, expectedChange istio.IstioCRChange) {
				type appliedConfig struct {
					operatorv1alpha1.IstioSpec
					IstioTag string
				}

				if b.Annotations == nil {
					b.Annotations = make(map[string]string)
				}

				newAppliedConfig := appliedConfig{
					IstioSpec: a.Spec,
					IstioTag:  mockIstioTag,
				}

				config, err := json.Marshal(newAppliedConfig)
				Expect(err).ToNot(HaveOccurred())

				b.Annotations[lastAppliedConfiguration] = string(config)

				change, err := istio.EvaluateIstioCRChanges(b, mockIstioTag)
				Expect(err).ToNot(HaveOccurred())
				Expect(change).To(Equal(expectedChange))
			},
				Entry("When a field changes value should return ConfigurationChanged", operatorv1alpha1.Istio{Spec: operatorv1alpha1.IstioSpec{
					Components: &operatorv1alpha1.Components{
						Pilot: &operatorv1alpha1.IstioComponent{K8s: &operatorv1alpha1.KubernetesResourcesConfig{
							HPASpec: &operatorv1alpha1.HPASpec{
								MaxReplicas: newInt32WithValue(5),
							},
						}},
					},
				}}, operatorv1alpha1.Istio{Spec: operatorv1alpha1.IstioSpec{
					Components: &operatorv1alpha1.Components{
						Pilot: &operatorv1alpha1.IstioComponent{K8s: &operatorv1alpha1.KubernetesResourcesConfig{
							HPASpec: &operatorv1alpha1.HPASpec{
								MaxReplicas: newInt32WithValue(21),
							},
						}},
					},
				}}, istio.ConfigurationUpdate),

				Entry("When a field changes from nil to not nil should return ConfigurationChanged", operatorv1alpha1.Istio{Spec: operatorv1alpha1.IstioSpec{
					Components: &operatorv1alpha1.Components{
						Pilot: &operatorv1alpha1.IstioComponent{K8s: &operatorv1alpha1.KubernetesResourcesConfig{
							HPASpec: &operatorv1alpha1.HPASpec{
								MaxReplicas: nil,
							},
						}},
					},
				}}, operatorv1alpha1.Istio{Spec: operatorv1alpha1.IstioSpec{
					Components: &operatorv1alpha1.Components{
						Pilot: &operatorv1alpha1.IstioComponent{K8s: &operatorv1alpha1.KubernetesResourcesConfig{
							HPASpec: &operatorv1alpha1.HPASpec{
								MaxReplicas: newInt32WithValue(21),
							},
						}},
					},
				}}, istio.ConfigurationUpdate),

				Entry("When a field changes from not nil to nil should return ConfigurationChanged", operatorv1alpha1.Istio{Spec: operatorv1alpha1.IstioSpec{
					Components: &operatorv1alpha1.Components{
						Pilot: &operatorv1alpha1.IstioComponent{K8s: &operatorv1alpha1.KubernetesResourcesConfig{
							HPASpec: &operatorv1alpha1.HPASpec{
								MaxReplicas: newInt32WithValue(21),
							},
						}},
					},
				}}, operatorv1alpha1.Istio{Spec: operatorv1alpha1.IstioSpec{
					Components: &operatorv1alpha1.Components{
						Pilot: &operatorv1alpha1.IstioComponent{K8s: &operatorv1alpha1.KubernetesResourcesConfig{
							HPASpec: &operatorv1alpha1.HPASpec{
								MaxReplicas: nil,
							},
						}},
					},
				}}, istio.ConfigurationUpdate),

				Entry("When resources config changes should return ConfigurationChanged", operatorv1alpha1.Istio{Spec: operatorv1alpha1.IstioSpec{
					Components: &operatorv1alpha1.Components{
						Pilot: &operatorv1alpha1.IstioComponent{K8s: &operatorv1alpha1.KubernetesResourcesConfig{
							Resources: &operatorv1alpha1.Resources{
								Requests: &operatorv1alpha1.ResourceClaims{
									Cpu: newStringWithValue("100m"),
								},
							},
						}},
					},
				}}, operatorv1alpha1.Istio{Spec: operatorv1alpha1.IstioSpec{
					Components: &operatorv1alpha1.Components{
						Pilot: &operatorv1alpha1.IstioComponent{K8s: &operatorv1alpha1.KubernetesResourcesConfig{
							Resources: &operatorv1alpha1.Resources{
								Requests: &operatorv1alpha1.ResourceClaims{
									Cpu: newStringWithValue("10m"),
								},
							},
						}},
					},
				}}, istio.ConfigurationUpdate),

				Entry("When strategy config changes should return ConfigurationChanged", operatorv1alpha1.Istio{Spec: operatorv1alpha1.IstioSpec{
					Components: &operatorv1alpha1.Components{
						Pilot: &operatorv1alpha1.IstioComponent{K8s: &operatorv1alpha1.KubernetesResourcesConfig{
							Strategy: &operatorv1alpha1.Strategy{RollingUpdate: &operatorv1alpha1.RollingUpdate{
								MaxSurge: &intstr.IntOrString{
									Type:   intstr.Int,
									IntVal: 1,
								},
								MaxUnavailable: &intstr.IntOrString{
									Type:   intstr.String,
									StrVal: "50%",
								},
							}},
						}},
					},
				}}, operatorv1alpha1.Istio{Spec: operatorv1alpha1.IstioSpec{
					Components: &operatorv1alpha1.Components{
						Pilot: &operatorv1alpha1.IstioComponent{K8s: &operatorv1alpha1.KubernetesResourcesConfig{
							Strategy: &operatorv1alpha1.Strategy{RollingUpdate: &operatorv1alpha1.RollingUpdate{
								MaxSurge: &intstr.IntOrString{
									Type:   intstr.Int,
									IntVal: 1,
								},
								MaxUnavailable: &intstr.IntOrString{
									Type:   intstr.Int,
									IntVal: 20,
								},
							}},
						}},
					},
				}}, istio.ConfigurationUpdate),

				Entry("When ingress gateway configuration changed should return ConfigurationUpdate", operatorv1alpha1.Istio{Spec: operatorv1alpha1.IstioSpec{
					Components: &operatorv1alpha1.Components{
						IngressGateway: &operatorv1alpha1.IstioComponent{
							K8s: &operatorv1alpha1.KubernetesResourcesConfig{
								HPASpec: &operatorv1alpha1.HPASpec{
									MaxReplicas: newInt32WithValue(1),
								},
							},
						},
					},
				}}, operatorv1alpha1.Istio{Spec: operatorv1alpha1.IstioSpec{
					Components: &operatorv1alpha1.Components{
						IngressGateway: &operatorv1alpha1.IstioComponent{
							K8s: &operatorv1alpha1.KubernetesResourcesConfig{
								HPASpec: &operatorv1alpha1.HPASpec{
									MaxReplicas: newInt32WithValue(2),
								},
							},
						},
					},
				}}, istio.ConfigurationUpdate),

				Entry("If no change occurred should return NoChange", operatorv1alpha1.Istio{Spec: operatorv1alpha1.IstioSpec{
					Components: &operatorv1alpha1.Components{
						Pilot: &operatorv1alpha1.IstioComponent{K8s: &operatorv1alpha1.KubernetesResourcesConfig{
							HPASpec: &operatorv1alpha1.HPASpec{
								MaxReplicas: newInt32WithValue(21),
							},
						}},
					},
				}}, operatorv1alpha1.Istio{Spec: operatorv1alpha1.IstioSpec{
					Components: &operatorv1alpha1.Components{
						Pilot: &operatorv1alpha1.IstioComponent{K8s: &operatorv1alpha1.KubernetesResourcesConfig{
							HPASpec: &operatorv1alpha1.HPASpec{
								MaxReplicas: newInt32WithValue(21),
							},
						}},
					},
				}}, istio.NoChange),
			)
		})
	})
})

func newInt32WithValue(value int) *int32 {
	ret := int32(value)
	return &ret
}

func newStringWithValue(value string) *string {
	ret := value
	return &ret
}
