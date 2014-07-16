package backends

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/libswarm"
	"github.com/docker/libswarm/utils"
)

type logForwarder struct {
	service         *libswarm.Server
	dockerInstances map[string]struct{}
	logFacility     libswarm.Sender
}

// This attaches to any/all containers and gets the stdout/stderr streams from them.
//TODO: decouple from dockerclient backend
func LogForwarder() libswarm.Sender {
	l := &logForwarder{
		service:         libswarm.NewServer(),
		dockerInstances: map[string]struct{}{},
	}

	l.service.OnLog(l.log)
	l.service.OnVerb(libswarm.Spawn, libswarm.Handler(l.spawn))
	l.service.OnStart(l.start)
	return l.service
}

func (l *logForwarder) spawn(msg *libswarm.Message) (err error) {
	for _, host := range msg.Args {
		l.dockerInstances[host] = struct{}{}
	}

	instance := utils.Task(func(in libswarm.Receiver, out libswarm.Sender) {
		l.logFacility = out
	})

	msg.Ret.Send(&libswarm.Message{
		Verb: libswarm.Ack,
		Ret:  instance,
	})

	return libswarm.AsClient(l.service).Start()
}

func (l *logForwarder) log(msg ...string) error {
	libswarm.AsClient(l.logFacility).Log(strings.Join(msg, "\t"))
	return nil
}

func (l *logForwarder) start() error {
	return l.getAllLogs()
}

func (l *logForwarder) getContainerLog(client *libswarm.Client, host, name string) error {
	_, out, err := client.Attach(name)
	if err != nil {
		return err
	}
	c := libswarm.AsClient(out)
	logs, _, err := c.Attach("")
	if err != nil {
		return err
	}
	prefix := []string{host, name}
	var tasks sync.WaitGroup
	go func() {
		defer tasks.Done()
		err := l.DecodeStream(logs, "stdout", prefix)
		if err != nil {
			fmt.Printf("decodestream: %v\n", err)
		}
	}()
	tasks.Add(1)
	go func() {
		defer tasks.Done()
		err := l.DecodeStream(logs, "stderr", prefix)
		if err != nil {
			fmt.Printf("decodestream: %v\n", err)
		}
	}()
	tasks.Add(1)
	tasks.Wait()
	fmt.Println("Stopped logging", name)
	return nil
}

func (l *logForwarder) getAllLogs() error {
	dockerBackend := libswarm.AsClient(DockerClient())
	for host := range l.dockerInstances {
		b, err := dockerBackend.Spawn(host)
		if err != nil {
			fmt.Errorf("Could not spawn %s", host)
		}
		backend := libswarm.AsClient(b)
		names, err := backend.Ls()
		if err != nil {
			return err
		}
		fmt.Println("Getting logs", backend, names) // DEBUG
		for _, name := range names {
			go l.getContainerLog(backend, host, name)
		}
	}
	return nil
}

func (l *logForwarder) DecodeStream(src libswarm.Receiver, tag string, prefix []string) error {
	dst := libswarm.AsClient(l.service)
	for {
		msg, err := src.Receive(libswarm.Ret)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if tag == msg.Args[0] {
			var logTag string
			switch tag {
			case "stdout":
				logTag = "INFO"
			case "stderr":
				logTag = "ERROR"
			}
			logEntry := fmt.Sprintf("%s\t%s\t%s\t%s", time.Now(), strings.Join(prefix, "\t"), logTag, msg.Args[1])
			if err := dst.Log(logEntry); err != nil {
				return err
			}
		}
	}
}
