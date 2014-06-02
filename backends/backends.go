package backends

import (
	"fmt"
	"github.com/docker/libswarm/beam"
	beamutils "github.com/docker/libswarm/beam/utils"
	"io"
	"strings"
	"time"
)

// New returns a new engine, with all backends
// registered but not activated.
// To activate a backend, call a job on the resulting
// engine, named after the desired backend.
//
// Example: `New().Job("debug").Run()`
func New() beam.Sender {
	backends := beamutils.NewHub()
	backends.RegisterName("cd", func(msg *beam.Message, out beam.Sender) (bool, error) {
		return false, fmt.Errorf("no such backend: %s\n", strings.Join(msg.Args, " "))
	})
	backends.RegisterName("cd", func(msg *beam.Message, out beam.Sender) (bool, error) {
		if len(msg.Args) > 0 && msg.Args[0] == "debug" {
			debug, err := out.Send(&beam.Message{Name: "register", Ret: beam.RetPipe})
			if err != nil {
				return false, err
			}
			// Spawn the debug object
			go func() {
				for {
					msg, err := debug.Receive(beam.Ret)
					if err == io.EOF {
						return
					}
					if err != nil {
						return
					}
					fmt.Printf("[DEBUG] %s %s\n", msg.Name, strings.Join(msg.Args, " "))
					if _, err := out.Send(msg); err != nil {
						return
					}
				}
			}()
			return false, nil
		}
		return true, nil
	})
	backends.RegisterName("cd", func(msg *beam.Message, out beam.Sender) (bool, error) {
		if len(msg.Args) > 0 && msg.Args[0] == "fakeclient" {
			_, err := out.Send(&beam.Message{Name: "register", Ret: beam.NopSender{}})
			if err != nil {
				return false, err
			}
			// Spawm the fakeclient task
			// FIXME: only do this after started?
			go func() {
				out.Send(&beam.Message{Name: "log", Args: []string{"fake client starting"}})
				defer out.Send(&beam.Message{Name: "log", Args: []string{"fake client terminating"}})
				for {
					time.Sleep(1 * time.Second)
					_, err := out.Send(&beam.Message{Name: "log", Args: []string{"fake client reporting for duty"}})
					if err != nil {
						return
					}
					if _, err := out.Send(&beam.Message{Name: "children", Ret: beam.NopSender{}}); err != nil {
						return
					}
				}
			}()
			return false, nil
		}
		return true, nil
	})
	return backends
}
