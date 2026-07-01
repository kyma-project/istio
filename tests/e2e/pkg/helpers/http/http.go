package httphelper

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

type TestLogTransportWrapperOptions struct {
	DisableTLog       bool
	AdditionalOutputs io.Writer
}

type TestLogTransportOption func(*TestLogTransportWrapperOptions)

func DisableTLog() TestLogTransportOption {
	return func(o *TestLogTransportWrapperOptions) {
		o.DisableTLog = true
	}
}

func WithAdditionalOutputs(outputs io.Writer) TestLogTransportOption {
	return func(o *TestLogTransportWrapperOptions) {
		o.AdditionalOutputs = outputs
	}
}

func logfWithOptions(t *testing.T, prefix string, opts *TestLogTransportWrapperOptions, format string, args ...interface{}) {
	sbuilder := &strings.Builder{}
	sbuilder.WriteString(fmt.Sprintf("[%s] ", prefix))
	sbuilder.WriteString(fmt.Sprintf(format, args...))
	toLog := sbuilder.String()

	if !opts.DisableTLog {
		t.Log(toLog)
	}
	if opts.AdditionalOutputs != nil {
		if _, err := io.WriteString(opts.AdditionalOutputs, toLog+"\n"); err != nil {
			t.Logf("Warning: failed to write to additional output: %v", err)
		}
	}
}

func TestLogTransportWrapper(t *testing.T, prefix string, host string, headers map[string]string, rt http.RoundTripper, option ...TestLogTransportOption) RoundTripFunc {
	opts := &TestLogTransportWrapperOptions{}
	for _, opt := range option {
		opt(opts)
	}

	return func(req *http.Request) (*http.Response, error) {
		// Set Host header if specified
		if host != "" {
			req.Host = host
		}

		logfWithOptions(t, prefix, opts, "request Host header set to: %s", req.Host)

		// Set custom headers if specified
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		logfWithOptions(t, prefix, opts, "request method: %s, url: %s, host: %s", req.Method, req.URL, req.Host)
		logfWithOptions(t, prefix, opts, "request headers: %v", req.Header)

		resp, err := rt.RoundTrip(req)
		if err != nil {
			logfWithOptions(t, prefix, opts, "request failed; method: %s, url: %s, err: %v", req.Method, req.URL, err)
			return nil, err
		}
		logfWithOptions(t, prefix, opts, "received response; status code: %d, status text: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
		return resp, nil
	}
}

const (
	artifactBaseDir = "test-artifacts"
	httpLogsDir     = "http_logs"
)

var (
	testRunTimestamp string
	timestampOnce    sync.Once
)

func getTestRunTimestamp() string {
	timestampOnce.Do(func() {
		testRunTimestamp = time.Now().Format("02_01_2006-15_04_05CET")
	})
	return testRunTimestamp
}

func sanitizePathComponent(name string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_",
		"|", "_", " ", "_", "(", "", ")", "", ",", "",
	)
	return replacer.Replace(name)
}

func OpenTestArtifactLog(t *testing.T, name string) io.Writer {
	t.Helper()

	dir := filepath.Join(".", artifactBaseDir, getTestRunTimestamp(), sanitizePathComponent(t.Name()), httpLogsDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Logf("Warning: failed to create artifact dir %s: %v", dir, err)
		return nil
	}

	filePath := filepath.Join(dir, sanitizePathComponent(name)+".log")
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Logf("Warning: failed to open artifact log %s: %v", filePath, err)
		return nil
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Logf("Warning: failed to close artifact log %s: %v", filePath, err)
		}
	})
	return f
}
