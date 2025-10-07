package search

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Params represents a collection of ASF search filters.
type Params struct {
	Platform        Platform
	BeamMode        BeamMode
	Polarization    string
	ProcessingLevel string
	FlightDirection string
	RelativeOrbit   int
	Start           time.Time
	End             time.Time
	IntersectsWith  string
	LookDirection   string
	Dataset         string
	Collections     []string
	GranuleList     []string
	ProductList     []string
	MaxResults      int
	Additional      map[string][]string
}

// New returns a Params instance with sensible defaults.
func New() Params {
	return Params{
		Additional: make(map[string][]string),
		MaxResults: 100,
	}
}

// Set adds a custom parameter value.
func (p *Params) Set(key string, value string) {
	if p.Additional == nil {
		p.Additional = make(map[string][]string)
	}
	p.Additional[key] = []string{value}
}

// Add appends a value to a multi-value parameter.
func (p *Params) Add(key string, value string) {
	if p.Additional == nil {
		p.Additional = make(map[string][]string)
	}
	p.Additional[key] = append(p.Additional[key], value)
}

func (p Params) hasAdditional(key string) bool {
	if p.Additional == nil {
		return false
	}
	_, ok := p.Additional[key]
	return ok
}

// Encode serialises the parameters into query string values expected by ASF.
func (p Params) Encode() (url.Values, error) {
	if !p.Start.IsZero() && p.End.IsZero() {
		return nil, fmt.Errorf("end time must be provided when start time is set")
	}
	if p.Start.After(p.End) && !p.End.IsZero() {
		return nil, fmt.Errorf("start time must be before end time")
	}

	values := make(url.Values)
	values.Set("output", "jsonlite")

	if p.Platform != "" {
		values.Set("platform", p.Platform.String())
	}
	if p.BeamMode != "" {
		values.Set("beamMode", p.BeamMode.String())
	}
	if p.Polarization != "" {
		values.Set("polarization", p.Polarization)
	}
	if p.ProcessingLevel != "" {
		values.Set("processingLevel", p.ProcessingLevel)
	}
	if p.FlightDirection != "" {
		values.Set("flightDirection", p.FlightDirection)
	}
	if p.RelativeOrbit > 0 {
		values.Set("relativeOrbit", fmt.Sprintf("%d", p.RelativeOrbit))
	}
	if !p.Start.IsZero() {
		values.Set("start", p.Start.UTC().Format(time.RFC3339))
	}
	if !p.End.IsZero() {
		values.Set("end", p.End.UTC().Format(time.RFC3339))
	}
	if p.IntersectsWith != "" {
		values.Set("intersectsWith", p.IntersectsWith)
	}
	if p.LookDirection != "" {
		values.Set("lookDirection", p.LookDirection)
	}
	if p.Dataset != "" {
		values.Set("dataset", p.Dataset)
	}
	if len(p.Collections) > 0 {
		sorted := append([]string(nil), p.Collections...)
		sort.Strings(sorted)
		for _, c := range sorted {
			if c != "" {
				values.Add("collections", c)
			}
		}
	}
	if len(p.GranuleList) > 0 {
		values.Set("granule_list", strings.Join(p.GranuleList, ","))
	}
	if len(p.ProductList) > 0 {
		values.Set("product_list", strings.Join(p.ProductList, ","))
	}

	includeMax := p.MaxResults > 0 && len(p.ProductList) == 0 && len(p.GranuleList) == 0 && !p.hasAdditional("product_list") && !p.hasAdditional("granule_list")
	if includeMax {
		values.Set("maxResults", fmt.Sprintf("%d", p.MaxResults))
	}

	for k, vals := range p.Additional {
		for _, v := range vals {
			values.Add(k, v)
		}
	}

	return values, nil
}

// PageSize returns the requested max results per page.
func (p Params) PageSize() int {
	if p.MaxResults <= 0 {
		return 100
	}
	return p.MaxResults
}
