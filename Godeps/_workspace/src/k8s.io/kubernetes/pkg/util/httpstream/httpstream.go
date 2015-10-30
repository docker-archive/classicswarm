/*
Copyright 2015 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package httpstream

import (
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	HeaderConnection = "Connection"
	HeaderUpgrade    = "Upgrade"
)

// NewStreamHandler defines a function that is called when a new Stream is
// received. If no error is returned, the Stream is accepted; otherwise,
// the stream is rejected.
type NewStreamHandler func(Stream) error

// NoOpNewStreamHandler is a stream handler that accepts a new stream and
// performs no other logic.
func NoOpNewStreamHandler(stream Stream) error { return nil }

// UpgradeRoundTripper is a type of http.RoundTripper that is able to upgrade
// HTTP requests to support multiplexed bidirectional streams. After RoundTrip()
// is invoked, if the upgrade is successful, clients may retrieve the upgraded
// connection by calling UpgradeRoundTripper.Connection().
type UpgradeRoundTripper interface {
	http.RoundTripper
	// NewConnection validates the response and creates a new Connection.
	NewConnection(resp *http.Response) (Connection, error)
}

// ResponseUpgrader knows how to upgrade HTTP requests and responses to
// add streaming support to them.
type ResponseUpgrader interface {
	// UpgradeResponse upgrades an HTTP response to one that supports multiplexed
	// streams. newStreamHandler will be called synchronously whenever the
	// other end of the upgraded connection creates a new stream.
	UpgradeResponse(w http.ResponseWriter, req *http.Request, newStreamHandler NewStreamHandler) Connection
}

// Connection represents an upgraded HTTP connection.
type Connection interface {
	// CreateStream creates a new Stream with the supplied headers.
	CreateStream(headers http.Header) (Stream, error)
	// Close resets all streams and closes the connection.
	Close() error
	// CloseChan returns a channel that is closed when the underlying connection is closed.
	CloseChan() <-chan bool
	// SetIdleTimeout sets the amount of time the connection may remain idle before
	// it is automatically closed.
	SetIdleTimeout(timeout time.Duration)
}

// Stream represents a bidirectional communications channel that is part of an
// upgraded connection.
type Stream interface {
	io.ReadWriteCloser
	// Reset closes both directions of the stream, indicating that neither client
	// or server can use it any more.
	Reset() error
	// Headers returns the headers used to create the stream.
	Headers() http.Header
	// Identifier returns the stream's ID.
	Identifier() uint32
}

// IsUpgradeRequest returns true if the given request is a connection upgrade request
func IsUpgradeRequest(req *http.Request) bool {
	for _, h := range req.Header[http.CanonicalHeaderKey(HeaderConnection)] {
		if strings.Contains(strings.ToLower(h), strings.ToLower(HeaderUpgrade)) {
			return true
		}
	}
	return false
}
