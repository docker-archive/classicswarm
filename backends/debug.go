package backends

import (
	"github.com/dotcloud/docker/engine"
	"strings"
)

func Debug() engine.Installer {
	return &debug{}
}

type debug struct {
}

func (d *debug) Install(eng *engine.Engine) error {
	eng.Register("debug", func(job *engine.Job) engine.Status {
		job.Eng.RegisterCatchall(func(job *engine.Job) engine.Status {
			job.Logf("--> %s %s\n", job.Name, strings.Join(job.Args, " "))
			for k, v := range job.Env().Map() {
				job.Logf("        %s=%s\n", k, v)
			}
			return engine.StatusOK
		})
		return engine.StatusOK
	})
	return nil
}
