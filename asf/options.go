package asf

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	internalhttp "github.com/example/go-asf/asf/internal/http"
)

// Option configures a Client.
type Option func(*Client) error

// WithHTTPClient allows providing a custom HTTP client implementation.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) error {
		if hc == nil {
			return fmt.Errorf("http client cannot be nil")
		}
		c.httpClient = hc
		if c.httpClient.Timeout == 0 {
			c.httpClient.Timeout = 30 * time.Second
		}
		return nil
	}
}

// WithBaseURL overrides the default ASF API base URL.
func WithBaseURL(raw string) Option {
	return func(c *Client) error {
		if raw == "" {
			return fmt.Errorf("base url cannot be empty")
		}
		u, err := url.Parse(raw)
		if err != nil {
			return fmt.Errorf("parse base url: %w", err)
		}
		c.baseURL = u
		return nil
	}
}

// WithUserAgent sets a custom user-agent header for outbound requests.
func WithUserAgent(ua string) Option {
	return func(c *Client) error {
		if ua != "" {
			c.userAgent = ua
		}
		return nil
	}
}

// WithRetryPolicy sets a custom retry policy.
func WithRetryPolicy(policy internalhttp.RetryPolicy) Option {
	return func(c *Client) error {
		if policy == nil {
			return fmt.Errorf("retry policy cannot be nil")
		}
		c.retry = policy
		return nil
	}
}
