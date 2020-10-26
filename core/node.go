package core

import (
	"bytes"
	"encoding/xml"
	"html/template"
	"net/url"
	"strconv"
	"strings"

	"github.com/golang-commonmark/markdown"
)

type XMLProperty struct {
	XMLName xml.Name `xml:"property"`
	Key     string   `xml:"key,attr"`
	Value   string   `xml:"value,attr"`
}

type XMLNode struct {
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

func (n *Node) Read(r []byte, name string) error {
	n.name = name
	return xml.Unmarshal(r, &n.xmlNode)
}

func (n *Node) Name() string {
	return n.name
}

func (n *Node) Title() string {
	return strings.TrimSpace(n.xmlNode.Title)
}

func (n *Node) Slug() string {
	return strings.ToLower(url.PathEscape(n.Name()))
}

func (n *Node) Path() template.URL {
	var s bytes.Buffer

	for _, node := range n.Parents() {
		s.WriteString("/")
		s.WriteString(node.Slug())
	}

	s.WriteString("/")
	s.WriteString(n.Slug())

	return template.URL(s.String())
}

func (n *Node) Weight() int {
	i, err := strconv.Atoi(n.xmlNode.Weight)
	if err == nil {
		return i
	}

	return -1
}

func (n *Node) Created() int {
	t, err := strconv.Atoi(n.xmlNode.Created)
	if err == nil {
		return t
	}

	return -1
}

func (n *Node) LastModified() int {
	t, err := strconv.Atoi(n.xmlNode.LastModified)
	if err == nil {
		return t
	}

	return -1
}

func (n *Node) Engine() string {
	return strings.ToLower(strings.TrimSpace(n.xmlNode.Engine))
}

func (n *Node) Language() string {
	l := strings.ToLower(strings.TrimSpace(n.xmlNode.Language))
	if l == "" && n.Parent() != nil {
		l = n.Parent().Language()
	}
	return l
}

func (n *Node) Description() string {
	return strings.TrimSpace(n.xmlNode.Description)
}

func (n *Node) Template() string {
	t := strings.ToLower(n.xmlNode.Template)
	if t == "" && n.Parent() != nil {
		t = n.Parent().Template()
	}
	return t
}

func (n *Node) Navigable() bool {
	nav := strings.ToLower(strings.TrimSpace(n.xmlNode.Navigable))

	if nav == "" && n.Parent() != nil {
		return n.Parent().Navigable()
	}

	if nav == "1" || nav == "on" || strings.HasPrefix(nav, "enable") || nav == "true" {
		return true
	}

	return false
}

func (n *Node) Enabled() bool {
	enabled := strings.ToLower(strings.TrimSpace(n.xmlNode.Enabled))

	if enabled == "" && n.Parent() != nil {
		return n.Parent().Enabled()
	}

	if enabled == "1" || enabled == "on" || strings.HasPrefix(enabled, "enable") || enabled == "true" {
		return true
	}

	return false
}

func (n *Node) Content() string {
	return n.xmlNode.Content
}

func (n *Node) SetContent(content string) {
	n.xmlNode.Content = content
}

func (n *Node) Parent() *Node {
	return n.parent
}

func (n *Node) SetParent(parent *Node) {
	n.parent = parent
}

func (n *Node) Parents() []*Node {
	var parentNodes []*Node
	for n != nil && n.Parent() != nil {
		parentNodes = append([]*Node{n.Parent()}, parentNodes...)

		n = n.Parent()
	}

	return parentNodes
}

func (n *Node) ParentsAndSelf() []*Node {
	return append(n.Parents(), n)
}

func (n *Node) Root() *Node {
	node := n

	for node.Parent() != nil {
		node = node.Parent()
	}

	return node
}

func (n *Node) HasChildren() bool {
	if len(n.Children()) == 0 {
		return false
	}

	return true
}

func (n *Node) Children() []*Node {
	return n.children
}

func (n *Node) CustomProperty(key string, parent bool) string {
	for _, p := range n.xmlNode.Property {
		if p.Key == key {
			return p.Value
		}
	}

	if parent && n.Parent() != nil {
		return n.Parent().CustomProperty(key, parent)
	}

	return ""
}

func (n *Node) Render() string {
	switch n.Engine() {
	case "markdown":
		md := markdown.New(markdown.HTML(true), markdown.Nofollow(true))
		return md.RenderToString([]byte(n.Content()))
	default:
		return n.Content()
	}

}

func (n *Node) RedirectTo() string {
	return strings.TrimSpace(n.xmlNode.RedirectTo)
}

func (n *Node) ApplicationEndpoint() bool {
	appep := strings.ToLower(strings.TrimSpace(n.xmlNode.ApplicationEndpoint))

	if appep == "1" || appep == "on" || strings.HasPrefix(appep, "enable") || appep == "true" {
		return true
	}

	return false
}
