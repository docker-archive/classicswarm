package inmem

import (
	"testing"
	"time"
)

func TestSimpleSend(t *testing.T) {
	r, w := Pipe()
	onTimeout := time.After(100 * time.Millisecond)
	onRcv := make(chan bool)
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
		close(onRcv)
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
	select {
	case <-onTimeout:
		t.Fatalf("timeout")
	case <-onRcv:
	}
}
