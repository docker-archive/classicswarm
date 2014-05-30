package backends

import (
	"fmt"

	"github.com/dotcloud/docker/engine"
)

func NewMux() *EngineMux {
	engineMux := &EngineMux{
		enabled: make(map[string]*engine.Engine),
		engines: make(map[string]*engine.Engine),
	}

	// Register all backends here
	engineMux.Register("debug", Debug())
	engineMux.Register("simulator", Simulator())
	engineMux.Register("forward", Forward())
	engineMux.Register("cloud", CloudBackend())

	return engineMux
}

type EngineMux struct {
	enabled map[string]*engine.Engine
	engines map[string]*engine.Engine
}

func (em *EngineMux) Install(eng *engine.Engine) (err error) {
	eng.RegisterCatchall(em.handler)
	return
}

func (em *EngineMux) handler(parentJob *engine.Job) (status engine.Status) {
	for name, eng := range em.enabled {
		// FIXME: This could get hairy if more than one backend tries to write
		// to stdout.
		nextJob := eng.Job(parentJob.Name, parentJob.Args...)
		nextJob.Stdout.Add(parentJob.Stdout)
		nextJob.Stderr.Add(parentJob.Stderr)
		nextJob.Stdin.Add(parentJob.Stdin)

		for key, val := range parentJob.Env().Map() {
			nextJob.Setenv(key, val)
		}

		if err := nextJob.Run(); err != nil {
			parentJob.Logf("Error occured while dispatching job to engine: %s. Reason: %v", name, err)
		}
	}

	return engine.StatusOK
}

func (em *EngineMux) GetEngine(name string) (eng *engine.Engine, found bool) {
	eng, found = em.engines[name]
	return
}

func (em *EngineMux) Enable(name string, args ...string) (err error) {
	if _, found := em.enabled[name]; found {
		err = fmt.Errorf("Engine %s already enabled.", name)
	} else {
		if targetEngine, found := em.engines[name]; found {
			if err = targetEngine.Job(name, args...).Run(); err == nil {
				em.enabled[name] = targetEngine
			}
		} else {
			err = fmt.Errorf("No engine registered as: %s.", name)
		}
	}

	return
}

func (em *EngineMux) Register(name string, installer engine.Installer) (err error) {
	targetEngine, found := em.engines[name]

	if !found {
		targetEngine = engine.New()
		em.engines[name] = targetEngine
	}

	return installer.Install(targetEngine)
}
