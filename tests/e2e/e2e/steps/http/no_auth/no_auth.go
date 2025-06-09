package no_auth

import (
	"context"
	"log"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync/atomic"
)

type Request struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`

	Response atomic.Pointer[http.Response] `json:"-"`
}

func (r *Request) Description() string {
	return "HTTP Request: " + r.Method + " " + r.URL
}

func (r *Request) Execute(_ context.Context, _ client.Client, debugLogger *log.Logger) error {
	debugLogger.Printf("Executing HTTP request: %s %s", r.Method, r.URL)
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

	debugLogger.Printf("Received response: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	r.Response.Store(resp)
	return nil
}

func (r *Request) Cleanup(context.Context, client.Client) error {
	return nil
}
