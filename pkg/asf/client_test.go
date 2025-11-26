package asf

import (
	"context"
	"net/http"
	"net/http/httptest" // Import the httptest package
	"os"                // Import the os package to read the file
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSearchSuccess(t *testing.T) {
	ctx := context.Background()

	// Load the realistic JSON response from the file
	payloadBytes, err := os.ReadFile("asf_response.json")
	if err != nil {
		t.Fatalf("failed to read asf_response.json: %v", err)
	}

	// --- Expected values from the first feature in asf_response.json ---
	const (
		expPlatform        = "Sentinel-1C"
		expProcessingLevel = "SLC"
		expBeamMode        = "IW"
		expIntersectsWith  = "POLYGON((-123.8 49.1,-123.4 49.1,-123.4 49.5,-123.8 49.5,-123.8 49.1))"
		expSceneName       = "S1C_IW_SLC__1SDV_20251028T021014_20251028T021042_004756_00963D_8B2E"
		expFileURL         = "https://datapool.asf.alaska.edu/SLC/SC/S1C_IW_SLC__1SDV_20251028T021014_20251028T021042_004756_00963D_8B2E.zip"
		expMd5sum          = "10f2083f7b859bde5fb985722c4fe0b0"
		expFileID          = "S1C_IW_SLC__1SDV_20251028T021014_20251028T021042_004756_00963D_8B2E-SLC"
		expFileName        = "S1C_IW_SLC__1SDV_20251028T021014_20251028T021042_004756_00963D_8B2E.zip"
		expGroupID         = "S1C_IWDV_0160_0165_004756_035"
		expS3URL           = "s3://asf-ngap2w-p-s1-slc-7b420b89/S1C_IW_SLC__1SDV_20251028T021014_20251028T021042_004756_00963D_8B2E.zip"
	)

	// Helper to parse times, failing the test on error
	mustParseTime := func(value string) time.Time {
		t.Helper()
		ts, err := time.Parse(time.RFC3339, value)
		if err != nil {
			t.Fatalf("failed to parse time %q: %v", value, err)
		}
		return ts
	}

	// Expected times
	expSearchStart := mustParseTime("2024-01-01T00:00:00Z")
	expSearchEnd := mustParseTime("2026-01-31T23:59:59Z")
	expStartTime := mustParseTime("2025-10-28T02:10:14Z")
	expStopTime := mustParseTime("2025-10-28T02:10:42Z")
	expProcessingDate := mustParseTime("2025-10-28T02:10:14Z")

	// Create a new test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate the request path
		if r.URL.Path != "/services/search/param" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Validate auth header
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Errorf("expected bearer token, got %q", got)
		}

		// Validate query parameters
		q := r.URL.Query()
		if got := q.Get("platform"); got != expPlatform {
			t.Errorf("expected platform %q, got %q", expPlatform, got)
		}
		if got := q.Get("processingLevel"); got != expProcessingLevel {
			t.Errorf("expected processingLevel %q, got %q", expProcessingLevel, got)
		}
		if got := q.Get("beamMode"); got != expBeamMode {
			t.Errorf("expected beamMode %q, got %q", expBeamMode, got)
		}
		if got := q.Get("start"); got != expSearchStart.Format(time.RFC3339) {
			t.Errorf("expected start %q, got %q", expSearchStart.Format(time.RFC3339), got)
		}
		if got := q.Get("end"); got != expSearchEnd.Format(time.RFC3339) {
			t.Errorf("expected end %q, got %q", expSearchEnd.Format(time.RFC3339), got)
		}
		if got := q.Get("intersectsWith"); got != expIntersectsWith {
			t.Errorf("expected intersectsWith %q, got %q", expIntersectsWith, got)
		}
		if got := q.Get("output"); got != "geojson" {
			t.Errorf("expected output=geojson, got %s", got)
		}

		// Write the realistic response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(payloadBytes)
	}))
	defer server.Close()

	// --- Client-side setup ---
	opts := SearchOptions{
		Platforms:       []Platform{Platform(expPlatform)},
		ProcessingLevel: []ProcessingLevel{ProcessingLevel(expProcessingLevel)},
		BeamModes:       []BeamMode{BeamMode(expBeamMode)},
		Start:           expSearchStart,
		End:             expSearchEnd,
		IntersectsWith:  expIntersectsWith,
	}

	client := NewClient(
		WithBaseURL(server.URL), // Use the test server's URL
		WithAuthToken("token"),
	)
	results, err := client.Search(ctx, opts)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Check the first product
	got := results[0]
	props := got.Properties
	if props.SceneName != expSceneName {
		t.Fatalf("unexpected scene name: %s", props.SceneName)
	}
	if !props.StartTime.Equal(expStartTime) {
		t.Fatalf("unexpected start time: %s", props.StartTime)
	}
	if !props.StopTime.Equal(expStopTime) {
		t.Fatalf("unexpected stop time: %s", props.StopTime)
	}
	if !props.ProcessingDate.Equal(expProcessingDate) {
		t.Fatalf("unexpected processing date: %s", props.ProcessingDate)
	}
	if props.URL != expFileURL {
		t.Fatalf("unexpected primary download URL: %s", props.URL)
	}

	// Check combined URLs
	urls := append([]string{props.URL}, props.S3Urls...)
	if len(urls) != 2 {
		t.Fatalf("unexpected file URLs: %v", urls)
	}

	if props.ProcessingLevel != expProcessingLevel {
		t.Fatalf("unexpected processing level: %s", props.ProcessingLevel)
	}
	if props.Bytes != 4636443928 {
		t.Fatalf("unexpected bytes value: %d", props.Bytes)
	}
	if props.Md5sum != expMd5sum {
		t.Fatalf("unexpected md5: %s", props.Md5sum)
	}
	if props.PgeVersion != "004.00" {
		t.Fatalf("unexpected pge version: %s", props.PgeVersion)
	}
	if props.FileID != expFileID {
		t.Fatalf("unexpected file id: %s", props.FileID)
	}
	if props.FileName != expFileName {
		t.Fatalf("unexpected file name: %s", props.FileName)
	}
	if props.GroupID != expGroupID {
		t.Fatalf("unexpected group id: %s", props.GroupID)
	}
	if props.Sensor != "C-SAR" {
		t.Fatalf("unexpected sensor: %s", props.Sensor)
	}
	if props.GranuleType != "SENTINEL_1C_FRAME" {
		t.Fatalf("unexpected granule type: %s", props.GranuleType)
	}
	if props.FlightDirection != string(FlightDirectionAscending) {
		t.Fatalf("unexpected flight direction: %s", props.FlightDirection)
	}
	if props.PathNumber != 35 {
		t.Fatalf("unexpected path number: %d", props.PathNumber)
	}
	if props.Orbit != 4756 {
		t.Fatalf("unexpected orbit: %d", props.Orbit)
	}
	if props.FrameNumber != 160 {
		t.Fatalf("unexpected frame number: %d", props.FrameNumber)
	}
	if props.CenterLat != 50.0631 {
		t.Fatalf("unexpected center lat: %f", props.CenterLat)
	}
	if props.CenterLon != -125.4036 {
		t.Fatalf("unexpected center lon: %f", props.CenterLon)
	}
	// "browse" is null in the JSON, which unmarshals to an empty string
	if props.Browse != "" {
		t.Fatalf("expected empty browse url, got: %s", props.Browse)
	}
	if len(props.S3Urls) != 1 || props.S3Urls[0] != expS3URL {
		t.Fatalf("unexpected s3 urls: %v", props.S3Urls)
	}
}

func TestSearchErrorStatus(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("boom"))
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL), // Use the test server's URL
	)
	_, err := client.Search(context.Background(), SearchOptions{})
	if err == nil || !strings.Contains(err.Error(), "unexpected status") {
		t.Fatalf("expected unexpected status error, got %v", err)
	}
}

func TestDownloadSuccess(t *testing.T) {
	ctx := context.Background()
	const fileContent = "This is the file content"
	const token = "dl-token"

	// Keep track of which files were requested
	var requestedFiles sync.Map

	// Create a test server that mimics a file download
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check auth
		if got := r.Header.Get("Authorization"); got != "Bearer "+token {
			t.Errorf("expected bearer token, got %q", got)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Record the request path
		requestedFiles.Store(r.URL.Path, true)

		// Send back file content
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fileContent))
	}))
	defer server.Close()

	products := []Product{
		{Properties: Properties{
			SceneName: "scene1",
			FileName:  "file1.zip",
			URL:       server.URL + "/file1.zip",
		}},
		{Properties: Properties{
			SceneName: "scene2",
			FileName:  "file2.zip",
			URL:       server.URL + "/file2.zip",
		}},
	}

	targetDir := t.TempDir()
	client := NewClient(WithAuthToken(token))

	err := client.Download(ctx, targetDir, products...)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	// Verify files were created and content is correct
	for _, p := range products {
		// Check if file was requested
		if _, ok := requestedFiles.Load("/" + p.Properties.FileName); !ok {
			t.Errorf("server did not receive request for %q", p.Properties.FileName)
		}

		// Check file on disk
		destPath := filepath.Join(targetDir, p.Properties.FileName)
		content, err := os.ReadFile(destPath)
		if err != nil {
			t.Fatalf("failed to read downloaded file %q: %v", destPath, err)
		}
		if string(content) != fileContent {
			t.Fatalf("file content mismatch for %q: got %q, want %q", destPath, string(content), fileContent)
		}
	}
}

func TestDownloadErrors(t *testing.T) {
	ctx := context.Background()

	// Test case 1: Target folder cannot be created
	t.Run("BadTargetFolder", func(t *testing.T) {
		// Create a file to block directory creation
		badDir := filepath.Join(t.TempDir(), "a_file")
		f, _ := os.Create(badDir)
		f.Close()

		client := NewClient()
		products := []Product{{Properties: Properties{SceneName: "s", FileName: "f", URL: "u"}}}
		err := client.Download(ctx, badDir, products...)
		if err == nil || !strings.Contains(err.Error(), "create target folder") {
			t.Fatalf("expected folder creation error, got: %v", err)
		}
	})

	// Test case 2: Product has no URL
	t.Run("MissingURL", func(t *testing.T) {
		client := NewClient()
		products := []Product{{Properties: Properties{SceneName: "s", FileName: "f"}}} // No URL
		err := client.Download(ctx, t.TempDir(), products...)
		if err == nil || !strings.Contains(err.Error(), "has no URL") {
			t.Fatalf("expected 'has no URL' error, got: %v", err)
		}
	})

	// Test case 3: Product has no FileName
	t.Run("MissingFileName", func(t *testing.T) {
		client := NewClient()
		products := []Product{{Properties: Properties{SceneName: "s", URL: "u"}}} // No FileName
		err := client.Download(ctx, t.TempDir(), products...)
		if err == nil || !strings.Contains(err.Error(), "has no FileName") {
			t.Fatalf("expected 'has no FileName' error, got: %v", err)
		}
	})

	// Test case 4: Server returns an error
	t.Run("ServerError", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/good.zip" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ok"))
				return
			}
			// Fail for file2
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}))
		defer server.Close()

		products := []Product{
			{Properties: Properties{SceneName: "good", FileName: "good.zip", URL: server.URL + "/good.zip"}},
			{Properties: Properties{SceneName: "bad", FileName: "bad.zip", URL: server.URL + "/bad.zip"}},
		}

		client := NewClient()
		err := client.Download(ctx, t.TempDir(), products...)
		if err == nil || !strings.Contains(err.Error(), "unexpected download status") {
			t.Fatalf("expected 'unexpected download status' error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "bad.zip") {
			t.Fatalf("error message did not contain the failing file name: %v", err)
		}
	})

	// Test case 5: Context cancelled
	t.Run("ContextCancelled", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond) // Make download slow
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		products := []Product{
			{Properties: Properties{SceneName: "s1", FileName: "f1.zip", URL: server.URL + "/f1.zip"}},
			{Properties: Properties{SceneName: "s2", FileName: "f2.zip", URL: server.URL + "/f2.zip"}},
		}

		client := NewClient()
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond) // Very short timeout
		defer cancel()

		err := client.Download(ctx, t.TempDir(), products...)
		if err == nil || !strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
			t.Fatalf("expected context deadline error, got: %v", err)
		}
	})
}
