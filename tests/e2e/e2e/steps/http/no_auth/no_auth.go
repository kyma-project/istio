package no_auth

import (
	"context"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/executor"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

type Request struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`

	Response *http.Response `json:"-"`
}

func (r *Request) Description() string {
	return "Making HTTP Request: " + r.Method + " " + r.URL
}

func (r *Request) Execute(t *testing.T, _ context.Context, _ client.Client) error {
	executor.Debugf(t, "Executing HTTP request: %s %s", r.Method, r.URL)
	req, err := http.NewRequest(r.Method, r.URL, nil)
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

	executor.Debugf(t, "Received response: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	r.Response = resp
	return nil
}

func (r *Request) Cleanup(*testing.T, context.Context, client.Client) error {
	return nil
}
