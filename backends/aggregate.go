package backends

import (
	"github.com/docker/libswarm/beam"
	"github.com/flynn/go-shlex"

	"fmt"
	"log"
	"sync"
)

func Aggregate() beam.Sender {
	backend := beam.NewServer()
	backend.OnSpawn(beam.Handler(func(ctx *beam.Message) error {
		allBackends := New()
		instance := beam.NewServer()

		a, err := newAggregator(allBackends, instance, ctx.Args)
		if err != nil {
			return err
		}

		instance.OnAttach(beam.Handler(a.attach))
		instance.OnStart(beam.Handler(a.start))
		instance.OnLs(beam.Handler(a.ls))

		_, err = ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: instance})
		return err
	}))
	return backend
}

type aggregator struct {
	backends []*beam.Object
	server   *beam.Server
}

func newAggregator(allBackends *beam.Object, server *beam.Server, args []string) (*aggregator, error) {
	a := &aggregator{server: server}

	for _, argString := range args {
		args, err := shlex.Split(argString)
		if err != nil {
			return nil, err
		}
		if len(args) == 0 {
			return nil, fmt.Errorf("empty backend string")
		}
		log.Printf("aggregator: spawning %s(%#v)\n", args[0], args[1:])
		_, b, err := allBackends.Attach(args[0])
		if err != nil {
			return nil, err
		}
		i, err := b.Spawn(args[1:]...)
		if err != nil {
			return nil, err
		}
		a.backends = append(a.backends, i)
	}

	return a, nil
}

func (a *aggregator) attach(ctx *beam.Message) error {
	if ctx.Args[0] != "" {
		// TODO: implement this?
		return fmt.Errorf("attaching to a child is not implemented")
	}

	if _, err := ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: a.server}); err != nil {
		return err
	}

	var copies sync.WaitGroup

	for _, b := range a.backends {
		r, _, err := b.Attach("")
		if err != nil {
			return err
		}
		copies.Add(1)
		go func() {
			log.Printf("copying output from %#v\n", b)
			beam.Copy(ctx.Ret, r)
			log.Printf("finished output from %#v\n", b)
			copies.Done()
		}()
	}

	copies.Wait()
	return nil
}

func (a *aggregator) start(ctx *beam.Message) error {
	for _, b := range a.backends {
		err := b.Start()
		if err != nil {
			return err
		}
	}
	_, err := ctx.Ret.Send(&beam.Message{Verb: beam.Ack})
	return err
}

func (a *aggregator) ls(ctx *beam.Message) error {
	var children []string

	for _, b := range a.backends {
		bChildren, err := b.Ls()
		if err != nil {
			return err
		}
		children = append(children, bChildren...)
	}

	ctx.Ret.Send(&beam.Message{Verb: beam.Set, Args: children})

	return nil
}
