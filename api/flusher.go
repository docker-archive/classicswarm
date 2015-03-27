package api

import (
	"io"
	"net/http"
	"sync"

	"github.com/docker/docker/pkg/ioutils"
)

// WriteFlusher is exported
type WriteFlusher struct {
	sync.Mutex
	w       io.Writer
	flusher http.Flusher
}

func (wf *WriteFlusher) Write(b []byte) (n int, err error) {
	wf.Lock()
	defer wf.Unlock()
	n, err = wf.w.Write(b)
	wf.flusher.Flush()
	return n, err
}

// Flush the stream immediately.
func (wf *WriteFlusher) Flush() {
	wf.Lock()
	defer wf.Unlock()
	wf.flusher.Flush()
}

// NewWriteFlusher is exported
func NewWriteFlusher(w io.Writer) *WriteFlusher {
	var flusher http.Flusher
	if f, ok := w.(http.Flusher); ok {
		flusher = f
	} else {
		flusher = &ioutils.NopFlusher{}
	}
	return &WriteFlusher{w: w, flusher: flusher}
}
