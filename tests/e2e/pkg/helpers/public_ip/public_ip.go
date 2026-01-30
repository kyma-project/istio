package public_ip

import (
	"io"
	"net/http"
	"testing"
)

// FetchPublicIP returns the public IP address of the caller by using the ipify API.
func FetchPublicIP(t *testing.T) (string, error) {
	url := "https://api.ipify.org?format=text"
	t.Logf("Getting IP address of client from  ipify ...")
	resp, err := http.Get(url)
	if err != nil {
		t.Logf("Failed to fetch public IP of client: %v", err)
		return "", err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("Failed to read response body: %v", err)
		return "", err
	}

	return string(ip), nil
}
