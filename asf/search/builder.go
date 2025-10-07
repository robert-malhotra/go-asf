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

// Platform filters by satellite platform (e.g. Sentinel-1A).
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

// ProcessingLevel filters by processing level (e.g. RAW, METADATA).
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

// Dataset restricts results to a specific dataset short name.
func (b Builder) Dataset(v string) Builder {
	b.params.Dataset = v
	return b
}

// AddCollection appends a CMR collection identifier to the filter.
func (b Builder) AddCollection(v string) Builder {
	b.params.Collections = append(b.params.Collections, v)
	return b
}

// GranuleList restricts results to the provided granule identifiers.
func (b Builder) GranuleList(ids ...string) Builder {
	b.params.GranuleList = append([]string{}, ids...)
	return b
}

// ProductList restricts results to the provided product identifiers.
func (b Builder) ProductList(ids ...string) Builder {
	b.params.ProductList = append([]string{}, ids...)
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

// MaxResults sets the maximum number of results to return.
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
