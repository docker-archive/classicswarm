package backends

import (
	"fmt"
	"strings"

	"github.com/docker/libswarm"
)

type stdoutLogger struct {
	*libswarm.Server
}

func StdoutLogger() libswarm.Sender {
	backend := libswarm.NewServer()

	backend.OnSpawn(func(cmd ...string) (libswarm.Sender, error) {
		fl := &stdoutLogger{Server: libswarm.NewServer()}

		fl.OnAttach(fl.attach)
		fl.OnStart(fl.start)
		fl.OnLog(fl.log)

		return fl, nil
	})
	return backend
}

func (l *stdoutLogger) attach(name string, ret libswarm.Sender) error {
	ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: l.Server})
	<-make(chan struct{})
	return nil
}

func (l *stdoutLogger) start() error {
	fmt.Errorf("logger: start not implemented")
	return nil
}

func (l *stdoutLogger) log(msg ...string) error {
	fmt.Println(strings.Join(msg, "\t"))
	return nil
}
