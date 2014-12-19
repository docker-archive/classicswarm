package filter

import (
	"fmt"
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

// ConstraintFilter selects only nodes that match certain labels.
type ConstraintFilter struct {
}

func (f *ConstraintFilter) extractConstraints(env []string) map[string]string {
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

// Create the regex for globbing (ex: ub*t* -> ^ub.*t.*$)
// and match.
func (f *ConstraintFilter) match(pattern, s string) bool {
	regex := "^" + strings.Replace(pattern, "*", ".*", -1) + "$"
	matched, err := regexp.MatchString(regex, strings.ToLower(s))
	if err != nil {
		log.Error(err)
	}
	return matched
}

func (f *ConstraintFilter) Filter(config *dockerclient.ContainerConfig, nodes []*cluster.Node) ([]*cluster.Node, error) {
	constraints := f.extractConstraints(config.Env)
	for k, v := range constraints {
		log.Debugf("matching constraint: %s=%s", k, v)
		candidates := []*cluster.Node{}
		for _, node := range nodes {
			switch k {
			case "node":
				// "node" label is a special case pinning a container to a specific node.
				if f.match(v, node.ID) || f.match(v, node.Name) {
					candidates = append(candidates, node)
				}
			default:
				// By default match the node labels.
				if label, ok := node.Labels[k]; ok {
					if f.match(v, label) {
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
