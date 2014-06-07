package http2

import (
	"encoding/base64"
	"fmt"
	"github.com/docker/libswarm/beam"
	"github.com/docker/libswarm/beam/data"
	"github.com/docker/spdystream"
	"io"
	"net"
	"net/http"
	"os"
	"syscall"
)

type Authenticator func(conn net.Conn) (spdystream.AuthHandler, error)

func NoAuthenticator(conn net.Conn) (spdystream.AuthHandler, error) {
	return func(header http.Header, slot uint8, parent uint32) bool {
		return true
	}, nil
}

type streamChanProvider interface {
	addStreamChan(stream *spdystream.Stream, streamChan chan *spdystream.Stream)
	getStreamChan(stream *spdystream.Stream) chan *spdystream.Stream
}

func encodeArgs(args []string) string {
	encoded := data.Encode(map[string][]string{"args": args})
	return base64.URLEncoding.EncodeToString([]byte(encoded))
}

func decodeArgs(argString string) ([]string, error) {
	decoded, decodeErr := base64.URLEncoding.DecodeString(argString)
	if decodeErr != nil {
		return []string{}, decodeErr
	}
	dataMap, dataErr := data.Decode(string(decoded))
	if dataErr != nil {
		return []string{}, dataErr
	}
	return dataMap["args"], nil
}

func createStreamMessage(stream *spdystream.Stream, mode int, streamChans streamChanProvider, ret beam.Sender) (*beam.Message, error) {
	verbString := stream.Headers()["Verb"]
	if len(verbString) != 1 {
		if len(verbString) == 0 {
			return nil, fmt.Errorf("Stream(%s) is missing verb header", stream)
		} else {
			return nil, fmt.Errorf("Stream(%s) has multiple verb headers", stream)
		}

	}
	verb, verbOk := verbs[verbString[0]]
	if !verbOk {
		return nil, fmt.Errorf("Unknown verb: %s", verbString[0])
	}

	var args []string
	argString := stream.Headers()["Args"]
	if len(argString) > 1 {
		return nil, fmt.Errorf("Stream(%s) has multiple args headers", stream)
	}
	if len(argString) == 1 {
		var err error
		args, err = decodeArgs(argString[0])
		if err != nil {
			return nil, err
		}
	}

	var attach *os.File
	if !stream.IsFinished() {
		socketFds, socketErr := syscall.Socketpair(syscall.AF_LOCAL, syscall.SOCK_STREAM|syscall.FD_CLOEXEC, 0)
		if socketErr != nil {
			return nil, socketErr
		}
		attach = os.NewFile(uintptr(socketFds[0]), "")
		conn, connErr := net.FileConn(os.NewFile(uintptr(socketFds[1]), ""))
		if connErr != nil {
			return nil, connErr
		}

		go func() {
			io.Copy(conn, stream)
		}()
		go func() {
			io.Copy(stream, conn)
		}()
	}

	retSender := ret
	if retSender == nil || beam.RetPipe.Equals(retSender) {
		retSender = &StreamSender{stream: stream, streamChans: streamChans}
	}

	if mode&beam.Ret == 0 {
		retSender.Close()
	}

	return &beam.Message{
		Verb: verb,
		Args: args,
		Att:  attach,
		Ret:  retSender,
	}, nil
}
