package strategy

import (
	"errors"
	"sort"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

type RoundRobinPlacementStrategy struct{}

func (ps *RoundRobinPlacementStrategy) Initialize() error {
	return nil
}

func (ps *RoundRobinPlacementStrategy) PlaceContainer(config *dockerclient.ContainerConfig, nodes []*cluster.Node) (*cluster.Node, error) {
	scores := lowScores{}

	for _, node := range nodes {
		nodeMemory := node.UsableMemory()
		nodeCpus := node.UsableCpus()
		// Skip nodes that are smaller than the requested resources.
		if nodeMemory < int64(config.Memory) || nodeCpus < config.CpuShares {
			continue
		}

		var (
			cpuScore    int64 = 100
			memoryScore int64 = 100
		)

		if config.CpuShares > 0 {
			cpuScore = (node.ReservedCpus() + config.CpuShares) * 100 / nodeCpus
		}
		if config.Memory > 0 {
			memoryScore = (node.ReservedMemory() + config.Memory) * 100 / nodeMemory
		}

		if cpuScore <= 100 && memoryScore <= 100 {
			scores = append(scores, &score{node: node, score: int64(len(node.Containers()))})
		}
	}

	if len(scores) == 0 {
		return nil, errors.New(ErrNoResourcesAvailable)
	}

	sort.Sort(scores)
	return scores[0].node, nil
}

type lowScores []*score

func (s lowScores) Len() int {
	return len(s)
}

func (s lowScores) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s lowScores) Less(i, j int) bool {
	var (
		ip = s[i]
		jp = s[j]
	)

	return ip.score < jp.score
}
