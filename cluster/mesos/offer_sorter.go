package mesos

import "github.com/mesos/mesos-go/mesosproto"

// OfferSorter implements the Sort interface to sort offers.
// It is not guaranteed to be a stable sort.
type offerSorter []*mesosproto.Offer

// Len returns the number of engines to be sorted.
func (s offerSorter) Len() int {
	return len(s)
}

// Swap exchanges the engine elements with indices i and j.
func (s offerSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less reports whether the engine with index i should sort before the engine with index j.
// Offers are sorted chronologically by name.
func (s offerSorter) Less(i, j int) bool {
	return s[i].Id.GetValue() < s[j].Id.GetValue()
}
