package steps

import (
	"bytes"
	"context"
	"fmt"
	"github.com/avast/retry-go"
	istioCR "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/yaml"
	"text/template"
)

const templateFileName string = "manifests/istio_cr_template.yaml"

func IstioCRDIsInstalled(ctx context.Context) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	crdGVK := schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinition",
	}

	var crd unstructured.Unstructured
	crd.SetGroupVersionKind(crdGVK)
	return retry.Do(func() error {
		return k8sClient.Get(context.TODO(), types.NamespacedName{Name: "istios.operator.kyma-project.io"}, &crd)
	}, testcontext.GetRetryOpts()...)
}

func IstioCRInNamespaceHasStatus(ctx context.Context, name, namespace, status string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

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
	}, testcontext.GetRetryOpts()...)
}

type TemplatedIstioCr struct {
	templateValues map[string]string
}

func (t *TemplatedIstioCr) SetTemplateValue(name, value string) {
	if len(t.templateValues) == 0 {
		t.templateValues = make(map[string]string)
	}
	t.templateValues[name] = value
}

func (t *TemplatedIstioCr) IstioCRIsAppliedInNamespace(ctx context.Context, name, namespace string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	istio, err := createIstioCrFromTemplate(name, namespace, t.templateValues)
	if err != nil {
		return ctx, err
	}

	err = retry.Do(func() error {
		err := k8sClient.Create(context.TODO(), &istio)
		if err != nil {
			return err
		}
		ctx = testcontext.SetIstioCrInContext(ctx, &istio)
		return nil
	}, testcontext.GetRetryOpts()...)

	return ctx, err
}

func (t *TemplatedIstioCr) IstioCrIsUpdatedInNamespace(ctx context.Context, name, namespace string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	istio, err := createIstioCrFromTemplate(name, namespace, t.templateValues)
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
	}, testcontext.GetRetryOpts()...)
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
