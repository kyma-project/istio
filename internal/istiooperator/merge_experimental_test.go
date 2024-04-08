//go:build experimental

package istiooperator_test

import (
	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/istiooperator"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"istio.io/api/operator/v1alpha1"
	istiov1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
)

var _ = Describe("Merge", func() {
	It("should succeed and have experimental variables set in config file", func() {
		istioCR := v1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-test",
			Namespace: "namespace",
		},
			Spec: v1alpha2.IstioSpec{
				Experimental: &v1alpha2.Experimental{PilotFeatures: v1alpha2.PilotFeatures{
					EnableAlphaGatewayAPI:                true,
					EnableMultiNetworkDiscoverGatewayAPI: true,
				}},
			},
		}
		merger := istiooperator.NewDefaultIstioMerger()

		p, err := merger.Merge(clusterconfig.Evaluation, &istioCR, clusterconfig.ClusterConfiguration{})
		Expect(err).ShouldNot(HaveOccurred())
		iop := readIOP(p)
		Expect(iop.Spec.Components.Pilot).ToNot(BeNil())
		// I check the expected size of the env struct, because Merge()
		// populates the state object which invalidates strict reflect
		// validation. If loaded CR from file (Evaluation) changes, this
		// size needs to be updated...
		Expect(len(iop.Spec.Components.Pilot.K8S.Env)).To(Equal(3))
	})
	Context("ParseExperimentalFeatures", func() {
		It("should update IstioOperator with managed environment variables when all experimental options are set to true and source struct is populated", func() {
			istioCR := v1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
				Name:      "istio-test",
				Namespace: "namespace",
			},
				Spec: v1alpha2.IstioSpec{
					Experimental: &v1alpha2.Experimental{PilotFeatures: v1alpha2.PilotFeatures{
						EnableAlphaGatewayAPI:                true,
						EnableMultiNetworkDiscoverGatewayAPI: true,
					}},
				},
			}
			// doesn't matter which file is used, use light
			iop := readIOP("../../internal/istiooperator/istio-operator-light.yaml")
			Expect(istiooperator.ParseExperimentalFeatures(&istioCR, &iop)).To(Succeed())
			Expect(iop.Spec.Components.Pilot).ToNot(BeNil())
			Expect(iop.Spec.Components.Pilot.K8S.Env).To(ContainElements(
				&v1alpha1.EnvVar{Name: "PILOT_ENABLE_ALPHA_GATEWAY_API", Value: "true"},
				&v1alpha1.EnvVar{Name: "PILOT_MULTI_NETWORK_DISCOVER_GATEWAY_API", Value: "true"}))
		})
		It("should update IstioOperator with managed environment variables when all experimental options are set to true and source struct is empty", func() {
			istioCR := v1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
				Name:      "istio-test",
				Namespace: "namespace",
			},
				Spec: v1alpha2.IstioSpec{
					Experimental: &v1alpha2.Experimental{PilotFeatures: v1alpha2.PilotFeatures{
						EnableAlphaGatewayAPI:                true,
						EnableMultiNetworkDiscoverGatewayAPI: true,
					}},
				},
			}
			iop := istiov1alpha1.IstioOperator{}
			Expect(istiooperator.ParseExperimentalFeatures(&istioCR, &iop)).To(Succeed())
			Expect(iop.Spec.Components.Pilot).ToNot(BeNil())
			Expect(iop.Spec.Components.Pilot.K8S.Env).To(ContainElements(
				&v1alpha1.EnvVar{Name: "PILOT_ENABLE_ALPHA_GATEWAY_API", Value: "true"},
				&v1alpha1.EnvVar{Name: "PILOT_MULTI_NETWORK_DISCOVER_GATEWAY_API", Value: "true"}))
		})
		It("should update IstioOperator with managed environment variables when all experimental options are set to true source struct already contains those variables set to non-managed ones", func() {
			istioCR := v1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
				Name:      "istio-test",
				Namespace: "namespace",
			},
				Spec: v1alpha2.IstioSpec{
					Experimental: &v1alpha2.Experimental{PilotFeatures: v1alpha2.PilotFeatures{
						EnableAlphaGatewayAPI:                true,
						EnableMultiNetworkDiscoverGatewayAPI: true,
					}},
				},
			}
			iop := istiov1alpha1.IstioOperator{
				Spec: &v1alpha1.IstioOperatorSpec{Components: &v1alpha1.IstioComponentSetSpec{Pilot: &v1alpha1.ComponentSpec{K8S: &v1alpha1.KubernetesResourcesSpec{Env: []*v1alpha1.EnvVar{
					{Name: "PILOT_ENABLE_ALPHA_GATEWAY_API", Value: "asdasd"},
					{Name: "PILOT_MULTI_NETWORK_DISCOVER_GATEWAY_API", Value: "asdasd"},
				}}}}},
			}
			Expect(istiooperator.ParseExperimentalFeatures(&istioCR, &iop)).To(Succeed())
			Expect(iop.Spec.Components.Pilot).ToNot(BeNil())
			Expect(iop.Spec.Components.Pilot.K8S.Env).To(ContainElements(
				&v1alpha1.EnvVar{Name: "PILOT_ENABLE_ALPHA_GATEWAY_API", Value: "true"},
				&v1alpha1.EnvVar{Name: "PILOT_MULTI_NETWORK_DISCOVER_GATEWAY_API", Value: "true"}))
		})
		It("should succeed if experimental fields are not defined", func() {
			istioCR := v1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
				Name:      "istio-test",
				Namespace: "namespace",
			}}
			iop := istiov1alpha1.IstioOperator{}
			expectiop := istiov1alpha1.IstioOperator{}
			Expect(istiooperator.ParseExperimentalFeatures(&istioCR, &iop)).To(Succeed())
			Expect(reflect.DeepEqual(iop, expectiop)).To(BeTrue())
			// condition is not applied, or Unknown state because there was no experimental features applied
		})
	})

})
