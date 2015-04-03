package cluster

import "fmt"

// Node is exported
type Node interface {
	ID() string
	Name() string

	IP() string   //to inject the actual IP of the machine in docker ps (hostname:port or ip:port)
	Addr() string //to know where to connect with the proxy

	Images() []*Image                     //used by the API
	Image(IDOrName string) *Image         //used by the filters
	Containers() []*Container             //used by the filters
	Container(IDOrName string) *Container //used by the filters

	TotalCpus() int64   //used by the strategy
	UsedCpus() int64    //used by the strategy
	TotalMemory() int64 //used by the strategy
	UsedMemory() int64  //used by the strategy

	Labels() map[string]string //used by the filters

	IsHealthy() bool
}

// SerializeNode is exported
func SerializeNode(node Node) string {
	return fmt.Sprintf("{%q:%q,%q:%q,%q:%q,%q:%q}",
		"Name", node.Name(),
		"Id", node.ID(),
		"Addr", node.Addr(),
		"Ip", node.IP())
}

// NodeSorter implements the Sort interface to sort Cluster.Node.
// It is not guaranteed to be a stable sort.
type NodeSorter []Node

// Len returns the number of nodes to be sorted.
func (s NodeSorter) Len() int {
	return len(s)
}

// Swap exchanges the node elements with indices i and j.
func (s NodeSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less reports whether the node with index i should sort before the node with index j.
// Nodes are sorted chronologically by name.
func (s NodeSorter) Less(i, j int) bool {
	return s[i].Name() < s[j].Name()
}
