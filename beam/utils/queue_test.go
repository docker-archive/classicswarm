package utils

import (
	"github.com/docker/libswarm/beam"
	"github.com/docker/libswarm/beam/inmem"
	"testing"
)

func TestSendRet(t *testing.T) {
	r, w := inmem.Pipe()
	defer r.Close()
	defer w.Close()
	q := NewQueue(w, 1)
	defer q.Close()
	ret, err := q.Send(&beam.Message{Name: "ping", Ret: beam.RetPipe})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		ping, err := r.Receive(beam.Ret)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ping.Ret.Send(&beam.Message{Name: "pong"}); err != nil {
			t.Fatal(err)
		}
	}()
	pong, err := ret.Receive(0)
	if err != nil {
		t.Fatal(err)
	}
	if pong.Name != "pong" {
		t.Fatal(err)
	}
}

func TestSendClose(t *testing.T) {
	q := NewQueue(beam.NopSender{}, 1)
	q.Send(&beam.Message{Name: "hello"})
	q.Close()
	if _, err := q.Send(&beam.Message{Name: "again"}); err == nil {
		t.Fatal("send on closed queue should return an error")
	}
}
