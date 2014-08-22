package backends

import (
	"fmt"
	"net"
	"net/url"

	"github.com/docker/libchan/http2"
	"github.com/docker/libswarm"
)

type libchanClient struct {
	*libswarm.Server
	url    string
	remote libswarm.Sender
}

func LibchanClient() libswarm.Sender {
	backend := libswarm.NewServer()
	backend.OnSpawn(func(cmd ...string) (libswarm.Sender, error) {

		s := &libchanClient{Server: libswarm.NewServer()}

		s.url = "http2://localhost:9000"
		if len(cmd) > 0 {
			s.url = cmd[0]
		}

		s.OnAttach(s.attach)
		s.OnVerb(libswarm.Start, libswarm.Handler(s.start))
		s.Catchall(libswarm.Handler(s.catchall))
		return s.Server, nil
	})
	return backend
}

func (s *libchanClient) attach(name string, ret libswarm.Sender) error {
	ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: s})
	<-make(chan struct{})
	return nil
}

func (s *libchanClient) start(msg *libswarm.Message) error {
	remote, err := s.dial()
	if err != nil {
		return err
	}
	forwardedMsg := msg
	remote.Send(forwardedMsg)

	return nil
}

func (s *libchanClient) catchall(msg *libswarm.Message) (err error) {
	forwardedMsg := msg
	s.remote.Send(forwardedMsg)

	return nil
}

func (s *libchanClient) dial() (libswarm.Sender, error) {
	parsedUrl, err := url.Parse(s.url)
	if err != nil {
		return nil, err
	}

	switch parsedUrl.Scheme {
	case "http2":
		return s.dialHttp2(parsedUrl.Host)
	default:
		return nil, fmt.Errorf("libchanClient: Protocol not implemented - %s", parsedUrl.Scheme)
	}
}

func (s *libchanClient) dialHttp2(host string) (libswarm.Sender, error) {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, fmt.Errorf("libchanClient http2: Error dialing %v - %v", host, err)
	}

	session, err := http2.NewStreamSession(conn)
	if err != nil {
		return nil, fmt.Errorf("libchanClient http2: Error establishing spdy stream to %v - %v", host, err)
	}

	sender, err := session.NewSender()
	return libswarm.WrapSender(sender), err

}
