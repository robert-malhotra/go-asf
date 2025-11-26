# go-asf

Small Go client and CLI for querying and downloading products from the Alaska Satellite Facility (ASF) Search API.

## Why this exists
- Wraps the ASF search endpoint with typed options instead of raw query strings.
- Handles authentication (bearer, basic, or custom headers) and redirect-safe HTTP client setup.
- Downloads matching products concurrently with sensible error messages.
- Ships a simple CLI (`asfcli`) for quick searches or scripted downloads.

## Install
- Library: add `github.com/robert-malhotra/go-asf` to your module and `go get` as usual.
- CLI: `go install ./cmd/asfcli`

## Using the library
```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/robert-malhotra/go-asf/pkg/asf"
)

func main() {
	ctx := context.Background()
	client := asf.NewClient(asf.WithAuthToken("<ASF_TOKEN>"))

	opts := asf.SearchOptions{
		Platforms:       []asf.Platform{asf.PlatformSentinel1},
		ProcessingLevel: []asf.ProcessingLevel{asf.ProcessingLevelSLC},
		Start:           time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		End:             time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC),
		IntersectsWith:  "POLYGON ((-64.8 32.3, -65.5 18.3, -80.3 25.2, -64.8 32.3))",
		MaxResults:      10,
	}

	products, err := client.Search(ctx, opts)
	if err != nil {
		log.Fatalf("search failed: %v", err)
	}
	fmt.Printf("found %d products\n", len(products))

	// Download the first product (requires auth token)
	if len(products) > 0 {
		if err := client.Download(ctx, "./downloads", products[0]); err != nil {
			log.Fatalf("download failed: %v", err)
		}
	}
}
```

## Using the CLI
- Set `ASF_TOKEN` if you need authenticated downloads.
- Common searches:
  - `asfcli search --platform Sentinel-1 --processing-level SLC --start 2024-01-01T00:00:00Z --end 2025-01-31T23:59:59Z`
  - `asfcli search --platform Sentinel-1 --beam-mode IW --intersects "POLYGON ((-64.8 32.3, -65.5 18.3, -80.3 25.2, -64.8 32.3))" --max-results 5`
- Output formats:
  - Table (default): `--output text`
  - JSON: `--output json`
- Download results: append `--download-dir ./data` to fetch all matched products.

## Authentication
- Anonymous searches work for most filters.
- Downloads often require an ASF bearer token: set `ASF_TOKEN` or pass `--token` to the CLI.
- Library helpers:
  - `asf.WithAuthToken(token)`
  - `asf.BasicAuth(user, pass)`
  - `asf.HeaderAuth(map[string]string{...})`

## Tests
- Unit tests: `go test ./...`
- `pkg/asf/live_test.go` hits the real ASF API; it runs without auth for search validation. Download coverage in that test is skipped unless `ASF_TOKEN` is set.
