package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/cucumber/godog"
	istioCR "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/tests/integration/manifests"
	"github.com/mitchellh/mapstructure"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"strings"
	"text/template"
)

const (
	defaultIopName      string = "installed-state-default-operator"
	defaultIopNamespace string = "istio-system"

	templateFileName string = "manifests/istio_cr_template.yaml"
)

type TestWithTemplatedManifest struct {
	TemplateValues map[string]string
}

func (t *TestWithTemplatedManifest) initIstioScenarios(ctx *godog.ScenarioContext) {

	ctx.After(istioCrTearDown)
	ctx.After(testAppTearDown)

	ctx.Step(`^"([^"]*)" "([^"]*)" in namespace "([^"]*)" is ready`, resourceIsReady)
	ctx.Step(`^Istio CRD is installed$`, istioCRDIsInstalled)
	ctx.Step(`^Istio CR "([^"]*)" in namespace "([^"]*)" has status "([^"]*)"$`, istioCRInNamespaceHasStatus)
	ctx.Step(`^Template value "([^"]*)" is set to "([^"]*)"$`, t.setTemplateValue)
	ctx.Step(`^Istio CR "([^"]*)" is applied in namespace "([^"]*)"$`, t.istioCRIsAppliedInNamespace)
	ctx.Step(`^Istio CR "([^"]*)" is updated in namespace "([^"]*)"$`, t.istioCrIsUpdatedInNamespace)
	ctx.Step(`^Namespace "([^"]*)" is "([^"]*)"$`, namespaceIsPresent)
	ctx.Step(`^Istio CRDs "([^"]*)" be present on cluster$`, istioCRDsBePresentOnCluster)
	ctx.Step(`^"([^"]*)" has "([^"]*)" set to cpu - "([^"]*)" and memory - "([^"]*)"$`, componentHasResourcesSetToCpuAndMemory)
	ctx.Step(`^"([^"]*)" "([^"]*)" in namespace "([^"]*)" is deleted$`, resourceInNamespaceIsDeleted)
	ctx.Step(`^"([^"]*)" is not present on cluster$`, resourceNotPresent)
	ctx.Step(`^Istio injection is enabled in namespace "([^"]*)"$`, enableIstioInjection)
	ctx.Step(`^Application "([^"]*)" is running in namespace "([^"]*)"$`, createApplicationDeployment)
	ctx.Step(`^Application "([^"]*)" in namespace "([^"]*)" has proxy with "([^"]*)" set to cpu - "([^"]*)" and memory - "([^"]*)"$`, applicationHasProxyResourcesSetToCpuAndMemory)
}

func (t *TestWithTemplatedManifest) setTemplateValue(name, value string) {
	if len(t.TemplateValues) == 0 {
		t.TemplateValues = make(map[string]string)
	}
	t.TemplateValues[name] = value
}

func resourceIsReady(kind, name, namespace string) error {
	return retry.Do(func() error {
		var object client.Object
		switch kind {
		case Deployment.String():
			object = &v1.Deployment{}
		case DaemonSet.String():
			object = &v1.DaemonSet{}
		default:
			return godog.ErrUndefined
		}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: name}, object)
		if err != nil {
			return err
		}

		if err != nil {
			return err
		}

		switch kind {
		case Deployment.String():
			if object.(*v1.Deployment).Status.Replicas != object.(*v1.Deployment).Status.ReadyReplicas {
				return fmt.Errorf("%s %s/%s is not ready",
					kind, namespace, name)
			}
		case DaemonSet.String():
			if object.(*v1.DaemonSet).Status.NumberReady != object.(*v1.DaemonSet).Status.DesiredNumberScheduled {
				return fmt.Errorf("%s %s/%s is not ready",
					kind, namespace, name)
			}
		default:
			return godog.ErrUndefined
		}

		return nil
	}, retryOpts...)
}

func istioCRDIsInstalled() error {
	var crd unstructured.Unstructured
	crd.SetGroupVersionKind(CRDGroupVersionKind)
	return retry.Do(func() error {
		return k8sClient.Get(context.TODO(), types.NamespacedName{Name: "istios.operator.kyma-project.io"}, &crd)
	}, retryOpts...)
}

func istioCRInNamespaceHasStatus(name, namespace, status string) error {
	var cr istioCR.Istio
	return retry.Do(func() error {
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &cr)
		if err != nil {
			return err
		}
		if string(cr.Status.State) != status {
			return fmt.Errorf("status %s of Istio CR is not equal to %s", cr.Status.State, status)
		}
		return nil
	}, retryOpts...)
}

func (t *TestWithTemplatedManifest) istioCRIsAppliedInNamespace(ctx context.Context, name, namespace string) (context.Context, error) {
	istio, err := createIstioCrFromTemplate(name, namespace, t.TemplateValues)
	if err != nil {
		return ctx, err
	}

	err = retry.Do(func() error {
		err := k8sClient.Create(context.TODO(), &istio)
		if err != nil {
			return err
		}
		ctx = setIstioCrInContext(ctx, &istio)
		return nil
	}, retryOpts...)

	return ctx, err
}

func (t *TestWithTemplatedManifest) istioCrIsUpdatedInNamespace(name, namespace string) error {
	istio, err := createIstioCrFromTemplate(name, namespace, t.TemplateValues)
	if err != nil {
		return err
	}

	return retry.Do(func() error {
		var existingIstio istioCR.Istio
		if err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &existingIstio); err != nil {
			return err
		}
		istio.Spec.DeepCopyInto(&existingIstio.Spec)

		return k8sClient.Update(context.TODO(), &existingIstio)
	}, retryOpts...)
}

func createIstioCrFromTemplate(name string, namespace string, templateValues map[string]string) (istioCR.Istio, error) {
	istioCRYaml, err := os.ReadFile(templateFileName)
	if err != nil {
		return istioCR.Istio{}, err
	}

	crTemplate, err := template.New("tmpl").Option("missingkey=zero").Parse(string(istioCRYaml))
	if err != nil {
		return istioCR.Istio{}, err
	}

	var resource bytes.Buffer
	err = crTemplate.Execute(&resource, templateValues)
	if err != nil {
		return istioCR.Istio{}, err
	}

	var istio istioCR.Istio
	err = yaml.Unmarshal(resource.Bytes(), &istio)
	if err != nil {
		return istioCR.Istio{}, err
	}

	istio.Namespace = namespace
	istio.Name = name
	return istio, nil
}

func namespaceIsPresent(name, shouldBePresent string) error {
	var ns corev1.Namespace
	return retry.Do(func() error {
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name}, &ns)
		if shouldBePresent != "present" {
			if !k8serrors.IsNotFound(err) {
				return fmt.Errorf("namespace %s is present but shouldn't", name)
			}
			return nil
		}
		return err
	}, retryOpts...)
}

func istioCRDsBePresentOnCluster(should string) error {
	shouldHave := true
	if should != "should" {
		shouldHave = false
	}
	lister, err := manifests.NewCRDListerFromFile(k8sClient, crdListPath)
	if err != nil {
		return err
	}
	return retry.Do(func() error {
		wrongs, err := lister.CheckForCRDs(context.TODO(), shouldHave)
		if err != nil {
			return err
		}
		if len(wrongs) > 0 {
			if shouldHave {
				return fmt.Errorf("CRDs %s where not present", strings.Join(wrongs, ";"))
			} else {
				return fmt.Errorf("CRDs %s where present", strings.Join(wrongs, ";"))
			}
		}
		return nil
	}, retryOpts...)
}

func resourceInNamespaceIsDeleted(kind, name, namespace string) error {
	switch kind {
	case IstioCR.String():
		return retry.Do(func() error {
			var istioCr istioCR.Istio
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &istioCr)
			if err != nil {
				return err
			}

			return k8sClient.Delete(context.TODO(), &istioCr)
		})
	default:
		return godog.ErrUndefined
	}
}

func resourceNotPresent(kind string) error {
	return retry.Do(func() error {
		switch kind {
		case IstioCR.String():
			var istioList istioCR.IstioList
			err := k8sClient.List(context.TODO(), &istioList)
			if err != nil {
				return err
			}
			if len(istioList.Items) > 0 {
				return fmt.Errorf("there are %d %s present but shouldn't", len(istioList.Items), kind)
			}
		}
		return nil
	}, retryOpts...)
}

func componentHasResourcesSetToCpuAndMemory(component, resourceType, cpu, memory string) error {
	operator, err := getIstioOperatorFromCluster()
	if err != nil {
		return err
	}
	resources, err := getResourcesForComponent(operator, component, resourceType)
	if err != nil {
		return err
	}

	if resources.Cpu != cpu {
		return fmt.Errorf("cpu %s for component %s wasn't expected; expected=%s got=%s", resourceType, component, cpu, resources.Cpu)
	}

	if resources.Memory != memory {
		return fmt.Errorf("memory %s for component %s wasn't expected; expected=%s got=%s", resourceType, component, memory, resources.Memory)
	}

	return nil
}

func getIstioOperatorFromCluster() (*istioOperator.IstioOperator, error) {
	iop := istioOperator.IstioOperator{}

	err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: defaultIopName, Namespace: defaultIopNamespace}, &iop)
	if err != nil {
		return nil, fmt.Errorf("default Kyma IstioOperator CR wasn't found err=%s", err)
	}

	return &iop, nil
}

type ResourceStruct struct {
	Cpu    string
	Memory string
}

func getResourcesForComponent(operator *istioOperator.IstioOperator, component, resourceType string) (*ResourceStruct, error) {
	res := ResourceStruct{}

	switch component {
	case "proxy_init":
		fallthrough
	case "proxy":
		jsonResources, err := json.Marshal(operator.Spec.Values.Fields["global"].GetStructValue().Fields[component].GetStructValue().
			Fields["resources"].GetStructValue().Fields[resourceType].GetStructValue())

		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(jsonResources, &res)
		if err != nil {
			return nil, err
		}
		return &res, nil
	case "ingress-gateway":
		if resourceType == "limits" {
			err := mapstructure.Decode(operator.Spec.Components.IngressGateways[0].K8S.Resources.Limits, &res)
			if err != nil {
				return nil, err
			}
		} else {
			err := mapstructure.Decode(operator.Spec.Components.IngressGateways[0].K8S.Resources.Requests, &res)
			if err != nil {
				return nil, err
			}
		}

		return &res, nil
	case "egress-gateway":
		if resourceType == "limits" {
			err := mapstructure.Decode(operator.Spec.Components.EgressGateways[0].K8S.Resources.Limits, &res)
			if err != nil {
				return nil, err
			}
		} else {
			err := mapstructure.Decode(operator.Spec.Components.EgressGateways[0].K8S.Resources.Requests, &res)
			if err != nil {
				return nil, err
			}
		}
		return &res, nil
	case "pilot":
		if resourceType == "limits" {
			err := mapstructure.Decode(operator.Spec.Components.Pilot.K8S.Resources.Limits, &res)
			if err != nil {
				return nil, err
			}
		} else {
			err := mapstructure.Decode(operator.Spec.Components.Pilot.K8S.Resources.Requests, &res)
			if err != nil {
				return nil, err
			}
		}
		return &res, nil
	default:
		return nil, godog.ErrPending
	}
}

func enableIstioInjection(namespace string) error {
	var ns corev1.Namespace
	return retry.Do(func() error {
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: namespace}, &ns)
		if err != nil {
			return err
		}
		ns.Labels["istio-injection"] = "enabled"
		return k8sClient.Update(context.TODO(), &ns)
	}, retryOpts...)
}

func createApplicationDeployment(ctx context.Context, appName, namespace string) (context.Context, error) {
	dep := v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: namespace,
		},
		Spec: v1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": appName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": appName},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  appName,
							Image: "europe-docker.pkg.dev/kyma-project/prod/external/kennethreitz/httpbin",
						},
					},
				},
			},
		},
	}

	err := retry.Do(func() error {
		err := k8sClient.Create(context.TODO(), &dep)
		if err != nil {
			return err
		}
		ctx = setTestAppInContext(ctx, &dep)
		return nil
	}, retryOpts...)

	return ctx, err
}

func applicationHasProxyResourcesSetToCpuAndMemory(appName, appNamespace, resourceType, cpu, memory string) error {
	var podList corev1.PodList
	return retry.Do(func() error {
		err := k8sClient.List(context.TODO(), &podList, &client.ListOptions{
			Namespace: appNamespace,
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"app": appName,
			}),
		})
		if err != nil {
			return err
		}

		if len(podList.Items) == 0 {
			return fmt.Errorf("no pods found for app %s in namespace %s", appName, appNamespace)
		}

		hasResources := false
		for _, pod := range podList.Items {
			for _, container := range pod.Spec.Containers {
				if container.Name != "istio-proxy" {
					continue
				}

				switch resourceType {
				case "limits":
					hasResources = container.Resources.Limits.Cpu().String() == cpu &&
						container.Resources.Limits.Memory().String() == memory
				case "requests":
					hasResources = container.Resources.Requests.Cpu().String() == cpu &&
						container.Resources.Requests.Memory().String() == memory
				default:
					return fmt.Errorf("resource type %s is not supported", resourceType)

				}
			}
		}

		if !hasResources {
			return fmt.Errorf("the resources of proxy of app %s in namespace %s does not match %s cpu %s memory %s",
				appName, appNamespace, resourceType, cpu, memory)
		}

		return nil
	}, retryOpts...)
}
