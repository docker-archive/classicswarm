package utils

import (
	"github.com/docker/libswarm/beam"
	"github.com/dotcloud/docker/pkg/testutils"
	"testing"
)

func TestHubSendEmpty(t *testing.T) {
	hub := NewHub()
	// Send to empty hub should silently drop
	ret, err := hub.Send(&beam.Message{Name: "hello", Args: nil, Ret: beam.RetPipe})
	// Send must not return an error
	if err != nil {
		t.Fatal(err)
	}
	// We set beam.R, so a valid return pipe must be returned
	if ret == nil {
		t.Fatalf("%#v", ret)
	}
}

type CountSender int

func (s *CountSender) Send(msg *beam.Message) (beam.Receiver, error) {
	(*s)++
	return nil, nil
}

func TestHubSendOneHandler(t *testing.T) {
	hub := NewHub()
	defer hub.Close()
	testutils.Timeout(t, func() {
		handlerIn, err := hub.Send(&beam.Message{Name: "register", Args: nil, Ret: beam.RetPipe})
		if err != nil {
			t.Fatal(err)
		}
		ack, err := handlerIn.Receive(beam.Ret)
		if err != nil {
			t.Fatal(err)
		}
		if ack.Name != "register" {
			t.Fatalf("%#v", err)
		}
		handlerOut := ack.Ret
		if handlerOut == nil {
			t.Fatalf("nil handler out")
		}
		go func() {
			if _, err := hub.Send(&beam.Message{Name: "hello", Args: nil}); err != nil {
				t.Fatal(err)
			}
		}()
		msg, err := handlerIn.Receive(0)
		if err != nil {
			t.Fatal(err)
		}
		if msg.Name != "hello" {
			t.Fatalf("%#v", msg)
		}
	})
}
