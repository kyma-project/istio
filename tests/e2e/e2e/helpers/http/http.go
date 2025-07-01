package http

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

type Request struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    string
}

type Response struct {
	Headers    map[string]string
	Body       string
	StatusCode int
	Status     string
}

func DoRequest(t *testing.T, request *Request) (*Response, error) {
	t.Helper()
	t.Logf("Executing HTTP request: method: %s, url: %s", request.Method, request.URL)
	requestBody := bytes.NewBufferString(request.Body)
	httpreq, err := http.NewRequestWithContext(t.Context(), request.Method, request.URL, requestBody)
	if err != nil {
		return nil, err
	}

	for key, value := range request.Headers {
		httpreq.Header.Set(key, value)
	}

	c := &http.Client{}

	httpResp, err := c.Do(httpreq)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := httpResp.Body.Close(); closeErr != nil {
			t.Logf("Failed to close response body: %v", closeErr)
		}
	}()
	t.Logf("Received response: status: %d", httpResp.StatusCode)

	response := &Response{}
	response.Status = httpResp.Status
	response.StatusCode = httpResp.StatusCode
	response.Headers = make(map[string]string)
	for key, value := range httpResp.Header {
		response.Headers[key] = value[0]
	}
	respBodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	response.Body = string(respBodyBytes)
	return response, nil
}
