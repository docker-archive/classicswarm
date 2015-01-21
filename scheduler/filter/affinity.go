package filter

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

// AffinityFilter selects only nodes based on other containers on the node.
type AffinityFilter struct {
}

func (f *AffinityFilter) Filter(config *dockerclient.ContainerConfig, nodes []*cluster.Node) ([]*cluster.Node, error) {
	affinities, err := parseExprs("affinity", config.Env)
	if err != nil {
		return nil, err
	}

	for _, affinity := range affinities {
		log.Debugf("matching affinity: %s%s%s", affinity.key, OPERATORS[affinity.operator], affinity.value)

		candidates := []*cluster.Node{}
		for _, node := range nodes {
			switch affinity.key {
			case "container":
				for _, container := range node.Containers() {
					if affinity.Match(container.Id, container.Names[0]) {
						candidates = append(candidates, node)
						break
					}
				}
			case "image":
			done:
				for _, image := range node.Images() {
					if affinity.Match(image.Id) {
						candidates = append(candidates, node)
						break
					}
					for _, tag := range image.RepoTags {
						if affinity.Match(tag) {
							candidates = append(candidates, node)
							break done
						}
					}
				}
			}
		}
		if len(candidates) == 0 {
			return nil, fmt.Errorf("unable to find a node that satisfies %s%s%s", affinity.key, OPERATORS[affinity.operator], affinity.value)
		}
		nodes = candidates
	}
	return nodes, nil
}
