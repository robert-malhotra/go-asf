package search

import (
	"testing"
	"time"
)

func TestBuilderSetsFields(t *testing.T) {
	start := time.Date(2022, 5, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	params := ParamsBuilder().
		Platform(PlatformSentinel1B).
		BeamMode(BeamModeEW).
		Polarization("VH").
		ProcessingLevel("L1").
		FlightDirection("ASCENDING").
		RelativeOrbit(128).
		StartTime(start).
		EndTime(end).
		IntersectsWith("POLYGON((0 0,1 0,1 1,0 1,0 0))").
		LookDirection("RIGHT").
		Dataset("TEST").
		AddCollection("C1").
		GranuleList("G1", "G2").
		ProductList("P1").
		MaxResults(10).
		Set("k", "v").
		Add("k", "v2").
		Build()

	if params.Platform != PlatformSentinel1B {
		t.Fatalf("platform mismatch")
	}
	if params.BeamMode != BeamModeEW {
		t.Fatalf("beam mode mismatch")
	}
	if params.Polarization != "VH" || params.ProcessingLevel != "L1" {
		t.Fatalf("string fields not set")
	}
	if params.FlightDirection != "ASCENDING" || params.LookDirection != "RIGHT" {
		t.Fatalf("direction fields not set")
	}
	if params.RelativeOrbit != 128 {
		t.Fatalf("relative orbit not set")
	}
	if !params.Start.Equal(start) || !params.End.Equal(end) {
		t.Fatalf("time fields mismatch")
	}
	if params.Dataset != "TEST" {
		t.Fatalf("dataset not set")
	}
	if len(params.Collections) != 1 || params.Collections[0] != "C1" {
		t.Fatalf("collections not set")
	}
	if len(params.GranuleList) != 2 || params.GranuleList[0] != "G1" {
		t.Fatalf("granule list not set")
	}
	if len(params.ProductList) != 1 || params.ProductList[0] != "P1" {
		t.Fatalf("product list not set")
	}
	if params.MaxResults != 10 {
		t.Fatalf("max results not set")
	}
	if got := params.Additional["k"]; len(got) != 2 || got[0] != "v" || got[1] != "v2" {
		t.Fatalf("additional values mismatch: %#v", got)
	}
}
