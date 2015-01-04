package filter

import (
	"github.com/docker/swarm/cluster"
	"strings"
)

// CollocationFilter provides the helper to find the node to locate the containers.

type CollocationFilter struct {
}

func (c *CollocationFilter) FindNode(IdOrName string, nodes []*cluster.Node) *cluster.Node {

	for _, node := range nodes {
		for _, container := range node.Containers() {
			// Match ID prefix.
			if strings.HasPrefix(container.Id, IdOrName) {
				return node
			}
			// Match name, /name or engine/name.
			for _, name := range container.Names {
				if name == IdOrName || name == "/"+IdOrName || node.ID+name == IdOrName || node.Name+name == IdOrName {
					return node
				}
			}
		}
	}
	return nil
}
