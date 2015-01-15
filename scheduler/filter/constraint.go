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
func (f *ConstraintFilter) match(pattern, s string, useRegex bool) bool {
	regex := pattern
	if !useRegex {
		regex = "^" + strings.Replace(pattern, "*", ".*", -1) + "$"
	}
	matched, err := regexp.MatchString(regex, s)
	if err != nil {
		log.Error(err)
	}
	return matched
}

func (f *ConstraintFilter) Filter(config *dockerclient.ContainerConfig, nodes []*cluster.Node) ([]*cluster.Node, error) {
	constraints := extractEnv("constraint", config.Env)
	for k, v := range constraints {
		log.Debugf("matching constraint: %s=%s", k, v)

		// keep the original for display in case of error
		v0 := v
		k0 := k

		// support case of constraint:k==v
		// try to check = on both sides, "k= : = : v" and "k : = : =v", to make sure it works everywhere
		if strings.HasSuffix(k, "=") {
			k = strings.TrimSuffix(k, "=")
		} else if strings.HasPrefix(v, "=") {
			v = strings.TrimPrefix(v, "=")
		}

		negate := false
		if strings.HasPrefix(v, "!") {
			log.Debugf("negate detected in value")
			v = strings.TrimPrefix(v, "!")
			negate = true
		} else if strings.HasSuffix(k, "!") {
			log.Debugf("negate detected in key")
			k = strings.TrimSuffix(k, "!")
			negate = true
		}

		gte := strings.HasSuffix(k, ">")
		lte := strings.HasSuffix(k, "<")
		if gte {
			log.Debugf("gt (>) detected in key")
			k = strings.TrimSuffix(k, ">")
		} else if lte {
			log.Debugf("lt (<) detected in key")
			k = strings.TrimSuffix(k, "<")
		}

		useRegex := false
		if strings.HasPrefix(v, "/") && strings.HasSuffix(v, "/") {
			log.Debugf("regex detected")
			v = strings.TrimPrefix(strings.TrimSuffix(v, "/"), "/")
			useRegex = true
		}

		candidates := []*cluster.Node{}
		for _, node := range nodes {
			switch k {
			case "node":
				// "node" label is a special case pinning a container to a specific node.
				matchResult := f.match(v, node.ID, useRegex) || f.match(v, node.Name, useRegex)
				if (negate && !matchResult) || (!negate && matchResult) {

				if gte && node.ID >= v {
					candidates = append(candidates, node)
				} else if lte && node.ID <= v {
					candidates = append(candidates, node)
				} else {
					// "node" label is a special case pinning a container to a specific node.
					matchResult := f.match(v, node.ID, useRegex) || f.match(v, node.Name, useRegex)
					if (negate && !matchResult) || (!negate && matchResult) {
						candidates = append(candidates, node)
					}
				}
			default:
				// By default match the node labels.
				if label, ok := node.Labels[k]; ok {
					if gte && label >= v {
						candidates = append(candidates, node)
					} else if lte && label <= v {
						candidates = append(candidates, node)
					} else {
						matchResult := f.match(v, label, useRegex)
						if (negate && !matchResult) || (!negate && matchResult) {
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
