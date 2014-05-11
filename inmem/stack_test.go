package inmem

import (
	"github.com/dotcloud/docker/pkg/testutils"
	"strings"
	"testing"
)

func TestSendStack(t *testing.T) {
	r, w := Pipe()
	defer r.Close()
	defer w.Close()
	s := NewStackSender()
	s.Add(w)
	testutils.Timeout(t, func() {
		go func() {
			msg, _, _, err := r.Receive(0)
			if err != nil {
				t.Fatal(err)
			}
			if msg.Name != "hello" {
				t.Fatalf("%#v", msg)
			}
			if strings.Join(msg.Args, " ") != "wonderful world" {
				t.Fatalf("%#v", msg)
			}
		}()
		_, _, err := s.Send(&Message{"hello", []string{"wonderful", "world"}, nil}, 0)
		if err != nil {
			t.Fatal(err)
		}
	})
}
