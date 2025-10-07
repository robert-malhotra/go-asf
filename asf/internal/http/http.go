package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RetryPolicy controls retry behaviour for HTTP requests.
type RetryPolicy interface {
	NextDelay(attempt int, resp *http.Response, err error) (time.Duration, bool)
}

type retryPolicy struct {
	maxAttempts int
	baseDelay   time.Duration
	statuses    map[int]struct{}
}

// DefaultRetryPolicy returns a conservative retry policy suitable for ASF APIs.
func DefaultRetryPolicy() RetryPolicy {
	return &retryPolicy{
		maxAttempts: 3,
		baseDelay:   500 * time.Millisecond,
		statuses: map[int]struct{}{
			http.StatusTooManyRequests:     {},
			http.StatusInternalServerError: {},
			http.StatusBadGateway:          {},
			http.StatusServiceUnavailable:  {},
			http.StatusGatewayTimeout:      {},
		},
	}
}

// NoRetryPolicy disables retries.
type NoRetryPolicy struct{}

// NextDelay implements RetryPolicy.
func (NoRetryPolicy) NextDelay(int, *http.Response, error) (time.Duration, bool) {
	return 0, false
}

func (p *retryPolicy) NextDelay(attempt int, resp *http.Response, err error) (time.Duration, bool) {
	if attempt >= p.maxAttempts {
		return 0, false
	}
	if err != nil {
		return backoff(p.baseDelay, attempt), true
	}
	if resp != nil {
		if _, ok := p.statuses[resp.StatusCode]; ok {
			return backoff(p.baseDelay, attempt), true
		}
	}
	return 0, false
}

func backoff(base time.Duration, attempt int) time.Duration {
	shift := attempt - 1
	if shift < 0 {
		shift = 0
	}
	return base * time.Duration(1<<uint(shift))
}

// Do issues the HTTP request honouring the provided retry policy.
func Do(ctx context.Context, client *http.Client, req *http.Request, policy RetryPolicy) (*http.Response, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if client == nil {
		return nil, errors.New("http client is required")
	}
	if policy == nil {
		policy = DefaultRetryPolicy()
	}

	attempt := 1
	for {
		attemptReq, err := cloneRequest(req, ctx)
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(attemptReq)
		if err == nil && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		delay, retry := policy.NextDelay(attempt, resp, err)
		if !retry {
			if err != nil {
				return nil, err
			}
			return resp, nil
		}

		if resp != nil {
			resp.Body.Close()
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
		attempt++
	}
}

func cloneRequest(req *http.Request, ctx context.Context) (*http.Request, error) {
	clone := req.Clone(ctx)
	if req.Body != nil && req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		clone.Body = body
	}
	return clone, nil
}

// HTTPError returns a descriptive error for non-successful responses.
func HTTPError(resp *http.Response) error {
	if resp == nil {
		return errors.New("nil response")
	}
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("http error: %s: %s", resp.Status, string(data))
}

// DecodeJSON decodes a JSON payload from r into v.
func DecodeJSON(r io.Reader, v interface{}) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}
