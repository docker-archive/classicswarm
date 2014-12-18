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

func (f *ConstraintFilter) Filter(config *dockerclient.ContainerConfig, nodes []*cluster.Node) ([]*cluster.Node, error) {
	constraints := f.extractConstraints(config.Env)
	for k, v := range constraints {
		regex := "^" + strings.Replace(v, "*", ".*", -1) + "$"
		log.Debugf("matching constraint: %s=%s", k, regex)
		candidates := []*cluster.Node{}
		for _, node := range nodes {
			switch k {
			case "node":
				// "node" label is a special case pinning a container to a specific node.
				matchedID, err := regexp.MatchString(regex, strings.ToLower(node.ID))
				if err != nil {
					log.Error(err)
				}
				matchedName, err := regexp.MatchString(regex, strings.ToLower(node.Name))
				if err != nil {
					log.Error(err)
				}
				if matchedID || matchedName {
					candidates = append(candidates, node)
				}
			default:
				// By default match the node labels.
				if label, ok := node.Labels[k]; ok {
					matched, err := regexp.MatchString(regex, strings.ToLower(label))
					if err != nil {
						log.Error(err)
					}
					if matched {
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
