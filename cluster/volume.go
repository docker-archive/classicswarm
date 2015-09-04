package cluster

import "github.com/samalba/dockerclient"

// Volume is exported
type Volume struct {
	dockerclient.Volume

	Engine *Engine
}
