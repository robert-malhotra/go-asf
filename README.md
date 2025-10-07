# go-asf

Go client for interacting with the Alaska Satellite Facility (ASF) search API and downloading product files.

## Features

- Query the ASF search endpoint with strongly-typed parameters and a fluent builder.
- Iterate search results transparently across paginated responses.
- Download product files with configurable concurrency, checksum verification, and progress callbacks.

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
    "time"

    "github.com/example/go-asf/asf"
    "github.com/example/go-asf/asf/search"
)

func main() {
    ctx := context.Background()
    client, err := asf.NewClient(
        asf.WithUserAgent("my-app/1.0"),
    )
    if err != nil {
        log.Fatal(err)
    }

    params := search.ParamsBuilder().
        Platform("S1A").
        BeamMode("IW").
        Polarization("VV").
        StartTime(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)).
        EndTime(time.Date(2023, 1, 31, 23, 59, 59, 0, time.UTC)).
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

## License

MIT

