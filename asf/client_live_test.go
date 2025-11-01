//go:build live

package asf

import (
	"context"
	"strings"
	"testing"
	"time"
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

const sentinelAOIWKT = "POLYGON((-180 -90, 180 -90, 180 90, -180 90, -180 -90))"

var (
	sentinelAOIStart = time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)
	sentinelAOIEnd   = time.Date(2021, time.April, 1, 0, 0, 0, 0, time.UTC)
)
