package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/kyma-project/istio/operator/tests/integration/pkg/ip"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"log"
	"net/http"
)

// ValidatePublicClientIpInHeader validates that the header expectedHeaderName contains the public client IP.
func ValidatePublicClientIpInHeader(ctx context.Context, expectedHeaderName string) (context.Context, error) {
	return ctx, retry.Do(func() error {
		clientIp, err := ip.FetchPublic()
		if err != nil {
			return err
		}

		log.Printf("Public IP address of the caller: %s\n", clientIp)

		ingressAddress, err := fetchIstioIngressGatewayAddress(ctx)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("http://%s/get?show_env=true", ingressAddress)

		r, err := http.Get(url)
		if err != nil {
			return err
		}

		var resp map[string]interface{}
		err = json.NewDecoder(r.Body).Decode(&resp)
		if err != nil {
			return err
		}
		defer func() {
			err := r.Body.Close()
			if err != nil {
				log.Printf("Failed to close response body: %s", err)
			}
		}()

		hv := fmt.Sprintf("%v", resp["headers"].(map[string]interface{})[expectedHeaderName])
	
		if hv != clientIp {
			return fmt.Errorf("expected header %s to contain %s, but got %s", expectedHeaderName, clientIp, hv)
		}

		return nil
	}, testcontext.GetRetryOpts()...)
}
