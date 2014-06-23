package backends

import (
	"github.com/docker/libswarm"
	"github.com/flynn/go-shlex"

	"fmt"
	"log"
	"sync"
)

func Aggregate() libswarm.Sender {
	backend := libswarm.NewServer()
	backend.OnSpawn(func(cmd ...string) (libswarm.Sender, error) {
		allBackends := New()
		instance := libswarm.NewServer()

		a, err := newAggregator(allBackends, instance, cmd)
		if err != nil {
			return nil, err
		}

		instance.OnAttach(a.attach)
		instance.OnStart(a.start)
		instance.OnLs(a.ls)

		return instance, nil
	})
	return backend
}

type aggregator struct {
	backends []*libswarm.Object
	server   *libswarm.Server
}

func newAggregator(allBackends *libswarm.Object, server *libswarm.Server, args []string) (*aggregator, error) {
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

func (a *aggregator) attach(name string, ret libswarm.Sender) error {
	if name != "" {
		// TODO: implement this?
		return fmt.Errorf("attaching to a child is not implemented")
	}

	if _, err := ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: a.server}); err != nil {
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
			libswarm.Copy(ret, r)
			log.Printf("finished output from %#v\n", b)
			copies.Done()
		}()
	}

	copies.Wait()
	return nil
}

func (a *aggregator) start() error {
	for _, b := range a.backends {
		err := b.Start()
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *aggregator) ls() ([]string, error) {
	var children []string

	for _, b := range a.backends {
		bChildren, err := b.Ls()
		if err != nil {
			return nil, err
		}
		children = append(children, bChildren...)
	}

	return children, nil
}
