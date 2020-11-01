package core

import (
	"encoding/xml"
	"strings"

	"github.com/bmatcuk/doublestar"
)

// URI struct mapping header/expression
type URI struct {
	Expression string   `xml:"expression,attr"`
	Header     []string `xml:"header"`
}

// HTTPHeaders struct
type HTTPHeaders struct {
	URI []URI `xml:"uri"`
}

// Read read http headers from []byte
func (h *HTTPHeaders) Read(r []byte) error {
	return xml.Unmarshal(r, &h)
}

// Match return matching headers for slug
func (h *HTTPHeaders) Match(slug string) []string {
	var result []string
	slug = strings.ToLower(slug)

	for _, uri := range h.URI {
		m, err := doublestar.Match(uri.Expression, slug)
		if m && err == nil {
			for _, h := range uri.Header {
				result = append(result, h)
			}
		}
	}

	return result
}
