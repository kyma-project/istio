package steps

import (
	"context"
	"fmt"

	"github.com/kyma-project/istio/operator/tests/testcontext"

	"github.com/avast/retry-go"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kyma-project/istio/operator/tests/integration/pkg/ip"
	"github.com/kyma-project/istio/operator/tests/integration/testsupport"
)

// ValidateHeader validates that the header givenHeaderName with value givenHeaderValue is forwarded to the application as header expectedHeaderName with the value expectedHeaderValue.
func ValidateHeader(ctx context.Context, givenHeaderName, givenHeaderValue, expectedHeaderName, expectedHeaderValue string) (context.Context, error) {
	ingressAddress, err := fetchIstioIngressGatewayAddress(ctx)
	if err != nil {
		return ctx, err
	}

	c := testsupport.NewHTTPClientWithRetry()
	headers := map[string]string{
		givenHeaderName: givenHeaderValue,
	}
	url := fmt.Sprintf("http://%s/get?show_env=true", ingressAddress)
	asserter := testsupport.BodyContainsAsserter{
		Expected: []string{
			fmt.Sprintf(`"%s": "%s"`, expectedHeaderName, expectedHeaderValue),
		},
	}

	return ctx, c.GetWithHeaders(url, headers, asserter)
}

// ValidateHeaderInBody validates that the header givenHeaderName with value givenHeaderValue is contained in the body of the response.
func ValidateHeaderInBody(ctx context.Context, path string, expectedHeaderName, expectedHeaderValue string) (context.Context, error) {
	ingressAddress, err := fetchIstioIngressGatewayAddress(ctx)
	if err != nil {
		return ctx, err
	}

	c := testsupport.NewHTTPClientWithRetry()
	url := fmt.Sprintf("http://%s%s", ingressAddress, path)
	asserter := testsupport.BodyContainsAsserter{
		Expected: []string{
			fmt.Sprintf(`"%s": "%s"`, expectedHeaderName, expectedHeaderValue),
		},
	}

	return ctx, c.Get(url, asserter)
}

// ValidateResponseStatusCode validates that the response status code is the expected one.
func ValidateResponseStatusCode(ctx context.Context, path string, expectedCode int) (context.Context, error) {
	ingressAddress, err := fetchIstioIngressGatewayAddress(ctx)
	if err != nil {
		return ctx, err
	}

	c := testsupport.NewHTTPClientWithRetry()
	url := fmt.Sprintf("http://%s%s", ingressAddress, path)
	asserter := testsupport.ResponseStatusCodeAsserter{
		Code: expectedCode,
	}

	return ctx, c.Get(url, asserter)
}

func ValidateResponseCodeForRequestWithHeader(ctx context.Context, givenHeaderName, givenHeaderValue, path string, expectedCode int) (context.Context, error) {
	ingressAddress, err := fetchIstioIngressGatewayAddress(ctx)
	if err != nil {
		return ctx, err
	}

	c := testsupport.NewHTTPClientWithRetry()
	headers := map[string]string{
		givenHeaderName: givenHeaderValue,
	}
	url := fmt.Sprintf("http://%s%s", ingressAddress, path)
	asserter := testsupport.ResponseStatusCodeAsserter{
		Code: expectedCode,
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

	var ingressIP string
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
			}
			lbIP, err := ip.GetLoadBalancerIp(svc.Status.LoadBalancer.Ingress[0])
			if err != nil {
				return err
			}

			ingressIP = lbIP.String()

			for _, port := range svc.Spec.Ports {
				if port.Name == "http2" {
					ingressPort = port.Port
				}
			}
			return nil
		}
		// In case we are not running on Gardener we assume that it's a k3d cluster that has 127.0.0.1 as default address
		ingressIP = "localhost"
		ingressPort = 80

		return nil
	}, testcontext.GetRetryOpts()...)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%d", ingressIP, ingressPort), nil
}
