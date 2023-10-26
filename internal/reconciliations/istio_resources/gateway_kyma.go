package istio_resources

import (
	"bytes"
	"context"
	_ "embed"
	"github.com/kyma-project/istio/operator/internal/resources"
	"text/template"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

//go:embed gateway_kyma.yaml
var manifest_gateway_kyma []byte

type GatewayKyma struct {
	k8sClient client.Client
}

func NewGatewayKyma(k8sClient client.Client) GatewayKyma {
	return GatewayKyma{k8sClient: k8sClient}
}

func (GatewayKyma) apply(ctx context.Context, k8sClient client.Client, _ metav1.OwnerReference, templateValues map[string]string) (controllerutil.OperationResult, error) {
	resourceTemplate, err := template.New("tmpl").Option("missingkey=error").Parse(string(manifest_gateway_kyma))
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	var resourceBuffer bytes.Buffer
	err = resourceTemplate.Execute(&resourceBuffer, templateValues)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	var resource unstructured.Unstructured
	err = yaml.Unmarshal(resourceBuffer.Bytes(), &resource)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	spec := resource.Object["spec"]
	result, err := controllerutil.CreateOrUpdate(ctx, k8sClient, &resource, func() error {
		resource.Object["spec"] = spec
		return nil
	})
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	var daFound bool
	if resource.GetAnnotations() != nil {
		_, daFound = resource.GetAnnotations()[resources.DisclaimerKey]
	}
	if !daFound {
		err := resources.AnnotateWithDisclaimer(ctx, resource, k8sClient)
		if err != nil {
			return controllerutil.OperationResultNone, err
		}
	}

	return result, nil
}

func (GatewayKyma) Name() string {
	return "Gateway/kyma-gateway"
}
