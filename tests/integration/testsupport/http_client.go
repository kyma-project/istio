package testsupport

import (
	"crypto/tls"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"github.com/pkg/errors"
	"net/http"
	"time"
)

type RetryableHttpClient struct {
	client *http.Client
	opts   []retry.Option
}

func NewHttpClientWithRetry() *RetryableHttpClient {

	c := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: time.Second * 10,
	}

	return &RetryableHttpClient{
		client: c,
		opts:   testcontext.GetRetryOpts(),
	}
}

// Get returns returns error if the validator returns false
func (h *RetryableHttpClient) Get(url string, validator HttpResponseAsserter) error {
	err := h.withRetries(func() (*http.Response, error) {
		return h.client.Get(url)
	}, validator)

	if err != nil {
		return fmt.Errorf("error calling endpoint %s err=%s", url, err)
	}

	return nil
}

// GetWithHeaders returns error if the validator returns false
func (h *RetryableHttpClient) GetWithHeaders(url string, requestHeaders map[string]string, asserter HttpResponseAsserter) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	for headerName, headerValue := range requestHeaders {
		req.Header.Set(headerName, headerValue)
	}

	err = h.withRetries(func() (*http.Response, error) {
		return h.client.Do(req)
	}, asserter)

	if err != nil {
		return fmt.Errorf("error calling endpoint %s err=%s", url, err)
	}

	return nil
}

func (h *RetryableHttpClient) withRetries(httpCall func() (*http.Response, error), asserter HttpResponseAsserter) error {

	if err := retry.Do(func() error {

		response, callErr := httpCall()
		if callErr != nil {
			return callErr
		}

		if isValid, failureMsg := asserter.Assert(*response); !isValid {
			return errors.New(failureMsg)
		}

		return nil
	},
		h.opts...,
	); err != nil {
		return err
	}

	return nil
}
