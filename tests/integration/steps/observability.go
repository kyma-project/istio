package steps

import (
	"context"
	"strconv"

	"github.com/avast/retry-go"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"istio.io/api/telemetry/v1alpha1"
	v1alpha12 "istio.io/client-go/pkg/apis/telemetry/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func EnableAccessLogging(ctx context.Context, provider string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}
	err = retry.Do(func() error {
		tm := &v1alpha12.Telemetry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "access-logs",
				Namespace: "istio-system",
			},
			Spec: v1alpha1.Telemetry{
				AccessLogging: []*v1alpha1.AccessLogging{
					{
						Providers: []*v1alpha1.ProviderRef{
							{Name: provider},
						},
					},
				},
			},
		}

		err := k8sClient.Create(context.Background(), tm)
		if err != nil {
			return err
		}
		ctx = testcontext.AddCreatedTestObjectInContext(ctx, tm)
		return nil
	}, testcontext.GetRetryOpts()...)
	return ctx, err
}

func EnableTracing(ctx context.Context, tracingProvider string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}
	err = retry.Do(func() error {
		tm := &v1alpha12.Telemetry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "enable-tracing",
				Namespace: "istio-system",
			},
			Spec: v1alpha1.Telemetry{
				Tracing: []*v1alpha1.Tracing{
					{
						Providers: []*v1alpha1.ProviderRef{
							{Name: tracingProvider},
						},
					},
				},
			},
		}

		err := k8sClient.Create(context.Background(), tm)
		if err != nil {
			return err
		}
		ctx = testcontext.AddCreatedTestObjectInContext(ctx, tm)
		return nil
	}, testcontext.GetRetryOpts()...)
	return ctx, err
}

func CreateTelemetryCollectorMock(ctx context.Context, appName, namespace string) (context.Context, error) {
	exposedPort := 4317
	c := corev1.Container{
		Name:  appName,
		Image: "docker.io/istio/tcp-echo-server:1.2",
		Args:  []string{strconv.Itoa(exposedPort), "one"},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: int32(exposedPort),
				Protocol:      corev1.ProtocolTCP,
			},
		},
	}

	return CreateDeployment(ctx, appName, namespace, c)
}

func CreateOpenTelemetryService(ctx context.Context, collectorDepName, namespace string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}
	err = retry.Do(func() error {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "telemetry-otlp-traces",
				Namespace: namespace,
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app": collectorDepName,
				},
				Ports: []corev1.ServicePort{
					{
						Name:       "otlp-grpc",
						Port:       4317,
						Protocol:   "TCP",
						TargetPort: intstr.FromInt32(4317),
					},
				},
			},
		}

		err := k8sClient.Create(context.Background(), svc)
		if err != nil {
			return err
		}
		ctx = testcontext.AddCreatedTestObjectInContext(ctx, svc)
		return nil
	}, testcontext.GetRetryOpts()...)
	return ctx, err
}
