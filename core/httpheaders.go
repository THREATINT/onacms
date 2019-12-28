package core

import (
	"encoding/xml"
)

type HTTPHeaders struct {
	XMLName             xml.Name      `xml:"node"`
	Title               string        `xml:"title"`
	Description         string        `xml:"description"`
	Weight              string        `xml:"weight"`
	Created             string        `xml:"created"`
	LastModified        string        `xml:"lastmodified"`
	Language            string        `xml:"language"`
	Engine              string        `xml:"engine"`
	Template            string        `xml:"template"`
	Navigable           string        `xml:"navigable"`
	Enabled             string        `xml:"enabled"`
	Content             string        `xml:"content"`
	ContentFile         string        `xml:"content-file"`
	RedirectTo          string        `xml:"redirect-to"`
	ApplicationEndpoint string        `xml:"application-endpoint"`
	Property            []XMLProperty `xml:"property"`
}

type Node struct {
	xmlNode  XMLNode
	name     string
	parent   *Node
	children []*Node
}
