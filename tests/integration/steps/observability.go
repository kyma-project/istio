package steps

import (
	"bytes"
	"context"
	"github.com/avast/retry-go"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"github.com/pkg/errors"
	"io"
	"istio.io/api/telemetry/v1alpha1"
	v1alpha12 "istio.io/client-go/pkg/apis/telemetry/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	"log"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const collectorMockImage = "europe-docker.pkg.dev/kyma-project/prod/external/kennethreitz/httpbin"

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

func EnableTracingAndAccessLogging(ctx context.Context, logsProvider, tracingProvider string) (context.Context, error) {
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
							{Name: logsProvider},
						},
					},
				},
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

func CreateTelemetryCollectorMock(ctx context.Context, name, namespace string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	dep := v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.DeploymentSpec{
			Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       name,
					"component": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name, "component": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: collectorMockImage,
						},
					},
				},
			},
		},
	}

	err = retry.Do(func() error {
		err := k8sClient.Create(context.TODO(), &dep)
		if err != nil {
			return err
		}
		ctx = testcontext.AddCreatedTestObjectInContext(ctx, &dep)
		return nil
	}, testcontext.GetRetryOpts()...)

	return ctx, err
}

func CreateTelemetryService(ctx context.Context, collectorDepName, namespace string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}
	err = retry.Do(func() error {
		svc := &v12.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "telemetry-otlp-traces",
				Namespace: namespace,
			},
			Spec: v12.ServiceSpec{
				Selector: map[string]string{
					"component": collectorDepName,
				},
				Ports: []v12.ServicePort{
					{
						Name:       "otlp-grpc",
						Port:       4317,
						Protocol:   "TCP",
						TargetPort: intstr.FromInt32(4317),
					},
					{
						Name:       "otlp-http",
						Port:       4318,
						Protocol:   "TCP",
						TargetPort: intstr.FromInt32(4318),
					},
					{
						Name: "metrics",
						Port: 8888,
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

func VerifyLogEntryForDeployment(ctx context.Context, name, namespace, logKey string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}
	err = retry.Do(func() error {

		var dep v1.Deployment
		err = k8sClient.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}, &dep)
		if err != nil {
			return err
		}

		var pods v12.PodList
		err = k8sClient.List(ctx, &pods, client.MatchingLabels{
			"app": name,
		})
		if err != nil {
			return err
		}

		found := false
		for _, pod := range pods.Items {
			str, err := getLogsFromPodsContainer(ctx, pod, "istio-proxy")
			if err != nil {
				return err
			}
			if sub := strings.Contains(str, logKey); sub {
				found = true
			}
		}
		if !found {
			return errors.New("log entry not found")
		}

		return nil
	}, testcontext.GetRetryOpts()...)
	return ctx, err
}

func getLogsFromPodsContainer(ctx context.Context, pod v12.Pod, containerName string) (string, error) {
	conf := config.GetConfigOrDie()
	c := kubernetes.NewForConfigOrDie(conf)

	logOpt := &v12.PodLogOptions{
		Container: containerName,
	}
	req := c.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, logOpt)
	logs, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer func() {
		e := logs.Close()
		if e != nil {
			log.Printf("error closing logs stream: %s", err.Error())
		}
	}()
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, logs)
	if err != nil {
		return "", err
	}
	str := buf.String()
	return str, nil
}
