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
					// "node" label is a special case pinning a container to a specific node.
					if match(v, container.Id) || match(v, container.Names[0]) {
						candidates = append(candidates, node)
						break
					}
				}
			case "image":
				//TODO use cache
				images, err := node.ListImages()
				if err != nil {
					break
				}
				for _, image := range images {
					if match(v, image) {
						candidates = append(candidates, node)
						break
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
