package cluster

// EngineSorter implements the Sort interface to sort Cluster.Engine.
// It is not guaranteed to be a stable sort.
type EngineSorter []*Engine

// Len returns the number of engines to be sorted.
func (s EngineSorter) Len() int {
	return len(s)
}

// Swap exchanges the engine elements with indices i and j.
func (s EngineSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less reports whether the engine with index i should sort before the engine with index j.
// Engines are sorted chronologically by name.
func (s EngineSorter) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}
