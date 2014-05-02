package backends

import (
	"github.com/dotcloud/docker/engine"
)

func Simulator() engine.Installer {
	return &simulator{}
}

type simulator struct {
	containers []string
}

func (s *simulator) Install(eng *engine.Engine) error {
	eng.Register("simulator", func(job *engine.Job) engine.Status {
		s.containers = job.Args
		job.Eng.Register("containers", func(job *engine.Job) engine.Status {
			t := engine.NewTable("Id", len(s.containers))
			for _, c := range s.containers {
				e := &engine.Env{}
				e.Set("Id", c)
				e.Set("Image", "foobar")
				t.Add(e)
			}
			t.WriteListTo(job.Stdout)
			return engine.StatusOK
		})
		return engine.StatusOK
	})
	return nil
}
