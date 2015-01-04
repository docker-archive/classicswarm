package filter

import (
	"fmt"
	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
	"strings"
)

// VolumeFilter guarantees that, when scheduling a container with the volumes from other containers
// , it will be located on the same node.
type NetFilter struct {
	CollocationFilter
}

const CONTAINER_PREFIX = "container:"

func (v *NetFilter) Filter(config *dockerclient.ContainerConfig, nodes []*cluster.Node) ([]*cluster.Node, error) {
	candidates := []*cluster.Node{}
	networkMode := config.HostConfig.NetworkMode
	if strings.HasPrefix(networkMode, CONTAINER_PREFIX) {
		IdOrName := networkMode[len(CONTAINER_PREFIX):]
		node := v.FindNode(IdOrName, nodes)
		if node == nil {
			return nil, fmt.Errorf("unable to find a node with container %s", IdOrName)
		}
		candidates = append(candidates, node)
	} else {
		candidates = append(candidates, nodes...)
	}
	return candidates, nil
}
