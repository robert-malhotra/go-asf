package asf

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSearchSuccess(t *testing.T) {
	ctx := context.Background()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/services/search/param" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("expected bearer token, got %q", got)
		}
		q := r.URL.Query()
		if got := q.Get("platform"); got != PlatformSentinel1A {
			t.Fatalf("expected platform Sentinel-1A, got %s", got)
		}
		if got := q.Get("output"); got != "geojson" {
			t.Fatalf("expected output=geojson, got %s", got)
		}
		if got := q.Get("start"); got == "" {
			t.Fatalf("expected start query to be set")
		}
		if got := q.Get("extra"); got != "value" {
			t.Fatalf("expected extra query value, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		payload := featureCollection{
			Features: []feature{{
				ID: "S1",
				Properties: featureProps{
					Platform:        PlatformSentinel1A,
					BeamMode:        BeamModeIW,
					Polarization:    PolarizationVV,
					ProcessingLevel: ProcessingLevelL1,
					ProductType:     ProductTypeSLC,
					StartTime:       "2023-01-01T00:00:00Z",
					SizeMB:          numericValue(50.5),
					Files:           []fileInfo{{URL: server.URL + "/file"}},
				},
			}},
		}
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			t.Fatalf("encode payload: %v", err)
		}
	}))
	defer server.Close()

	opts := SearchOptions{
		Platforms: []Platform{PlatformSentinel1A},
		Start:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		Extra:     url.Values{"extra": {"value"}},
	}

	client := NewClient(WithBaseURL(server.URL), WithAuthToken("token"))
	results, err := client.Search(ctx, opts)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	got := results[0]
	if got.GranuleID != "S1" {
		t.Fatalf("unexpected granule: %+v", got)
	}
	if got.DownloadURL != server.URL+"/file" {
		t.Fatalf("unexpected download URL: %s", got.DownloadURL)
	}
	if len(got.FileURLs) != 1 || got.FileURLs[0] != server.URL+"/file" {
		t.Fatalf("unexpected file URLs: %v", got.FileURLs)
	}
	if got.ProductType != ProductTypeSLC {
		t.Fatalf("unexpected product type: %s", got.ProductType)
	}
}

func TestSearchErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	_, err := client.Search(context.Background(), SearchOptions{})
	if err == nil || !strings.Contains(err.Error(), "unexpected status") {
		t.Fatalf("expected unexpected status error, got %v", err)
	}
}

func TestDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/download" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data"))
	}))
	defer server.Close()

	client := NewClient()
	product := Product{DownloadURL: server.URL + "/download"}
	dest := filepath.Join(t.TempDir(), "file.bin")
	if err := client.Download(context.Background(), product, dest); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}
	contents, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(contents) != "data" {
		t.Fatalf("unexpected file contents: %q", contents)
	}
}

func TestDownloadMissingURL(t *testing.T) {
	client := NewClient()
	err := client.Download(context.Background(), Product{}, "ignored")
	if !errors.Is(err, ErrMissingDownloadURL) {
		t.Fatalf("expected missing download URL error, got %v", err)
	}
}

func TestGeoSearchHelper(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("intersectsWith") != "POLYGON" {
			t.Fatalf("expected intersectsWith query")
		}
		if r.URL.Query().Get("platform") != PlatformSentinel1A {
			t.Fatalf("expected platform parameter")
		}
		json.NewEncoder(w).Encode(featureCollection{})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	if _, err := client.GeoSearch(ctx, "POLYGON", GeoSearchOptions{SearchOptions: SearchOptions{Platforms: []Platform{PlatformSentinel1A}}}); err != nil {
		t.Fatalf("GeoSearch returned error: %v", err)
	}
}

func TestGranuleSearchHelper(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := r.URL.Query()["granule_list"]
		if len(values) != 2 {
			t.Fatalf("expected two granule_list entries, got %v", values)
		}
		json.NewEncoder(w).Encode(featureCollection{})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	if _, err := client.GranuleSearch(ctx, []string{"A", "B"}, GranuleSearchOptions{}); err != nil {
		t.Fatalf("GranuleSearch returned error: %v", err)
	}
}

func TestDownloadURLsAggregatesErrors(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.Write([]byte("data"))
		default:
			http.Error(w, "fail", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	client := NewClient()
	urls := []string{server.URL + "/ok", server.URL + "/fail"}
	err := client.DownloadURLs(ctx, urls, dir, 2)
	if err == nil {
		t.Fatalf("expected error from DownloadURLs")
	}
	var batch BatchError
	if !errors.As(err, &batch) {
		t.Fatalf("expected BatchError, got %T", err)
	}
	if len(batch.Errors) != 1 {
		t.Fatalf("expected single error, got %d", len(batch.Errors))
	}
	content, err := os.ReadFile(filepath.Join(dir, "ok"))
	if err != nil {
		t.Fatalf("expected ok file to be written: %v", err)
	}
	if string(content) != "data" {
		t.Fatalf("unexpected file content: %s", content)
	}
}

func TestDownloadAll(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.URL.Path))
	}))
	defer server.Close()

	product := Product{FileURLs: []string{server.URL + "/a.bin", server.URL + "/b.bin"}}
	client := NewClient()
	if err := client.DownloadAll(ctx, []Product{product}, dir); err != nil {
		t.Fatalf("DownloadAll returned error: %v", err)
	}
	for _, name := range []string{"a.bin", "b.bin"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("expected file %s: %v", name, err)
		}
		if string(data) == "" {
			t.Fatalf("expected data for %s", name)
		}
	}
}

func TestCustomAuthenticator(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test") != "value" {
			t.Fatalf("expected custom header")
		}
		json.NewEncoder(w).Encode(featureCollection{})
	}))
	defer server.Close()

	auth := AuthenticatorFunc(func(req *http.Request) error {
		req.Header.Set("X-Test", "value")
		return nil
	})
	session := NewSession(WithSessionAuthenticator(auth))
	client := NewClient(WithBaseURL(server.URL), WithSession(session))
	if _, err := client.Search(ctx, SearchOptions{}); err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
}

func TestParseTime(t *testing.T) {
	cases := []string{
		"2023-01-01T00:00:00Z",
		"2023-01-01T00:00:00.000Z",
		time.Now().UTC().Format(time.RFC3339Nano),
	}
	for _, tc := range cases {
		if got := parseTime(tc); got.IsZero() {
			t.Fatalf("parseTime failed for %s", tc)
		}
	}
}
