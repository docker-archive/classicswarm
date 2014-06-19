package backends

import (
	"github.com/docker/libswarm/beam"
)

func Simulator() beam.Sender {
	s := beam.NewServer()
	s.OnVerb(beam.Spawn, beam.Handler(func(ctx *beam.Message) error {
		containers := ctx.Args
		instance := beam.Task(func(in beam.Receiver, out beam.Sender) {
			beam.Obj(out).Log("[simulator] starting\n")
			s := beam.NewServer()
			s.OnVerb(beam.Ls, beam.Handler(func(msg *beam.Message) error {
				beam.Obj(out).Log("[simulator] generating fake list of objects...\n")
				beam.Obj(msg.Ret).Set(containers...)
				return nil
			}))
			beam.Copy(s, in)
		})
		ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: instance})
		return nil
	}))
	return s
}
