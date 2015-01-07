package strategy

import (
	"sort"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

type BalancedPlacementStrategy struct {
	overcommitRatio int64
}

func (p *BalancedPlacementStrategy) Initialize() error {
	return nil
}

func (p *BalancedPlacementStrategy) PlaceContainer(config *dockerclient.ContainerConfig, nodes []*cluster.Node) (*cluster.Node, error) {
	scores := balancedScores{}

	for _, node := range nodes {
		nodeMemory := node.UsableMemory()
		nodeCpus := node.UsableCpus()

		// Skip nodes that are smaller than the requested resources.
		if nodeMemory < int64(config.Memory) || nodeCpus < config.CpuShares {
			continue
		}

		var cpuScore = (node.ReservedCpus() + config.CpuShares) * 100 / nodeCpus
		var memoryScore = (node.ReservedMemory() + config.Memory) * 100 / nodeMemory
		var containerScore = int64(len(node.Containers())) + 1

		var total = cpuScore + memoryScore + containerScore

		scores = append(scores, &balancedScore{node: node, score: total})
	}

	if len(scores) == 0 {
		return nil, ErrNoResourcesAvailable
	}

	sort.Sort(scores)

	return scores[0].node, nil
}

type balancedScore struct {
	node  *cluster.Node
	score int64
}

type balancedScores []*balancedScore

func (s balancedScores) Len() int {
	return len(s)
}

func (s balancedScores) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s balancedScores) Less(i, j int) bool {
	var (
		ip = s[i]
		jp = s[j]
	)

	return ip.score < jp.score
}
