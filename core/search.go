package core

import (
	"strings"

	"golang.org/x/net/html"
)

// NodeSearchable struct
type NodeSearchable struct {
	Content string
}

// SearchResult struct
type SearchResult struct {
	Index   int
	URL     string
	Score   string
	Content string
}

// NewNodeSearchable initialiser
func NewNodeSearchable(node *Node) *NodeSearchable {
	nodeSearchable := &NodeSearchable{}

	doc, err := html.Parse(strings.NewReader(string(node.Render())))
	if err == nil {
		var c string
		var f func(*html.Node)
		f = func(n *html.Node) {
			if n.Type == html.TextNode {
				c += n.Data
			}

			for child := n.FirstChild; child != nil; child = child.NextSibling {
				f(child)
			}
		}
		f(doc)

		nodeSearchable.Content = c
	}

	return nodeSearchable
}
