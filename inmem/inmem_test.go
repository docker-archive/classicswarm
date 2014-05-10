package inmem

import (
	"github.com/dotcloud/docker/pkg/testutils"
	"testing"
)

func TestSimpleSend(t *testing.T) {
	r, w := Pipe()
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
