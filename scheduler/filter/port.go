package filter

import (
	"fmt"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

// PortFilter guarantees that, when scheduling a container binding a public
// port, only nodes that have not already allocated that same port will be
// considered.
type PortFilter struct {
}

func (p *PortFilter) Filter(config *dockerclient.ContainerConfig, nodes []*cluster.Node) ([]*cluster.Node, error) {
	for _, port := range config.HostConfig.PortBindings {
		for _, binding := range port {
			candidates := []*cluster.Node{}
			for _, node := range nodes {
				if !p.portAlreadyInUse(node, binding) {
					candidates = append(candidates, node)
				}
			}
			if len(candidates) == 0 {
				return nil, fmt.Errorf("unable to find a node with port %s available", binding.HostPort)
			}
			nodes = candidates
		}
	}
	return nodes, nil
}

func (p *PortFilter) portAlreadyInUse(node *cluster.Node, requested dockerclient.PortBinding) bool {
	for _, c := range node.Containers() {
		for _, port := range c.Info.NetworkSettings.Ports {
			for _, binding := range port {
				if binding.HostPort == requested.HostPort {
					// Another container on the same host is binding on the same
					// port/protocol.  Verify if they are requesting the same
					// binding IP, or if the other container is already binding on
					// every interface.
					if requested.HostIp == binding.HostIp || bindsAllInterfaces(requested) || bindsAllInterfaces(binding) {
						return true
					}
				}
			}
		}
	}
	return false
}

func bindsAllInterfaces(binding dockerclient.PortBinding) bool {
	return binding.HostIp == "0.0.0.0" || binding.HostIp == ""
}
