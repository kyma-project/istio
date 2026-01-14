package nginx

import (
	"context"
	_ "embed"
	"fmt"
	"testing"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"

	rc "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
)

//go:embed nginx_config_forward.yaml
var ConfigMap string

// CreateForwardRequestNginx returns the full DNS name of the created nginx service.
// It waits until the deployment has at least one ready replica.
// ForwardTarget is used with http protocol with Service on port 80
func CreateForwardRequestNginx(t *testing.T, name, namespace, forwardTarget string) (string, error) {
	t.Helper()
	c, err := rc.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return "", err
	}
	_, err = infrastructure.CreateResourceWithTemplateValues(
		t,
		ConfigMap,
		map[string]any{
			"ForwardTo": forwardTarget,
		},
		decoder.MutateNamespace(namespace),
	)
	if err != nil {
		t.Logf("Failed to create nginx config map template: %v", err)
		return "", err
	}

	err = createNginxDeployment(t, c, name, namespace)
	if err != nil {
		t.Logf("Failed to create nginx deployment: %v", err)
		return "", err
	}
	err = createNginxService(t, c, name, namespace)
	if err != nil {
		t.Logf("Failed to create nginx service: %v", err)
		return "", err
	}
	err = wait.For(func(ctx context.Context) (done bool, err error) {
		dep := &v1.Deployment{}
		err = c.Get(ctx, name, namespace, dep)
		if err != nil {
			t.Logf("Failed to get nginx deployment: %v", err)
			return false, err
		}
		if dep.Status.ReadyReplicas < 1 {
			t.Logf("Nginx deployment is not ready yet. Ready replicas: %d", dep.Status.ReadyReplicas)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Logf("Failed to wait for nginx deployment to be ready: %v", err)
		return "", err
	}

	return fmt.Sprintf("%s.%s.svc.cluster.local", name, namespace), nil
}

func createNginxDeployment(t *testing.T, k8sClient *resources.Resources, name, namespace string) error {
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
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: "nginx:alpine",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "nginx-config",
									MountPath: "/etc/nginx/nginx.conf",
									SubPath:   "nginx.conf",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							// This name should match the name from the yaml manifest of the configmap
							Name: "nginx-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: "nginx-config"},
								},
							},
						},
					},
				},
			},
		},
	}

	err := k8sClient.Create(t.Context(), &dep)
	if err != nil {
		t.Logf("Failed to create nginx deployment: %v", err)
		return err
	}

	return nil
}

func createNginxService(t *testing.T, k8sClient *resources.Resources, name, namespace string) error {
	svc := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": name,
			},
			Ports: []corev1.ServicePort{
				{
					Port: 80,
				},
			},
		},
	}

	err := k8sClient.Create(t.Context(), &svc)
	if err != nil {
		t.Logf("Failed to create nginx service: %v", err)
		return err
	}

	return nil
}
