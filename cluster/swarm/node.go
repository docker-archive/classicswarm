package swarm

import (
	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

// NewNode is exported
func NewNode(addr string, overcommitRatio float64) *node {
	return &node{Engine: *cluster.NewEngine(addr, "", overcommitRatio)}
}

type node struct {
	cluster.Engine
}

func (n *node) create(config *dockerclient.ContainerConfig, name string, pullImage bool) (*cluster.Container, error) {
	var (
		err    error
		id     string
		client = n.Client
	)

	newConfig := *config

	// nb of CPUs -> real CpuShares
	newConfig.CpuShares = config.CpuShares * 100 / n.Cpus

	if id, err = client.CreateContainer(&newConfig, name); err != nil {
		// If the error is other than not found, abort immediately.
		if err != dockerclient.ErrNotFound || !pullImage {
			return nil, err
		}
		// Otherwise, try to pull the image...
		if err = n.Pull(config.Image); err != nil {
			return nil, err
		}
		// ...And try again.
		if id, err = client.CreateContainer(&newConfig, name); err != nil {
			return nil, err
		}
	}

	// Register the container immediately while waiting for a state refresh.
	// Force a state refresh to pick up the newly created container.
	n.RefreshContainer(id, true)

	return n.Container(id), nil
}

func (n *node) destroy(container *cluster.Container, force bool) error {
	if err := n.Client.RemoveContainer(container.Id, force, true); err != nil {
		return err
	}

	// Remove the container from the state. Eventually, the state refresh loop
	// will rewrite this.
	return n.RemoveContainer(container)
}
