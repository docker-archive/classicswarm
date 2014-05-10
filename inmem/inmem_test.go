package inmem

import (
	"github.com/dotcloud/docker/pkg/testutils"
	"testing"
)

func TestModes(t *testing.T) {
	if R == W {
		t.Fatalf("0")
	}
	if R == 0 {
		t.Fatalf("0")
	}
	if W == 0 {
		t.Fatalf("0")
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
			if msg.Data != "hello world" {
				t.Fatalf("%#v", *msg)
			}
			if msg.Name != "print" {
				t.Fatalf("%#v", *msg)
			}
			if len(msg.Args) != 0 {
				t.Fatalf("%#v", *msg)
			}
			if in != nil {
				t.Fatalf("%#v", in)
			}
			if out != nil {
				t.Fatalf("%#v", out)
			}
		}()
		in, out, err := w.Send(&Message{Name: "print", Data: "hello world"}, 0)
		if err != nil {
			t.Fatal(err)
		}
		if in != nil {
			t.Fatalf("%#v", in)
		}
		if out != nil {
			t.Fatalf("%#v", out)
		}
	})
}

func TestSendReply(t *testing.T) {
	r, w := Pipe()
	defer r.Close()
	defer w.Close()
	testutils.Timeout(t, func() {
		// Send
		go func() {
			// Send a message with mode=R
			in, out, err := w.Send(&Message{Data: "this is the request"}, R)
			if err != nil {
				t.Fatal(err)
			}
			if out != nil {
				t.Fatalf("%#v", out)
			}
			if in == nil {
				t.Fatalf("%#v", in)
			}
			// Read for a reply
			resp, _, _, err := in.Receive(0)
			if err != nil {
				t.Fatal(err)
			}
			if resp.Data != "this is the reply" {
				t.Fatalf("%#v", resp)
			}
		}()
		// Receive a message with mode=W
		msg, in, out, err := r.Receive(W)
		if err != nil {
			t.Fatal(err)
		}
		if msg.Data != "this is the request" {
			t.Fatalf("%#v", msg)
		}
		if out == nil {
			t.Fatalf("%#v", out)
		}
		if in != nil {
			t.Fatalf("%#v", in)
		}
		// Send a reply
		_, _, err = out.Send(&Message{Data: "this is the reply"}, 0)
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
			in, out, err := w.Send(&Message{Data: "this is the request"}, W)
			if err != nil {
				t.Fatal(err)
			}
			if out == nil {
				t.Fatalf("%#v", out)
			}
			if in != nil {
				t.Fatalf("%#v", in)
			}
			// Send a nested message
			_, _, err = out.Send(&Message{Data: "this is the nested message"}, 0)
			if err != nil {
				t.Fatal(err)
			}
		}()
		// Receive a message with mode=R
		msg, in, out, err := r.Receive(R)
		if err != nil {
			t.Fatal(err)
		}
		if msg.Data != "this is the request" {
			t.Fatalf("%#v", msg)
		}
		if out != nil {
			t.Fatalf("%#v", out)
		}
		if in == nil {
			t.Fatalf("%#v", in)
		}
		// Read for a nested message
		nested, _, _, err := in.Receive(0)
		if err != nil {
			t.Fatal(err)
		}
		if nested.Data != "this is the nested message" {
			t.Fatalf("%#v", nested)
		}
	})
}
