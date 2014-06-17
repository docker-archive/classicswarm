package beam

import (
	"fmt"
	"io"
)

func EncodeStream(sender Sender, reader io.Reader, tag string) {
	chunk := make([]byte, 4096)
	for {
		n, err := reader.Read(chunk)
		if n > 0 {
			sender.Send(&Message{Verb: Log, Args: []string{tag, string(chunk[0:n])}})
		}
		if err != nil {
			message := fmt.Sprintf("Error reading from stream: %v", err)
			sender.Send(&Message{Verb: Error, Args: []string{message}})
			break
		}
	}
}

func DecodeStream(dst io.Writer, src Receiver, tag string) error {
	for {
		msg, err := src.Receive(Ret)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if tag == msg.Args[0] {
			if _, err := dst.Write([]byte(msg.Args[1])); err != nil {
				return err
			}
		}
	}
}
