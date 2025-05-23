package testsupport

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/avast/retry-go"
	"github.com/pkg/errors"

	"github.com/kyma-project/istio/operator/tests/testcontext"
)

type RetryableHTTPClient struct {
	client *http.Client
	opts   []retry.Option
}

func NewHTTPClientWithRetry() *RetryableHTTPClient {
	c := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: time.Second * 10,
	}

	return &RetryableHTTPClient{
		client: c,
		opts:   testcontext.GetRetryOpts(),
	}
}

// Get returns returns error if the validator returns false.
func (h *RetryableHTTPClient) Get(url string, validator HTTPResponseAsserter) error {
	err := h.withRetries(func() (*http.Response, error) {
		return h.client.Get(url)
	}, validator)

	if err != nil {
		return fmt.Errorf("error calling endpoint %s err=%w", url, err)
	}

	return nil
}

// GetWithHeaders returns error if the validator returns false.
func (h *RetryableHTTPClient) GetWithHeaders(url string, requestHeaders map[string]string, asserter HTTPResponseAsserter) error {
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
		return fmt.Errorf("error calling endpoint %s err=%w", url, err)
	}

	return nil
}

func (h *RetryableHTTPClient) withRetries(httpCall func() (*http.Response, error), asserter HTTPResponseAsserter) error {
	if err := retry.Do(func() error {
		response, callErr := httpCall()
		if callErr != nil {
			return callErr
		}
		defer response.Body.Close()
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
