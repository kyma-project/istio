package steps

import (
	"context"
	"fmt"
	"github.com/kyma-project/istio/operator/tests/integration/pkg/ip"
	"github.com/kyma-project/istio/operator/tests/integration/testsupport"
	"log"
)

// ValidatePublicClientIpInHeader validates that the header expectedHeaderName contains the public client IP.
func ValidatePublicClientIpInHeader(ctx context.Context, host, expectedHeaderName string) (context.Context, error) {

	clientIp, err := ip.FetchPublic()
	if err != nil {
		return ctx, err
	}

	log.Printf("Public IP address of the caller: %s\n", clientIp)

	ingressAddress, err := fetchIstioIngressGatewayAddress(ctx)
	if err != nil {
		return ctx, err
	}

	c := testsupport.NewHttpClientWithRetry()
	url := fmt.Sprintf("http://%s/get?show_env=true", ingressAddress)

	asserter := testsupport.BodyContainsAsserter{
		Expected: []string{
			fmt.Sprintf(`"%s": "%s"`, expectedHeaderName, clientIp),
		},
	}
	return ctx, c.GetWithHeaders(url, map[string]string{"Host": host}, asserter)
}
