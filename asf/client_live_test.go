//go:build live

package asf

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestLiveSearchSentinel1SLC(t *testing.T) {
	if testing.Short() {
		t.Skip("live search requires network access")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	client := NewClient()
	basicOpts := SearchOptions{
		Platforms:       []Platform{PlatformSentinel1A, PlatformSentinel1B},
		ProcessingLevel: []ProcessingLevel{ProcessingLevelSLC},
		MaxResults:      5,
	}

	products := searchSentinelProducts(t, ctx, client, basicOpts)
	if !hasProcessingLevel(products, "SLC") {
		t.Fatalf("expected an SLC product, got processing levels: %v", collectProcessingLevels(products))
	}

	t.Run("AreaAndTimeCoverage", func(t *testing.T) {
		areaOpts := basicOpts
		areaOpts.IntersectsWith = sentinelAOIWKT
		areaOpts.Start = sentinelAOIStart
		areaOpts.End = sentinelAOIEnd

		areaProducts := searchSentinelProducts(t, ctx, client, areaOpts)
		ensureAcquisitionInRange(t, areaProducts, sentinelAOIStart, sentinelAOIEnd)
	})

	t.Run("GranuleLookup", func(t *testing.T) {
		granuleID := products[0].GranuleID
		if granuleID == "" {
			t.Fatalf("expected granule ID in results: %#v", products[0])
		}
		granuleProducts := searchGranuleByID(t, ctx, client, granuleID)
		found := false
		for _, product := range granuleProducts {
			if product.GranuleID == granuleID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("granule search did not return requested ID %q", granuleID)
		}
	})
}

func TestLiveSearchSentinel1GRD(t *testing.T) {
	if testing.Short() {
		t.Skip("live search requires network access")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	client := NewClient()
	basicOpts := SearchOptions{
		Platforms:       []Platform{PlatformSentinel1A, PlatformSentinel1B},
		ProcessingLevel: []ProcessingLevel{ProcessingLevelGRDMD},
		MaxResults:      5,
	}

	products := searchSentinelProducts(t, ctx, client, basicOpts)
	if !hasProcessingLevel(products, "GRD") {
		t.Fatalf("expected a GRD product, got processing levels: %v", collectProcessingLevels(products))
	}

	t.Run("AreaAndTimeCoverage", func(t *testing.T) {
		areaOpts := basicOpts
		areaOpts.IntersectsWith = sentinelAOIWKT
		areaOpts.Start = sentinelAOIStart
		areaOpts.End = sentinelAOIEnd

		areaProducts := searchSentinelProducts(t, ctx, client, areaOpts)
		ensureAcquisitionInRange(t, areaProducts, sentinelAOIStart, sentinelAOIEnd)
	})

	t.Run("GranuleLookup", func(t *testing.T) {
		granuleID := products[0].GranuleID
		if granuleID == "" {
			t.Fatalf("expected granule ID in results: %#v", products[0])
		}
		granuleProducts := searchGranuleByID(t, ctx, client, granuleID)
		found := false
		for _, product := range granuleProducts {
			if product.GranuleID == granuleID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("granule search did not return requested ID %q", granuleID)
		}
	})
}

func TestLiveDownloadWithBearerToken(t *testing.T) {
	if testing.Short() {
		t.Skip("live download requires network access")
	}
	token := os.Getenv("ASF_TEST_BEARER_TOKEN")
	if token == "" {
		t.Skip("set ASF_TEST_BEARER_TOKEN to run live download test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	client := NewClient(WithAuthToken(token))
	opts := SearchOptions{
		Platforms:       []Platform{PlatformSentinel1A, PlatformSentinel1B},
		ProcessingLevel: []ProcessingLevel{ProcessingLevelGRDMD},
		MaxResults:      5,
	}

	products := searchSentinelProducts(t, ctx, client, opts)
	downloadURL := firstDownloadURL(products)
	if downloadURL == "" {
		t.Fatalf("no download URL found in products: %+v", products)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		t.Fatalf("create download request: %v", err)
	}
	req.Header.Set("Range", "bytes=0-0")

	resp, err := client.session.Do(req)
	if err != nil {
		t.Fatalf("download request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected download status: %d", resp.StatusCode)
	}
	if _, err := io.CopyN(io.Discard, resp.Body, 1); err != nil && err != io.EOF {
		t.Fatalf("reading download response: %v", err)
	}
}

func TestLiveDownloadS3WithTemporaryCredentials(t *testing.T) {
	if testing.Short() {
		t.Skip("live download requires network access")
	}
	token := os.Getenv("ASF_TEST_BEARER_TOKEN")
	if token == "" {
		t.Skip("set ASF_TEST_BEARER_TOKEN to run live S3 download test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	client := NewClient(WithAuthToken(token))
	opts := SearchOptions{
		Platforms:       []Platform{PlatformSentinel1A, PlatformSentinel1B},
		ProcessingLevel: []ProcessingLevel{ProcessingLevelSLC},
		MaxResults:      5,
	}

	products := searchSentinelProducts(t, ctx, client, opts)

	var (
		s3URL  string
		target Product
	)
	for _, product := range products {
		for _, candidate := range product.FileURLs {
			if strings.HasPrefix(strings.ToLower(candidate), "s3://") {
				s3URL = candidate
				target = product
				break
			}
		}
		if s3URL != "" {
			break
		}
	}
	if s3URL == "" {
		t.Fatalf("expected at least one S3 URL in search results")
	}

	parsed, err := url.Parse(s3URL)
	if err != nil {
		t.Fatalf("parse s3 url: %v", err)
	}
	if parsed.Host == "" {
		t.Fatalf("s3 url missing bucket: %s", s3URL)
	}
	if strings.TrimPrefix(parsed.Path, "/") == "" {
		t.Fatalf("s3 url missing key: %s", s3URL)
	}

	s3cfg, ok := client.s3.(*s3Config)
	if !ok {
		t.Fatalf("unexpected s3 configuration type %T", client.s3)
	}

	s3cfg.mu.Lock()
	originalFactory := s3cfg.newDownloader
	s3cfg.newDownloader = func(cfg aws.Config) s3Downloader {
		return &rangeDownloader{
			client: s3.NewFromConfig(cfg),
			limit:  1024,
		}
	}
	s3cfg.downloaders = make(map[string]s3Downloader)
	s3cfg.mu.Unlock()
	defer func() {
		s3cfg.mu.Lock()
		s3cfg.newDownloader = originalFactory
		s3cfg.downloaders = make(map[string]s3Downloader)
		s3cfg.mu.Unlock()
	}()

	target.FileURLs = []string{s3URL}
	target.DownloadURL = ""

	dest := filepath.Join(t.TempDir(), fmt.Sprintf("%s.partial", target.GranuleID))
	if err := client.Download(ctx, target, dest); err != nil {
		message := err.Error()
		if strings.Contains(message, "AccessDenied") || strings.Contains(message, "SignatureDoesNotMatch") {
			t.Skipf("s3 download requires running within the authorized AWS region: %v", err)
		}
		t.Fatalf("download s3 url: %v", err)
	}

	info, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("stat downloaded file: %v", err)
	}
	if info.Size() == 0 || info.Size() > 2048 {
		t.Fatalf("unexpected download size %d", info.Size())
	}
}

func hasProcessingLevel(products []Product, want string) bool {
	want = strings.ToUpper(want)
	for _, product := range products {
		level := strings.ToUpper(product.ProcessingLevel)
		if level == want || strings.HasPrefix(level, want+"_") {
			return true
		}
		productType := strings.ToUpper(product.ProductType)
		if productType == want || strings.HasPrefix(productType, want+"_") {
			return true
		}
	}
	return false
}

func collectProcessingLevels(products []Product) []string {
	levels := make([]string, 0, len(products))
	for _, product := range products {
		if product.ProcessingLevel != "" {
			levels = append(levels, product.ProcessingLevel)
			continue
		}
		if product.ProductType != "" {
			levels = append(levels, product.ProductType)
		}
	}
	return levels
}

func searchSentinelProducts(t *testing.T, ctx context.Context, client *Client, opts SearchOptions) []Product {
	t.Helper()
	products, err := client.Search(ctx, opts)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(products) == 0 {
		t.Fatalf("expected at least one product for opts %+v", opts)
	}
	return products
}

func searchGranuleByID(t *testing.T, ctx context.Context, client *Client, granuleID string) []Product {
	t.Helper()
	products, err := client.GranuleSearch(ctx, []string{granuleID}, GranuleSearchOptions{})
	if err != nil {
		t.Fatalf("granule search returned error: %v", err)
	}
	if len(products) == 0 {
		t.Fatalf("granule search returned no products for %q", granuleID)
	}
	return products
}

func ensureAcquisitionInRange(t *testing.T, products []Product, start, end time.Time) {
	t.Helper()
	if end.Before(start) {
		t.Fatalf("invalid acquisition range: %s before %s", end, start)
	}
	for _, product := range products {
		if product.Acquisition.IsZero() {
			continue
		}
		if (product.Acquisition.Before(start) && !product.Acquisition.Equal(start)) || (product.Acquisition.After(end) && !product.Acquisition.Equal(end)) {
			t.Fatalf("product acquisition %s outside range %s - %s", product.Acquisition, start, end)
		}
	}
	if !hasAcquisitionInRange(products, start, end) {
		t.Fatalf("expected at least one acquisition within %s - %s, got %v", start, end, collectAcquisitionTimes(products))
	}
}

func hasAcquisitionInRange(products []Product, start, end time.Time) bool {
	for _, product := range products {
		if product.Acquisition.IsZero() {
			continue
		}
		if (product.Acquisition.Equal(start) || product.Acquisition.After(start)) && (product.Acquisition.Equal(end) || product.Acquisition.Before(end)) {
			return true
		}
	}
	return false
}

func collectAcquisitionTimes(products []Product) []time.Time {
	times := make([]time.Time, 0, len(products))
	for _, product := range products {
		if !product.Acquisition.IsZero() {
			times = append(times, product.Acquisition)
		}
	}
	return times
}

type rangeDownloader struct {
	client *s3.Client
	limit  int64
}

func (d *rangeDownloader) Download(ctx context.Context, w io.WriterAt, input *s3.GetObjectInput, optFns ...func(*manager.Downloader)) (int64, error) {
	if d == nil || d.client == nil {
		return 0, fmt.Errorf("nil range downloader")
	}
	if input == nil {
		return 0, fmt.Errorf("nil download input")
	}
	limit := d.limit
	if limit <= 0 {
		limit = 1024
	}
	input.Range = aws.String(fmt.Sprintf("bytes=0-%d", limit-1))
	resp, err := d.client.GetObject(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("get object: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return int64(len(data)), fmt.Errorf("read object: %w", err)
	}
	if len(data) > 0 {
		if _, err := w.WriteAt(data, 0); err != nil {
			return int64(len(data)), fmt.Errorf("write data: %w", err)
		}
	}
	return int64(len(data)), nil
}

func firstDownloadURL(products []Product) string {
	for _, product := range products {
		for _, url := range product.FileURLs {
			if url != "" {
				return url
			}
		}
		if product.DownloadURL != "" {
			return product.DownloadURL
		}
	}
	return ""
}

const sentinelAOIWKT = "POLYGON((-180 -90, 180 -90, 180 90, -180 90, -180 -90))"

var (
	sentinelAOIStart = time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)
	sentinelAOIEnd   = time.Date(2021, time.April, 1, 0, 0, 0, 0, time.UTC)
)
