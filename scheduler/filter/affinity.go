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
	affinities := extractEnv("affinity", config.Env)
	for k, v := range affinities {
		log.Debugf("matching affinity: %s=%s", k, v)
		candidates := []*cluster.Node{}
		for _, node := range nodes {
			switch k {
			case "container":
				for _, container := range node.Containers() {
					if match(v, container.Id) || match(v, container.Names[0]) {
						candidates = append(candidates, node)
						break
					}
				}
			case "image":
			done:
				for _, image := range node.Images() {
					if match(v, image.Id) {
						candidates = append(candidates, node)
						break
					}
					for _, t := range image.RepoTags {
						if match(v, t) {
							candidates = append(candidates, node)
							break done
						}
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
