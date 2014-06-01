package utils

import (
	"github.com/docker/beam"
	"github.com/dotcloud/docker/pkg/testutils"
	"testing"
)

func TestHubSendEmpty(t *testing.T) {
	hub := NewHub()
	// Send to empty hub should silently drop
	r, w, err := hub.Send(&beam.Message{Name: "hello", Args: nil}, beam.R|beam.W)
	// Send must not return an error
	if err != nil {
		t.Fatal(err)
	}
	// We set beam.R, so a valid receiver must be returned
	if r == nil {
		t.Fatalf("%#v", r)
	}
	// We set beam.W, so a valid receiver must be returned
	if w == nil {
		t.Fatalf("%#v", w)
	}
}

type CountSender int

func (s *CountSender) Send(msg *beam.Message, mode int) (beam.Receiver, beam.Sender, error) {
	(*s)++
	return nil, nil, nil
}

func TestHubSendOneHandler(t *testing.T) {
	hub := NewHub()
	defer hub.Close()
	testutils.Timeout(t, func() {
		in, _, err := hub.Send(&beam.Message{Name: "register", Args: nil}, beam.R)
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			if _, _, err := hub.Send(&beam.Message{Name: "hello", Args: nil}, 0); err != nil {
				t.Fatal(err)
			}
		}()
		msg, _, _, err := in.Receive(0)
		if err != nil {
			t.Fatal(err)
		}
		if msg.Name != "hello" {
			t.Fatalf("%#v", msg)
		}
	})
}
