package filter

import (
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

// AffinityFilter selects only nodes based on other containers on the node.
type AffinityFilter struct {
}

// Name returns the name of the filter
func (f *AffinityFilter) Name() string {
	return "affinity"
}

// Filter is exported
func (f *AffinityFilter) Filter(config *cluster.ContainerConfig, nodes []*node.Node) ([]*node.Node, error) {
	affinities, err := parseExprs(config.Affinities())
	if err != nil {
		return nil, err
	}

	for _, affinity := range affinities {
		log.Debugf("matching affinity: %s%s%s", affinity.key, OPERATORS[affinity.operator], affinity.value)

		candidates := []*node.Node{}
		for _, node := range nodes {
			switch affinity.key {
			case "container":
				containers := []string{}
				for _, container := range node.Containers {
					containers = append(containers, container.Id, strings.TrimPrefix(container.Names[0], "/"))
				}
				for _, container := range node.ScheduledList("container") {
					containers = append(containers, container)
				}
				if affinity.Match(containers...) {
					candidates = append(candidates, node)
				}
			case "image":
				images := []string{}
				for _, image := range node.Images {
					images = append(images, image.Id)
					images = append(images, image.RepoTags...)
					for _, tag := range image.RepoTags {
						images = append(images, strings.Split(tag, ":")[0])
					}
				}
				for _, image := range node.ScheduledList("images") {
					images = append(images, image)
				}
				if affinity.Match(images...) {
					candidates = append(candidates, node)
				}
			default:
				labels := []string{}
				for _, container := range node.Containers {
					labels = append(labels, container.Labels[affinity.key])
				}
				if affinity.Match(labels...) {
					candidates = append(candidates, node)
				}

			}
		}
		if len(candidates) == 0 {
			if affinity.isSoft {
				return nodes, nil
			}
			return nil, fmt.Errorf("unable to find a node that satisfies %s%s%s", affinity.key, OPERATORS[affinity.operator], affinity.value)
		}
		nodes = candidates
	}
	return nodes, nil
}
