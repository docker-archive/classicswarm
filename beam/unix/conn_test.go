package unix

import (
	"github.com/docker/beam"
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
			msg, in, out, err := r.Receive(0)
			if err != nil {
				t.Fatal(err)
			}
			if msg.Name != "print" {
				t.Fatalf("%#v", *msg)
			}
			if msg.Args[0] != "hello world" {
				t.Fatalf("%#v", *msg)
			}
			if in != nil && out != nil {
				t.Fatal("Unexpected return value")
			}
		}()
		_, _, err := w.Send(&beam.Message{Name: "print", Args: []string{"hello world"}}, 0)
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
			in, out, err := w.Send(&beam.Message{Args: []string{"this is the request"}}, beam.R)
			if err != nil {
				t.Fatal(err)
			}
			assertMode(t, in, out, beam.R)
			// Read for a reply
			resp, _, _, err := in.Receive(0)
			if err != nil {
				t.Fatal(err)
			}
			if resp.Args[0] != "this is the reply" {
				t.Fatalf("%#v", resp)
			}
		}()
		// Receive a message with mode=W
		msg, in, out, err := r.Receive(beam.W)
		if err != nil {
			t.Fatal(err)
		}
		if msg.Args[0] != "this is the request" {
			t.Fatalf("%#v", msg)
		}
		assertMode(t, in, out, beam.W)
		// Send a reply
		_, _, err = out.Send(&beam.Message{Args: []string{"this is the reply"}}, 0)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestSendNested(t *testing.T) {
	r, w, err := Pair()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	testutils.Timeout(t, func() {
		// Send
		go func() {
			// Send a message with mode=W
			in, out, err := w.Send(&beam.Message{Args: []string{"this is the request"}}, beam.W)
			if err != nil {
				t.Fatal(err)
			}
			assertMode(t, in, out, beam.W)
			// Send a nested message
			_, _, err = out.Send(&beam.Message{Args: []string{"this is the nested message"}}, 0)
			if err != nil {
				t.Fatal(err)
			}
		}()
		// Receive a message with mode=R
		msg, in, out, err := r.Receive(beam.R)
		if err != nil {
			t.Fatal(err)
		}
		if msg.Args[0] != "this is the request" {
			t.Fatalf("%#v", msg)
		}
		assertMode(t, in, out, beam.R)
		// Read for a nested message
		nested, _, _, err := in.Receive(0)
		if err != nil {
			t.Fatal(err)
		}
		if nested.Args[0] != "this is the nested message" {
			t.Fatalf("%#v", nested)
		}
	})
}

// FIXME: duplicate from inmem/inmem_test.go
// assertMode verifies that the values of r and w match
// mode.
// If mode has the R bit set, r must be non-nil. Otherwise it must be nil.
// If mode has the W bit set, w must be non-nil. Otherwise it must be nil.
//
// If any of these conditions are not met, t.Fatal is called and the active
// test fails.
func assertMode(t *testing.T, r beam.Receiver, w beam.Sender, mode int) {
	// If mode has the R bit set, r must be non-nil
	if mode&beam.R != 0 {
		if r == nil {
			t.Fatalf("should be non-nil: %#v", r)
		}
		// Otherwise it must be nil.
	} else {
		if r != nil {
			t.Fatalf("should be nil: %#v", r)
		}
	}
	// If mode has the W bit set, w must be non-nil
	if mode&beam.W != 0 {
		if w == nil {
			t.Fatalf("should be non-nil: %#v", w)
		}
		// Otherwise it must be nil.
	} else {
		if w != nil {
			t.Fatalf("should be nil: %#v", w)
		}
	}
}
