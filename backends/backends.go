package backends

import (
	"github.com/dotcloud/docker/engine"
)

// New returns a new engine, with all backends
// registered but not activated.
// To activate a backend, call a job on the resulting
// engine, named after the desired backend.
//
// Example: `New().Job("debug").Run()`
func New() *engine.Engine {
	back := engine.New()
	back.Logging = false
	// Register all backends here
	Debug().Install(back)
	Simulator().Install(back)
	Forward().Install(back)
	CloudBackend().Install(back)
	Tutum().Install(back)
	return back
}
