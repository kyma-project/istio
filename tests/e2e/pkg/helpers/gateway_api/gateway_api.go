package gateway_api

import (
	"context"
	"fmt"
	"testing"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	k8sclient "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
)

const (
	// IstioGatewayClassName is the name of the GatewayClass registered by Istio
	IstioGatewayClassName = "istio"

	gatewayClassReadyTimeout  = 2 * time.Minute
	gatewayClassReadyInterval = 5 * time.Second
	gatewayAddressTimeout     = 5 * time.Minute
	gatewayAddressInterval    = 5 * time.Second
)

// WaitForGatewayClassReady waits until the "istio" GatewayClass is accepted by the controller.
func WaitForGatewayClassReady(t *testing.T) error {
	t.Helper()
	t.Log("Waiting for 'istio' GatewayClass to be ready")

	r, err := k8sclient.ResourcesClient(t)
	if err != nil {
		return fmt.Errorf("failed to get resources client: %w", err)
	}
	c := r.GetControllerRuntimeClient()

	return wait.PollUntilContextTimeout(t.Context(), gatewayClassReadyInterval, gatewayClassReadyTimeout, true,
		func(ctx context.Context) (bool, error) {
			var gc gatewayv1.GatewayClass
			if err := c.Get(ctx, types.NamespacedName{Name: IstioGatewayClassName}, &gc); err != nil {
				if k8serrors.IsNotFound(err) {
					t.Logf("GatewayClass '%s' not found yet, retrying...", IstioGatewayClassName)
					return false, nil
				}
				return false, err
			}

			for _, cond := range gc.Status.Conditions {
				if cond.Type == string(gatewayv1.GatewayClassConditionStatusAccepted) &&
					cond.Status == metav1.ConditionTrue {
					t.Logf("GatewayClass '%s' is accepted", IstioGatewayClassName)
					return true, nil
				}
			}
			t.Logf("GatewayClass '%s' not yet accepted, retrying...", IstioGatewayClassName)
			return false, nil
		})
}

// CreateGateway creates a gateway.networking.k8s.io/v1 Gateway that listens on HTTP port 80
// using the "istio" GatewayClass and registers cleanup.
func CreateGateway(t *testing.T, name, namespace string) error {
	t.Helper()
	t.Logf("Creating Gateway API Gateway %s/%s", namespace, name)

	r, err := k8sclient.ResourcesClient(t)
	if err != nil {
		return fmt.Errorf("failed to get resources client: %w", err)
	}
	c := r.GetControllerRuntimeClient()

	portNumber := gatewayv1.PortNumber(80)
	fromSame := gatewayv1.NamespacesFromSame
	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName(IstioGatewayClassName),
			Listeners: []gatewayv1.Listener{
				{
					Name:     "http",
					Port:     portNumber,
					Protocol: gatewayv1.HTTPProtocolType,
					AllowedRoutes: &gatewayv1.AllowedRoutes{
						Namespaces: &gatewayv1.RouteNamespaces{
							From: &fromSame,
						},
					},
				},
			},
		},
	}

	if err := c.Create(t.Context(), gw); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create Gateway: %w", err)
		}
		t.Logf("Gateway %s/%s already exists", namespace, name)
	} else {
		t.Logf("Gateway %s/%s created", namespace, name)
	}

	setup.DeclareCleanup(t, func() {
		if err := c.Delete(setup.GetCleanupContext(), gw); err != nil {
			t.Logf("Failed to delete Gateway %s/%s: %v", namespace, name, err)
		} else {
			t.Logf("Gateway %s/%s deleted", namespace, name)
		}
	})

	return nil
}

// CreateHTTPRoute creates a gateway.networking.k8s.io/v1 HTTPRoute that routes all traffic
// to the given backend service and registers cleanup.
func CreateHTTPRoute(t *testing.T, name, namespace, gatewayName, backendService string, backendPort int) error {
	t.Helper()
	t.Logf("Creating HTTPRoute %s/%s -> %s:%d via Gateway %s", namespace, name, backendService, backendPort, gatewayName)

	r, err := k8sclient.ResourcesClient(t)
	if err != nil {
		return fmt.Errorf("failed to get resources client: %w", err)
	}
	c := r.GetControllerRuntimeClient()

	port := gatewayv1.PortNumber(backendPort)
	ns := gatewayv1.Namespace(namespace)
	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name:      gatewayv1.ObjectName(gatewayName),
						Namespace: &ns,
					},
				},
			},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: gatewayv1.ObjectName(backendService),
									Port: &port,
								},
							},
						},
					},
				},
			},
		},
	}

	if err := c.Create(t.Context(), route); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create HTTPRoute: %w", err)
		}
		t.Logf("HTTPRoute %s/%s already exists", namespace, name)
	} else {
		t.Logf("HTTPRoute %s/%s created", namespace, name)
	}

	setup.DeclareCleanup(t, func() {
		if err := c.Delete(setup.GetCleanupContext(), route); err != nil {
			t.Logf("Failed to delete HTTPRoute %s/%s: %v", namespace, name, err)
		} else {
			t.Logf("HTTPRoute %s/%s deleted", namespace, name)
		}
	})

	return nil
}

// GetGatewayAddress waits for the Gateway to receive an address from the controller
// and returns it as "ip:port" (or "hostname:port") on port 80.
func GetGatewayAddress(t *testing.T, name, namespace string) (string, error) {
	t.Helper()
	t.Logf("Waiting for Gateway %s/%s to receive an address", namespace, name)

	r, err := k8sclient.ResourcesClient(t)
	if err != nil {
		return "", fmt.Errorf("failed to get resources client: %w", err)
	}
	c := r.GetControllerRuntimeClient()

	var addr string
	err = wait.PollUntilContextTimeout(t.Context(), gatewayAddressInterval, gatewayAddressTimeout, true,
		func(ctx context.Context) (bool, error) {
			var gw gatewayv1.Gateway
			if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &gw); err != nil {
				if k8serrors.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}

			if len(gw.Status.Addresses) == 0 {
				t.Logf("Gateway %s/%s has no addresses yet, retrying...", namespace, name)
				return false, nil
			}

			addr = fmt.Sprintf("%s:80", gw.Status.Addresses[0].Value)
			t.Logf("Gateway %s/%s address: %s", namespace, name, addr)
			return true, nil
		})

	return addr, err
}