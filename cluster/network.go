package cluster

import (
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/pkg/stringid"
)

// Network is exported
type Network struct {
	types.NetworkResource

	Engine *Engine
}

func (network *Network) isPreDefined() bool {
	return (network.Name == "none" || network.Name == "host" || network.Name == "bridge")
}

// Networks represents an array of networks
type Networks []*Network

// Uniq returns all uniq networks
func (networks Networks) Uniq() Networks {
	tmp := make(map[string]*Network)
	for _, network := range networks {
		if _, ok := tmp[network.ID]; ok {
			for id, endpoint := range network.Containers {
				tmp[network.ID].Containers[id] = endpoint
			}
		} else {
			netCopy := *network
			netCopy.Containers = make(map[string]types.EndpointResource)
			for key, value := range network.Containers {
				netCopy.Containers[key] = value
			}
			tmp[network.ID] = &netCopy
		}
	}
	uniq := Networks{}
	for _, network := range tmp {
		uniq = append(uniq, network)
	}
	return uniq
}

// Filter returns networks filtered by names, IDs, nodes, labels, etc.
func (networks Networks) Filter(filter filters.Args) Networks {
	includeFilter := func(network *Network) bool {
		for _, typ := range filter.Get("type") {
			if typ == "custom" && network.isPreDefined() {
				return false
			}
			if typ == "builtin" && !network.isPreDefined() {
				return false
			}
		}
		if filter.Include("label") {
			if !filter.MatchKVList("label", network.Labels) {
				return false
			}
		}
		if filter.Include("driver") {
			if !filter.ExactMatch("driver", network.Driver) {
				return false
			}
		}
		for _, node := range filter.Get("node") {
			if network.Engine.Name != node {
				return false
			}
		}
		return true
	}

	names := filter.Get("name")
	ids := filter.Get("id")
	out := Networks{}
	if len(names) == 0 && len(ids) == 0 {
		for _, network := range networks.Uniq() {
			if includeFilter(network) {
				out = append(out, network)
			}
		}
	} else {
		for _, idOrName := range append(names, ids...) {
			if network := networks.Get(idOrName); network != nil {
				if includeFilter(network) {
					out = append(out, network)
				}
			}
		}
		// substring match based on name filter
		for _, idOrName := range append(names, ids...) {
			if networkList := networks.matchByName(idOrName, filter); len(networkList) != 0 {
				for _, network := range networkList.Uniq() {
					if includeFilter(network) {
						out = append(out, network)
					}
				}
			}
		}
	}

	return out.Uniq()
}

// RemoveDuplicateEndpoints returns a copy of input network
// where duplicate container endpoints in the network are removed.
// See https://github.com/docker/swarm/issues/1969
// This function should be disabled when the bug is fixed in Docker network
func (network *Network) RemoveDuplicateEndpoints() *Network {
	// build a map from endpointID -> endpointIndex
	endpointMap := make(map[string]string)
	// traverse the endpoints to find the correct endpointIndex for each endpointID
	for endpointIndex, endpointResource := range network.NetworkResource.Containers {
		endpointID := endpointResource.EndpointID
		// if this endpointID doesn't exist yet, add it
		// if this endpointID exists, but endpointIndex is not a duplicate, use
		// this endpointIndex
		if _, ok := endpointMap[endpointID]; !ok || !strings.Contains(endpointIndex, endpointID) {
			endpointMap[endpointID] = endpointIndex
		}
	}
	// Make a copy of the network
	netCopy := *network
	// clean up existing endpoints
	netCopy.Containers = make(map[string]types.EndpointResource)
	// add the endpoint index from endpointMap
	for _, index := range endpointMap {
		netCopy.Containers[index] = network.Containers[index]
	}
	return &netCopy
}

// matchByName checks if any networks match by substring
func (networks Networks) matchByName(IDOrName string, filter filters.Args) Networks {
	candidates := Networks{}
	for _, network := range networks {
		if filter.Match("name", network.Name) {
			candidates = append(candidates, network)
		}
	}
	return candidates
}

// Get returns a network using its ID or Name
func (networks Networks) Get(IDOrName string) *Network {
	// Abort immediately if the name is empty.
	if len(IDOrName) == 0 {
		return nil
	}

	// Match exact or short Network ID.
	for _, network := range networks {
		if network.ID == IDOrName || stringid.TruncateID(network.ID) == IDOrName {
			return network
		}
	}

	candidates := Networks{}

	// Match name or engine/name.
	for _, network := range networks {
		if network.Name == IDOrName || network.Engine.ID+"/"+network.Name == IDOrName || network.Engine.Name+"/"+network.Name == IDOrName {
			candidates = append(candidates, network)
		}
	}

	// Return if we found a unique match.
	if size := len(candidates.Uniq()); size == 1 {
		return candidates[0]
	} else if size > 1 {
		return nil
	}

	// Match /name and return as soon as we find one.
	for _, network := range networks {
		if network.Name == "/"+IDOrName {
			return network
		}
	}

	// Match Network ID prefix.
	for _, network := range networks {
		if strings.HasPrefix(network.ID, IDOrName) {
			candidates = append(candidates, network)
		}
	}

	if len(candidates.Uniq()) == 1 {
		return candidates[0]
	}

	return nil
}
