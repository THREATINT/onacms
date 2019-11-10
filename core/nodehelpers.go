package core

import (
	"strings"
)

/**
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

/**
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
	} else {
		path = path[0:i]

		for _, node := range nodes {
			if string(node.Path()) == path && node.Enabled() && node.ApplicationEndpoint() {
				return node
			}
		}

		return FindApplicationEndpointNode(path, nodes)
	}
}

/**
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
	} else {
		path = path[0:i]

		for _, node := range nodes {
			if string(node.Path()) == path && node.Enabled() {
				return node
			}
		}

		return FindFallbackNode(path, nodes)
	}
}

func RootNodes(nodes []*Node) []*Node {
	if nodes != nil && len(nodes) > 0 {
		return SiblingsAndSelf(nodes[0].Root(), nodes)
	}
	return nil
}

/*
func Siblings(node *Node, nodes []*Node) []*Node {
	siblings := []*Node{}
	for _, n := range SiblingsAndSelf(node, nodes) {
		if n != node {
			siblings = append(siblings, n)
		}
	}
	return siblings
}
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
