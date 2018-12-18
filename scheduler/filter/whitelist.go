package filter

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

// WhitelistFilter selects only nodes that are defined in a whitelist by the user
type WhitelistFilter struct {
}

// Name returns the name of the filter
func (f *WhitelistFilter) Name() string {
	return "whitelist"
}

// Filter is exported
func (f *WhitelistFilter) Filter(config *cluster.ContainerConfig, nodes []*node.Node, soft bool) ([]*node.Node, error) {
	whitelists, err := parseExprs(config.Whitelists())
	if err != nil {
		return nil, err
	}

	for _, whitelist := range whitelists {
		if !soft && whitelist.isSoft {
			continue
		}
		log.Debugf("matching whitelist: %s%s%s (soft=%t)", whitelist.key, OPERATORS[whitelist.operator], whitelist.value, whitelist.isSoft)

		candidates := []*node.Node{}

		// Handle |-separated node names in the same whitelist
		whiteNodes := strings.Split(whitelist.value, "|")

		for _, node := range nodes {
			switch whitelist.key {
			// Treat all keys as "node name" keys
			default:
				for _, whiteNode := range whiteNodes {
					if node.Name == whiteNode {
						candidates = append(candidates, node)
						break
					}
				}
			}
		}
		if len(candidates) == 0 {
			return nil, fmt.Errorf("unable to find a node that satisfies the whitelist %s%s%s", whitelist.key, OPERATORS[whitelist.operator], whitelist.value)
		}
		nodes = candidates
	}

	return nodes, nil
}

// GetFilters returns a list of the whitelists found in the container config.
func (f *WhitelistFilter) GetFilters(config *cluster.ContainerConfig) ([]string, error) {
	allWhitelists := []string{}
	whitelists, err := parseExprs(config.Whitelists())
	if err != nil {
		return nil, err
	}
	for _, whitelist := range whitelists {
		allWhitelists = append(allWhitelists, fmt.Sprintf("%s%s%s (soft=%t)", whitelist.key, OPERATORS[whitelist.operator], whitelist.value, whitelist.isSoft))
	}
	return allWhitelists, nil
}
