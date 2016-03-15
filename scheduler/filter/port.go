package filter

import (
	"fmt"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
	"github.com/samalba/dockerclient"
)

// PortFilter guarantees that, when scheduling a container binding a public
// port, only nodes that have not already allocated that same port will be
// considered.
type PortFilter struct {
}

// Name returns the name of the filter
func (p *PortFilter) Name() string {
	return "port"
}

// Filter is exported
func (p *PortFilter) Filter(config *cluster.ContainerConfig, nodes []*node.Node, _ bool) ([]*node.Node, error) {
	if config.HostConfig.NetworkMode == "host" {
		return p.filterHost(config, nodes)
	}

	return p.filterBridge(config, nodes)
}

func (p *PortFilter) filterHost(config *cluster.ContainerConfig, nodes []*node.Node) ([]*node.Node, error) {
	for port := range config.ExposedPorts {
		candidates := []*node.Node{}
		for _, node := range nodes {
			if !p.portAlreadyExposed(node, port) {
				candidates = append(candidates, node)
			}
		}
		if len(candidates) == 0 {
			return nil, fmt.Errorf("unable to find a node with port %s available in the Host mode", port)
		}
		nodes = candidates
	}
	return nodes, nil
}

func (p *PortFilter) filterBridge(config *cluster.ContainerConfig, nodes []*node.Node) ([]*node.Node, error) {
	for _, port := range config.HostConfig.PortBindings {
		for _, binding := range port {
			candidates := []*node.Node{}
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

func (p *PortFilter) portAlreadyExposed(node *node.Node, requestedPort string) bool {
	for _, c := range node.Containers {
		if c.Info.HostConfig.NetworkMode == "host" {
			for port := range c.Info.Config.ExposedPorts {
				if port == requestedPort {
					return true
				}
			}
		}
	}
	return false
}

func (p *PortFilter) portAlreadyInUse(node *node.Node, requested dockerclient.PortBinding) bool {
	for _, c := range node.Containers {
		// HostConfig.PortBindings contains the requested ports.
		// NetworkSettings.Ports contains the actual ports.
		//
		// We have to check both because:
		// 1/ If the port was not specifically bound (e.g. -p 80), then
		//    HostConfig.PortBindings.HostPort will be empty and we have to check
		//    NetworkSettings.Port.HostPort to find out which port got dynamically
		//    allocated.
		// 2/ If the port was bound (e.g. -p 80:80) but the container is stopped,
		//    NetworkSettings.Port will be null and we have to check
		//    HostConfig.PortBindings to find out the mapping.

		if p.compare(requested, c.Info.HostConfig.PortBindings) || p.compare(requested, c.Info.NetworkSettings.Ports) {
			return true
		}
	}
	return false
}

func (p *PortFilter) compare(requested dockerclient.PortBinding, bindings map[string][]dockerclient.PortBinding) bool {
	for _, binding := range bindings {
		for _, b := range binding {
			if b.HostPort == "" {
				// Skip undefined HostPorts. This happens in bindings that
				// didn't explicitly specify an external port.
				continue
			}

			if b.HostPort == requested.HostPort {
				// Another container on the same host is binding on the same
				// port/protocol.  Verify if they are requesting the same
				// binding IP, or if the other container is already binding on
				// every interface.
				if requested.HostIp == b.HostIp || bindsAllInterfaces(requested) || bindsAllInterfaces(b) {
					return true
				}
			}
		}
	}
	return false
}

// GetFilters returns a list of the port constraints found in the container config.
func (p *PortFilter) GetFilters(config *cluster.ContainerConfig) ([]string, error) {
	allPortConstraints := []string{}
	if config.HostConfig.NetworkMode == "host" {
		for port := range config.ExposedPorts {
			allPortConstraints = append(allPortConstraints, fmt.Sprintf("port %s (Host mode)", port))
		}
		return allPortConstraints, nil
	}

	for _, port := range config.HostConfig.PortBindings {
		for _, binding := range port {
			allPortConstraints = append(allPortConstraints, fmt.Sprintf("port %s (Bridge mode)", binding.HostPort))
		}
	}
	return allPortConstraints, nil
}

func bindsAllInterfaces(binding dockerclient.PortBinding) bool {
	return binding.HostIp == "0.0.0.0" || binding.HostIp == ""
}
