package search

// Platform represents a supported satellite platform identifier.
type Platform string

const (
	PlatformSentinel1A Platform = "S1A"
	PlatformSentinel1B Platform = "S1B"
	PlatformSentinel1C Platform = "S1C"
	PlatformSentinel2A Platform = "S2A"
	PlatformSentinel2B Platform = "S2B"
	PlatformSentinel3A Platform = "S3A"
	PlatformSentinel3B Platform = "S3B"
	PlatformALOS       Platform = "ALOS"
	PlatformRADARSAT1  Platform = "RADARSAT-1"
	PlatformRADARSAT2  Platform = "RADARSAT-2"
)

// String returns the underlying string value.
func (p Platform) String() string {
	return string(p)
}

// BeamMode represents a supported beam mode identifier.
type BeamMode string

const (
	BeamModeEW            BeamMode = "EW"
	BeamModeIW            BeamMode = "IW"
	BeamModeSM            BeamMode = "SM"
	BeamModeWV            BeamMode = "WV"
	BeamModeScanSARWide   BeamMode = "SCAN SAR WIDE"
	BeamModeScanSARNarrow BeamMode = "SCAN SAR NARROW"
)

// String returns the underlying string value.
func (b BeamMode) String() string {
	return string(b)
}
