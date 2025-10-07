package search

import "time"

// Builder provides a fluent way to construct Params.
type Builder struct {
	params Params
}

// ParamsBuilder creates a new Builder with default Params.
func ParamsBuilder() Builder {
	return Builder{params: New()}
}

// Platform filters by satellite platform (e.g. S1A).
func (b Builder) Platform(v Platform) Builder {
	b.params.Platform = v
	return b
}

// BeamMode filters by beam mode (e.g. IW, EW).
func (b Builder) BeamMode(v BeamMode) Builder {
	b.params.BeamMode = v
	return b
}

// Polarization filters by polarization (VV, VH, etc).
func (b Builder) Polarization(v string) Builder {
	b.params.Polarization = v
	return b
}

// ProductType filters by product type.
func (b Builder) ProductType(v string) Builder {
	b.params.ProductType = v
	return b
}

// ProcessingLevel filters by processing level (e.g. L1, L2).
func (b Builder) ProcessingLevel(v string) Builder {
	b.params.ProcessingLevel = v
	return b
}

// FlightDirection filters by flight direction.
func (b Builder) FlightDirection(v string) Builder {
	b.params.FlightDirection = v
	return b
}

// RelativeOrbit filters by relative orbit number.
func (b Builder) RelativeOrbit(v int) Builder {
	b.params.RelativeOrbit = v
	return b
}

// StartTime sets the inclusive search start time.
func (b Builder) StartTime(t time.Time) Builder {
	b.params.Start = t
	return b
}

// EndTime sets the inclusive search end time.
func (b Builder) EndTime(t time.Time) Builder {
	b.params.End = t
	return b
}

// IntersectsWith sets a WKT geometry filter.
func (b Builder) IntersectsWith(wkt string) Builder {
	b.params.IntersectsWith = wkt
	return b
}

// LookDirection filters by look direction (e.g. RIGHT, LEFT).
func (b Builder) LookDirection(dir string) Builder {
	b.params.LookDirection = dir
	return b
}

// MaxResults sets the maximum number of results per page.
func (b Builder) MaxResults(n int) Builder {
	b.params.MaxResults = n
	return b
}

// Set assigns a custom parameter value.
func (b Builder) Set(key, value string) Builder {
	b.params.Set(key, value)
	return b
}

// Add appends a custom parameter value.
func (b Builder) Add(key, value string) Builder {
	b.params.Add(key, value)
	return b
}

// Build returns the composed Params.
func (b Builder) Build() Params {
	return b.params
}
