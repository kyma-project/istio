//go:build !experimental

package controllers

import (
	"context"
	"github.com/go-logr/logr"
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/status"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Istio Controller", func() {
	It("should set CR status to warning if experimental fields have been defined", func() {
		// given
		istioCR := &operatorv1alpha2.Istio{
			ObjectMeta: metav1.ObjectMeta{
				Name:              istioCrName,
				Namespace:         testNamespace,
				UID:               "1",
				CreationTimestamp: metav1.Unix(1494505756, 0),
				Finalizers: []string{
					"istios.operator.kyma-project.io/istio-installation",
				},
			},
			Spec: operatorv1alpha2.IstioSpec{
				Experimental: &operatorv1alpha2.Experimental{
					PilotFeatures: operatorv1alpha2.PilotFeatures{
						EnableMultiNetworkDiscoverGatewayAPI: true,
						EnableAlphaGatewayAPI:                true,
					},
				}},
		}

		fakeClient := createFakeClient(istioCR)
		sut := &IstioReconciler{
			Client:                 fakeClient,
			Scheme:                 getTestScheme(),
			istioInstallation:      &istioInstallationReconciliationMock{},
			istioResources:         &istioResourcesReconciliationMock{},
			userResources:          &UserResourcesMock{},
			log:                    logr.Discard(),
			statusHandler:          status.NewStatusHandler(fakeClient),
			reconciliationInterval: testReconciliationInterval,
		}

		// when
		result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

		// then
		Expect(err).ToNot(HaveOccurred())
		Expect(result.Requeue).To(BeFalse())

		updatedIstioCR := operatorv1alpha2.Istio{}
		err = fakeClient.Get(context.Background(), client.ObjectKeyFromObject(istioCR), &updatedIstioCR)
		Expect(err).To(Not(HaveOccurred()))

		Expect(updatedIstioCR.Status.State).Should(Equal(operatorv1alpha2.Warning))
		Expect(updatedIstioCR.Status.Description).To(ContainSubstring("Experimental features are not supported in this image flavour"))
	})
})
