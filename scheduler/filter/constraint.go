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

func (f *ConstraintFilter) Filter(config *dockerclient.ContainerConfig, nodes []cluster.Node) ([]cluster.Node, error) {
	constraints, err := parseExprs("constraint", config.Env)
	if err != nil {
		return nil, err
	}

	for _, constraint := range constraints {
		log.Debugf("matching constraint: %s %s %s", constraint.key, OPERATORS[constraint.operator], constraint.value)

		candidates := []cluster.Node{}
		for _, node := range nodes {
			switch constraint.key {
			case "node":
				// "node" label is a special case pinning a container to a specific node.
				if constraint.Match(node.ID(), node.Name()) {
					candidates = append(candidates, node)
				}
			default:
				if constraint.Match(node.Labels()[constraint.key]) {
					candidates = append(candidates, node)
				}
			}
		}
		if len(candidates) == 0 {
			return nil, fmt.Errorf("unable to find a node that satisfies %s%s%s", constraint.key, OPERATORS[constraint.operator], constraint.value)
		}
		nodes = candidates
	}
	return nodes, nil
}
