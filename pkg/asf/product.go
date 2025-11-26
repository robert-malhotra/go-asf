package asf

import (
	"encoding/json"
	"time"
)

// FeatureCollectionResponse represents the top-level GeoJSON FeatureCollection
type FeatureCollection struct {
	Features []Product `json:"features"`
}

// Feature represents a single feature in the collection
type Product struct {
	Geometry   json.RawMessage `json:"geometry"`
	Properties Properties      `json:"properties"`
}

// Properties represents the metadata associated with a feature
type Properties struct {
	CenterLat       float64   `json:"centerLat"`
	CenterLon       float64   `json:"centerLon"`
	StopTime        time.Time `json:"stopTime"`
	FileID          string    `json:"fileID"`
	FlightDirection string    `json:"flightDirection"`
	PathNumber      int       `json:"pathNumber"`
	ProcessingLevel string    `json:"processingLevel"`
	URL             string    `json:"url"`
	StartTime       time.Time `json:"startTime"`
	SceneName       string    `json:"sceneName"`
	Browse          string    `json:"browse"`
	Platform        string    `json:"platform"`
	Bytes           int64     `json:"bytes"`
	Md5sum          string    `json:"md5sum"`
	FrameNumber     int       `json:"frameNumber"`
	GranuleType     string    `json:"granuleType"`
	Orbit           int       `json:"orbit"`
	Polarization    string    `json:"polarization"`
	ProcessingDate  time.Time `json:"processingDate"`
	Sensor          string    `json:"sensor"`
	GroupID         string    `json:"groupID"`
	PgeVersion      string    `json:"pgeVersion"`
	FileName        string    `json:"fileName"`
	BeamModeType    string    `json:"beamModeType"`
	S3Urls          []string  `json:"s3Urls"`
}
