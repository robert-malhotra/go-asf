package asf

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"time"

	"github.com/example/go-asf/asf/download"
	internalhttp "github.com/example/go-asf/asf/internal/http"
	"github.com/example/go-asf/asf/model"
	"github.com/example/go-asf/asf/search"
)

const (
	defaultBaseURL   = "https://api.daac.asf.alaska.edu/"
	defaultUserAgent = "go-asf-client/0.1"
)

// Client is the entry point for interacting with ASF search and download endpoints.
type Client struct {
	httpClient *http.Client
	baseURL    *url.URL
	userAgent  string
	retry      internalhttp.RetryPolicy
	basicAuth  *download.BasicAuth
}

// NewClient constructs a Client configured with the provided options.
func NewClient(opts ...Option) (*Client, error) {
	base, err := url.Parse(defaultBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse default base url: %w", err)
	}

	c := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    base,
		userAgent:  defaultUserAgent,
		retry:      internalhttp.DefaultRetryPolicy(),
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	if c.httpClient.Jar == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, fmt.Errorf("create cookie jar: %w", err)
		}
		c.httpClient.Jar = jar
	}

	return c, nil
}

// Search executes a query against the ASF search endpoint and returns an iterator over products.
func (c *Client) Search(ctx context.Context, params search.Params) (*ResultIterator, error) {
	if ctx == nil {
		return nil, errors.New("nil context")
	}

	encoded, err := params.Encode()
	if err != nil {
		return nil, err
	}

	return newResultIterator(c, encoded, params.PageSize()), nil
}

func (c *Client) doSearchRequest(ctx context.Context, query url.Values) (*model.SearchResponse, error) {
	rel := &url.URL{Path: path.Join(c.baseURL.Path, "services/search/param")}
	u := c.baseURL.ResolveReference(rel)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create search request: %w", err)
	}

	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	var resp *http.Response
	if resp, err = internalhttp.Do(ctx, c.httpClient, req, c.retry); err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, internalhttp.HTTPError(resp)
	}

	var payload model.SearchResponse
	if err := internalhttp.DecodeJSON(resp.Body, &payload); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}

	return &payload, nil
}

// DownloadProduct fetches all files for a given product into destDir.
func (c *Client) DownloadProduct(ctx context.Context, product model.Product, destDir string, opts ...DownloadOption) error {
	cfg := downloadConfig{
		concurrency: 2,
		verify:      true,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	cfg.ensureDefaults(c.basicAuth)
	return cfg.downloader.Download(ctx, c.httpClient, c.userAgent, product, destDir)
}
