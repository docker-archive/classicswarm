package beam

import (
	"sort"
)

type Tree struct {
	*Server
	children map[string]Sender
}

func NewTree() *Tree {
	t := &Tree{
		Server:   NewServer(),
		children: make(map[string]Sender),
	}
	t.OnVerb(Attach, Handler(func(msg *Message) error {
		if len(msg.Args) == 0 || msg.Args[0] == "" {
			msg.Ret.Send(&Message{Verb: Ack, Ret: t})
			return nil
		}
		if child, exists := t.children[msg.Args[0]]; exists {
			msg.Ret.Send(&Message{Verb: Ack, Ret: child})
			return nil
		}
		Obj(msg.Ret).Error("not found")
		return nil
	}))
	t.OnVerb(Ls, Handler(func(msg *Message) error {
		names := make([]string, 0, len(t.children))
		for name := range t.children {
			names = append(names, name)
		}
		sort.Strings(names)
		Obj(msg.Ret).Set(names...)
		return nil
	}))
	return t
}

func (t *Tree) Bind(name string, dst Sender) *Tree {
	t.children[name] = dst
	return t
}
