package predicates

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
)

var _ = Describe("Prometheus Merge Predicate", func() {

	client := makeClientWithObjects(
		&v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "istio-system",
				Name:      "istio",
			},
			Data: map[string]string{
				"mesh": "|-\n    defaultConfig:\n      statusPort: 15020\n",
			},
		})
	Context("NewPrometheusMergeRestartPredicate", func() {
		It("should return value of prometheusMerge from istio CR", func() {
			predicate, err := NewPrometheusMergeRestartPredicate(context.Background(), client, &operatorv1alpha2.Istio{
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						Telemetry: operatorv1alpha2.Telemetry{
							Metrics: operatorv1alpha2.Metrics{
								PrometheusMerge: true,
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.prometheusMerge).To(BeTrue())
		})
		It("should return true if prometheusMerge is true but annotations are not updated", func() {
			pod := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"prometheus.io/port": "8080",
					},
				},
			}
			predicate, err := NewPrometheusMergeRestartPredicate(context.Background(), client, &operatorv1alpha2.Istio{
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						Telemetry: operatorv1alpha2.Telemetry{
							Metrics: operatorv1alpha2.Metrics{
								PrometheusMerge: true,
							},
						},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.Matches(pod)).To(BeTrue())
		})
		It("should return true if the new prometheusMerge is false but annotations are not updated", func() {
			pod := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"prometheus.io/path": "/stats/prometheus",
						"prometheus.io/port": "15020",
					},
				},
			}
			predicate, err := NewPrometheusMergeRestartPredicate(context.Background(), client, &operatorv1alpha2.Istio{
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						Telemetry: operatorv1alpha2.Telemetry{
							Metrics: operatorv1alpha2.Metrics{
								PrometheusMerge: false,
							},
						},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.Matches(pod)).To(BeTrue())
		})
		It("should return false if the new prometheusMerge is true and annotations are correctly updated", func() {
			pod := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"prometheus.io/path": "/stats/prometheus",
						"prometheus.io/port": "15020",
					},
				},
			}
			predicate, err := NewPrometheusMergeRestartPredicate(context.Background(), client, &operatorv1alpha2.Istio{
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						Telemetry: operatorv1alpha2.Telemetry{
							Metrics: operatorv1alpha2.Metrics{
								PrometheusMerge: true,
							},
						},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.Matches(pod)).To(BeFalse())
		})
		It("should return false if the new prometheusMerge is false and annotations are correctly updated", func() {
			pod := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"prometheus.io/port": "8080",
					},
				},
			}
			predicate, err := NewPrometheusMergeRestartPredicate(context.Background(), client, &operatorv1alpha2.Istio{
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						Telemetry: operatorv1alpha2.Telemetry{
							Metrics: operatorv1alpha2.Metrics{
								PrometheusMerge: false,
							},
						},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.Matches(pod)).To(BeFalse())
		})
		It("should use the status port configured in the istio default config", func() {
			client := makeClientWithObjects(&v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "istio-system",
					Name:      "istio",
				},
				Data: map[string]string{
					"mesh": "|-\n    defaultConfig:\n      statusPort: 15080\n",
				},
			})

			predicate, err := NewPrometheusMergeRestartPredicate(context.Background(), client, &operatorv1alpha2.Istio{
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						Telemetry: operatorv1alpha2.Telemetry{
							Metrics: operatorv1alpha2.Metrics{
								PrometheusMerge: false,
							},
						},
					},
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(predicate.statusPort).To(Equal("15080"))

		})
	})
})

func makeClientWithObjects(objects ...client.Object) client.Client {
	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()
}
