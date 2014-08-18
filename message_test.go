package libswarm

import (
	"github.com/docker/libswarm/iowrapper"

	"io"
	"io/ioutil"
	"reflect"
	"testing"
)

func TestVerbArgs(t *testing.T) {
	receiver, sender := Pipe()
	sentMsg := &Message{Verb: Set, Args: []string{"foo", "bar"}}

	go sender.Send(sentMsg)

	receivedMsg, err := receiver.Receive(0)
	if err != nil {
		t.Fatal(err)
	}
	if receivedMsg == nil {
		t.Fatalf("Didn't get a message")
	}
	if receivedMsg.Verb != sentMsg.Verb {
		t.Fatalf("Expected %s, got %s", sentMsg.Verb.String(), receivedMsg.Verb.String())
	}
	if !reflect.DeepEqual(receivedMsg.Args, sentMsg.Args) {
		t.Fatalf("Expected %#v, got %#v", sentMsg.Args, receivedMsg.Args)
	}
}

func TestReturnChannel(t *testing.T) {
	receiver, sender := Pipe()

	replyReceiver, replySender := Pipe()

	go func() {
		receivedMsg, err := receiver.Receive(Ret)
		if err != nil {
			t.Fatal(err)
		}
		if receivedMsg == nil {
			t.Fatalf("Didn't get a message")
		}
		receivedMsg.Ret.Send(&Message{Verb: Set})
	}()

	if _, err := sender.Send(&Message{Verb: Get, Ret: replySender}); err != nil {
		t.Fatal(err)
	}

	reply, err := replyReceiver.Receive(0)
	if err != nil {
		t.Fatal(err)
	}
	if reply == nil {
		t.Fatalf("Didn't get a reply")
	}
	if reply.Verb != Set {
		t.Fatalf("Expected Set, got %s", reply.Verb.String())
	}
}

func TestRetPipe(t *testing.T) {
	receiver, sender := Pipe()

	go func() {
		receivedMsg, err := receiver.Receive(Ret)
		if err != nil {
			t.Fatal(err)
		}
		if receivedMsg == nil {
			t.Fatalf("Didn't get a message")
		}
		receivedMsg.Ret.Send(&Message{Verb: Set})
		receivedMsg.Ret.Close()
	}()

	replyReceiver, err := sender.Send(&Message{Verb: Get, Ret: RetPipe})
	if err != nil {
		t.Fatal(err)
	}

	reply, err := replyReceiver.Receive(0)
	if err != nil {
		t.Fatal(err)
	}
	if reply == nil {
		t.Fatalf("Didn't get a reply")
	}
	if reply.Verb != Set {
		t.Fatalf("Expected Set, got %s", reply.Verb.String())
	}
}

func TestAttachment(t *testing.T) {
	expectedContents := "hello world\n"
	r, w := io.Pipe()

	receiver, sender := Pipe()

	go func() {
		msg, err := receiver.Receive(Ret)
		if err != nil {
			t.Fatal(err)
		}

		msg.Ret.Send(&Message{Verb: Connect, Att: iowrapper.Wrap(r)})
	}()

	ret, err := sender.Send(&Message{Verb: Connect, Ret: RetPipe})
	if err != nil {
		t.Fatal(err)
	}

	reply, err := ret.Receive(0)
	if err != nil {
		t.Fatal(err)
	}
	if reply == nil {
		t.Fatalf("Didn't get a reply")
	}
	if reply.Att == nil {
		t.Fatalf("Didn't get an attachment back")
	}

	go func() {
		w.Write([]byte(expectedContents))
		w.Close()
	}()
	contents, err := ioutil.ReadAll(reply.Att.(io.Reader))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != expectedContents {
		t.Fatalf("Expected %#v, got %#v", expectedContents, string(contents))
	}
}
