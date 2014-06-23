package backends

import (
	"github.com/docker/libswarm"
	"github.com/docker/libswarm/utils"
)

func Simulator() libswarm.Sender {
	s := libswarm.NewServer()
	s.OnVerb(libswarm.Spawn, libswarm.Handler(func(ctx *libswarm.Message) error {
		containers := ctx.Args
		instance := utils.Task(func(in libswarm.Receiver, out libswarm.Sender) {
			libswarm.Obj(out).Log("[simulator] starting\n")
			s := libswarm.NewServer()
			s.OnVerb(libswarm.Ls, libswarm.Handler(func(msg *libswarm.Message) error {
				libswarm.Obj(out).Log("[simulator] generating fake list of objects...\n")
				libswarm.Obj(msg.Ret).Set(containers...)
				return nil
			}))
			libswarm.Copy(s, in)
		})
		ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: instance})
		return nil
	}))
	return s
}
