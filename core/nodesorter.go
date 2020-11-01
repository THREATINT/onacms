package core

// NodeSorter type
type NodeSorter []*Node

// Len return length of node list
func (n NodeSorter) Len() int {
	return len(n)
}

// Swap swap position of two nodes
func (n NodeSorter) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

// Less compare weight of two nodes taking into consideration weight or, if it does not exist, creation time
func (n NodeSorter) Less(i, j int) bool {
	if n[i].Weight() != -1 && n[j].Weight() != -1 {
		// both nodes have a weight
		return n[i].Weight() < n[j].Weight()
	}

	if n[i].Weight() != -1 {
		// only weight for i is set
		return true
	}

	// fallback: use creation time
	return n[i].Created() < n[j].Created()
}
