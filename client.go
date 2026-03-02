package axon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// StatusError is returned when the server responds with an unexpected HTTP status code.
type StatusError struct {
	Code int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("unexpected status %d", e.Code)
}

// IsStatusError checks if err is a StatusError with the given code.
func IsStatusError(err error, code int) bool {
	var se *StatusError
	if errors.As(err, &se) {
		return se.Code == code
	}
	return false
}

// InternalClient is a lightweight HTTP client for internal service-to-service
// calls. It handles JSON marshalling/unmarshalling and status code checks.
type InternalClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewInternalClient creates a client with sensible defaults (10s timeout).
func NewInternalClient(baseURL string) *InternalClient {
	return &InternalClient{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Get performs a GET request and decodes the JSON response into result.
// Returns an error if the status code is not 200.
func (c *InternalClient) Get(ctx context.Context, path string, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &StatusError{Code: resp.StatusCode}
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// Post performs a POST request with a JSON body and decodes the response into result.
// Returns a *StatusError if the status code is not 2xx.
func (c *InternalClient) Post(ctx context.Context, path string, body, result any) error {
	var bodyReader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &StatusError{Code: resp.StatusCode}
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
