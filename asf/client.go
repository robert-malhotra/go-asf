package asf

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultBaseURL = "https://api.daac.asf.alaska.edu"

var (
	// ErrNilClient is returned when methods are invoked on a nil Client pointer.
	ErrNilClient = errors.New("asf: nil client")
	// ErrMissingDownloadURL indicates that a product does not include any downloadable files.
	ErrMissingDownloadURL = errors.New("asf: product missing download URL")
)

// Client provides access to ASF Search endpoints and product downloads.
type Client struct {
	baseURL string
	session *Session
}

// Platform represents a supported mission/platform identifier recognized by ASF search APIs.
// The provided constants cover commonly used missions but callers may supply any valid platform string.
type Platform = string

const (
	PlatformSentinel1A Platform = "Sentinel-1A"
	PlatformSentinel1B Platform = "Sentinel-1B"
	PlatformSentinel1  Platform = "Sentinel-1"
)

// BeamMode enumerates radar beam mode values for Sentinel-1 style missions.
type BeamMode = string

const (
	BeamModeIW BeamMode = "IW"
	BeamModeEW BeamMode = "EW"
	BeamModeSM BeamMode = "SM"
	BeamModeWV BeamMode = "WV"
)

// Polarization enumerates common SAR polarization strings.
type Polarization = string

const (
	PolarizationHH Polarization = "HH"
	PolarizationHV Polarization = "HV"
	PolarizationVV Polarization = "VV"
	PolarizationVH Polarization = "VH"
	PolarizationQP Polarization = "QP"
)

// ProductType represents an ASF product type identifier.
type ProductType = string

const (
	ProductTypeSLC      ProductType = "SLC"
	ProductTypeGRD      ProductType = "GRD"
	ProductTypeGRDMD    ProductType = "GRD_MD"
	ProductTypeOCN      ProductType = "OCN"
	ProductTypeRAW      ProductType = "RAW"
	ProductTypeMETADATA ProductType = "METADATA"
)

// CollectionName denotes an ASF collection value.
type CollectionName = string

const (
	CollectionSentinel1 CollectionName = "SENTINEL-1"
)

// ProcessingLevel enumerates the processing level strings reported by ASF search.
type ProcessingLevel = string

const (
	ProcessingLevelL0    ProcessingLevel = "L0"
	ProcessingLevelL1    ProcessingLevel = "L1"
	ProcessingLevelL2    ProcessingLevel = "L2"
	ProcessingLevelSLC   ProcessingLevel = "SLC"
	ProcessingLevelGRD   ProcessingLevel = "GRD"
	ProcessingLevelGRDMD ProcessingLevel = "GRD_MD"
)

// LookDirection describes the look direction parameter accepted by ASF search.
type LookDirection = string

const (
	LookDirectionLeft  LookDirection = "LEFT"
	LookDirectionRight LookDirection = "RIGHT"
)

// FlightDirection enumerates valid flight direction filters.
type FlightDirection = string

const (
	FlightDirectionAscending  FlightDirection = "ASCENDING"
	FlightDirectionDescending FlightDirection = "DESCENDING"
)

// StackSort specifies the stack sort key accepted by ASF stack searches.
type StackSort = string

const (
	StackSortPerpendicularBaselineAsc  StackSort = "BPERP_ASCENDING"
	StackSortPerpendicularBaselineDesc StackSort = "BPERP_DESCENDING"
	StackSortPerpendicularBaseline     StackSort = "BPERP"
)

// Option mutates the client when constructing it.
type Option func(*Client)

// WithBaseURL overrides the default API host.
func WithBaseURL(u string) Option {
	return func(c *Client) {
		c.baseURL = u
	}
}

// WithHTTPClient configures a custom HTTP client instance.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if c.session == nil {
			c.session = NewSession()
		}
		c.session.client = hc
	}
}

// WithAuthToken configures the bearer token used for authenticated requests.
func WithAuthToken(token string) Option {
	return WithAuthenticator(BearerToken(token))
}

// WithAuthenticator sets a custom authenticator for the client's session.
func WithAuthenticator(auth Authenticator) Option {
	return func(c *Client) {
		if c.session == nil {
			c.session = NewSession()
		}
		c.session.authenticator = auth
	}
}

// WithSession allows callers to provide a preconfigured session.
func WithSession(session *Session) Option {
	return func(c *Client) {
		c.session = session
	}
}

// NewClient creates a Client with sensible defaults.
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL: defaultBaseURL,
		session: NewSession(),
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.session == nil {
		c.session = NewSession()
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
	StackName       string
	Master          string
	Slave           string
	StackSort       StackSort
	StackSceneCount int
	MaxResults      int
	Page            int
	Extra           url.Values
}

// Product describes a normalized result from ASF search.
type Product struct {
	GranuleID       string
	Platform        Platform
	Acquisition     time.Time
	ProcessingLevel ProcessingLevel
	ProductType     ProductType
	Polarization    Polarization
	BeamMode        BeamMode
	SizeMB          float64
	DownloadURL     string
	FileURLs        []string
}

// Search queries the ASF search API using the supplied options.
func (c *Client) Search(ctx context.Context, opts SearchOptions) ([]Product, error) {
	if c == nil {
		return nil, ErrNilClient
	}
	if c.session == nil {
		return nil, fmt.Errorf("asf: client missing session")
	}
	endpoint, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("asf: invalid base URL: %w", err)
	}
	endpoint.Path = joinURLPath(endpoint.Path, "services/search/param")

	endpoint.RawQuery = encodeSearchOptions(opts).Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("asf: create request: %w", err)
	}

	resp, err := c.session.Do(req)
	if err != nil {
		return nil, fmt.Errorf("asf: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("asf: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var payload featureCollection
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("asf: decode response: %w", err)
	}

	products := make([]Product, 0, len(payload.Features))
	for _, feature := range payload.Features {
		products = append(products, feature.toProduct())
	}

	return products, nil
}

// Download streams the given product to the destination path.
func (c *Client) Download(ctx context.Context, product Product, destPath string) error {
	if c == nil {
		return ErrNilClient
	}
	if destPath == "" {
		return fmt.Errorf("asf: destination path required")
	}
	urls := product.FileURLs
	if len(urls) == 0 && product.DownloadURL != "" {
		urls = []string{product.DownloadURL}
	}
	if len(urls) == 0 {
		return ErrMissingDownloadURL
	}
	return c.downloadURL(ctx, urls[0], destPath)
}

// DownloadAll downloads every file referenced by the supplied products to destDir.
func (c *Client) DownloadAll(ctx context.Context, products []Product, destDir string) error {
	if c == nil {
		return ErrNilClient
	}
	var urls []string
	for _, product := range products {
		if len(product.FileURLs) == 0 && product.DownloadURL != "" {
			urls = append(urls, product.DownloadURL)
			continue
		}
		urls = append(urls, product.FileURLs...)
	}
	if len(urls) == 0 {
		return ErrMissingDownloadURL
	}
	return c.DownloadURLs(ctx, urls, destDir, 0)
}

// DownloadURLs downloads the provided URLs into destDir using up to concurrency workers.
func (c *Client) DownloadURLs(ctx context.Context, urls []string, destDir string, concurrency int) error {
	if c == nil {
		return ErrNilClient
	}
	if len(urls) == 0 {
		return fmt.Errorf("asf: no URLs supplied")
	}
	if destDir == "" {
		return fmt.Errorf("asf: destination directory required")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("asf: create destination directory: %w", err)
	}
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
		if concurrency < 1 {
			concurrency = 1
		}
	}

	type job struct {
		url string
	}

	jobs := make(chan job)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	worker := func() {
		defer wg.Done()
		for j := range jobs {
			select {
			case <-ctx.Done():
				mu.Lock()
				errs = append(errs, ctx.Err())
				mu.Unlock()
				continue
			default:
			}
			base := filepath.Base(j.url)
			if base == "." || base == "" || base == "/" {
				base = fmt.Sprintf("download-%d", time.Now().UnixNano())
			}
			dest := filepath.Join(destDir, base)
			if err := c.downloadURL(ctx, j.url, dest); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", j.url, err))
				mu.Unlock()
			}
		}
	}

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}

	go func() {
		defer close(jobs)
		seen := make(map[string]struct{})
		for _, u := range urls {
			if u == "" {
				continue
			}
			if _, ok := seen[u]; ok {
				continue
			}
			seen[u] = struct{}{}
			jobs <- job{url: u}
		}
	}()

	wg.Wait()
	if len(errs) > 0 {
		return BatchError{Errors: errs}
	}
	return nil
}

// GeoSearchOptions customizes geometry searches.
type GeoSearchOptions struct {
	SearchOptions
}

// GranuleSearchOptions customizes granule-based searches.
type GranuleSearchOptions struct {
	SearchOptions
}

// ProductSearchOptions customizes product searches.
type ProductSearchOptions struct {
	SearchOptions
	ProductTypes []ProductType
	Collections  []CollectionName
}

// StackSearchOptions customizes stack searches.
type StackSearchOptions struct {
	SearchOptions
	Master          string
	Slave           string
	StackSort       string
	StackSceneCount int
}

// GeoSearch performs a geometry-based search using the provided WKT or GeoJSON.
func (c *Client) GeoSearch(ctx context.Context, geometry string, opts GeoSearchOptions) ([]Product, error) {
	options := opts.SearchOptions
	options.IntersectsWith = geometry
	return c.Search(ctx, options)
}

// GranuleSearch performs a granule list search.
func (c *Client) GranuleSearch(ctx context.Context, ids []string, opts GranuleSearchOptions) ([]Product, error) {
	options := opts.SearchOptions
	options.GranuleIDs = append(options.GranuleIDs, ids...)
	return c.Search(ctx, options)
}

// ProductSearch searches for matching product types and collections.
func (c *Client) ProductSearch(ctx context.Context, opts ProductSearchOptions) ([]Product, error) {
	options := opts.SearchOptions
	options.ProductTypes = append(options.ProductTypes, opts.ProductTypes...)
	options.Collections = append(options.Collections, opts.Collections...)
	return c.Search(ctx, options)
}

// StackSearch mirrors asf_search stack helper behavior.
func (c *Client) StackSearch(ctx context.Context, stackName string, opts StackSearchOptions) ([]Product, error) {
	options := opts.SearchOptions
	options.StackName = stackName
	if opts.Master != "" {
		options.Master = opts.Master
	}
	if opts.Slave != "" {
		options.Slave = opts.Slave
	}
	if opts.StackSort != "" {
		options.StackSort = opts.StackSort
	}
	if opts.StackSceneCount > 0 {
		options.StackSceneCount = opts.StackSceneCount
	}
	return c.Search(ctx, options)
}

// encodeSearchOptions flattens search options into URL query parameters.
func encodeSearchOptions(opts SearchOptions) url.Values {
	q := url.Values{}
	addList := func(key string, values []string) {
		for _, value := range values {
			if value != "" {
				q.Add(key, value)
			}
		}
	}
	addList("platform", opts.Platforms)
	addList("beamMode", opts.BeamModes)
	addList("polarization", opts.Polarizations)
	addList("productType", opts.ProductTypes)
	addList("collectionName", opts.Collections)
	addList("processingLevel", opts.ProcessingLevel)
	addList("lookDirection", opts.LookDirections)
	addList("granule_list", opts.GranuleIDs)
	if opts.IntersectsWith != "" {
		q.Set("intersectsWith", opts.IntersectsWith)
	}
	if opts.RelativeOrbit != "" {
		q.Set("relativeOrbit", opts.RelativeOrbit)
	}
	if opts.FlightDirection != "" {
		q.Set("flightDirection", opts.FlightDirection)
	}
	if opts.StackName != "" {
		q.Set("stackName", opts.StackName)
	}
	if opts.Master != "" {
		q.Set("master", opts.Master)
	}
	if opts.Slave != "" {
		q.Set("slave", opts.Slave)
	}
	if opts.StackSort != "" {
		q.Set("sortby", opts.StackSort)
	}
	if opts.StackSceneCount > 0 {
		q.Set("stackSize", fmt.Sprintf("%d", opts.StackSceneCount))
	}
	if !opts.Start.IsZero() {
		q.Set("start", opts.Start.UTC().Format(time.RFC3339))
	}
	if !opts.End.IsZero() {
		q.Set("end", opts.End.UTC().Format(time.RFC3339))
	}
	if opts.MaxResults > 0 {
		q.Set("maxResults", fmt.Sprintf("%d", opts.MaxResults))
	}
	if opts.Page > 0 {
		q.Set("page", fmt.Sprintf("%d", opts.Page))
	}
	for key, values := range opts.Extra {
		for _, value := range values {
			if value != "" {
				q.Add(key, value)
			}
		}
	}
	if q.Get("output") == "" {
		q.Set("output", "geojson")
	}
	return q
}

// Common parameter helpers similar to asf_search enumerations.
var (
	PlatformsSentinel1 = []Platform{PlatformSentinel1A, PlatformSentinel1B}
	BeamModesIW        = []BeamMode{BeamModeIW}
	PolarizationsVVVH  = []Polarization{PolarizationVV, PolarizationVH}
)

func (c *Client) downloadURL(ctx context.Context, downloadURL, destPath string) error {
	if c.session == nil {
		return fmt.Errorf("asf: client missing session")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("asf: create download request: %w", err)
	}
	resp, err := c.session.Do(req)
	if err != nil {
		return fmt.Errorf("asf: download request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("asf: download failed %d: %s", resp.StatusCode, string(body))
	}

	tmp := destPath + ".part"
	file, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("asf: create temp file: %w", err)
	}
	success := false
	defer func() {
		file.Close()
		if !success {
			os.Remove(tmp)
		}
	}()

	if _, err = io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("asf: write file: %w", err)
	}
	if err = file.Sync(); err != nil {
		return fmt.Errorf("asf: sync file: %w", err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("asf: close file: %w", err)
	}

	if err = os.Rename(tmp, destPath); err != nil {
		return fmt.Errorf("asf: rename temp file: %w", err)
	}
	success = true
	return nil
}

type featureCollection struct {
	Features []feature `json:"features"`
}

type feature struct {
	ID         string       `json:"id"`
	Properties featureProps `json:"properties"`
}

type featureProps struct {
	Platform        string       `json:"platform"`
	BeamMode        string       `json:"beamMode"`
	BeamModeType    string       `json:"beamModeType"`
	Polarization    string       `json:"polarization"`
	ProcessingLevel string       `json:"processingLevel"`
	ProductType     string       `json:"productType"`
	StartTime       string       `json:"startTime"`
	SizeMB          numericValue `json:"sizeMB"`
	Bytes           numericValue `json:"bytes"`
	Files           []fileInfo   `json:"files"`
	URL             string       `json:"url"`
	SceneName       string       `json:"sceneName"`
	FileID          string       `json:"fileID"`
	FileName        string       `json:"fileName"`
	AdditionalURLs  []string     `json:"additionalUrls"`
	S3URLs          []string     `json:"s3Urls"`
}

type fileInfo struct {
	URL string `json:"url"`
}

func (f feature) toProduct() Product {
	acquisition := parseTime(f.Properties.StartTime)
	granuleID := f.ID
	if f.Properties.SceneName != "" {
		granuleID = f.Properties.SceneName
	} else if f.Properties.FileID != "" {
		granuleID = f.Properties.FileID
	} else if f.Properties.FileName != "" {
		granuleID = f.Properties.FileName
	}

	urls := collectURLs(f.Properties)
	downloadURL := ""
	if len(urls) > 0 {
		downloadURL = urls[0]
	}

	beamMode := f.Properties.BeamMode
	if beamMode == "" {
		beamMode = f.Properties.BeamModeType
	}

	sizeMB := float64(f.Properties.SizeMB)
	if sizeMB == 0 && f.Properties.Bytes > 0 {
		sizeMB = float64(f.Properties.Bytes) / (1024 * 1024)
	}

	productType := f.Properties.ProductType
	if productType == "" {
		productType = f.Properties.ProcessingLevel
	}

	return Product{
		GranuleID:       granuleID,
		Platform:        f.Properties.Platform,
		BeamMode:        beamMode,
		Polarization:    f.Properties.Polarization,
		ProcessingLevel: f.Properties.ProcessingLevel,
		ProductType:     productType,
		Acquisition:     acquisition,
		SizeMB:          sizeMB,
		DownloadURL:     downloadURL,
		FileURLs:        urls,
	}
}

func collectURLs(props featureProps) []string {
	dedupe := make(map[string]struct{})
	add := func(values ...string) {
		for _, value := range values {
			if value == "" {
				continue
			}
			dedupe[value] = struct{}{}
		}
	}
	add(props.URL)
	add(props.AdditionalURLs...)
	add(props.S3URLs...)
	for _, file := range props.Files {
		add(file.URL)
	}
	urls := make([]string, 0, len(dedupe))
	for url := range dedupe {
		urls = append(urls, url)
	}
	sort.Strings(urls)
	return urls
}

type numericValue float64

func (n *numericValue) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*n = 0
		return nil
	}
	if data[0] == '"' && data[len(data)-1] == '"' {
		str := string(data[1 : len(data)-1])
		if str == "" {
			*n = 0
			return nil
		}
		value, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return err
		}
		*n = numericValue(value)
		return nil
	}
	value, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return err
	}
	*n = numericValue(value)
	return nil
}

func joinURLPath(basePath string, elems ...string) string {
	parts := make([]string, 0, len(elems)+1)
	trimmedBase := strings.Trim(basePath, "/")
	if trimmedBase != "" {
		parts = append(parts, trimmedBase)
	}
	for _, elem := range elems {
		trimmed := strings.Trim(elem, "/")
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return "/" + path.Join(parts...)
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05.000Z",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t
		}
	}
	return time.Time{}
}

// BatchError aggregates multiple download errors.
type BatchError struct {
	Errors []error
}

// Error implements the error interface.
func (e BatchError) Error() string {
	if len(e.Errors) == 0 {
		return ""
	}
	messages := make([]string, 0, len(e.Errors))
	for _, err := range e.Errors {
		if err != nil {
			messages = append(messages, err.Error())
		}
	}
	return strings.Join(messages, "; ")
}

// Authenticator applies authentication information to a request.
type Authenticator interface {
	Authenticate(req *http.Request) error
}

// AuthenticatorFunc converts a function into an Authenticator.
type AuthenticatorFunc func(*http.Request) error

// Authenticate applies the function to the request.
func (f AuthenticatorFunc) Authenticate(req *http.Request) error {
	return f(req)
}

// BearerToken authenticates with a bearer token header.
type BearerToken string

// Authenticate applies the bearer token header.
func (b BearerToken) Authenticate(req *http.Request) error {
	if string(b) == "" {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+string(b))
	return nil
}

// BasicAuth uses HTTP Basic authentication.
type BasicAuth struct {
	Username string
	Password string
}

// Authenticate applies the basic auth header.
func (b BasicAuth) Authenticate(req *http.Request) error {
	req.SetBasicAuth(b.Username, b.Password)
	return nil
}

// HeaderAuth sets arbitrary headers.
type HeaderAuth map[string]string

// Authenticate applies stored headers to the request.
func (h HeaderAuth) Authenticate(req *http.Request) error {
	for key, value := range h {
		req.Header.Set(key, value)
	}
	return nil
}

// Session mediates authenticated HTTP traffic for ASF requests.
type Session struct {
	client        *http.Client
	authenticator Authenticator
}

// SessionOption configures a session.
type SessionOption func(*Session)

// WithSessionHTTPClient overrides the HTTP client used by the session.
func WithSessionHTTPClient(hc *http.Client) SessionOption {
	return func(s *Session) {
		s.client = hc
	}
}

// WithSessionAuthenticator sets the session authenticator.
func WithSessionAuthenticator(auth Authenticator) SessionOption {
	return func(s *Session) {
		s.authenticator = auth
	}
}

// NewSession constructs a session with cookie jar and timeout defaults.
func NewSession(opts ...SessionOption) *Session {
	jar, _ := cookiejar.New(nil)
	httpClient := &http.Client{Timeout: 30 * time.Second, Jar: jar}
	session := &Session{client: httpClient}
	for _, opt := range opts {
		opt(session)
	}
	if session.client == nil {
		session.client = http.DefaultClient
	}
	return session
}

// Do issues an HTTP request with authentication applied.
func (s *Session) Do(req *http.Request) (*http.Response, error) {
	if s == nil {
		return nil, fmt.Errorf("asf: nil session")
	}
	if s.authenticator != nil {
		if err := s.authenticator.Authenticate(req); err != nil {
			return nil, fmt.Errorf("asf: authenticate request: %w", err)
		}
	}
	return s.client.Do(req)
}
