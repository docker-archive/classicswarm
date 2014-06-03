package backends

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/dotcloud/docker/engine"
	"github.com/flynn/go-shlex"
)

type _EngineBus struct {
	name    string
	engines map[string]*engine.Engine
}

// Create and return a new EngineBus. Since EngineBus implements the
// engine.Installer interface, the return value can be treated as any other
// installer.
func EngineBus(name string) (eb *_EngineBus) {
	return &_EngineBus{
		name:    name,
		engines: make(map[string]*engine.Engine),
	}
}

// Links a frontend engine to a backend engine via a catch all handler.
func Link(frontend, backend *engine.Engine) {
	frontend.RegisterCatchall(func(job *engine.Job) (status engine.Status) {
		// Rewrist the engine to point to the backend and then call dispatch
		job.Eng = backend
		return dispatch(job)
	})
}

// Lazy developer is lazy. This just lets me get at the id field which is
// thankfully a string for type wrangling purposes. This function is only
// here to provide more informative debug output.
func getEngineId(eng *engine.Engine) (id string) {
	engValue := reflect.ValueOf(eng).Elem()
	return engValue.FieldByName("id").String()
}

func inheritJobComponents(parent, child *engine.Job) {
	child.Stdout.Add(parent.Stdout)
	child.Stderr.Add(parent.Stderr)
	child.Stdin.Add(parent.Stdin)

	for key, val := range parent.Env().Map() {
		child.Setenv(key, val)
	}
}

func dispatch(job *engine.Job) (status engine.Status) {
	// reappend the name of the job as part of the args
	args := append([]string{job.Name}, job.Args...)

	// Set up the route job and run it
	route := job.Eng.Job("route", args...)
	inheritJobComponents(job, route)

	if err := route.Run(); err != nil {
		job.Printf("Failed job route. Reason: %v", err)
	}

	return engine.Status(route.StatusCode())
}

func parseCmd(txt string) (string, []string, error) {
	l, err := shlex.NewLexer(strings.NewReader(txt))
	if err != nil {
		return "", nil, err
	}

	var cmd []string
	for {
		word, err := l.NextWord()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", nil, err
		}

		cmd = append(cmd, word)
	}

	if len(cmd) == 0 {
		return "", nil, fmt.Errorf("parse error: empty command")
	}

	return cmd[0], cmd[1:], nil
}

func (eb *_EngineBus) Install(eng *engine.Engine) (err error) {
	eng.Register(eb.name, eb.init)
	eng.Register("route", eb.route)
	return
}

func (eb *_EngineBus) Register(name string, eng *engine.Engine) {
	eb.engines[name] = eng
}

func (eb *_EngineBus) init(job *engine.Job) (status engine.Status) {
	if len(job.Args) == 0 {
		status = job.Errorf("%s \"backend_1 arg1 -arg 2; backend_2 arg3 --arg 4 -a\"")
	} else {
		eb.processArgs(job)
	}

	return
}

func (eb *_EngineBus) processArgs(job *engine.Job) (status engine.Status) {
	status = engine.StatusOK
	engineInitCommands := strings.Split(job.Args[0], ";")

	// Backends are treated as jobs that are started up in addition to
	// adding the backend to the engine multiplexer.
	for _, eiCmd := range engineInitCommands {
		if eiName, eiArgs, err := parseCmd(strings.TrimSpace(eiCmd)); err == nil {
			// Init a new engine for this backend
			eng := engine.New()
			eng.Logging = false

			// Add the engine to our collection for tracking purposes
			eb.Register(eiName, eng)

			// Snag the initjob, modify its engine to point to the new one
			initJob := job.Eng.Job(eiName, eiArgs...)
			initJob.Eng = eng

			// Inherit Stdout and Stderr
			inheritJobComponents(job, initJob)

			// Enable the backend engine in the engine bu
			if err := initJob.Run(); err != nil {
				status = job.Errorf("Failed to load %s: %v\n", eiName, err)
				break
			}
		} else {
			status = job.Errorf("Failed to parse command: %s, %v\n", eiCmd, err)
			break
		}
	}

	return
}

func (eb *_EngineBus) route(job *engine.Job) (status engine.Status) {
	cmd := job.Args[0]
	args := job.Args[1:]

	for name, eng := range eb.engines {
		fmt.Printf("Routing (%s %v) to: %s\n", cmd, args, name)

		// FIXME: This could get hairy if more than one backend tries to write
		// to stdout. Alternatives...?
		nextJob := eng.Job(cmd, args...)
		inheritJobComponents(job, nextJob)

		for key, val := range job.Env().Map() {
			nextJob.Setenv(key, val)
		}

		if err := nextJob.Run(); err != nil {
			job.Logf("Error occured while dispatching job to engine. Reason: %v", err)
		}

		fmt.Printf("Engine(%s)::Job(%s) returned %v\n", getEngineId(eng), nextJob.Name, nextJob.StatusCode())
	}

	return engine.StatusOK
}
