package istio_resources

import (
	"bytes"
	"context"
	_ "embed"
	"github.com/kyma-project/istio/operator/internal/resources"
	"text/template"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//go:embed virtual_service_healthz.yaml
var manifest_vs_healthz []byte

type VirtualServiceHealthz struct {
	k8sClient client.Client
}

func NewVirtualServiceHealthz(k8sClient client.Client) VirtualServiceHealthz {
	return VirtualServiceHealthz{k8sClient: k8sClient}
}

func (VirtualServiceHealthz) apply(ctx context.Context, k8sClient client.Client, _ metav1.OwnerReference, templateValues map[string]string) (controllerutil.OperationResult, error) {
	resourceTemplate, err := template.New("tmpl").Option("missingkey=error").Parse(string(manifest_vs_healthz))
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	var resourceBuffer bytes.Buffer
	err = resourceTemplate.Execute(&resourceBuffer, templateValues)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	return resources.ApplyResource(ctx, k8sClient, resourceBuffer.Bytes(), nil)
}

func (VirtualServiceHealthz) Name() string {
	return "VirtualService/istio-healthz"
}
