package utils

import (
	"github.com/docker/libswarm/beam"
	"github.com/docker/libswarm/beam/unix"
	"github.com/dotcloud/docker/pkg/testutils"
	"strings"
	"testing"
)

func TestStackWithPipe(t *testing.T) {
	r, w := beam.Pipe()
	defer r.Close()
	defer w.Close()
	s := NewStackSender()
	s.Add(w)
	testutils.Timeout(t, func() {
		go func() {
			msg, err := r.Receive(0)
			if err != nil {
				t.Fatal(err)
			}
			if msg.Verb != beam.Log {
				t.Fatalf("%#v", msg)
			}
			if strings.Join(msg.Args, " ") != "wonderful world" {
				t.Fatalf("%#v", msg)
			}
		}()
		_, err := s.Send(&beam.Message{Verb: beam.Log, Args: []string{"wonderful", "world"}})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestStackWithPair(t *testing.T) {
	r, w, err := unix.Pair()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	s := NewStackSender()
	s.Add(w)
	testutils.Timeout(t, func() {
		go func() {
			msg, err := r.Receive(0)
			if err != nil {
				t.Fatal(err)
			}
			if msg.Verb != beam.Log {
				t.Fatalf("%#v", msg)
			}
			if strings.Join(msg.Args, " ") != "wonderful world" {
				t.Fatalf("%#v", msg)
			}
		}()
		_, err := s.Send(&beam.Message{Verb: beam.Log, Args: []string{"wonderful", "world"}})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestStackLen(t *testing.T) {
	s := NewStackSender()
	if s.Len() != 0 {
		t.Fatalf("empty StackSender has length %d", s.Len())
	}
}

func TestStackAdd(t *testing.T) {
	s := NewStackSender()
	a := Buffer{}
	beforeA := s.Add(&a)
	// Add on an empty StackSender should return an empty StackSender
	if beforeA.Len() != 0 {
		t.Fatalf("%s has %d elements", beforeA, beforeA.Len())
	}
	if s.Len() != 1 {
		t.Fatalf("%#v", beforeA)
	}
	// Add a 2nd element
	b := Buffer{}
	beforeB := s.Add(&b)
	if beforeB.Len() != 1 {
		t.Fatalf("%#v", beforeA)
	}
	if s.Len() != 2 {
		t.Fatalf("%#v", beforeA)
	}
	s.Send(&beam.Message{Verb: beam.Log, Args: []string{"for b"}})
	beforeB.Send(&beam.Message{Verb: beam.Log, Args: []string{"for a"}})
	beforeA.Send(&beam.Message{Verb: beam.Log, Args: []string{"for nobody"}})
	if len(a) != 1 {
		t.Fatalf("%#v", a)
	}
	if len(b) != 1 {
		t.Fatalf("%#v", b)
	}
}

// Misbehaving backends must be removed
func TestStackAddBad(t *testing.T) {
	s := NewStackSender()
	buf := Buffer{}
	s.Add(&buf)
	r, w := beam.Pipe()
	s.Add(w)
	if s.Len() != 2 {
		t.Fatalf("%#v", s)
	}
	r.Close()
	if _, err := s.Send(&beam.Message{Verb: beam.Log, Args: []string{"for the buffer"}}); err != nil {
		t.Fatal(err)
	}
	if s.Len() != 1 {
		t.Fatalf("%#v")
	}
	if len(buf) != 1 {
		t.Fatalf("%#v", buf)
	}
	if buf[0].Args[0] != "for the buffer" {
		t.Fatalf("%#v", buf)
	}
}
