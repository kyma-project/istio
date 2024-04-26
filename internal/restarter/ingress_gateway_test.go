package restarter_test

import (
	"context"
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/filter"
	"github.com/kyma-project/istio/operator/internal/restarter"
	"github.com/kyma-project/istio/operator/internal/status"
	"github.com/kyma-project/istio/operator/pkg/lib/annotations"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var _ = Describe("Istio Ingress Gateway restart", func() {
	It("should successfully restart istio ingress-gateway when predicate requires restart", func() {
		// given
		istioCR := &operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations:     map[string]string{},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{},
			},
		}

		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "1.16.1")
		igDep := createIngressGatewayDep(time.Now().Add(-time.Hour))
		igPod := createIgPodWithCreationTimestamp("istio-ingressgateway", gatherer.IstioNamespace, "discovery", "1.16.1", time.Now().Add(-time.Hour))
		fakeClient := createFakeClient(istioCR, istiod, igPod, igDep)
		statusHandler := status.NewStatusHandler(fakeClient)
		igRestarter := restarter.NewIngressGatewayRestarter(fakeClient, []filter.IngressGatewayPredicate{mockIgPredicate{shouldRestart: true}}, statusHandler)

		//when
		err := igRestarter.Restart(context.Background(), istioCR)

		//then
		e := fakeClient.Get(context.Background(), client.ObjectKey{Namespace: gatherer.IstioNamespace, Name: "istio-ingressgateway"}, igDep)
		Expect(err).Should(Not(HaveOccurred()))
		Expect(e).Should(Not(HaveOccurred()))

		Expect(annotations.HasRestartAnnotation(igDep.Spec.Template.Annotations)).To(BeTrue())
		Expect((*istioCR.Status.Conditions)[0].Reason).Should(Equal(string(operatorv1alpha2.ConditionReasonIngressGatewayRestartSucceeded)))
		Expect((*istioCR.Status.Conditions)[0].Message).Should(Equal(operatorv1alpha2.ConditionReasonIngressGatewayRestartSucceededMessage))
	})

	It("does not restart ingress gateway when predicate does not require it", func() {
		// given
		istioCR := &operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations:     map[string]string{},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{},
			},
		}

		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "1.16.1")
		igDep := createIngressGatewayDep(time.Now())
		igPod := createIgPodWithCreationTimestamp("istio-ingressgateway", gatherer.IstioNamespace, "discovery", "1.16.1", time.Now())
		fakeClient := createFakeClient(istioCR, istiod, igDep, igPod)
		statusHandler := status.NewStatusHandler(fakeClient)
		igRestarter := restarter.NewIngressGatewayRestarter(fakeClient, []filter.IngressGatewayPredicate{mockIgPredicate{shouldRestart: false}}, statusHandler)

		//when
		err := igRestarter.Restart(context.Background(), istioCR)

		//then
		e := fakeClient.Get(context.Background(), client.ObjectKey{Namespace: gatherer.IstioNamespace, Name: "istio-ingressgateway"}, igDep)
		Expect(err).Should(Not(HaveOccurred()))
		Expect(e).Should(Not(HaveOccurred()))

		Expect(annotations.HasRestartAnnotation(igDep.Spec.Template.Annotations)).To(BeFalse())
		Expect((*istioCR.Status.Conditions)[0].Reason).Should(Equal(string(operatorv1alpha2.ConditionReasonIngressGatewayRestartSucceeded)))
		Expect((*istioCR.Status.Conditions)[0].Message).Should(Equal(operatorv1alpha2.ConditionReasonIngressGatewayRestartSucceededMessage))
	})

	It("should not fail ingress gateway restarting when there is no ingress gateway pods found", func() {
		// given
		istioCR := &operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations:     map[string]string{},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{},
			},
		}

		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "1.16.1")
		fakeClient := createFakeClient(istioCR, istiod)
		statusHandler := status.NewStatusHandler(fakeClient)
		igRestarter := restarter.NewIngressGatewayRestarter(fakeClient, []filter.IngressGatewayPredicate{mockIgPredicate{shouldRestart: true}}, statusHandler)

		//when
		err := igRestarter.Restart(context.Background(), istioCR)

		//then
		Expect(err).Should(Not(HaveOccurred()))
		Expect((*istioCR.Status.Conditions)[0].Reason).Should(Equal(string(operatorv1alpha2.ConditionReasonIngressGatewayRestartSucceeded)))
		Expect((*istioCR.Status.Conditions)[0].Message).Should(Equal(operatorv1alpha2.ConditionReasonIngressGatewayRestartSucceededMessage))

	})

})

func createIngressGatewayDep(creationTimestamp time.Time) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "istio-ingressgateway",
			Namespace:         gatherer.IstioNamespace,
			CreationTimestamp: metav1.Unix(creationTimestamp.Unix(), 0),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: func(i int32) *int32 { return &i }(1),
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "istio-ingressgateway",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "discovery",
							Image: "image:1.16.1",
						},
					},
				},
			},
		},
	}
}

type mockIgPredicate struct {
	shouldRestart bool
}

func (m mockIgPredicate) RequiresIngressGatewayRestart() bool {
	return m.shouldRestart
}

func (m mockIgPredicate) NewIngressGatewayEvaluator(_ context.Context) (filter.IngressGatewayRestartEvaluator, error) {
	return m, nil
}

func createIgPodWithCreationTimestamp(name, namespace, containerName, imageVersion string, t time.Time) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.Unix(t.Unix(), 0),
			Labels: map[string]string{
				"app": "istio-ingressgateway",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  containerName,
					Image: "image:" + imageVersion,
				},
			},
		},
	}
}
