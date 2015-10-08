package filter

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

// ConstraintFilter selects only nodes that match certain labels.
type ConstraintFilter struct {
}

// Name returns the name of the filter
func (f *ConstraintFilter) Name() string {
	return "constraint"
}

// Filter is exported
func (f *ConstraintFilter) Filter(config *cluster.ContainerConfig, nodes []*node.Node) ([]*node.Node, error) {
	constraints, err := parseExprs(config.Constraints())
	if err != nil {
		return nil, err
	}

	for _, constraint := range constraints {
		log.Debugf("matching constraint: %s %s %s", constraint.key, OPERATORS[constraint.operator], constraint.value)

		candidates := []*node.Node{}
		for _, node := range nodes {
			switch constraint.key {
			case "node":
				// "node" label is a special case pinning a container to a specific node.
				if constraint.Match(node.ID, node.Name) {
					candidates = append(candidates, node)
				}
			default:
				if constraint.Match(node.Labels[constraint.key]) {
					candidates = append(candidates, node)
				}
			}
		}
		if len(candidates) == 0 {
			if constraint.isSoft {
				continue
			}
			return nil, fmt.Errorf("unable to find a node that satisfies %s%s%s", constraint.key, OPERATORS[constraint.operator], constraint.value)
		}
		nodes = candidates
	}
	return nodes, nil
}
