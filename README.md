# go-asf

Go client for interacting with the Alaska Satellite Facility (ASF) search API and downloading product files.

## Features

- Query the ASF search endpoint with strongly-typed parameters and a fluent builder.
- Iterate search results from ASF's `jsonlite` output format with convenient Go structs.
- Download product files with configurable concurrency, checksum verification, and progress callbacks.
- Ship a CLI utility (`asfcli`) for ad-hoc searches and product downloads.

## Installation

```bash
go get github.com/example/go-asf
```

## Usage

```go
package main

import (
    "context"
    "log"
    "os"
    "time"

    "github.com/example/go-asf/asf"
    "github.com/example/go-asf/asf/search"
)

func main() {
    ctx := context.Background()
    client, err := asf.NewClient(
        asf.WithUserAgent("my-app/1.0"),
        asf.WithBasicAuth(os.Getenv("ASF_EARTHDATA_USERNAME"), os.Getenv("ASF_EARTHDATA_PASSWORD")),
    )
    if err != nil {
        log.Fatal(err)
    }

    params := search.ParamsBuilder().
        Platform(search.PlatformSentinel1A).
        BeamMode(search.BeamModeIW).
        ProcessingLevel("METADATA").
        MaxResults(1).
        Build()

    iter, err := client.Search(ctx, params)
    if err != nil {
        log.Fatal(err)
    }

    for iter.Next(ctx) {
        product := iter.Product()
        if err := client.DownloadProduct(ctx, product, "./downloads"); err != nil {
            log.Printf("download failed: %v", err)
        }
    }
    if err := iter.Err(); err != nil {
        log.Fatal(err)
    }
}
```

> **Note:** ASF downloads require NASA Earthdata credentials. Provide them via the `ASF_EARTHDATA_USERNAME` and `ASF_EARTHDATA_PASSWORD` environment variables or the CLI flags shown below.

## CLI

Build the CLI and perform a small search against the live ASF service:

```bash
go build ./cmd/asfcli
./asfcli search --platform=S1A --beam-mode=IW --processing-level=METADATA --max-results=1
```

Download a product into a destination directory (credentials may be supplied via flags or the `ASF_USERNAME`/`ASF_PASSWORD` environment variables):

```bash
./asfcli download \
  --product-id "S1A_IW_RAW__0SDV_20251007T131700_20251007T131718_061320_07A682_5386-METADATA_RAW" \
  --dest ./downloads \
  --username "$ASF_EARTHDATA_USERNAME" \
  --password "$ASF_EARTHDATA_PASSWORD"
```

## Live test

A network-enabled integration test is available behind the `live` build tag. It requires valid Earthdata credentials exported as `ASF_EARTHDATA_USERNAME` and `ASF_EARTHDATA_PASSWORD`:

```bash
ASF_EARTHDATA_USERNAME=your-user ASF_EARTHDATA_PASSWORD=your-pass go test -tags=live ./...
```

## License

MIT
