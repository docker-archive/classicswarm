package backends

import (
	"fmt"
	"github.com/docker/beam"
	beamutils "github.com/docker/beam/utils"
	"strings"
	"sync"
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
	backends.RegisterName("debug", func(msg *beam.Message, in beam.Receiver, out beam.Sender, next beam.Sender) (bool, error) {
		backends.RegisterTask(func(r beam.Receiver, w beam.Sender) error {
			for {
				msg, msgr, msgw, err := r.Receive(beam.R | beam.W)
				if err != nil {
					return err
				}
				fmt.Printf("[DEBUG] %s %s\n", msg.Name, strings.Join(msg.Args, " "))
				// FIXME: goroutine?
				splice(w, msg, msgr, msgw)
			}
		})
		return true, nil
	})
	backends.RegisterName("fakeclient", func(msg *beam.Message, in beam.Receiver, out beam.Sender, next beam.Sender) (bool, error) {
		backends.RegisterTask(func(r beam.Receiver, w beam.Sender) error {
			for {
				time.Sleep(1 * time.Second)
				_, _, err := w.Send(&beam.Message{Name: "log", Args: []string{"fake client reporting for duty"}}, 0)
				if err != nil {
					return err
				}
				containersR, _, err := w.Send(&beam.Message{Name: "containers"}, beam.R)
				if err != nil {
					return err
				}
				go beamutils.Copy(beamutils.NopSender{}, containersR)
			}
		})
		return true, nil
	})
	return backends
}

func splice(dst beam.Sender, msg *beam.Message, r beam.Receiver, w beam.Sender) error {
	dstR, dstW, err := dst.Send(msg, beam.R|beam.W)
	if err != nil {
		return err
	}
	defer dstW.Close()
	var tasks sync.WaitGroup
	_copy := func(dst beam.Sender, src beam.Receiver) {
		tasks.Add(1)
		go func() {
			defer tasks.Done()
			beamutils.Copy(dst, src)
		}()
	}
	_copy(dstW, r)
	_copy(w, dstR)
	tasks.Wait()
	return nil
}
