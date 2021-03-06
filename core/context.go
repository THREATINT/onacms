package core

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/blevesearch/bleve"
)

// Context struct
type Context struct {
	HTTPRequest   *http.Request
	Node          *Node
	Content       string
	PublicFiles   map[string]*PublicFile
	AllNodes      []*Node
	FulltextIndex bleve.Index
}

// FindByPath find node by path
func (context *Context) FindByPath(path string) *Node {
	return FindNode(path, context.AllNodes)
}

// RootNodes return all root nodes
func (context *Context) RootNodes() []*Node {
	return RootNodes(context.AllNodes)
}

// Search search all nodes
func (context *Context) Search(term string, maxresults int) []SearchResult {
	term = strings.Replace(term, "*", " ", -1)
	term = strings.Replace(term, "?", " ", -1)
	term = strings.TrimSpace(term)

	var r []SearchResult

	q := bleve.NewQueryStringQuery(term)

	req := bleve.NewSearchRequest(q)
	req.Highlight = bleve.NewHighlightWithStyle("html")
	req.Fields = []string{""}

	resp, err := context.FulltextIndex.Search(req)
	if err != nil {
		return r
	}

	for i, m := range resp.Hits {
		if i == maxresults {
			break
		}

		c := ""
		for _, f := range m.Fragments {
			c = fmt.Sprintf("%s<br/>%s", c, f[0])
		}

		r = append(r, SearchResult{Index: i + 1, URL: m.ID, Score: fmt.Sprintf("%.4f", m.Score), Content: c})
	}

	return r
}
