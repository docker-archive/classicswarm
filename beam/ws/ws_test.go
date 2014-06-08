package ws

import (
	"github.com/docker/libswarm/beam"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServe(t *testing.T) {
	gotAck := make(chan bool)
	u := &Upgrader{}
	server := httptest.NewServer(Serve(u, func(r beam.Receiver) {
		msg, msgErr := r.Receive(beam.Ret)
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

		<-gotAck
	}))

	wsConn, _, err := websocket.DefaultDialer.Dial(strings.Replace(server.URL, "http://", "ws://", 1), http.Header{"Origin": {server.URL}})
	if err != nil {
		t.Fatal(err)
	}
	sender, senderErr := NewSender(wsConn)
	if senderErr != nil {
		t.Fatalf("Error creating sender: %s", senderErr)
	}

	receiver, sendErr := sender.Send(&beam.Message{Verb: beam.Attach, Ret: beam.RetPipe})
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

	gotAck <- true

	shutdownErr := sender.Close()
	if shutdownErr != nil && !strings.Contains(shutdownErr.Error(), "broken pipe") {
		t.Fatalf("Error closing: %s", shutdownErr)
	}
}
