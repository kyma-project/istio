package infrastructure

import (
	"fmt"
	"testing"

	v2 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
)

const (
	IstioNamespace              = "istio-system"
	IstiodDeploymentName        = "istiod"
	IngressGatewayDeployment    = "istio-ingressgateway"
	EgressGatewayDeployment     = "istio-egressgateway"
	CniDaemonSetName            = "istio-cni-node"
	IstiodLabelSelector         = "app=istiod"
	IngressGatewayLabelSelector = "app=istio-ingressgateway"
	EgressGatewayLabelSelector  = "app=istio-egressgateway"
)

// GetIstiodDeployment returns the Istiod deployment from the istio-system namespace
func GetIstiodDeployment(t *testing.T) (*v2.Deployment, error) {
	t.Helper()
	c, err := client.ResourcesClient(t)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources client: %w", err)
	}

	deployment := &v2.Deployment{}
	err = c.Get(t.Context(), IstiodDeploymentName, IstioNamespace, deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to get istiod deployment: %w", err)
	}

	return deployment, nil
}

// GetIngressGatewayDeployment returns the Istio ingress gateway deployment from the istio-system namespace
func GetIngressGatewayDeployment(t *testing.T) (*v2.Deployment, error) {
	t.Helper()
	c, err := client.ResourcesClient(t)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources client: %w", err)
	}

	deployment := &v2.Deployment{}
	err = c.Get(t.Context(), IngressGatewayDeployment, IstioNamespace, deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to get istio-ingressgateway deployment: %w", err)
	}

	return deployment, nil
}

// GetEgressGatewayDeployment returns the Istio egress gateway deployment from the istio-system namespace
func GetEgressGatewayDeployment(t *testing.T) (*v2.Deployment, error) {
	t.Helper()
	c, err := client.ResourcesClient(t)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources client: %w", err)
	}

	deployment := &v2.Deployment{}
	err = c.Get(t.Context(), EgressGatewayDeployment, IstioNamespace, deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to get istio-egressgateway deployment: %w", err)
	}

	return deployment, nil
}

// GetCniDaemonSet returns the Istio CNI DaemonSet from the istio-system namespace
func GetCniDaemonSet(t *testing.T) (*v2.DaemonSet, error) {
	t.Helper()
	c, err := client.ResourcesClient(t)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources client: %w", err)
	}

	daemonSet := &v2.DaemonSet{}
	err = c.Get(t.Context(), CniDaemonSetName, IstioNamespace, daemonSet)
	if err != nil {
		return nil, fmt.Errorf("failed to get istio-cni-node daemonset: %w", err)
	}

	return daemonSet, nil
}

// GetIstiodPods returns all Istiod pods from the istio-system namespace
func GetIstiodPods(t *testing.T) (*v1.PodList, error) {
	t.Helper()
	c, err := client.ResourcesClient(t)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources client: %w", err)
	}

	podList := &v1.PodList{}
	err = c.List(t.Context(), podList, resources.WithLabelSelector(IstiodLabelSelector))
	if err != nil {
		return nil, fmt.Errorf("failed to list istiod pods: %w", err)
	}

	return podList, nil
}

// GetIngressGatewayPods returns all Istio ingress gateway pods from the istio-system namespace
func GetIngressGatewayPods(t *testing.T) (*v1.PodList, error) {
	t.Helper()
	c, err := client.ResourcesClient(t)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources client: %w", err)
	}

	podList := &v1.PodList{}
	err = c.List(t.Context(), podList, resources.WithLabelSelector(IngressGatewayLabelSelector))
	if err != nil {
		return nil, fmt.Errorf("failed to list istio-ingressgateway pods: %w", err)
	}

	return podList, nil
}

// GetEgressGatewayPods returns all Istio egress gateway pods from the istio-system namespace
func GetEgressGatewayPods(t *testing.T) (*v1.PodList, error) {
	t.Helper()
	c, err := client.ResourcesClient(t)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources client: %w", err)
	}

	podList := &v1.PodList{}
	err = c.List(t.Context(), podList, resources.WithLabelSelector(EgressGatewayLabelSelector))
	if err != nil {
		return nil, fmt.Errorf("failed to list istio-egressgateway pods: %w", err)
	}

	return podList, nil
}

// GetIngressGatewayService returns the Istio ingress gateway service from the istio-system namespace
func GetIngressGatewayService(t *testing.T) (*v1.Service, error) {
	t.Helper()
	c, err := client.ResourcesClient(t)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources client: %w", err)
	}

	service := &v1.Service{}
	err = c.Get(t.Context(), IngressGatewayDeployment, IstioNamespace, service)
	if err != nil {
		return nil, fmt.Errorf("failed to get istio-ingressgateway service: %w", err)
	}

	return service, nil
}

// GetIstioSystemNamespace returns the istio-system namespace
func GetIstioSystemNamespace(t *testing.T) (*v1.Namespace, error) {
	t.Helper()
	c, err := client.ResourcesClient(t)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources client: %w", err)
	}

	namespace := &v1.Namespace{}
	err = c.Get(t.Context(), IstioNamespace, "", namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get istio-system namespace: %w", err)
	}

	return namespace, nil
}
