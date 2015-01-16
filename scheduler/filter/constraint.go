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
	constraints, err := extractEnv("constraint", config.Env)
	if err != nil {
		return nil, err
	}
	for k, v := range constraints {
		log.Debugf("matching constraint: %s=%s", k, v)

		// keep the original for display in case of error
		v0 := v
		k0 := k

		k, v, mode, useRegex := parse(k, v)

		candidates := []*cluster.Node{}
		for _, node := range nodes {
			switch k {
			case "node":
				if mode == gte && node.ID >= v {
					candidates = append(candidates, node)
				} else if mode == lte && node.ID <= v {
					candidates = append(candidates, node)
				} else {
					// "node" label is a special case pinning a container to a specific node.
					matchResult := match(v, node.ID, useRegex) || match(v, node.Name, useRegex)
					if (mode == neg && !matchResult) || (mode == equ && matchResult) {
						candidates = append(candidates, node)
					}
				}
			default:
				// By default match the node labels.
				if label, ok := node.Labels[k]; ok {
					if mode == gte && label >= v {
						candidates = append(candidates, node)
					} else if mode == lte && label <= v {
						candidates = append(candidates, node)
					} else {
						matchResult := match(v, label, useRegex)
						if (mode == neg && !matchResult) || (mode == equ && matchResult) {
							candidates = append(candidates, node)
						}
					}
				}
			}
		}
		if len(candidates) == 0 {
			return nil, fmt.Errorf("unable to find a node that satisfies %s=%s", k0, v0)
		}
		nodes = candidates
	}
	return nodes, nil
}
