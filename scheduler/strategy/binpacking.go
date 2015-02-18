package strategy

import (
	"errors"
	"sort"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

var (
	ErrNoResourcesAvailable = errors.New("no resources available to schedule container")
)

type BinPackingPlacementStrategy struct{}

func (p *BinPackingPlacementStrategy) Initialize() error {
	return nil
}

func (p *BinPackingPlacementStrategy) PlaceContainer(config *dockerclient.ContainerConfig, nodes []cluster.Node) (cluster.Node, error) {
	scores := scores{}

	for _, node := range nodes {
		nodeMemory := node.TotalMemory()
		nodeCpus := node.TotalCpus()

		// Skip nodes that are smaller than the requested resources.
		if nodeMemory < int64(config.Memory) || nodeCpus < config.CpuShares {
			continue
		}

		var (
			cpuScore    int64 = 100
			memoryScore int64 = 100
		)

		if config.CpuShares > 0 {
			cpuScore = (node.UsedCpus() + config.CpuShares) * 100 / nodeCpus
		}
		if config.Memory > 0 {
			memoryScore = (node.UsedMemory() + config.Memory) * 100 / nodeMemory
		}

		if cpuScore <= 100 && memoryScore <= 100 {
			scores = append(scores, &score{node: node, score: cpuScore + memoryScore})
		}
	}

	if len(scores) == 0 {
		return nil, ErrNoResourcesAvailable
	}

	sort.Sort(scores)

	return scores[0].node, nil
}

type score struct {
	node  cluster.Node
	score int64
}

type scores []*score

func (s scores) Len() int {
	return len(s)
}

func (s scores) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s scores) Less(i, j int) bool {
	var (
		ip = s[i]
		jp = s[j]
	)

	return ip.score > jp.score
}
