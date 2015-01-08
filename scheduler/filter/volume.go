package filter

import (
	"fmt"
	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

// VolumeFilter guarantees that, when scheduling a container with the volumes from other containers
// , it will be located on the same node.
type VolumeFilter struct {
	CollocationFilter
}

func (v *VolumeFilter) Filter(config *dockerclient.ContainerConfig, nodes []*cluster.Node) ([]*cluster.Node, error) {
	candidates := []*cluster.Node{}
	volumesFrom := config.HostConfig.VolumesFrom
	if volumesFrom != nil && len(volumesFrom) > 0 {
		var result *cluster.Node
		for _, IdOrName := range volumesFrom {
			node := v.FindNode(IdOrName, nodes)
			if node == nil {
				return nil, fmt.Errorf("unable to find a node with container %s", IdOrName)
			}
			if result == nil {
				result = node
			} else if result != node {
				return nil, fmt.Errorf("unable to find a node for conflicts of VolumesFrom %v", volumesFrom)
			}
		}
		candidates = append(candidates, result)
	} else {
		candidates = append(candidates, nodes...)
	}
	return candidates, nil
}
