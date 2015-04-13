package mesos

// SlaveSorter implements the Sort interface to sort slaves.
// It is not guaranteed to be a stable sort.
type SlaveSorter []*slave

// Len returns the number of engines to be sorted.
func (s SlaveSorter) Len() int {
	return len(s)
}

// Swap exchanges the engine elements with indices i and j.
func (s SlaveSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less reports whether the engine with index i should sort before the engine with index j.
// Slaves are sorted chronologically by name.
func (s SlaveSorter) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}
