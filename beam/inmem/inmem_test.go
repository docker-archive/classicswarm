package inmem

import (
	"fmt"
	"github.com/docker/libswarm/beam"
	"github.com/dotcloud/docker/pkg/testutils"
	"io/ioutil"
	"os"
	"testing"
)

func TestReceiveW(t *testing.T) {
	r, w := Pipe()
	go func() {
		w.Send(&beam.Message{Name: "hello"}, 0)
	}()
	_, _, ww, err := r.Receive(beam.W)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ww.Send(&beam.Message{Name: "this better not crash"}, 0); err != nil {
		t.Fatal(err)
	}
}

func TestSimpleSend(t *testing.T) {
	r, w := Pipe()
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
			assertMode(t, in, out, 0)
		}()
		in, out, err := w.Send(&beam.Message{Name: "print", Args: []string{"hello world"}}, 0)
		if err != nil {
			t.Fatal(err)
		}
		assertMode(t, in, out, 0)
	})
}

// assertMode verifies that the values of r and w match
// mode.
// If mode has the R bit set, r must be non-nil.
// If mode has the W bit set, w must be non-nil.
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
	}
	// If mode has the W bit set, w must be non-nil
	if mode&beam.W != 0 {
		if w == nil {
			t.Fatalf("should be non-nil: %#v", w)
		}
		// Otherwise it must be nil.
	}
}

func TestSendReply(t *testing.T) {
	r, w := Pipe()
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
	r, w := Pipe()
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

func TestSendFile(t *testing.T) {
	r, w := Pipe()
	defer r.Close()
	defer w.Close()
	tmp, err := ioutil.TempFile("", "beam-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp.Name())
	fmt.Fprintf(tmp, "hello world\n")
	tmp.Sync()
	tmp.Seek(0, 0)
	testutils.Timeout(t, func() {
		go func() {
			_, _, err := w.Send(&beam.Message{"file", []string{"path=" + tmp.Name()}, tmp}, 0)
			if err != nil {
				t.Fatal(err)
			}
		}()
		msg, _, _, err := r.Receive(0)
		if err != nil {
			t.Fatal(err)
		}
		if msg.Name != "file" {
			t.Fatalf("%#v", msg)
		}
		if msg.Args[0] != "path="+tmp.Name() {
			t.Fatalf("%#v", msg)
		}
		txt, err := ioutil.ReadAll(msg.Att)
		if err != nil {
			t.Fatal(err)
		}
		if string(txt) != "hello world\n" {
			t.Fatalf("%s\n", txt)
		}
	})
}
