package remote

import (
	"bytes"
	"fmt"
	"net/http"
	"time"
)

// HTTPSender handles HTTP requests

type HTTPSender struct {
	client *http.Client
}

// NewHTTPSender creates a new HTTPSender with default configuration

func NewHTTPSender() *HTTPSender {
	return &HTTPSender{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewHTTPSenderWithTimeout creates a new HTTPSender with custom timeout

func NewHTTPSenderWithTimeout(timeout time.Duration) *HTTPSender {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &HTTPSender{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Post sends a POST request with JSON payload

func (h *HTTPSender) Post(url string, jsonPayload string) (string, int, error) {
	if url == "" {
		return "", 0, fmt.Errorf("empty URL")
	}

	body := bytes.NewBufferString(jsonPayload)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	var responseBody bytes.Buffer
	_, err = responseBody.ReadFrom(resp.Body)
	if err != nil {
		return "", resp.StatusCode, fmt.Errorf("failed to read response: %v", err)
	}

	return responseBody.String(), resp.StatusCode, nil
}

// PostWithHeaders sends a POST request with JSON payload and custom headers

func (h *HTTPSender) PostWithHeaders(url string, jsonPayload string, headers map[string]string) (string, int, error) {
	if url == "" {
		return "", 0, fmt.Errorf("empty URL")
	}

	body := bytes.NewBufferString(jsonPayload)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create request: %v", err)
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	var responseBody bytes.Buffer
	_, err = responseBody.ReadFrom(resp.Body)
	if err != nil {
		return "", resp.StatusCode, fmt.Errorf("failed to read response: %v", err)
	}

	return responseBody.String(), resp.StatusCode, nil
}
