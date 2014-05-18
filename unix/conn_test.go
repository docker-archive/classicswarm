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
			if in != nil  && out != nil {
				t.Fatal("Unexpected return value")
			}
		}()
		_, _, err := w.Send(&beam.Message{Name: "print", Args: []string{"hello world"}}, 0)
		if err != nil {
			t.Fatal(err)
		}
	})
}
