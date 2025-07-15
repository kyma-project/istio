package http

import (
	"net/http"
	"testing"
)

func NewHTTPClient(t *testing.T) *http.Client {
	return &http.Client{
		Transport: TestLogTransportWrapper(t, "thttp", http.DefaultTransport),
	}
}

type RoundTripFunc func(*http.Request) (*http.Response, error)

func (fn RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestLogTransportWrapper(t *testing.T, prefix string, rt http.RoundTripper) RoundTripFunc {
	return func(req *http.Request) (*http.Response, error) {
		t.Logf("[%s] request method: %s, url: %s", prefix, req.Method, req.URL)
		resp, err := rt.RoundTrip(req)
		if err != nil {
			t.Logf("[%s] request error: method: %s, url: %s, err: %v", prefix, req.Method, req.URL, err)
			return nil, err
		}
		t.Logf("[%s] response: %d %s", prefix, resp.StatusCode, http.StatusText(resp.StatusCode))
		return resp, nil
	}
}
