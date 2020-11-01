package core

import (
	"strings"
)

// FindNode searches for an exact match for path in the array of nodes, return nil if there is none.
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

// FindApplicationEndpointNodesssSearch for the best match for a given path (right to left) where application-endpoint is set as property of that node, return nil if there is none.
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

// FindFallbackNode search for the best match for a given path (right to left), return nil if there is none.
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

// RootNodes return all nodes at the root (=top level) of the hierarchie
func RootNodes(nodes []*Node) []*Node {
	var rootNodes []*Node

	for _, n := range nodes {
		if n.Parent() == nil {
			rootNodes = append(rootNodes, n)
		}
	}

	return rootNodes
}

// SiblingsAndSelf return all nodes at the same hierarchie including own node
func SiblingsAndSelf(node *Node, nodes []*Node) []*Node {
	if node.Parent() != nil {
		return node.Parent().Children()
	}

	return RootNodes(nodes)
}
