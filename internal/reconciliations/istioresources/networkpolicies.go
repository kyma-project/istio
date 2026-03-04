package istioresources

import (
	"bytes"
	"context"
	_ "embed"
	"strconv"

	"github.com/kyma-project/istio/operator/internal/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//go:embed networkpolicies/allow-cni.yaml
var allowCni []byte

//go:embed networkpolicies/allow-egress-to-customer.yaml
var allowEgressToCustomer []byte

//go:embed networkpolicies/allow-customer-to-egress.yaml
var allowCustomerToEgress []byte

//go:embed networkpolicies/allow-ingressgateway.yaml
var allowIngressGateway []byte

//go:embed networkpolicies/allow-istio-controller-manager.yaml
var allowIstioControllerManager []byte

//go:embed networkpolicies/allow-istiod.yaml
var allowIstiod []byte

//go:embed networkpolicies/allow-jwks.yaml
var allowJwks []byte

type NetworkPolicies struct {
	shouldDelete bool
}

func NewNetworkPolicies(shouldDelete bool) NetworkPolicies {
	return NetworkPolicies{
		shouldDelete: shouldDelete,
	}
}

func (NetworkPolicies) Name() string {
	return "NetworkPolicies"
}

const apiServerPortPlaceholder = "__API_SERVER_PORT__"

func (np NetworkPolicies) reconcile(ctx context.Context, k8sClient client.Client, _ metav1.OwnerReference, _ map[string]string) (controllerutil.OperationResult, error) {
	networkPoliciesManifests := [][]byte{
		allowCni,
		allowEgressToCustomer,
		allowCustomerToEgress,
		allowIngressGateway,
		allowIstioControllerManager,
		allowIstiod,
		allowJwks,
	}

	apiServerTargetPort := 443
	kubernetesSvc := &corev1.Service{}
	err := k8sClient.Get(ctx, client.ObjectKey{Name: "kubernetes", Namespace: "default"}, kubernetesSvc)
	if err == nil {
		for _, port := range kubernetesSvc.Spec.Ports {
			if port.Name == "https" {
				apiServerTargetPort = int(port.TargetPort.IntVal)
				break
			}
		}
	} else {
		ctrl.Log.Error(err, "Failed to get kubernetes service to determine API server target port, defaulting to 443")
	}

	endResult := controllerutil.OperationResultNone
	for _, resource := range networkPoliciesManifests {
		toApply := bytes.ReplaceAll(resource, []byte(apiServerPortPlaceholder), []byte(strconv.Itoa(apiServerTargetPort)))
		var result controllerutil.OperationResult
		if np.shouldDelete {
			r, err := resources.DeleteIfPresent(ctx, k8sClient, toApply)
			if err != nil {
				return r, err
			}
			result = r
		} else {
			r, err := resources.Apply(ctx, k8sClient, toApply, nil)
			if err != nil {
				return r, err
			}
			result = r
		}
		endResult = result
	}
	return endResult, nil
}
