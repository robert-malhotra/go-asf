package asf

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
		browse := server.URL + "/browse.png"
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
					StopTime:        "2023-01-01T00:05:00Z",
					ProcessingDate:  "2023-01-01T00:10:00Z",
					SizeMB:          numericValue(50.5),
					Bytes:           numericValue(1048576),
					SceneName:       "S1_SCENE",
					FileID:          "FILE123",
					FileName:        "S1_SCENE.zip",
					GroupID:         "GROUP1",
					Sensor:          "C-SAR",
					GranuleType:     "SENTINEL_FRAME",
					FlightDirection: string(FlightDirectionAscending),
					PathNumber:      numericValue(120),
					Orbit:           numericValue(98765),
					FrameNumber:     numericValue(42),
					MD5Sum:          "abc123",
					PgeVersion:      "004.00",
					CenterLat:       10.5,
					CenterLon:       -20.25,
					Browse:          stringList{browse},
					AdditionalURLs:  []string{server.URL + "/zzalternate"},
					S3URLs:          []string{"s3://bucket/object"},
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
	expectedStart := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	expectedStop := time.Date(2023, 1, 1, 0, 5, 0, 0, time.UTC)
	expectedProcessing := time.Date(2023, 1, 1, 0, 10, 0, 0, time.UTC)
	if got.GranuleID != "S1_SCENE" {
		t.Fatalf("unexpected granule: %+v", got)
	}
	if !got.Acquisition.Equal(expectedStart) {
		t.Fatalf("unexpected acquisition time: %s", got.Acquisition)
	}
	if !got.StartTime.Equal(expectedStart) {
		t.Fatalf("unexpected start time: %s", got.StartTime)
	}
	if !got.StopTime.Equal(expectedStop) {
		t.Fatalf("unexpected stop time: %s", got.StopTime)
	}
	if !got.ProcessingDate.Equal(expectedProcessing) {
		t.Fatalf("unexpected processing date: %s", got.ProcessingDate)
	}
	if got.DownloadURL != server.URL+"/file" {
		t.Fatalf("unexpected download URL: %s", got.DownloadURL)
	}
	if len(got.FileURLs) != 3 {
		t.Fatalf("unexpected file URLs: %v", got.FileURLs)
	}
	foundFile := false
	for _, u := range got.FileURLs {
		if u == server.URL+"/file" {
			foundFile = true
			break
		}
	}
	if !foundFile {
		t.Fatalf("expected file URL in results: %v", got.FileURLs)
	}
	if got.ProductType != ProductTypeSLC {
		t.Fatalf("unexpected product type: %s", got.ProductType)
	}
	if got.Bytes != 1048576 {
		t.Fatalf("unexpected bytes value: %d", got.Bytes)
	}
	if got.MD5Sum != "abc123" {
		t.Fatalf("unexpected md5: %s", got.MD5Sum)
	}
	if got.PgeVersion != "004.00" {
		t.Fatalf("unexpected pge version: %s", got.PgeVersion)
	}
	if got.SceneName != "S1_SCENE" {
		t.Fatalf("unexpected scene name: %s", got.SceneName)
	}
	if got.FileID != "FILE123" {
		t.Fatalf("unexpected file id: %s", got.FileID)
	}
	if got.FileName != "S1_SCENE.zip" {
		t.Fatalf("unexpected file name: %s", got.FileName)
	}
	if got.GroupID != "GROUP1" {
		t.Fatalf("unexpected group id: %s", got.GroupID)
	}
	if got.Sensor != "C-SAR" {
		t.Fatalf("unexpected sensor: %s", got.Sensor)
	}
	if got.GranuleType != "SENTINEL_FRAME" {
		t.Fatalf("unexpected granule type: %s", got.GranuleType)
	}
	if got.FlightDirection != FlightDirectionAscending {
		t.Fatalf("unexpected flight direction: %s", got.FlightDirection)
	}
	if got.PathNumber != 120 {
		t.Fatalf("unexpected path number: %d", got.PathNumber)
	}
	if got.Orbit != 98765 {
		t.Fatalf("unexpected orbit: %d", got.Orbit)
	}
	if got.FrameNumber != 42 {
		t.Fatalf("unexpected frame number: %d", got.FrameNumber)
	}
	if got.CenterLat != 10.5 {
		t.Fatalf("unexpected center lat: %f", got.CenterLat)
	}
	if got.CenterLon != -20.25 {
		t.Fatalf("unexpected center lon: %f", got.CenterLon)
	}
	if got.BrowseURL != server.URL+"/browse.png" {
		t.Fatalf("unexpected browse url: %s", got.BrowseURL)
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

func TestDownloadS3(t *testing.T) {
	ctx := context.Background()
	expiration := time.Now().Add(30 * time.Minute).UTC().Format("2006-01-02 15:04:05-07:00")
	var credentialRequests int
	credServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		credentialRequests++
		fmt.Fprintf(w, `{"accessKeyId":"AKIA","secretAccessKey":"SECRET","sessionToken":"TOKEN","expiration":"%s"}`, expiration)
	}))
	defer credServer.Close()

	client := NewClient(WithS3CredentialsURL(credServer.URL), WithAuthToken("token"))
	cfg, ok := client.s3.(*s3Config)
	if !ok {
		t.Fatalf("expected *s3Config, got %T", client.s3)
	}

	mock := &mockS3Downloader{content: []byte("s3data")}
	cfg.newDownloader = func(cfg aws.Config) s3Downloader {
		mock.cfg = cfg
		return mock
	}

	product := Product{FileURLs: []string{"s3://sentinel-bucket/path/file.dat"}}
	dest := filepath.Join(t.TempDir(), "file.dat")
	if err := client.Download(ctx, product, dest); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "s3data" {
		t.Fatalf("unexpected s3 file contents: %q", data)
	}
	if credentialRequests != 1 {
		t.Fatalf("expected single credential request, got %d", credentialRequests)
	}
	if mock.input == nil {
		t.Fatalf("expected downloader input to be recorded")
	}
	if got := aws.ToString(mock.input.Bucket); got != "sentinel-bucket" {
		t.Fatalf("unexpected bucket: %s", got)
	}
	if got := aws.ToString(mock.input.Key); got != "path/file.dat" {
		t.Fatalf("unexpected key: %s", got)
	}
	if mock.cfg.Region != defaultS3Region {
		t.Fatalf("expected region %s, got %s", defaultS3Region, mock.cfg.Region)
	}

	mock.content = []byte("s3data2")
	dest2 := filepath.Join(t.TempDir(), "file2.dat")
	if err := client.Download(ctx, product, dest2); err != nil {
		t.Fatalf("second Download returned error: %v", err)
	}
	data2, err := os.ReadFile(dest2)
	if err != nil {
		t.Fatalf("ReadFile second: %v", err)
	}
	if string(data2) != "s3data2" {
		t.Fatalf("unexpected s3 file contents: %q", data2)
	}
	if credentialRequests != 1 {
		t.Fatalf("expected credentials reused, got %d requests", credentialRequests)
	}
}

func TestDownloadMissingURL(t *testing.T) {
	client := NewClient()
	err := client.Download(context.Background(), Product{}, "ignored")
	if err == nil || !strings.Contains(err.Error(), "product missing download URL") {
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

type mockS3Downloader struct {
	content []byte
	input   *s3.GetObjectInput
	cfg     aws.Config
}

func (m *mockS3Downloader) Download(ctx context.Context, w io.WriterAt, input *s3.GetObjectInput, _ ...func(*manager.Downloader)) (int64, error) {
	copied := *input
	m.input = &copied
	if len(m.content) == 0 {
		return 0, fmt.Errorf("no content configured")
	}
	if _, err := w.WriteAt(m.content, 0); err != nil {
		return 0, err
	}
	return int64(len(m.content)), nil
}
