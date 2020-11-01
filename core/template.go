package core

import (
	"encoding/xml"
	"strconv"
	"strings"
)

// Template struct
type Template struct {
	xmlTemplate XMLTemplate
	name        string
}

// XMLTemplate struct
// XML representation of the template
type XMLTemplate struct {
	XMLName      xml.Name `xml:"template"`
	Parent       string   `xml:"parent"`
	Description  string   `xml:"description"`
	Created      string   `xml:"created"`
	LastModified string   `xml:"lastmodified"`
	MimeType     string   `xml:"mime-type"`
	Engine       string   `xml:"engine"`
	Content      string   `xml:"content"`
	ContentFile  string   `xml:"content-file"`
}

func (t *Template) Read(r []byte) error {
	return xml.Unmarshal(r, &t.xmlTemplate)
}

// Parent return parent template (from field: 'parent')
func (t *Template) Parent() string {
	return strings.ToLower(strings.TrimSpace(t.xmlTemplate.Parent))
}

// Name return name (field: 'name')
func (t *Template) Name() string {
	return t.name
}

// Description return description (field: 'description')
func (t *Template) Description() string {
	return t.xmlTemplate.Description
}

// Created return created timestamp in unix format (!) (field: 'created')
func (t *Template) Created() (int, error) {
	return strconv.Atoi(t.xmlTemplate.Created)
}

// LastModified return LastModified timestamp in unix format (!) (field: 'lastmodified')
func (t *Template) LastModified() (int, error) {
	return strconv.Atoi(t.xmlTemplate.LastModified)
}

// Engine return (rendering) Engine (field: 'engine')
func (t *Template) Engine() string {
	if strings.TrimSpace(t.xmlTemplate.Engine) == "" {
		return ""
	}

	return strings.ToLower(t.xmlTemplate.Engine)
}

// MimeType return MimeType (field: 'mime-type')
func (t *Template) MimeType() string {
	return strings.TrimSpace(t.xmlTemplate.MimeType)
}

// Content return Content (field: 'content')
func (t *Template) Content() string {
	return t.xmlTemplate.Content
}

// SetContent set Content (field: 'content')
func (t *Template) SetContent(content string) {
	t.xmlTemplate.Content = content
}

// ContentFile return ContentFile (field: 'content-file')
func (t *Template) ContentFile() string {
	return t.xmlTemplate.ContentFile
}
