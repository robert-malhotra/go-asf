package model

import "encoding/json"

// SearchResponse mirrors the key fields returned by the ASF search endpoint.
type SearchResponse struct {
	Results []Product `json:"results"`
	Total   int       `json:"total"`
	Count   int       `json:"count"`
	Next    string    `json:"next"`
}

// Product represents an individual scene/granule returned from the search API.
type Product struct {
	ID          string                 `json:"product_id"`
	Name        string                 `json:"fileName"`
	Platform    string                 `json:"platform"`
	Acquisition string                 `json:"acquisition_date"`
	Files       []File                 `json:"files"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// File describes a downloadable file associated with a product.
type File struct {
	URL          string `json:"url"`
	Size         int64  `json:"size"`
	Checksum     string `json:"checksum"`
	ChecksumType string `json:"checksumType"`
	Name         string `json:"name"`
}

// UnmarshalJSON ensures the Files slice is initialised.
func (p *Product) UnmarshalJSON(data []byte) error {
	type alias Product
	var tmp alias
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*p = Product(tmp)
	if p.Files == nil {
		p.Files = []File{}
	}
	if p.Metadata == nil {
		p.Metadata = map[string]interface{}{}
	}
	return nil
}
