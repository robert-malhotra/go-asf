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
	params.ProductType = "GRD"
	params.RelativeOrbit = 42
	params.Start = time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
	params.End = time.Date(2023, 1, 2, 10, 0, 0, 0, time.UTC)
	params.Add("custom", "value1")
	params.Add("custom", "value2")

	got, err := params.Encode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := url.Values{
		"platform":      {"S1A"},
		"beamMode":      {"IW"},
		"polarization":  {"VV"},
		"productType":   {"GRD"},
		"relativeOrbit": {"42"},
		"start":         {"2023-01-01T10:00:00Z"},
		"end":           {"2023-01-02T10:00:00Z"},
		"custom":        {"value1", "value2"},
		"maxResults":    {"100"},
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
