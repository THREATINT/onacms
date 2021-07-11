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
	var (
		err            error
		doc            *html.Node
		nodeSearchable = &NodeSearchable{}
		c              string
		f              func(*html.Node)
	)

	if doc, err = html.Parse(strings.NewReader(string(node.Render()))); err == nil {
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
