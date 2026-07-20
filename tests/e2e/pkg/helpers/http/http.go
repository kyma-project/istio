package httphelper

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
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
	// Network, when set, forces the TCP family used for dialling. Values:
	// "tcp4" (IPv4 only), "tcp6" (IPv6 only). Empty means default ("tcp",
	// resolver-luck). Pair with ipfamily.From().DialNetworks().
	Network string
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

// WithNetwork pins the TCP family for dialing. Use "tcp4" or "tcp6". An
// empty value (the default) leaves family selection to the Go resolver.
func WithNetwork(network string) Option {
	return func(o *Options) {
		o.Network = network
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

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if opts.Network != "" {
		dialer := &net.Dialer{Timeout: 30 * time.Second}
		transport.DialContext = func(ctx context.Context, _network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, opts.Network, addr)
		}
	}
	client := &http.Client{
		Transport: TestLogTransportWrapper(t, opts.Prefix, opts.Host, opts.Headers, transport),
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
	SuppressTestLog bool
	Output          io.Writer
	outputMu        sync.Mutex
}

type TestLogTransportOption func(*TestLogTransportWrapperOptions)

func SuppressTestLog() TestLogTransportOption {
	return func(o *TestLogTransportWrapperOptions) {
		o.SuppressTestLog = true
	}
}

func WithOutput(output io.Writer) TestLogTransportOption {
	return func(o *TestLogTransportWrapperOptions) {
		o.Output = output
	}
}

func logfWithOptions(t *testing.T, prefix string, opts *TestLogTransportWrapperOptions, format string, args ...interface{}) {
	sbuilder := &strings.Builder{}
	sbuilder.WriteString(fmt.Sprintf("[%s] ", prefix))
	sbuilder.WriteString(fmt.Sprintf(format, args...))
	toLog := sbuilder.String()

	if !opts.SuppressTestLog {
		t.Log(toLog)
	}
	if opts.Output != nil {
		opts.outputMu.Lock()
		_, err := io.WriteString(opts.Output, toLog+"\n")
		opts.outputMu.Unlock()
		if err != nil {
			t.Logf("Warning: failed to write to output: %v", err)
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
