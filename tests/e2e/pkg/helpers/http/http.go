package httphelper

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"
)

type Options struct {
	Prefix  string
	Host    string
	Headers map[string]string
	Timeout time.Duration
}

type Option func(*Options)

func WithPrefix(prefix string) Option {
	return func(o *Options) {
		o.Prefix = prefix
	}
}

func WithHost(host string) Option {
	return func(o *Options) {
		o.Host = host
	}
}

func WithHeaders(headers map[string]string) Option {
	return func(o *Options) {
		o.Headers = headers
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.Timeout = timeout
	}
}

func NewHTTPClient(t *testing.T, options ...Option) *http.Client {
	t.Helper()
	opts := &Options{
		Prefix: "http-test-client",
	}
	for _, opt := range options {
		opt(opts)
	}

	transport := http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: TestLogTransportWrapper(t, opts.Prefix, opts.Host, opts.Headers, &transport),
	}
	if opts.Timeout > 0 {
		client.Timeout = opts.Timeout
	}
	return client
}

type RoundTripFunc func(*http.Request) (*http.Response, error)

func (fn RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestLogTransportWrapper(t *testing.T, prefix string, host string, headers map[string]string, rt http.RoundTripper) RoundTripFunc {
	return func(req *http.Request) (*http.Response, error) {
		// Set Host header if specified
		if host != "" {
			req.Host = host
		}
		t.Logf("[%s] request Host header set to: %s", prefix, req.Host)

		// Set custom headers if specified
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		t.Logf("[%s] request method: %s, url: %s, host: %s", prefix, req.Method, req.URL, req.Host)
		t.Logf("[%s] request headers: %v", prefix, req.Header)

		resp, err := rt.RoundTrip(req)
		if err != nil {
			t.Logf("[%s] request error: method: %s, url: %s, err: %v", prefix, req.Method, req.URL, err)
			return nil, err
		}
		t.Logf("[%s] response: %d %s", prefix, resp.StatusCode, http.StatusText(resp.StatusCode))
		return resp, nil
	}
}
