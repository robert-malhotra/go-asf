package asf

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFeatureCollectionUnmarshal(t *testing.T) {
	data := []byte(`{
		"features": [{
			"geometry": null,
			"properties": {
				"centerLat": 10.5,
				"centerLon": -20.25,
				"startTime": "2023-01-01T00:00:00Z",
				"stopTime": "2023-01-01T00:05:00Z",
				"processingDate": "2023-01-01T00:10:00Z",
				"url": "https://example.com/file",
				"s3Urls": ["s3://example/file"],
				"fileName": "FILE",
				"fileID": "FILE123",
				"sceneName": "SCENE1",
				"groupID": "GROUP1",
				"platform": "Sentinel-1A",
				"sensor": "C-SAR",
				"granuleType": "SENTINEL_FRAME",
				"processingLevel": "L1",
				"flightDirection": "ASCENDING",
				"browse": "https://example.com/browse.png"
			}
		}]
	}`)

	var fc FeatureCollection
	if err := json.Unmarshal(data, &fc); err != nil {
		t.Fatalf("unmarshal feature collection: %v", err)
	}
	if len(fc.Features) != 1 {
		t.Fatalf("expected one feature, got %d", len(fc.Features))
	}
	props := fc.Features[0].Properties

	wantStart := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	wantStop := time.Date(2023, 1, 1, 0, 5, 0, 0, time.UTC)
	wantProcessing := time.Date(2023, 1, 1, 0, 10, 0, 0, time.UTC)

	if !props.StartTime.Equal(wantStart) {
		t.Fatalf("unexpected start time: %s", props.StartTime)
	}
	if !props.StopTime.Equal(wantStop) {
		t.Fatalf("unexpected stop time: %s", props.StopTime)
	}
	if !props.ProcessingDate.Equal(wantProcessing) {
		t.Fatalf("unexpected processing date: %s", props.ProcessingDate)
	}
	if props.Browse != "https://example.com/browse.png" {
		t.Fatalf("unexpected browse: %s", props.Browse)
	}
	if props.URL != "https://example.com/file" {
		t.Fatalf("unexpected url: %s", props.URL)
	}
	if len(props.S3Urls) != 1 || props.S3Urls[0] != "s3://example/file" {
		t.Fatalf("unexpected s3 urls: %v", props.S3Urls)
	}
	if props.SceneName == "" {
		t.Fatalf("expected scene name to be populated")
	}
}

func TestFeatureCollectionMarshal(t *testing.T) {
	props := Properties{
		StartTime:      time.Date(2023, 1, 2, 3, 4, 5, 0, time.UTC),
		StopTime:       time.Date(2023, 1, 2, 3, 9, 5, 0, time.UTC),
		ProcessingDate: time.Date(2023, 1, 2, 4, 0, 0, 0, time.UTC),
		URL:            "https://example.com/file",
		Browse:         "https://example.com/browse.png",
		S3Urls:         []string{"s3://example/file"},
		FileName:       "FILE",
		FileID:         "FILE123",
		SceneName:      "SCENE1",
		GroupID:        "GROUP1",
	}
	payload := FeatureCollection{
		Features: []Product{{Geometry: json.RawMessage("null"), Properties: props}},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal feature collection: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, `"startTime":"2023-01-02T03:04:05Z"`) {
		t.Fatalf("expected start time in JSON, got %s", got)
	}
	if !strings.Contains(got, `"processingDate":"2023-01-02T04:00:00Z"`) {
		t.Fatalf("expected processing date in JSON, got %s", got)
	}
	if !strings.Contains(got, `"url":"https://example.com/file"`) {
		t.Fatalf("expected url in JSON, got %s", got)
	}
}
