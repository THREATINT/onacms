package core

import (
	"strings"
)

/*
FindNode (path ,nodes)
Searches for an exact match for path in the array of nodes, return nil if there is none.
*/
func FindNode(path string, nodes []*Node) *Node {
	path = strings.TrimSpace(strings.ToLower(path))

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	for _, node := range nodes {
		if string(node.Path()) == path && node.Enabled() {
			return node
		}
	}

	return nil
}

/*
FindApplicationEndpointNode (path, nodes)
Search for the best match for a given path (right to left) where application-endpoint is set as property of
that node, return nil if there is none.
*/
func FindApplicationEndpointNode(path string, nodes []*Node) *Node {
	path = strings.TrimSpace(strings.ToLower(path))

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	i := strings.LastIndex(path, "/")
	if i <= 0 {
		return nil
	}
	path = path[0:i]

	for _, node := range nodes {
		if string(node.Path()) == path && node.Enabled() && node.ApplicationEndpoint() {
			return node
		}
	}

	return FindApplicationEndpointNode(path, nodes)
}

/*
FindFallbackNode (path, nodes)
Search for the best match for a given path (right to left), return nil if there is none.
*/
func FindFallbackNode(path string, nodes []*Node) *Node {
	path = strings.TrimSpace(strings.ToLower(path))

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	i := strings.LastIndex(path, "/")

	if i <= 0 {
		return nil
	}

	path = path[0:i]

	for _, node := range nodes {
		if string(node.Path()) == path && node.Enabled() {
			return node
		}
	}

	return FindFallbackNode(path, nodes)
}

/*
RootNodes (nodes)
Return all nodes at the root (=top level) of the hierarchie
*/
func RootNodes(nodes []*Node) []*Node {
	if nodes != nil && len(nodes) > 0 {
		return SiblingsAndSelf(nodes[0].Root(), nodes)
	}
	return nil
}

/*
SiblingsAndSelf (node, nodes)
Return all nodes at the same hierarchie including own node
*/
func SiblingsAndSelf(node *Node, nodes []*Node) []*Node {
	if node.Parent() != nil {
		return node.Parent().Children()
	}

	var rootNodes []*Node

	for _, n := range nodes {
		if n.Parent() == nil {
			rootNodes = append(rootNodes, n)
		}
	}

	return rootNodes
}
