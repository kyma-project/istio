package steps

import (
	"bytes"
	"context"
	_ "embed"
	"github.com/avast/retry-go"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
	"text/template"
)

//go:embed nginx_config_template.yaml
var configMapTemplateYaml []byte

// CreateNginxApplication creates a deployment, service  and config map for a nginx application
func CreateNginxApplication(ctx context.Context, appName, namespace, forwardTo string) (context.Context, error) {

	configMapName := "nginx-conf"
	ctx, err := createConfigurationCm(ctx, configMapName, namespace, forwardTo)
	if err != nil {
		return ctx, err
	}

	ctx, err = createDeployment(ctx, appName, namespace, configMapName)
	if err != nil {
		return ctx, err
	}

	ctx, err = createService(ctx, appName, namespace)
	if err != nil {
		return ctx, err
	}

	return ctx, nil
}

func createService(ctx context.Context, appName string, namespace string) (context.Context, error) {
	svc := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": appName,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt32(80),
				},
			},
		},
	}

	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	err = retry.Do(func() error {
		err := k8sClient.Create(context.TODO(), &svc)
		if err != nil {
			return err
		}
		ctx = testcontext.AddCreatedTestObjectInContext(ctx, &svc)
		return nil
	}, testcontext.GetRetryOpts()...)
	return ctx, err
}

func createDeployment(ctx context.Context, name, namespace, configMapName string) (context.Context, error) {
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
							Name: "nginx-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: configMapName},
								},
							},
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

func createConfigurationCm(ctx context.Context, name string, namespace, forwardTo string) (context.Context, error) {
	confTemplateValues := map[string]string{
		"ForwardTo": forwardTo,
	}

	cm, err := createNginxConfigMapFromTemplate(name, namespace, confTemplateValues)
	if err != nil {
		return ctx, err
	}
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	err = retry.Do(func() error {
		err := k8sClient.Create(context.TODO(), &cm)
		if err != nil {
			return err
		}
		ctx = testcontext.AddCreatedTestObjectInContext(ctx, &cm)
		return nil
	}, testcontext.GetRetryOpts()...)

	return ctx, err
}

func createNginxConfigMapFromTemplate(name string, namespace string, templateValues map[string]string) (corev1.ConfigMap, error) {

	cmTemplate, err := template.New("tmpl").Option("missingkey=zero").Parse(string(configMapTemplateYaml))
	if err != nil {
		return corev1.ConfigMap{}, err
	}

	var resource bytes.Buffer
	err = cmTemplate.Execute(&resource, templateValues)
	if err != nil {
		return corev1.ConfigMap{}, err
	}

	var cm corev1.ConfigMap
	err = yaml.Unmarshal(resource.Bytes(), &cm)
	if err != nil {
		return corev1.ConfigMap{}, err
	}

	cm.Namespace = namespace
	cm.Name = name
	return cm, nil
}
