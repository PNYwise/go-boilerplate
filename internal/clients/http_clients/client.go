package http_clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go-boilerplate/internal/configs"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Client defines the interface for our robust HTTP client
type Client interface {
	Do(req *http.Request) (*http.Response, error)
	Get(ctx context.Context, url string, headers map[string]string) (*http.Response, []byte, error)
	PostJSON(ctx context.Context, url string, payload interface{}, headers map[string]string) (*http.Response, []byte, error)
}

type clientImpl struct {
	client *http.Client
	cfg    configs.Config
}

// NewHttpClient creates a new instrumented HTTP client
func NewHttpClient(cfg configs.Config) Client {
	// otelhttp.Transport automatically extracts the trace from the context
	// and injects it into the HTTP headers for distributed tracing!
	transport := otelhttp.NewTransport(http.DefaultTransport)

	timeoutSec := cfg.HttpClientTimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 60 // fallback default
	}

	c := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeoutSec) * time.Second,
	}

	return &clientImpl{
		client: c,
		cfg:    cfg,
	}
}

// Do executes a raw HTTP request.
// The caller MUST ensure the request has the context attached (req.WithContext(ctx))
// for tracing to work.
func (c *clientImpl) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

// Get is a convenience method for simple GET requests
func (c *clientImpl) Get(ctx context.Context, url string, headers map[string]string) (*http.Response, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return resp, body, nil
}

// PostJSON is a convenience method for JSON POST requests
func (c *clientImpl) PostJSON(ctx context.Context, url string, payload interface{}, headers map[string]string) (*http.Response, []byte, error) {
	var reqBody io.Reader

	if payload != nil {
		jsonBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		reqBody = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return resp, body, nil
}
