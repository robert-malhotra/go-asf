//go:build live

package asf_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/example/go-asf/asf"
	"github.com/example/go-asf/asf/search"
)

func TestLiveSearchAndDownload(t *testing.T) {
	username := firstNonEmpty(os.Getenv("ASF_EARTHDATA_USERNAME"), os.Getenv("ASF_USERNAME"))
	if username == "" {
		t.Skip("ASF_EARTHDATA_USERNAME/ASF_USERNAME not set; skipping live download test")
	}
	password := firstNonEmpty(os.Getenv("ASF_EARTHDATA_PASSWORD"), os.Getenv("ASF_PASSWORD"))
	if password == "" {
		t.Skip("ASF_EARTHDATA_PASSWORD/ASF_PASSWORD not set; skipping live download test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := asf.NewClient(asf.WithBasicAuth(username, password))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	params := search.ParamsBuilder().
		Platform(search.PlatformSentinel1A).
		BeamMode(search.BeamModeIW).
		ProcessingLevel("METADATA").
		MaxResults(1).
		Build()

	iter, err := client.Search(ctx, params)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if !iter.Next(ctx) {
		if err := iter.Err(); err != nil {
			t.Fatalf("iteration error: %v", err)
		}
		t.Fatal("no products returned from live search")
	}
	product := iter.Product()
	if product.DownloadURL == "" {
		t.Fatalf("product missing download URL: %+v", product)
	}

	dir := t.TempDir()

	if err := client.DownloadProduct(ctx, product, dir); err != nil {
		t.Fatalf("download: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, found %d", len(entries))
	}

	path := filepath.Join(dir, entries[0].Name())
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat downloaded file: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("downloaded file is empty")
	}

	fh, err := os.Open(path)
	if err != nil {
		t.Fatalf("open downloaded file: %v", err)
	}
	defer fh.Close()

	header := make([]byte, 256)
	n, err := fh.Read(header)
	if err != nil && err != io.EOF {
		t.Fatalf("read downloaded file: %v", err)
	}
	trimmed := bytes.TrimSpace(header[:n])
	if len(trimmed) > 0 {
		upper := bytes.ToUpper(trimmed)
		if bytes.HasPrefix(upper, []byte("<!DOCTYPE")) || bytes.HasPrefix(upper, []byte("<HTML")) {
			t.Fatalf("downloaded HTML login page instead of data: %q", string(trimmed))
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
