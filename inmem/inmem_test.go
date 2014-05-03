package inmem

import (
	"testing"
	"time"
)

func TestSimpleSend(t *testing.T) {
	a, b := Pipe()
	defer a.CloseWrite()
	defer b.CloseWrite()
	onTimeout := time.After(100 * time.Millisecond)
	onRcv := make(chan bool)
	go func() {
		msg, h, err := b.Receive(0)
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
		if h != nil {
			t.Fatalf("%#v", h)
		}
		close(onRcv)
	}()
	h, err := a.Send(&Message{Name:"print", Data: "hello world"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if h != nil {
		t.Fatalf("%#v", h)
	}
	select {
		case <-onTimeout: t.Fatalf("timeout")
		case <-onRcv:
	}
}
