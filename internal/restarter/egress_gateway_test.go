package restarter_test

import (
	"context"
	"time"

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
)

var _ = Describe("Istio Egress Gateway restart", func() {
	It("should successfully restart istio Egress-gateway when predicate requires restart", func() {
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
		egDep := createEgressGatewayDep(time.Now().Add(-time.Hour))
		egPod := createEgPodWithCreationTimestamp("istio-egressgateway", gatherer.IstioNamespace, "discovery", "1.16.1", time.Now().Add(-time.Hour))
		fakeClient := createFakeClient(istioCR, istiod, egPod, egDep)
		statusHandler := status.NewStatusHandler(fakeClient)
		egRestarter := restarter.NewEgressGatewayRestarter(fakeClient, []filter.EgressGatewayPredicate{mockEgPredicate{shouldRestart: true}}, statusHandler)

		//when
		err, requeue := egRestarter.Restart(context.Background(), istioCR)

		//then
		Expect(err).Should(Not(HaveOccurred()))
		Expect(requeue).To(BeFalse())

		e := fakeClient.Get(context.Background(), client.ObjectKey{Namespace: gatherer.IstioNamespace, Name: "istio-egressgateway"}, egDep)
		Expect(e).Should(Not(HaveOccurred()))

		Expect(annotations.HasRestartAnnotation(egDep.Spec.Template.Annotations)).To(BeTrue())
		Expect((*istioCR.Status.Conditions)[0].Reason).Should(Equal(string(operatorv1alpha2.ConditionReasonEgressGatewayRestartSucceeded)))
		Expect((*istioCR.Status.Conditions)[0].Message).Should(Equal(operatorv1alpha2.ConditionReasonEgressGatewayRestartSucceededMessage))
	})

	It("does not restart Egress gateway when predicate does not require it", func() {
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
		egDep := createEgressGatewayDep(time.Now())
		egPod := createEgPodWithCreationTimestamp("istio-egressgateway", gatherer.IstioNamespace, "discovery", "1.16.1", time.Now())
		fakeClient := createFakeClient(istioCR, istiod, egDep, egPod)
		statusHandler := status.NewStatusHandler(fakeClient)
		egRestarter := restarter.NewEgressGatewayRestarter(fakeClient, []filter.EgressGatewayPredicate{mockEgPredicate{shouldRestart: false}}, statusHandler)

		//when
		err, requeue := egRestarter.Restart(context.Background(), istioCR)

		//then
		Expect(err).Should(Not(HaveOccurred()))
		Expect(requeue).To(BeFalse())

		e := fakeClient.Get(context.Background(), client.ObjectKey{Namespace: gatherer.IstioNamespace, Name: "istio-egressgateway"}, egDep)
		Expect(e).Should(Not(HaveOccurred()))

		Expect(annotations.HasRestartAnnotation(egDep.Spec.Template.Annotations)).To(BeFalse())
		Expect((*istioCR.Status.Conditions)[0].Reason).Should(Equal(string(operatorv1alpha2.ConditionReasonEgressGatewayRestartSucceeded)))
		Expect((*istioCR.Status.Conditions)[0].Message).Should(Equal(operatorv1alpha2.ConditionReasonEgressGatewayRestartSucceededMessage))
	})

	It("should not fail Egress gateway restarting when there is no Egress gateway pods found", func() {
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
		egRestarter := restarter.NewEgressGatewayRestarter(fakeClient, []filter.EgressGatewayPredicate{mockEgPredicate{shouldRestart: true}}, statusHandler)

		//when
		err, requeue := egRestarter.Restart(context.Background(), istioCR)

		//then
		Expect(err).Should(Not(HaveOccurred()))
		Expect(requeue).To(BeFalse())
		Expect((*istioCR.Status.Conditions)[0].Reason).Should(Equal(string(operatorv1alpha2.ConditionReasonEgressGatewayRestartSucceeded)))
		Expect((*istioCR.Status.Conditions)[0].Message).Should(Equal(operatorv1alpha2.ConditionReasonEgressGatewayRestartSucceededMessage))

	})

})

func createEgressGatewayDep(creationTimestamp time.Time) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "istio-egressgateway",
			Namespace:         gatherer.IstioNamespace,
			CreationTimestamp: metav1.Unix(creationTimestamp.Unix(), 0),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: func(i int32) *int32 { return &i }(1),
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "istio-egressgateway",
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

type mockEgPredicate struct {
	shouldRestart bool
}

func (m mockEgPredicate) RequiresEgressGatewayRestart() bool {
	return m.shouldRestart
}

func (m mockEgPredicate) NewEgressGatewayEvaluator(_ context.Context) (filter.EgressGatewayRestartEvaluator, error) {
	return m, nil
}

func createEgPodWithCreationTimestamp(name, namespace, containerName, imageVersion string, t time.Time) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.Unix(t.Unix(), 0),
			Labels: map[string]string{
				"app": "istio-egressgateway",
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
