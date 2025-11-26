package asf

import (
	"encoding/json"
	"time"
)

// Platform represents a supported mission/platform identifier.
type Platform string

const (
	PlatformSentinel1A Platform = "Sentinel-1A"
	PlatformSentinel1B Platform = "Sentinel-1B"
	PlatformSentinel1C Platform = "Sentinel-1C"
	PlatformSentinel1  Platform = "Sentinel-1"
)

// BeamMode enumerates radar beam mode values.
type BeamMode string

const (
	BeamModeIW BeamMode = "IW"
	BeamModeEW BeamMode = "EW"
	BeamModeSM BeamMode = "SM"
	BeamModeWV BeamMode = "WV"
)

// Polarization enumerates common SAR polarization strings.
type Polarization string

const (
	PolarizationHH Polarization = "HH"
	PolarizationHV Polarization = "HV"
	PolarizationVV Polarization = "VV"
	PolarizationVH Polarization = "VH"
	PolarizationQP Polarization = "QP"
)

// ProductType represents an ASF product type identifier.
type ProductType string

const (
	ProductTypeSLC      ProductType = "SLC"
	ProductTypeGRD      ProductType = "GRD"
	ProductTypeGRDMD    ProductType = "GRD_MD"
	ProductTypeOCN      ProductType = "OCN"
	ProductTypeRAW      ProductType = "RAW"
	ProductTypeMETADATA ProductType = "METADATA"
)

// CollectionName denotes an ASF collection value.
type CollectionName string

const (
	CollectionSentinel1 CollectionName = "SENTINEL-1"
)

// ProcessingLevel enumerates the processing level strings.
type ProcessingLevel string

const (
	ProcessingLevelL0    ProcessingLevel = "L0"
	ProcessingLevelL1    ProcessingLevel = "L1"
	ProcessingLevelL2    ProcessingLevel = "L2"
	ProcessingLevelSLC   ProcessingLevel = "SLC"
	ProcessingLevelGRD   ProcessingLevel = "GRD"
	ProcessingLevelGRDMD ProcessingLevel = "GRD_MD"
	ProcessingLevelGRDHD ProcessingLevel = "GRD_HD"
)

// LookDirection describes the look direction parameter.
type LookDirection string

const (
	LookDirectionLeft  LookDirection = "LEFT"
	LookDirectionRight LookDirection = "RIGHT"
)

// FlightDirection enumerates valid flight direction filters.
type FlightDirection string

const (
	FlightDirectionAscending  FlightDirection = "ASCENDING"
	FlightDirectionDescending FlightDirection = "DESCENDING"
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
