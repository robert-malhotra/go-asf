package asf

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	defaultBaseURL = "https://api.daac.asf.alaska.edu"
)

// Client provides access to ASF Search endpoints.
type Client struct {
	baseURL       string
	httpClient    *http.Client
	authenticator Authenticator
}

// Platform represents a supported mission/platform identifier.
type Platform string

const (
	PlatformSentinel1A Platform = "Sentinel-1A"
	PlatformSentinel1B Platform = "Sentinel-1B"
	PlatformSentinel1C Platform = "Sentinel-1C"
	PlatformSentinel1  Platform = "Sentinel-1"
)

// BeamMode enumerates radar beam mode values.
type BeamMode string

const (
	BeamModeIW BeamMode = "IW"
	BeamModeEW BeamMode = "EW"
	BeamModeSM BeamMode = "SM"
	BeamModeWV BeamMode = "WV"
)

// Polarization enumerates common SAR polarization strings.
type Polarization string

const (
	PolarizationHH Polarization = "HH"
	PolarizationHV Polarization = "HV"
	PolarizationVV Polarization = "VV"
	PolarizationVH Polarization = "VH"
	PolarizationQP Polarization = "QP"
)

// ProductType represents an ASF product type identifier.
type ProductType string

const (
	ProductTypeSLC      ProductType = "SLC"
	ProductTypeGRD      ProductType = "GRD"
	ProductTypeGRDMD    ProductType = "GRD_MD"
	ProductTypeOCN      ProductType = "OCN"
	ProductTypeRAW      ProductType = "RAW"
	ProductTypeMETADATA ProductType = "METADATA"
)

// CollectionName denotes an ASF collection value.
type CollectionName string

const (
	CollectionSentinel1 CollectionName = "SENTINEL-1"
)

// ProcessingLevel enumerates the processing level strings.
type ProcessingLevel string

const (
	ProcessingLevelL0    ProcessingLevel = "L0"
	ProcessingLevelL1    ProcessingLevel = "L1"
	ProcessingLevelL2    ProcessingLevel = "L2"
	ProcessingLevelSLC   ProcessingLevel = "SLC"
	ProcessingLevelGRD   ProcessingLevel = "GRD"
	ProcessingLevelGRDMD ProcessingLevel = "GRD_MD"
	ProcessingLevelGRDHD ProcessingLevel = "GRD_HD"
)

// LookDirection describes the look direction parameter.
type LookDirection string

const (
	LookDirectionLeft  LookDirection = "LEFT"
	LookDirectionRight LookDirection = "RIGHT"
)

// FlightDirection enumerates valid flight direction filters.
type FlightDirection string

const (
	FlightDirectionAscending  FlightDirection = "ASCENDING"
	FlightDirectionDescending FlightDirection = "DESCENDING"
)

// Option mutates the client when constructing it.
type Option func(*Client)

// WithBaseURL overrides the default API host.
func WithBaseURL(u string) Option {
	return func(c *Client) {
		c.baseURL = u
	}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c == nil {
		return nil, fmt.Errorf("asf: client is nil")
	}
	if c.authenticator != nil {
		if err := c.authenticator(req); err != nil {
			return nil, fmt.Errorf("asf: authenticate request: %w", err)
		}
	}
	return c.httpClient.Do(req)
}

// WithHTTPClient configures a custom HTTP client instance.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc == nil {
			hc = newDefaultHTTPClient()
		}
		c.httpClient = hc
	}
}

// WithAuthToken configures the bearer token used for authenticated requests.
func WithAuthToken(token string) Option {
	return WithAuthenticator(BearerToken(token))
}

// WithAuthenticator sets a custom authenticator for the client's session.
func WithAuthenticator(auth Authenticator) Option {
	return func(c *Client) {
		c.authenticator = auth
	}
}

// NewClient creates a Client with sensible defaults.
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL:    defaultBaseURL,
		httpClient: newDefaultHTTPClient(),
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.httpClient == nil {
		c.httpClient = newDefaultHTTPClient()
	}
	return c
}

// SearchOptions captures supported query parameters for ASF search.
type SearchOptions struct {
	Platforms       []Platform
	BeamModes       []BeamMode
	Polarizations   []Polarization
	ProductTypes    []ProductType
	Collections     []CollectionName
	ProcessingLevel []ProcessingLevel
	LookDirections  []LookDirection
	Start           time.Time
	End             time.Time
	RelativeOrbit   string
	FlightDirection FlightDirection
	IntersectsWith  string
	GranuleIDs      []string
	MaxResults      int
}

// Search queries the ASF search API and returns a list of products.
func (c *Client) Search(ctx context.Context, opts SearchOptions) ([]Product, error) {
	endpoint, err := url.JoinPath(c.baseURL, "services", "search", "param")
	if err != nil {
		return nil, fmt.Errorf("asf: invalid base URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("asf: create request: %w", err)
	}
	req.URL.RawQuery = encodeSearchOptions(opts).Encode()

	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("asf: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("asf: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var payload FeatureCollection
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("asf: decode response: %w", err)
	}

	return payload.Features, nil
}

// encodeSearchOptions flattens search options into URL query parameters.
func encodeSearchOptions(opts SearchOptions) url.Values {
	q := url.Values{}
	addQueryValues(q, "platform", opts.Platforms)
	addQueryValues(q, "beamMode", opts.BeamModes)
	addQueryValues(q, "polarization", opts.Polarizations)
	addQueryValues(q, "productType", opts.ProductTypes)
	addQueryValues(q, "collectionName", opts.Collections)
	addQueryValues(q, "processingLevel", opts.ProcessingLevel)
	addQueryValues(q, "lookDirection", opts.LookDirections)
	addStringQueryValues(q, "granule_list", opts.GranuleIDs)
	setQueryIfNonEmpty(q, "intersectsWith", opts.IntersectsWith)
	setQueryIfNonEmpty(q, "relativeOrbit", opts.RelativeOrbit)
	setQueryIfNonEmpty(q, "flightDirection", opts.FlightDirection)
	setQueryTime(q, "start", opts.Start)
	setQueryTime(q, "end", opts.End)
	setPositiveInt(q, "maxResults", opts.MaxResults)
	q.Set("output", "geojson")
	return q
}

// addQueryValues appends non-empty values from a slice of string-based types.
func addQueryValues[T ~string](q url.Values, key string, values []T) {
	for _, value := range values {
		if s := string(value); s != "" {
			q.Add(key, s)
		}
	}
}

// addStringQueryValues appends non-empty values from a slice of strings.
func addStringQueryValues(q url.Values, key string, values []string) {
	for _, value := range values {
		if value != "" {
			q.Add(key, value)
		}
	}
}

// setQueryIfNonEmpty sets a query parameter if the string-based value is not empty.
func setQueryIfNonEmpty[T ~string](q url.Values, key string, value T) {
	if s := string(value); s != "" {
		q.Set(key, s)
	}
}

func setQueryTime(q url.Values, key string, value time.Time) {
	if value.IsZero() {
		return
	}
	q.Set(key, value.UTC().Format(time.RFC3339))
}

func setPositiveInt(q url.Values, key string, value int) {
	if value > 0 {
		q.Set(key, strconv.Itoa(value))
	}
}

// Common parameter helpers similar to asf_search enumerations.
var (
	PlatformsSentinel1 = []Platform{PlatformSentinel1A, PlatformSentinel1B, PlatformSentinel1C}
	BeamModesIW        = []BeamMode{BeamModeIW}
	PolarizationsVVVH  = []Polarization{PolarizationVV, PolarizationVH}
)

// Download fetches all products in the list and saves them to the targetFolder.
// It downloads files concurrently, limiting concurrency to runtime.NumCPU().
func (c *Client) Download(ctx context.Context, targetFolder string, products ...Product) error {
	if len(products) == 0 {
		return nil
	}

	if err := os.MkdirAll(targetFolder, 0755); err != nil {
		return fmt.Errorf("asf: create target folder %q: %w", targetFolder, err)
	}

	g, gctx := errgroup.WithContext(ctx)
	// Limit concurrency to avoid overwhelming the network or server.
	g.SetLimit(runtime.NumCPU())

	for _, p := range products {
		product := p // Capture loop variable for goroutine.
		g.Go(func() error {
			return c.downloadProduct(gctx, targetFolder, product)
		})
	}

	return g.Wait()
}

// downloadProduct handles the download of a single product.
func (c *Client) downloadProduct(ctx context.Context, targetFolder string, product Product) error {
	if product.Properties.URL == "" {
		return fmt.Errorf("asf: product %q has no URL", product.Properties.SceneName)
	}
	if product.Properties.FileName == "" {
		return fmt.Errorf("asf: product %q has no FileName", product.Properties.SceneName)
	}

	destPath := filepath.Join(targetFolder, product.Properties.FileName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, product.Properties.URL, nil)
	if err != nil {
		return fmt.Errorf("asf: create download request for %q: %w", product.Properties.FileName, err)
	}

	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("asf: send download request for %q: %w", product.Properties.FileName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("asf: unexpected download status for %q: %d: %s", product.Properties.FileName, resp.StatusCode, string(body))
	}

	// Create the destination file.
	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("asf: create file %q: %w", destPath, err)
	}
	defer file.Close()

	// Stream the response body to the file.
	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("asf: save file %q: %w", destPath, err)
	}

	return nil
}

// Authenticator applies authentication information to a request.
type Authenticator = func(*http.Request) error

// BearerToken returns an authenticator that adds an Authorization header.
func BearerToken(token string) Authenticator {
	return func(req *http.Request) error {
		if token == "" {
			return nil
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
}

// BasicAuth returns an authenticator that applies HTTP basic authentication.
func BasicAuth(username, password string) Authenticator {
	return func(req *http.Request) error {
		req.SetBasicAuth(username, password)
		return nil
	}
}

// HeaderAuth returns an authenticator that copies the provided headers.
func HeaderAuth(headers map[string]string) Authenticator {
	return func(req *http.Request) error {
		for key, value := range headers {
			if value == "" {
				continue
			}
			req.Header.Set(key, value)
		}
		return nil
	}
}

func newDefaultHTTPClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}
	httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) == 0 {
			return nil
		}
		prev := via[len(via)-1]

		// Only re-apply auth header on redirect
		if authHeader := prev.Header.Get("Authorization"); authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}
		return nil
	}
	return httpClient
}
