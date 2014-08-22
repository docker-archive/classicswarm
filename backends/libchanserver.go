package backends

import (
	"fmt"
	"io"
	"net"
	"net/url"

	"github.com/docker/libchan/http2"
	"github.com/docker/libswarm"
	"github.com/docker/libswarm/utils"
)

type libchanServer struct {
	out libswarm.Sender
	*libswarm.Server
}

type libchanHttp2Server struct {
	*libchanServer
}

func LibchanServer() libswarm.Sender {
	backend := libswarm.NewServer()
	backend.OnVerb(libswarm.Spawn, libswarm.Handler(func(ctx *libswarm.Message) error {
		s := &libchanServer{Server: libswarm.NewServer()}
		instance := utils.Task(func(in libswarm.Receiver, out libswarm.Sender) {

			s.out = out

			url := "http2://localhost:9000"
			if len(ctx.Args) > 0 {
				url = ctx.Args[0]
			}
			s.Catchall(libswarm.Handler(s.catchall))

			err := s.listenAndServe(url)
			if err != nil {
				fmt.Printf("libchanserver: %v", err)
			}
		})

		_, err := ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: instance})
		return err
	}))
	return backend
}

func (s *libchanServer) catchall(msg *libswarm.Message) error {
	forwardedMsg := msg
	//forwardedMsg.Ret = libswarm.RetPipe
	_, err := s.out.Send(forwardedMsg)
	if err != nil {
		return err
	}

	//if inbound, err := s.out.Send(forwardedMsg); err != nil {
	//	return fmt.Errorf("libchanserver: Failed to forward msg. Reason: %v\n", err)
	//} else if inbound == nil {
	//	return fmt.Errorf("libchanserver: Inbound channel nil.\n")
	//} else {
	//	for {
	//		// Relay all messages returned until the inbound channel is empty (EOF)
	//		var reply *libswarm.Message
	//		if reply, err = inbound.Receive(0); err != nil {
	//			if err == io.EOF {
	//				// EOF is expected
	//				err = nil
	//			}
	//			break
	//		}

	//		// Forward the message back downstream
	//		if _, err = msg.Ret.Send(reply); err != nil {
	//			return fmt.Errorf("libchanserver: Failed to forward msg. Reason: %v\n", err)
	//		}
	//	}
	//}
	return nil
}

func (s *libchanServer) listenAndServe(urlStr string) error {
	parsedUrl, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	switch parsedUrl.Scheme {
	case "http2":
		http2srv := &libchanHttp2Server{s}
		return http2srv.listenAndServe(parsedUrl.Host)
	default:
		return fmt.Errorf("libchanserver: Protocol not implemented - %s", parsedUrl.Scheme)
	}
}

func (s *libchanHttp2Server) listenAndServe(host string) error {
	listener, err := net.Listen("tcp", host)
	if err != nil {
		return err
	}

	session, err := http2.NewListenSession(listener, s.auth)
	if err != nil {
		return err
	}
	defer func() {
		session.Close()
	}()
	for {
		stream, err := session.AcceptSession()

		if err != nil {
			return err
		}
		go s.handleConn(stream)
	}
}

func (s *libchanHttp2Server) handleConn(stream *http2.StreamSession) {
	defer func() {
		stream.Close()
	}()
	r, err := stream.ReceiverWait()
	if err != nil {
		fmt.Errorf("libchanserver http2: %v", err)
		return
	}
	remote := libswarm.WrapReceiver(r)
	if err = s.Copy(remote); err != nil {
		fmt.Errorf("libchanserver http2: %v", err)
	}
}

func (s *libchanServer) Copy(src libswarm.Receiver) error {
	for {
		msg, err := src.Receive(0)
		if err == io.EOF {
			err = nil
		}
		if err != nil {
			return fmt.Errorf("libchanserver http2: %v", err)
		}
		if err = s.catchall(msg); err != nil {
			return fmt.Errorf("libchanserver http2: %v", err)
		}
	}
	return nil
}

func (s *libchanHttp2Server) auth(conn net.Conn) error {
	fmt.Printf("Verifying credentials...")
	fmt.Println("Much secure!")
	return nil
}
