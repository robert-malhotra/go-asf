package model

import "encoding/json"

// SearchResponse mirrors the fields returned by the ASF search endpoint for jsonlite output.
type SearchResponse struct {
	Results []Product `json:"results"`
}

// Product represents an individual scene/granule returned from the search API.
type Product struct {
	ID              string                 `json:"productID"`
	FileName        string                 `json:"fileName"`
	DownloadURL     string                 `json:"downloadUrl"`
	ProductType     string                 `json:"productType"`
	ProcessingLevel string                 `json:"processingLevel"`
	Platform        string                 `json:"platform"`
	BeamMode        string                 `json:"beamMode"`
	Polarization    string                 `json:"polarization"`
	FlightDirection string                 `json:"flightDirection"`
	Path            int                    `json:"path"`
	StartTime       string                 `json:"startTime"`
	StopTime        string                 `json:"stopTime"`
	SizeMB          float64                `json:"sizeMB"`
	MD5Sum          string                 `json:"md5sum"`
	StringFootprint string                 `json:"stringFootprint"`
	Metadata        map[string]interface{} `json:"-"`
	Files           []File                 `json:"files"`
}

// File describes a downloadable file associated with a product.
type File struct {
	URL          string `json:"url"`
	Size         int64  `json:"size"`
	Checksum     string `json:"checksum"`
	ChecksumType string `json:"checksumType"`
	Name         string `json:"name"`
}

// UnmarshalJSON ensures the Files slice and metadata are initialised.
func (p *Product) UnmarshalJSON(data []byte) error {
	type alias Product
	var tmp alias
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*p = Product(tmp)
	if p.Metadata == nil {
		var meta map[string]interface{}
		if err := json.Unmarshal(data, &meta); err == nil {
			p.Metadata = meta
		} else {
			p.Metadata = map[string]interface{}{}
		}
	}
	if p.Files == nil {
		p.Files = []File{}
	}
	if len(p.Files) == 0 && p.DownloadURL != "" {
		size := int64(p.SizeMB * 1024 * 1024)
		checksumType := ""
		if p.MD5Sum != "" {
			checksumType = "md5"
		}
		p.Files = append(p.Files, File{
			URL:          p.DownloadURL,
			Name:         p.FileName,
			Size:         size,
			Checksum:     p.MD5Sum,
			ChecksumType: checksumType,
		})
	}
	if p.Metadata == nil {
		p.Metadata = map[string]interface{}{}
	}
	return nil
}
