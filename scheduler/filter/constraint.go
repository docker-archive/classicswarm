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
	constraints, err := parseExprs("constraint", config.Env)
	if err != nil {
		return nil, err
	}

	for _, constraint := range constraints {
		log.Debugf("matching constraint: %s %s %s", constraint.key, OPERATORS[constraint.operator], constraint.value)

		candidates := []*cluster.Node{}
		for _, node := range nodes {
			switch constraint.key {
			case "node":
				// "node" label is a special case pinning a container to a specific node.
				if constraint.Match(node.ID, node.Name) {
					candidates = append(candidates, node)
				}
			default:
				if label, ok := node.Labels[constraint.key]; ok {
					if constraint.Match(label) {
						candidates = append(candidates, node)
					}
				} else {
					// The node doesn't have this particular label.
					if constraint.operator == NOTEQ {
						// Special case: If the operator is != and the node doesn't
						// have the label at all, consider it as a candidate.
						candidates = append(candidates, node)
					}
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
