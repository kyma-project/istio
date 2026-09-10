package public_ip

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"
)

// FetchPublicIP returns the caller's public IP for the given dial network
// ("tcp4" or "tcp6"). It queries ipify's family-specific endpoint over the
// matching socket family so the returned address is what the LB will see
// as X-Forwarded-For for a request dialled on `network`. Empty or unknown
// `network` falls back to the family-agnostic api.ipify.org and lets Go's
// resolver decide — matching the pre-dualstack behaviour.
func FetchPublicIP(t *testing.T, network string) (string, error) {
	t.Helper()

	url := endpointFor(network)
	t.Logf("Getting IP address of client from ipify (network=%q, url=%s) ...", network, url)

	client := httphelper.NewHTTPClient(t,
		httphelper.WithPrefix("public-ip-"+network),
		httphelper.WithNetwork(network),
		httphelper.WithTimeout(15*time.Second),
	)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build ipify request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Failed to fetch public IP of client: %v", err)
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ipify returned %d %s", resp.StatusCode, resp.Status)
	}

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("Failed to read response body: %v", err)
		return "", err
	}
	return strings.TrimSpace(string(ip)), nil
}

// ipify family-specific endpoints. `api4`/`api6` pin the resolver to a
// single family so the connection can't fall through to the other side of
// a dualstack host; `api` lets the resolver decide and is used only when
// the caller does not specify a network.
const (
	ipifyURLv4      = "https://api4.ipify.org?format=text"
	ipifyURLv6      = "https://api6.ipify.org?format=text"
	ipifyURLDefault = "https://api.ipify.org?format=text"
)

// endpointFor maps a dial network to ipify's family-specific host.
func endpointFor(network string) string {
	switch network {
	case "tcp4":
		return ipifyURLv4
	case "tcp6":
		return ipifyURLv6
	default:
		return ipifyURLDefault
	}
}
