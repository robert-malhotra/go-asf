package search

import (
	"net/url"
	"testing"
	"time"
)

func TestParamsEncodeValidation(t *testing.T) {
	params := New()
	params.Start = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := params.Encode(); err == nil {
		t.Fatalf("expected error when start is set without end")
	}

	params.End = params.Start.Add(-time.Hour)
	if _, err := params.Encode(); err == nil {
		t.Fatalf("expected error when start is after end")
	}
}

func TestParamsEncodeValues(t *testing.T) {
	params := New()
	params.Platform = PlatformSentinel1A
	params.BeamMode = BeamModeIW
	params.Polarization = "VV"
	params.ProcessingLevel = "METADATA"
	params.RelativeOrbit = 42
	params.Start = time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
	params.End = time.Date(2023, 1, 2, 10, 0, 0, 0, time.UTC)
	params.Dataset = "TEST"
	params.Collections = []string{"C456", "C123"}
	params.GranuleList = []string{"G1", "G2"}
	params.Add("custom", "value1")
	params.Add("custom", "value2")

	got, err := params.Encode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := url.Values{
		"output":          {"jsonlite"},
		"platform":        {"Sentinel-1A"},
		"beamMode":        {"IW"},
		"polarization":    {"VV"},
		"processingLevel": {"METADATA"},
		"relativeOrbit":   {"42"},
		"start":           {"2023-01-01T10:00:00Z"},
		"end":             {"2023-01-02T10:00:00Z"},
		"dataset":         {"TEST"},
		"collections":     {"C123", "C456"},
		"granule_list":    {"G1,G2"},
		"custom":          {"value1", "value2"},
	}

	if len(got) != len(want) {
		t.Fatalf("encoded values length mismatch: got %d want %d", len(got), len(want))
	}
	for key, wantVals := range want {
		gotVals, ok := got[key]
		if !ok {
			t.Fatalf("missing key %q", key)
		}
		if len(gotVals) != len(wantVals) {
			t.Fatalf("key %q length mismatch: got %v want %v", key, gotVals, wantVals)
		}
		for i := range gotVals {
			if gotVals[i] != wantVals[i] {
				t.Fatalf("key %q index %d mismatch: got %q want %q", key, i, gotVals[i], wantVals[i])
			}
		}
	}
	if _, ok := got["maxResults"]; ok {
		t.Fatalf("maxResults should be omitted when granule_list is set")
	}
}

func TestParamsEncodeOmitMaxResultsForProductList(t *testing.T) {
	params := New()
	params.ProductList = []string{"P1"}
	got, err := params.Encode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := got["maxResults"]; ok {
		t.Fatalf("maxResults should be omitted when product_list is provided")
	}
	if got.Get("product_list") != "P1" {
		t.Fatalf("unexpected product_list value: %q", got.Get("product_list"))
	}
}

func TestParamsPageSize(t *testing.T) {
	defaults := Params{}
	if got := defaults.PageSize(); got != 100 {
		t.Fatalf("default page size: got %d want 100", got)
	}

	params := Params{MaxResults: 50}
	if got := params.PageSize(); got != 50 {
		t.Fatalf("custom page size: got %d want 50", got)
	}
}
