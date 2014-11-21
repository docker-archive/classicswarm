package filter

import (
	"fmt"
	"strings"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

// LabelFilter selects only nodes that match certain labels.
type LabelFilter struct {
}

func (f *LabelFilter) extractConstraints(env []string) map[string]string {
	constraints := make(map[string]string)
	for _, e := range env {
		if strings.HasPrefix(e, "constraint:") {
			constraint := strings.TrimPrefix(e, "constraint:")
			parts := strings.SplitN(constraint, "=", 2)
			constraints[strings.ToLower(parts[0])] = strings.ToLower(parts[1])
		}
	}
	return constraints
}

func (f *LabelFilter) Filter(config *dockerclient.ContainerConfig, nodes []*cluster.Node) ([]*cluster.Node, error) {
	constraints := f.extractConstraints(config.Env)
	for k, v := range constraints {
		candidates := []*cluster.Node{}
		for _, node := range nodes {
			switch k {
			case "node":
				// "node" label is a special case pinning a container to a specific node.
				if strings.ToLower(node.ID) == v || strings.ToLower(node.Name) == v {
					candidates = append(candidates, node)
				}
			default:
				// By default match the node labels.
				if label, ok := node.Labels[k]; ok {
					if strings.Contains(strings.ToLower(label), v) {
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
