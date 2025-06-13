package noauth

import (
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/logging"
	"net/http"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Request struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`

	Response     *http.Response `json:"-"`
	ResponseBody []byte         `json:"response_body,omitempty"`
}

func (r *Request) Description() string {
	return "Making HTTP Request: " + r.Method + " " + r.URL
}

func (r *Request) Execute(t *testing.T, _ client.Client) error {
	logging.Debugf(t, "Executing HTTP request: %s %s", r.Method, r.URL)
	req, err := http.NewRequestWithContext(t.Context(), r.Method, r.URL, nil)
	if err != nil {
		return err
	}

	for key, value := range r.Headers {
		req.Header.Set(key, value)
	}

	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logging.Errorf(t, "Failed to close response body: %v", closeErr)
		}
	}()

	logging.Debugf(t, "Received response: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	r.Response = resp
	_, err = resp.Body.Read(r.ResponseBody)
	if err != nil {
		return err
	}
	return nil
}

func (r *Request) Cleanup(*testing.T, client.Client) error {
	return nil
}
