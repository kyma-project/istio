package istio

import (
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
			cr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
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
			cr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
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
			cr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
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
			cr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
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
			cr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
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
	Context("isIstioInstalled", func() {
		It("should return true when istiod deployment is present on cluster", func() {

			// given
			istiodDeployment := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "istiod",
					Namespace: "istio-system",
					Labels:    map[string]string{"app": "istiod"},
				},
			}
			fakeclient := fake.NewClientBuilder().WithObjects(&istiodDeployment).Build()

			// when
			isInstalled := isIstioInstalled(fakeclient)

			// then
			Expect(isInstalled).To(BeTrue())
		})

		It("should return false when there is no istiod deployment present on cluster", func() {

			// given
			fakeclient := fake.NewClientBuilder().Build()

			// when
			isInstalled := isIstioInstalled(fakeclient)

			// then
			Expect(isInstalled).To(BeFalse())
		})
	})
})
