package istio_resources

import (
	"context"
	_ "embed"

	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

//go:embed peer_authentication_mtls.yaml
var manifest_pa_mtls []byte

type PeerAuthenticationMtls struct {
	k8sClient client.Client
}

func NewPeerAuthenticationMtls(k8sClient client.Client) PeerAuthenticationMtls {
	return PeerAuthenticationMtls{k8sClient: k8sClient}
}

func (PeerAuthenticationMtls) apply(ctx context.Context, k8sClient client.Client, _ metav1.OwnerReference, _ map[string]string) (controllerutil.OperationResult, error) {
	var resource unstructured.Unstructured
	err := yaml.Unmarshal(manifest_pa_mtls, &resource)
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
		_, daFound = resource.GetAnnotations()[istio.DisclaimerKey]
	}
	if !daFound {
		err := annotateWithDisclaimer(ctx, resource, k8sClient)
		if err != nil {
			return controllerutil.OperationResultNone, err
		}
	}

	return result, nil
}

func (PeerAuthenticationMtls) Name() string {
	return "peer authentication mtls"
}
