package utils

import (
	"github.com/docker/beam"
	"github.com/dotcloud/docker/pkg/testutils"
	"testing"
)

func TestHubSendEmpty(t *testing.T) {
	hub := NewHub()
	// Send to empty hub should silently drop
	if r, w, err := hub.Send(&beam.Message{"hello", nil}, beam.R|beam.W); err != nil {
		t.Fatal(err)
	} else if r != nil {
		t.Fatalf("%#v", r)
	} else if w != nil {
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
		in, _, err := hub.Send(&beam.Message{"register", nil}, beam.R)
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			if _, _, err := hub.Send(&beam.Message{"hello", nil}, 0); err != nil {
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
