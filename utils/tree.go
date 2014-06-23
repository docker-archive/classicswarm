package utils

import (
	"github.com/docker/libswarm"

	"sort"
)

type Tree struct {
	*libswarm.Server
	children map[string]libswarm.Sender
}

func NewTree() *Tree {
	t := &Tree{
		Server:   libswarm.NewServer(),
		children: make(map[string]libswarm.Sender),
	}
	t.OnVerb(libswarm.Attach, libswarm.Handler(func(msg *libswarm.Message) error {
		if len(msg.Args) == 0 || msg.Args[0] == "" {
			msg.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: t})
			return nil
		}
		if child, exists := t.children[msg.Args[0]]; exists {
			msg.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: child})
			return nil
		}
		libswarm.AsClient(msg.Ret).Error("not found")
		return nil
	}))
	t.OnVerb(libswarm.Ls, libswarm.Handler(func(msg *libswarm.Message) error {
		names := make([]string, 0, len(t.children))
		for name := range t.children {
			names = append(names, name)
		}
		sort.Strings(names)
		libswarm.AsClient(msg.Ret).Set(names...)
		return nil
	}))
	return t
}

func (t *Tree) Bind(name string, dst libswarm.Sender) *Tree {
	t.children[name] = dst
	return t
}
