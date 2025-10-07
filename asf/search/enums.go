package search

// Platform represents a supported satellite platform identifier.
type Platform string

const (
	PlatformSentinel1A Platform = "Sentinel-1A"
	PlatformSentinel1B Platform = "Sentinel-1B"
	PlatformSentinel1C Platform = "Sentinel-1C"
	PlatformSentinel1  Platform = "SENTINEL-1"
	PlatformALOS       Platform = "ALOS"
	PlatformAIRSAR     Platform = "AIRSAR"
	PlatformERS1       Platform = "ERS-1"
	PlatformERS2       Platform = "ERS-2"
	PlatformJERS1      Platform = "JERS-1"
	PlatformRADARSAT1  Platform = "RADARSAT-1"
	PlatformSIRC       Platform = "SIR-C"
	PlatformSMAP       Platform = "SMAP"
	PlatformUAVSAR     Platform = "UAVSAR"
)

// String returns the underlying string value.
func (p Platform) String() string {
	return string(p)
}

// BeamMode represents a supported beam mode identifier.
type BeamMode string

const (
	BeamModeEW  BeamMode = "EW"
	BeamModeIW  BeamMode = "IW"
	BeamModeWV  BeamMode = "WV"
	BeamModeSM  BeamMode = "SM"
	BeamModeDSN BeamMode = "DSN"
)

// String returns the underlying string value.
func (b BeamMode) String() string {
	return string(b)
}
