package filter

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

// ConstraintFilter selects only nodes that match certain labels.
type ConstraintFilter struct {
}

func (f *ConstraintFilter) Filter(config *dockerclient.ContainerConfig, nodes []*cluster.Node) ([]*cluster.Node, error) {
	constraints := extractEnv("constraint", config.Env)
	for k, v := range constraints {
		log.Debugf("matching constraint: %s=%s", k, v)
		candidates := []*cluster.Node{}
		for _, node := range nodes {
			switch k {
			case "node":
				// "node" label is a special case pinning a container to a specific node.
				if match(v, node.ID) || match(v, node.Name) {
					candidates = append(candidates, node)
				}
			default:
				// By default match the node labels.
				if label, ok := node.Labels[k]; ok {
					if match(v, label) {
						candidates = append(candidates, node)
					}
				}
			}
		}
		if len(candidates) == 0 {
			return nil, fmt.Errorf("unable to find a node that satisfies %s == %s", k, v)
		}
		nodes = candidates
	}
	return nodes, nil
}
