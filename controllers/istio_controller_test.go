package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/manifest"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	"github.com/kyma-project/istio/operator/internal/reconciliations/proxy"
	"github.com/kyma-project/istio/operator/internal/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Istio Controller", func() {
	Context("Reconcile", func() {
		It("should fail to reconcile Istio CR in different than kyma-system namespace and set error state", func() {
			//given"
			numTrustedProxies := 1
			istioCR := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
				Name:            "default",
				ResourceVersion: "1",
			},
				Spec: operatorv1alpha1.IstioSpec{
					Config: operatorv1alpha1.Config{
						NumTrustedProxies: &numTrustedProxies,
					},
				},
			}

			client := createFakeClient(&istioCR)
			merger := manifest.NewDefaultIstioMerger()
			logger := logr.Discard()
			istioController := &IstioReconciler{
				Client:                 client,
				Scheme:                 scheme.Scheme,
				istioInstallation:      istio.Installation{Client: client, IstioClient: istio.NewIstioClient(), IstioVersion: IstioVersion, IstioImageBase: IstioImageBase, Merger: &merger, StatusHandler: status.NewDefaultStatusHandler()},
				proxySidecars:          proxy.Sidecars{IstioVersion: IstioVersion, IstioImageBase: IstioImageBase, Log: logger, Client: client, Merger: &merger, StatusHandler: status.NewDefaultStatusHandler()},
				log:                    logger,
				statusHandler:          status.NewDefaultStatusHandler(),
				reconciliationInterval: 10 * time.Hour,
			}
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "default",
				},
			}

			//when
			res, err := istioController.Reconcile(context.TODO(), req)

			//then
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Requeue).To(BeFalse())

			processedIstioCR := operatorv1alpha1.Istio{}
			err = client.Get(context.TODO(), types.NamespacedName{Name: "default"}, &processedIstioCR)
			Expect(err).To(Not(HaveOccurred()))
			Expect(processedIstioCR.Status.State).To(Equal(operatorv1alpha1.Error))
			Expect(processedIstioCR.Status.Description).To(Equal("Error occurred during reconciliation of Istio CR: Istio CR is not in kyma-system namespace"))
		})
	})
})

func createFakeClient(objects ...client.Object) client.Client {
	err := operatorv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())

	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()
}
