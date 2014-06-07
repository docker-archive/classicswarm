package http2

import (
	"github.com/docker/libswarm/beam"
	"io"
	"net"
	"testing"
)

func TestListenSession(t *testing.T) {
	listen := "localhost:7743"
	listener, listenErr := net.Listen("tcp", listen)
	if listenErr != nil {
		t.Fatalf("Error creating listener: %s", listenErr)
	}

	session, sessionErr := NewListenSession(listener, NoAuthenticator)
	if sessionErr != nil {
		t.Fatalf("Error creating session: %s", sessionErr)
	}

	go session.Serve()

	end := make(chan bool)
	go exerciseServer(t, listen, end)

	msg, msgErr := session.Receive(beam.Ret)
	if msgErr != nil {
		t.Fatalf("Error receiving message: %s", msgErr)
	}
	if msg.Att == nil {
		t.Fatalf("Error message missing attachment")
	}
	if msg.Verb != beam.Attach {
		t.Fatalf("Wrong verb\nActual: %s\nExpecting: %s", msg.Verb, beam.Attach)
	}

	receiver, sendErr := msg.Ret.Send(&beam.Message{Verb: beam.Ack})
	if sendErr != nil {
		t.Fatalf("Error sending return message: %s", sendErr)
	}
	_, ackErr := receiver.Receive(0)
	if ackErr == nil {
		t.Fatalf("No error receiving from message with no return pipe")
	}
	if ackErr != io.EOF {
		t.Fatalf("Unexpected error receiving from message: %s", ackErr)
	}

	<-end
	shutdownErr := session.Shutdown()
	if shutdownErr != nil {
		t.Fatalf("Error shutting down: %s", shutdownErr)
	}
}

func exerciseServer(t *testing.T, server string, endChan chan bool) {
	defer close(endChan)

	conn, connErr := net.Dial("tcp", server)
	if connErr != nil {
		t.Fatalf("Error dialing server: %s", connErr)
	}

	session, sessionErr := NewStreamSession(conn)
	if sessionErr != nil {
		t.Fatalf("Error creating session: %s", sessionErr)
	}

	receiver, sendErr := session.Send(&beam.Message{Verb: beam.Attach, Ret: beam.RetPipe})
	if sendErr != nil {
		t.Fatalf("Error sending message: %s", sendErr)
	}

	msg, receiveErr := receiver.Receive(beam.Ret)
	if receiveErr != nil {
		t.Fatalf("Error receiving message")
	}

	if msg.Verb != beam.Ack {
		t.Fatalf("Wrong verb\nActual: %s\nExpecting: %s", msg.Verb, beam.Ack)
	}
}
