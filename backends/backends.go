package backends

import (
	"github.com/docker/libswarm/beam"
)

// New returns a new engine, with all backends
// registered but not activated.
// To activate a backend, call a job on the resulting
// engine, named after the desired backend.
//
// Example: `New().Job("debug").Run()`
func New() *beam.Object {
	backends := beam.NewTree()
	backends.Bind("simulator", Simulator())
	backends.Bind("debug", Debug())
	backends.Bind("fakeclient", FakeClient())
	backends.Bind("dockerclient", DockerClient())
	backends.Bind("exec", Exec())
	backends.Bind("dockerserver", DockerServer())
	backends.Bind("orchard", Orchard())
	backends.Bind("aggregate", Aggregate())
	backends.Bind("shipyard", Shipyard())
	backends.Bind("ec2", Ec2())
	backends.Bind("tutum", Tutum())
	return beam.Obj(backends)
}
