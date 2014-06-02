package beam

import (
	"fmt"
	"github.com/dotcloud/docker/pkg/testutils"
	"io/ioutil"
	"os"
	"testing"
)

func TestInmemRetPipe(t *testing.T) {
	r, w := Pipe()
	defer r.Close()
	defer w.Close()
	wait := make(chan struct{})
	go func() {
		ret, err := w.Send(&Message{Name: "hello", Ret: RetPipe})
		if err != nil {
			t.Fatal(err)
		}
		msg, err := ret.Receive(0)
		if err != nil {
			t.Fatal(err)
		}
		if msg.Name != "this better not crash" {
			t.Fatalf("%#v", msg)
		}
		close(wait)
	}()
	msg, err := r.Receive(Ret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := msg.Ret.Send(&Message{Name: "this better not crash"}); err != nil {
		t.Fatal(err)
	}
	<-wait
}

func TestSimpleSend(t *testing.T) {
	r, w := Pipe()
	defer r.Close()
	defer w.Close()
	testutils.Timeout(t, func() {
		go func() {
			msg, err := r.Receive(0)
			if err != nil {
				t.Fatal(err)
			}
			if msg.Name != "print" {
				t.Fatalf("%#v", *msg)
			}
			if msg.Args[0] != "hello world" {
				t.Fatalf("%#v", *msg)
			}
		}()
		if _, err := w.Send(&Message{Name: "print", Args: []string{"hello world"}}); err != nil {
			t.Fatal(err)
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
			ret, err := w.Send(&Message{Args: []string{"this is the request"}, Ret: RetPipe})
			if err != nil {
				t.Fatal(err)
			}
			if ret == nil {
				t.Fatalf("ret = nil\n")
			}
			// Read for a reply
			msg, err := ret.Receive(0)
			if err != nil {
				t.Fatal(err)
			}
			if msg.Args[0] != "this is the reply" {
				t.Fatalf("%#v", msg)
			}
		}()
		// Receive a message with mode=Ret
		msg, err := r.Receive(Ret)
		if err != nil {
			t.Fatal(err)
		}
		if msg.Args[0] != "this is the request" {
			t.Fatalf("%#v", msg)
		}
		if msg.Ret == nil {
			t.Fatalf("%#v", msg)
		}
		// Send a reply
		_, err = msg.Ret.Send(&Message{Args: []string{"this is the reply"}})
		if err != nil {
			t.Fatal(err)
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
			_, err := w.Send(&Message{Name: "file", Args: []string{"path=" + tmp.Name()}, Att: tmp})
			if err != nil {
				t.Fatal(err)
			}
		}()
		msg, err := r.Receive(0)
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
