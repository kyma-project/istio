package load_balancer

import (
	"context"
	"fmt"
	"net"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetLoadBalancerIP(ctx context.Context, c client.Client) (string, error) {
	istioIngressGatewayNamespaceName := types.NamespacedName{
		Name:      "istio-ingressgateway",
		Namespace: "istio-system",
	}

	var ingressIp string
	var ingressPort int32

	runsOnGardener, err := runsOnGardener(ctx, c)
	if err != nil {
		return "", err
	}

	if runsOnGardener {
		var svc corev1.Service

		// Wait for the LoadBalancer to be provisioned (up to 5 minutes)
		err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
			if err := c.Get(ctx, istioIngressGatewayNamespaceName, &svc); err != nil {
				if k8serrors.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}

			if len(svc.Status.LoadBalancer.Ingress) > 0 {
				ingress := svc.Status.LoadBalancer.Ingress[0]
				if ingress.IP != "" || ingress.Hostname != "" {
					return true, nil
				}
			}

			return false, nil
		})

		if err != nil {
			return "", fmt.Errorf("failed to wait for LoadBalancer to be provisioned: %w", err)
		}

		// Retry DNS resolution as DNS propagation might take time
		var lbIp net.IP
		err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
			ip, err := getLoadBalancerIp(svc.Status.LoadBalancer.Ingress[0])
			if err != nil {
				// Continue retrying on DNS resolution failures
				return false, nil
			}
			lbIp = ip
			return true, nil
		})

		if err != nil {
			return "", fmt.Errorf("failed to resolve LoadBalancer IP/hostname after retries: %w", err)
		}

		ingressIp = lbIp.String()

		for _, port := range svc.Spec.Ports {
			if port.Name == "http2" {
				ingressPort = port.Port
			}
		}
	} else {
		// In case we are not running on Gardener we assume that it's a k3d cluster that has 127.0.0.1 as default address
		ingressIp = "localhost"
		ingressPort = 80
	}

	return fmt.Sprintf("%s:%d", ingressIp, ingressPort), nil
}

func runsOnGardener(ctx context.Context, k8sClient client.Client) (bool, error) {
	cmShootInfo := corev1.ConfigMap{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "kube-system", Name: "shoot-info"}, &cmShootInfo)

	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func getLoadBalancerIp(loadBalancerIngress corev1.LoadBalancerIngress) (net.IP, error) {
	loadBalancerIP, err := getIpBasedLoadBalancerIp(loadBalancerIngress)

	if err == nil {
		return loadBalancerIP, nil
	}

	return getDnsBasedLoadBalancerIp(loadBalancerIngress)
}

func getIpBasedLoadBalancerIp(lbIngress corev1.LoadBalancerIngress) (net.IP, error) {
	ip := net.ParseIP(lbIngress.IP)
	if ip == nil {
		return nil, fmt.Errorf("failed to parse IP from load balancer IP %s", lbIngress.IP)
	}

	return ip, nil
}

func getDnsBasedLoadBalancerIp(lbIngress corev1.LoadBalancerIngress) (net.IP, error) {
	ips, err := net.LookupIP(lbIngress.Hostname)
	if err != nil || len(ips) < 1 {
		return nil, fmt.Errorf("could not get IPs by load balancer hostname: %v", err)
	}

	return ips[0], nil
}
