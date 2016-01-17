package cluster

import "github.com/samalba/dockerclient"

// Volume is exported
type Volume struct {
	dockerclient.Volume

	Engine *Engine
}

// Volumes represents an array of volumes
type Volumes []*Volume

// Get returns a volume using it's ID or Name
func (volumes Volumes) Get(name string) *Volume {
	// Abort immediately if the name is empty.
	if len(name) == 0 {
		return nil
	}

	candidates := []*Volume{}

	// Match name or engine/name.
	for _, volume := range volumes {
		if volume.Name == name || volume.Engine.ID+"/"+volume.Name == name || volume.Engine.Name+"/"+volume.Name == name {
			candidates = append(candidates, volume)
		}
	}

	// Return if we found a unique match.
	if size := len(candidates); size == 1 {
		return candidates[0]
	} else if size > 1 {
		return nil
	}

	// Match /name and return as soon as we find one.
	for _, volume := range volumes {
		if volume.Name == "/"+name {
			return volume
		}
	}

	if len(candidates) == 1 {
		return candidates[0]
	}

	return nil
}
