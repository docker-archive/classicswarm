package simplifiedschema

import (
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

type Config struct {
	Image     string
	CpuShares int
	MemShares int
}

type Container struct {
	Names  []string          `json:"names"`
	Image  string            `json:"image"`
	State  string            `json:"state"`
	Ports  []int             `json:"ports"`
	Labels map[string]string `json:"labels"`
}

type Image struct {
	RepoTags    []string          `json:"repo_tags"`
	RepoDigests []string          `json:"repo_digests"`
	Labels      map[string]string `json:"labels"`
}

type Resource struct {
	Used  int64 `json:"used"`
	Total int64 `json:"total"`
}

type Node struct {
	ID              string            `json:"id"`
	HealthIndicator int64             `json:"scheduling_health"`
	Name            string            `json:"name"`
	Addr            string            `json:"addr"`
	Labels          map[string]string `json:"labels"`
	Containers      []Container       `json:"containers"`
	Images          []Image           `json:"images"`
	CPUs            Resource          `json:"cpus"`
	Memory          Resource          `json:"memory"`
	//EngineVersion   string            `json:"engine_version"`
}

type Placement struct {
	Config cluster.ContainerConfig `json:"config"`
	Nodes  []Node                  `json:"nodes"`
}

func SimplifyPlacement(config *cluster.ContainerConfig, nodes []*node.Node) *Placement {
	placement := &Placement{}
	placement.Config = *config

	for _, node := range nodes {
		placement.Nodes = append(placement.Nodes, SimplifyNode(node))
	}

	return placement
}

func SimplifyNode(node *node.Node) Node {
	var images []Image
	for _, image := range node.Images {
		simple_image := Image{
			RepoTags:    image.RepoTags,
			RepoDigests: image.RepoDigests,
			Labels:      image.Labels,
		}
		images = append(images, simple_image)
	}

	var containers []Container
	for _, container := range node.Containers {
		simple_container := Container{
			Names: container.Names,
			Image: container.Image,
			State: container.State,
			//Ports:  container.Ports,
			Labels: container.Labels,
		}
		containers = append(containers, simple_container)
	}

	return Node{
		ID:              node.ID,
		HealthIndicator: node.HealthIndicator,
		Name:            node.Name,
		Addr:            node.Addr,
		CPUs: Resource{
			Total: node.TotalCpus,
			Used:  node.UsedCpus,
		},
		Memory: Resource{
			Total: node.TotalMemory,
			Used:  node.UsedMemory,
		},
		Labels:     node.Labels,
		Containers: containers,
		Images:     images,
	}
}
