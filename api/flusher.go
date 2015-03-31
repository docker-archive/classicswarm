package api

import (
	"io"
	"net/http"
	"sync"

	"github.com/docker/docker/pkg/ioutils"
)

// A WriteFlusher provides synchronized write access to the writer's underlying data stream and ensures that each write is flushed immediately.
type WriteFlusher struct {
	sync.Mutex
	w       io.Writer
	flusher http.Flusher
}

// Write writes the bytes to a stream and flushes the stream.
func (wf *WriteFlusher) Write(b []byte) (n int, err error) {
	wf.Lock()
	defer wf.Unlock()
	n, err = wf.w.Write(b)
	wf.flusher.Flush()
	return n, err
}

// Flush flushes the stream immediately.
func (wf *WriteFlusher) Flush() {
	wf.Lock()
	defer wf.Unlock()
	wf.flusher.Flush()
}

// NewWriteFlusher creates a new WriteFlusher for the writer.
func NewWriteFlusher(w io.Writer) *WriteFlusher {
	var flusher http.Flusher
	if f, ok := w.(http.Flusher); ok {
		flusher = f
	} else {
		flusher = &ioutils.NopFlusher{}
	}
	return &WriteFlusher{w: w, flusher: flusher}
}
