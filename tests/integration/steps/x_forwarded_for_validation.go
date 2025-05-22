package steps

import (
	"context"
	"fmt"
	"log"

	"github.com/kyma-project/istio/operator/tests/integration/pkg/ip"
	"github.com/kyma-project/istio/operator/tests/integration/testsupport"
)

// ValidatePublicClientIpInHeader validates that the header expectedHeaderName contains the public client IP.
func ValidatePublicClientIpInHeader(ctx context.Context, expectedHeaderName string) (context.Context, error) {
	clientIP, err := ip.FetchPublic()
	if err != nil {
		return ctx, err
	}

	log.Printf("Public IP address of the caller: %s\n", clientIP)

	ingressAddress, err := fetchIstioIngressGatewayAddress(ctx)
	if err != nil {
		return ctx, err
	}

	c := testsupport.NewHTTPClientWithRetry()
	url := fmt.Sprintf("http://%s/get?show_env=true", ingressAddress)

	asserter := testsupport.BodyContainsAsserter{
		Expected: []string{
			fmt.Sprintf(`"%s": "%s"`, expectedHeaderName, clientIP),
		},
	}

	return ctx, c.Get(url, asserter)
}
