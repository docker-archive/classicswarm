package cluster

import (
	"strings"

	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/engine-api/types"
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
			tmp[network.ID] = network
		}
	}
	uniq := Networks{}
	for _, network := range tmp {
		uniq = append(uniq, network)
	}
	return uniq
}

// Filter returns networks filtered by names or ids
func (networks Networks) Filter(names []string, ids []string, types []string) Networks {
	typeFilter := func(network *Network) bool {
		if len(types) > 0 {
			for _, typ := range types {
				if typ == "custom" && !network.isPreDefined() {
					return true
				}
				if typ == "builtin" && network.isPreDefined() {
					return true
				}
			}
		} else {
			return true
		}
		return false
	}

	out := Networks{}
	if len(names) == 0 && len(ids) == 0 {
		for _, network := range networks.Uniq() {
			if typeFilter(network) {
				out = append(out, network)
			}
		}
	} else {
		for _, idOrName := range append(names, ids...) {
			if network := networks.Get(idOrName); network != nil {
				if typeFilter(network) {
					out = append(out, network)
				}
			}
		}
	}

	return out
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
