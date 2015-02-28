package strategy

import "github.com/docker/swarm/cluster"

// WeightedNode represents a node in the cluster with a given weight, typically used for sorting
// purposes.
type weightedNode struct {
	Node cluster.Node
	// Weight is the inherent value of this node.
	Weight int64
}

type weightedNodeList []*weightedNode

func (n weightedNodeList) Len() int {
	return len(n)
}

func (n weightedNodeList) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

func (n weightedNodeList) Less(i, j int) bool {
	var (
		ip = n[i]
		jp = n[j]
	)

	return ip.Weight < jp.Weight
}
