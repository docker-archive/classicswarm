package iowrapper

import (
	"io"
)

func Wrap(obj interface{}) io.ReadWriteCloser {
	return iowrapper{obj}
}

type iowrapper struct {
	obj interface{}
}

func (w iowrapper) Read(p []byte) (int, error) {
	if reader, ok := w.obj.(io.Reader); ok {
		return reader.Read(p)
	} else {
		return 0, io.ErrClosedPipe
	}
}

func (w iowrapper) Write(p []byte) (int, error) {
	if writer, ok := w.obj.(io.Writer); ok {
		return writer.Write(p)
	} else {
		return 0, io.ErrClosedPipe
	}
}

func (w iowrapper) Close() error {
	if closer, ok := w.obj.(io.Closer); ok {
		return closer.Close()
	} else {
		return nil
	}
}
