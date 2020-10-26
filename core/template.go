package core

import (
	"encoding/xml"
	"strconv"
	"strings"
)

type Template struct {
	xmlTemplate XMLTemplate
	name        string
}

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

func (t *Template) Parent() string {
	return strings.ToLower(strings.TrimSpace(t.xmlTemplate.Parent))
}

func (t *Template) Name() string {
	return t.name
}

func (t *Template) Description() string {
	return t.xmlTemplate.Description
}

func (t *Template) Created() (int, error) {
	return strconv.Atoi(t.xmlTemplate.Created)
}

func (t *Template) LastModified() (int, error) {
	return strconv.Atoi(t.xmlTemplate.LastModified)
}

func (t *Template) Engine() string {
	if strings.TrimSpace(t.xmlTemplate.Engine) == "" {
		return ""
	}

	return strings.ToLower(t.xmlTemplate.Engine)
}

func (t *Template) MimeType() string {
	return strings.TrimSpace(t.xmlTemplate.MimeType)
}

func (t *Template) Content() string {
	return t.xmlTemplate.Content
}

func (t *Template) SetContent(content string) {
	t.xmlTemplate.Content = content
}

func (t *Template) ContentFile() string {
	return t.xmlTemplate.ContentFile
}
