package steps

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"text/template"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/avast/retry-go"
	istioCR "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
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
			return fmt.Errorf("status %s of Istio CR is not equal to %s\n Description: %s", cr.Status.State, status, cr.Status.Description)
		}
		return nil
	}, testcontext.GetRetryOpts()...)
}

func IstioCRInNamespaceHasStatusCondition(ctx context.Context, name, namespace, reason, conditionType, status string) error {
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
		found := false
		for _, condition := range *cr.Status.Conditions {
			if condition.Reason == reason && condition.Type == conditionType && string(condition.Status) == status {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("status condition reason %s (type: %s, status: %s) not found", reason, conditionType, status)
		}

		return nil
	}, testcontext.GetRetryOpts()...)
}

func IstioCRInNamespaceHasDescription(ctx context.Context, name, namespace, desc string) error {
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
		if cr.Status.Description != desc {
			return fmt.Errorf("description %s of Istio CR is not equal to %s", cr.Status.Description, desc)
		}
		return nil
	}, testcontext.GetRetryOpts()...)
}

func IstioCRInNamespaceHasConditionMessage(ctx context.Context, name, namespace, message string) error {
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
		var conditionMessages []string
		for _, condition := range *cr.Status.Conditions {
			if condition.Message == message {
				return nil
			} else {
				// TODO: Should we fail here as there is a different condition than expected?
				conditionMessages = append(conditionMessages, condition.Message)
			}
		}

		return fmt.Errorf("condition messages %s of Istio CR is not equal to %s", conditionMessages, message)
	}, testcontext.GetRetryOpts()...)
}

func IstioCRInNamespaceHasConditionReason(ctx context.Context, name, namespace, reason string) error {
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
		var conditionMessages []string
		for _, condition := range *cr.Status.Conditions {
			if condition.Reason == reason {
				return nil
			} else {
				// TODO: Should we fail here as there is a different condition than expected?
				conditionMessages = append(conditionMessages, condition.Reason)
			}
		}

		return fmt.Errorf("condition messages %s of Istio CR is not equal to %s", conditionMessages, reason)
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

	istio, err := createIstioCRFromTemplate(name, namespace, t.templateValues)
	if err != nil {
		return ctx, err
	}

	err = retry.Do(func() error {
		err := k8sClient.Create(context.TODO(), &istio)
		if err != nil {
			return err
		}
		ctx = testcontext.AddIstioCRIntoContext(ctx, &istio)
		return nil
	}, testcontext.GetRetryOpts()...)

	return ctx, err
}

func (t *TemplatedIstioCr) IstioCRIsUpdatedInNamespace(ctx context.Context, name, namespace string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	istio, err := createIstioCRFromTemplate(name, namespace, t.templateValues)
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

func createIstioCRFromTemplate(name string, namespace string, templateValues map[string]string) (istioCR.Istio, error) {
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

func IstioCrStatusUpdateHappened(ctx context.Context, name, namespace string) error {
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

		for _, field := range cr.ManagedFields {

			// We only consider an update of the status owned by the manager as relevant, since we want to verify the manager is reconciling the CR.
			if field.Subresource == "status" && field.Manager == "manager" && field.Operation == metav1.ManagedFieldsOperationUpdate {
				timestamp, err := time.Parse(time.RFC3339, field.Time.UTC().Format(time.RFC3339))
				if err != nil {
					return err
				}

				// Check if the operation occurred within the last 20 seconds.
				if time.Since(timestamp).Seconds() <= 20 {
					return nil
				}
			}
		}

		return fmt.Errorf("no server-side update occurred for the CR '%s' within the last 20 seconds", name)
	}, testcontext.GetRetryOpts()...)
}
