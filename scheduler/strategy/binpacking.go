package strategy

import (
	"errors"
	"sort"

	"github.com/docker/libcluster/swarm"
	"github.com/samalba/dockerclient"
)

var (
	ErrNoResourcesAvailable = errors.New("no resources avaliable to schedule container")
)

type BinPackingPlacementStrategy struct {
}

func (p *BinPackingPlacementStrategy) PlaceContainer(config *dockerclient.ContainerConfig, nodes []*libcluster.Node) (*libcluster.Node, error) {
	scores := scores{}

	for _, node := range nodes {
		// Skip nodes that are smaller than the requested resources.
		if node.Memory < int64(config.Memory) || node.Cpus < config.CpuShares {
			continue
		}

		var (
			memory = int64(config.Memory)
			cpus   = float64(config.CpuShares) / 100.0 * float64(node.Cpus)
		)

		var (
			cpuScore    = ((node.ReservedCpus() + cpus) / float64(node.Cpus)) * 100.0
			memoryScore = (float64(node.ReservedMemory()+memory) / float64(node.Memory)) * 100.0
			total       = ((cpuScore + memoryScore) / 200.0) * 100.0
		)

		if cpuScore <= 100.0 && memoryScore <= 100.0 {
			scores = append(scores, &score{node: node, score: total})
		}
	}

	if len(scores) == 0 {
		return nil, ErrNoResourcesAvailable
	}

	sort.Sort(scores)

	return scores[0].node, nil
}

type score struct {
	node  *libcluster.Node
	score float64
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
