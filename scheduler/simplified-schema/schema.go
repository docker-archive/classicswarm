package simplifiedschema

import (
	apitypes "github.com/docker/engine-api/types"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

// Container represents a container in simplified form
type Container struct {
	Id      string            `json:"id"`
	Names   []string          `json:"names"`
	Image   string            `json:"image"`
	ImageId string            `json:"image_id"`
	State   string            `json:"state"`
	Ports   []apitypes.Port   `json:"ports"`
	Labels  map[string]string `json:"labels"`
}

// Image represents a image in simplified form
type Image struct {
	Id      string            `json:"id"`
	Tags    []string          `json:"repo_tags"`
	Digests []string          `json:"repo_digests"`
	Labels  map[string]string `json:"labels"`
}

// Resource represents some kind of consumable resource (e.g. cpu / ram)
type Resource struct {
	Used  int64 `json:"used"`
	Total int64 `json:"total"`
}

// Node represents a node in simplified form
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
	EngineVersion   string            `json:"engine_version"`
}

// Placement is the top level simplified placement structure
type Placement struct {
	Config cluster.ContainerConfig `json:"config"`
	Nodes  []Node                  `json:"nodes"`
}

// SimplifyPlacement takes a placement request and a list of nodes and simplfies it for wire transfer
func SimplifyPlacement(config *cluster.ContainerConfig, nodes []*node.Node) *Placement {
	placement := &Placement{}
	placement.Config = *config

	for _, node := range nodes {
		placement.Nodes = append(placement.Nodes, simplifyNode(node))
	}

	return placement
}

// simplifyNode simplifies an individual node record
func simplifyNode(node *node.Node) Node {
	var images []Image
	for _, image := range node.Images {
		simple_image := Image{
			Id:      image.ID,
			Tags:    image.RepoTags,
			Digests: image.RepoDigests,
			Labels:  image.Labels,
		}
		images = append(images, simple_image)
	}

	var containers []Container
	for _, container := range node.Containers {
		simple_container := Container{
			Id:      container.ID,
			Names:   container.Names,
			Image:   container.Image,
			ImageId: container.ImageID,
			State:   container.State,
			Ports:   container.Ports,
			Labels:  container.Labels,
		}
		containers = append(containers, simple_container)
	}

	return Node{
		ID:              node.ID,
		HealthIndicator: node.HealthIndicator,
		Name:            node.Name,
		Addr:            node.Addr,
		Labels:          node.Labels,
		Containers:      containers,
		Images:          images,
		EngineVersion:   node.Engine.Version,
		CPUs: Resource{
			Total: node.TotalCpus,
			Used:  node.UsedCpus,
		},
		Memory: Resource{
			Total: node.TotalMemory,
			Used:  node.UsedMemory,
		},
	}
}
