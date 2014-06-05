package unix

import (
	"github.com/docker/libswarm/beam"
	"github.com/dotcloud/docker/pkg/testutils"
	"testing"
)

func TestPair(t *testing.T) {
	r, w, err := Pair()
	if err != nil {
		t.Fatal("Unexpected error")
	}
	defer r.Close()
	defer w.Close()
	testutils.Timeout(t, func() {
		go func() {
			msg, err := r.Receive(0)
			if err != nil {
				t.Fatal(err)
			}
			if msg.Verb != beam.Log {
				t.Fatalf("%#v", *msg)
			}
			if msg.Args[0] != "hello world" {
				t.Fatalf("%#v", *msg)
			}
		}()
		_, err := w.Send(&beam.Message{Verb: beam.Log, Args: []string{"hello world"}})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestSendReply(t *testing.T) {
	r, w, err := Pair()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	testutils.Timeout(t, func() {
		// Send
		go func() {
			// Send a message with mode=R
			ret, err := w.Send(&beam.Message{Args: []string{"this is the request"}, Ret: beam.RetPipe})
			if err != nil {
				t.Fatal(err)
			}
			// Read for a reply
			msg, err := ret.Receive(0)
			if err != nil {
				t.Fatal(err)
			}
			if msg.Args[0] != "this is the reply" {
				t.Fatalf("%#v", msg)
			}
		}()
		// Receive a message with mode=W
		msg, err := r.Receive(beam.Ret)
		if err != nil {
			t.Fatal(err)
		}
		if msg.Args[0] != "this is the request" {
			t.Fatalf("%#v", msg)
		}
		// Send a reply
		_, err = msg.Ret.Send(&beam.Message{Args: []string{"this is the reply"}})
		if err != nil {
			t.Fatal(err)
		}
	})
}
