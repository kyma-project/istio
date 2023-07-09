package steps

import (
	"context"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"github.com/kyma-project/istio/operator/tests/integration/testsupport"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ValidateExternalAddressForwarding validates that the expectedExternalAddress in the X-Forwarded-For header value is forwarded to the application as X-Envoy-External-Address header.
func ValidateExternalAddressForwarding(ctx context.Context, givenXForwardedForValue, expectedExternalAddress string) (context.Context, error) {

	ingressAddress, err := fetchIstioIngressGatewayAddress(ctx)
	if err != nil {
		return ctx, err
	}

	c := testsupport.NewHttpClientWithRetry()
	headers := map[string]string{
		"X-Forwarded-For": givenXForwardedForValue,
	}
	url := fmt.Sprintf("http://%s/get?show_env=true", ingressAddress)
	asserter := testsupport.BodyContainsAsserter{
		Expected: []string{
			fmt.Sprintf(`"X-Envoy-External-Address": "%s"`, expectedExternalAddress),
		},
	}

	return ctx, c.GetWithHeaders(url, headers, asserter)
}

func fetchIstioIngressGatewayAddress(ctx context.Context) (string, error) {

	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return "", err
	}

	istioIngressGatewayNamespaceName := types.NamespacedName{
		Name:      "istio-ingressgateway",
		Namespace: "istio-system",
	}

	var ingressIp string
	var ingressPort int32

	err = retry.Do(func() error {

		runsOnGardener, err := testsupport.RunsOnGardener(ctx, k8sClient)
		if err != nil {
			return err
		}

		if runsOnGardener {
			svc := corev1.Service{}
			if err := k8sClient.Get(context.TODO(), istioIngressGatewayNamespaceName, &svc); err != nil {
				return err
			}

			if len(svc.Status.LoadBalancer.Ingress) == 0 {
				return errors.New("no ingress ip found")
			} else {
				ingressIp = svc.Status.LoadBalancer.Ingress[0].IP
				for _, port := range svc.Spec.Ports {
					if port.Name == "http2" {
						ingressPort = port.Port
					}
				}
				return nil
			}
		} else {
			// In case we are not running on Gardener we assume that it's a k3d cluster that has 127.0.0.1 as default address
			ingressIp = "localhost"
			ingressPort = 80
		}

		return nil
	}, testcontext.GetRetryOpts()...)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%d", ingressIp, ingressPort), nil
}
