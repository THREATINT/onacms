package core

type NodeSorter []*Node

func (n NodeSorter) Len() int {
	return len(n)
}

func (n NodeSorter) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

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
