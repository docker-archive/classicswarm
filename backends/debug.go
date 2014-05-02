package backends

import (
	"fmt"
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
			fmt.Printf("--> %s %s\n", job.Name, strings.Join(job.Args, " "))
			for k, v := range job.Env().Map() {
				fmt.Printf("        %s=%s\n", k, v)
			}
			// This helps us detect the race condition if our time.Sleep
			// missed it. (see comment in main)
			if job.Name == "acceptconnections" {
				panic("race condition in github.com/dotcloud/docker/api/server/ServeApi")
			}
			return engine.StatusOK
		})
		return engine.StatusOK
	})
	return nil
}
