package backends

import (
	"io"
	"fmt"
	"github.com/docker/libswarm/beam"
	beamutils "github.com/docker/libswarm/beam/utils"
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
	backends.RegisterName("cd", func(msg *beam.Message, in beam.Receiver, out beam.Sender, next beam.Sender) (bool, error) {
		return false, fmt.Errorf("no such backend: %s\n", strings.Join(msg.Args, " "))
	})
	backends.RegisterName("cd", func(msg *beam.Message, in beam.Receiver, out beam.Sender, next beam.Sender) (bool, error) {
		if len(msg.Args) > 0 && msg.Args[0] == "debug" {
			debugr, debugw, err := out.Send(&beam.Message{Name: "register"}, beam.R|beam.W)
			if err != nil {
				return false, err
			}
			go func() {
				for {
					msg, msgr, msgw, err := debugr.Receive(beam.R | beam.W)
					if err == io.EOF {
						return
					}
					if err != nil {
						return
					}
					fmt.Printf("[DEBUG] %s %s\n", msg.Name, strings.Join(msg.Args, " "))
					// FIXME: goroutine?
					Splice(debugw, msg, msgr, msgw)
				}
			}()
			return false, nil
		}
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
				go beamutils.Copy(beam.NopSender{}, containersR)
			}
		})
		return true, nil
	})
	return backends
}

func Splice(dst beam.Sender, msg *beam.Message, r beam.Receiver, w beam.Sender) error {
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
			dst.Close()
		}()
	}
	_copy(dstW, r)
	_copy(w, dstR)
	tasks.Wait()
	return nil
}
