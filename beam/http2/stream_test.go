package http2

import (
	//"bytes"
	"github.com/docker/libswarm/beam"
	//"github.com/docker/spdystream"
	"io"
	"net"
	"testing"
)

func TestBeamSession(t *testing.T) {
	end := make(chan bool)
	listen := "localhost:7543"
	server, serverErr := runServer(listen, t, end)
	if serverErr != nil {
		t.Fatalf("Error initializing server: %s", serverErr)
	}

	conn, connErr := net.Dial("tcp", listen)
	if connErr != nil {
		t.Fatalf("Error dialing server: %s", connErr)
	}

	sender, senderErr := NewStreamSession(conn)
	if senderErr != nil {
		t.Fatalf("Error creating sender: %s", senderErr)
	}

	// Ls interaction
	receiver, sendErr := sender.Send(&beam.Message{Verb: beam.Ls, Ret: beam.RetPipe})
	if sendErr != nil {
		t.Fatalf("Error sending beam message: %s", sendErr)
	}
	message, receiveErr := receiver.Receive(0)
	if receiveErr != nil {
		t.Fatalf("Error receiving beam message: %s", receiveErr)
	}
	if message.Verb != beam.Set {
		t.Errorf("Unexpected message name:\nActual: %s\nExpected: %s", message.Verb, beam.Ls.String())
	}
	if len(message.Args) != 3 {
		t.Fatalf("Unexpected args length\nActual: %d\nExpected: %d", len(message.Args), 3)
	}
	if message.Args[0] != "file1" {
		t.Errorf("Unexpected arg[0]\nActual: %s\nExpected: %s", message.Args[0], "file1")
	}
	if message.Args[1] != "file2" {
		t.Errorf("Unexpected arg[0]\nActual: %s\nExpected: %s", message.Args[1], "file2")
	}
	if message.Args[2] != string([]byte{0x00, 0x00, 0x00}) {
		t.Errorf("Unexpected arg[0]\nActual: %s\nExpected: %s", message.Args[2], []byte{0x00, 0x00, 0x00})
	}

	// Attach interactions
	receiver, sendErr = sender.Send(&beam.Message{Verb: beam.Attach, Ret: beam.RetPipe})
	if sendErr != nil {
		t.Fatalf("Error sending beam message: %s", sendErr)
	}
	message, receiveErr = receiver.Receive(beam.Ret)
	if receiveErr != nil {
		t.Fatalf("Error receiving beam message: %s", receiveErr)
	}
	if message.Verb != beam.Ack {
		t.Errorf("Unexpected message name:\nActual: %s\nExpected: %s", message.Verb, beam.Ack.String())
	}

	// TODO full connect interaction
	//if message.Att == nil {
	//	t.Fatalf("Missing attachment on message")
	//}

	//testBytes := []byte("Hello")
	//n, writeErr := message.Att.Write(testBytes)
	//if writeErr != nil {
	//	t.Fatalf("Error writing bytes: %s", writeErr)
	//}
	//if n != 5 {
	//	t.Fatalf("Unexpected number of bytes read:\nActual: %d\nExpected: 5", n)
	//}

	//buf := make([]byte, 10)
	//n, readErr := message.Att.Read(buf)
	//if readErr != nil {
	//	t.Fatalf("Error writing bytes: %s", readErr)
	//}
	//if n != 5 {
	//	t.Fatalf("Unexpected number of bytes read:\nActual: %d\nExpected: 5", n)
	//}
	//if bytes.Compare(buf[:n], testBytes) != 0 {
	//	t.Fatalf("Did not receive expected message:\nActual: %s\nExpectd: %s", buf, testBytes)
	//}

	closeErr := server.Close()
	if closeErr != nil {
		t.Fatalf("Error closing server: %s", closeErr)
	}

	closeErr = sender.Close()
	if closeErr != nil {
		t.Fatalf("Error closing sender: %s", closeErr)
	}
	<-end
}

func runServer(listen string, t *testing.T, endChan chan bool) (io.Closer, error) {
	listener, lErr := net.Listen("tcp", listen)
	if lErr != nil {
		return nil, lErr
	}

	session, sessionErr := NewListenSession(listener, NoAuthenticator)
	if sessionErr != nil {
		t.Fatalf("Error creating session: %s", sessionErr)
	}

	go session.Serve()

	go func() {
		defer close(endChan)
		// Ls exchange
		message, receiveErr := session.Receive(beam.Ret)
		if receiveErr != nil {
			t.Fatalf("Error receiving on server: %s", receiveErr)
		}
		if message.Verb != beam.Ls {
			t.Fatalf("Unexpected verb: %s", message.Verb)
		}
		receiver, sendErr := message.Ret.Send(&beam.Message{Verb: beam.Set, Args: []string{"file1", "file2", string([]byte{0x00, 0x00, 0x00})}})
		if sendErr != nil {
			t.Fatalf("Error sending set message: %s", sendErr)
		}
		_, receiveErr = receiver.Receive(0)
		if receiveErr == nil {
			t.Fatalf("No error received from empty receiver")
		}
		if receiveErr != io.EOF {
			t.Fatalf("Expected error from empty receiver: %s", receiveErr)
		}

		// Connect exchange
		message, receiveErr = session.Receive(beam.Ret)
		if receiveErr != nil {
			t.Fatalf("Error receiving on server: %s", receiveErr)
		}
		if message.Verb != beam.Attach {
			t.Fatalf("Unexpected verb: %s", message.Verb)
		}
		receiver, sendErr = message.Ret.Send(&beam.Message{Verb: beam.Ack})
		if sendErr != nil {
			t.Fatalf("Error sending set message: %s", sendErr)
		}

		// TODO full connect interaction

	}()

	return listener, nil
}
