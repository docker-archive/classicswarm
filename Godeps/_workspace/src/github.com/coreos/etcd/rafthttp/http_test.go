// Copyright 2015 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rafthttp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coreos/etcd/pkg/pbutil"
	"github.com/coreos/etcd/pkg/types"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/coreos/etcd/version"
)

func TestServeRaftPrefix(t *testing.T) {
	testCases := []struct {
		method    string
		body      io.Reader
		p         Raft
		clusterID string

		wcode int
	}{
		{
			// bad method
			"GET",
			bytes.NewReader(
				pbutil.MustMarshal(&raftpb.Message{}),
			),
			&fakeRaft{},
			"0",
			http.StatusMethodNotAllowed,
		},
		{
			// bad method
			"PUT",
			bytes.NewReader(
				pbutil.MustMarshal(&raftpb.Message{}),
			),
			&fakeRaft{},
			"0",
			http.StatusMethodNotAllowed,
		},
		{
			// bad method
			"DELETE",
			bytes.NewReader(
				pbutil.MustMarshal(&raftpb.Message{}),
			),
			&fakeRaft{},
			"0",
			http.StatusMethodNotAllowed,
		},
		{
			// bad request body
			"POST",
			&errReader{},
			&fakeRaft{},
			"0",
			http.StatusBadRequest,
		},
		{
			// bad request protobuf
			"POST",
			strings.NewReader("malformed garbage"),
			&fakeRaft{},
			"0",
			http.StatusBadRequest,
		},
		{
			// good request, wrong cluster ID
			"POST",
			bytes.NewReader(
				pbutil.MustMarshal(&raftpb.Message{}),
			),
			&fakeRaft{},
			"1",
			http.StatusPreconditionFailed,
		},
		{
			// good request, Processor failure
			"POST",
			bytes.NewReader(
				pbutil.MustMarshal(&raftpb.Message{}),
			),
			&fakeRaft{
				err: &resWriterToError{code: http.StatusForbidden},
			},
			"0",
			http.StatusForbidden,
		},
		{
			// good request, Processor failure
			"POST",
			bytes.NewReader(
				pbutil.MustMarshal(&raftpb.Message{}),
			),
			&fakeRaft{
				err: &resWriterToError{code: http.StatusInternalServerError},
			},
			"0",
			http.StatusInternalServerError,
		},
		{
			// good request, Processor failure
			"POST",
			bytes.NewReader(
				pbutil.MustMarshal(&raftpb.Message{}),
			),
			&fakeRaft{err: errors.New("blah")},
			"0",
			http.StatusInternalServerError,
		},
		{
			// good request
			"POST",
			bytes.NewReader(
				pbutil.MustMarshal(&raftpb.Message{}),
			),
			&fakeRaft{},
			"0",
			http.StatusNoContent,
		},
	}
	for i, tt := range testCases {
		req, err := http.NewRequest(tt.method, "foo", tt.body)
		if err != nil {
			t.Fatalf("#%d: could not create request: %#v", i, err)
		}
		req.Header.Set("X-Etcd-Cluster-ID", tt.clusterID)
		req.Header.Set("X-Server-Version", version.Version)
		rw := httptest.NewRecorder()
		h := NewHandler(tt.p, types.ID(0))
		h.ServeHTTP(rw, req)
		if rw.Code != tt.wcode {
			t.Errorf("#%d: got code=%d, want %d", i, rw.Code, tt.wcode)
		}
	}
}

func TestServeRaftStreamPrefix(t *testing.T) {
	tests := []struct {
		path  string
		wtype streamType
	}{
		{
			RaftStreamPrefix + "/message/1",
			streamTypeMessage,
		},
		{
			RaftStreamPrefix + "/msgapp/1",
			streamTypeMsgAppV2,
		},
		// backward compatibility
		{
			RaftStreamPrefix + "/1",
			streamTypeMsgApp,
		},
	}
	for i, tt := range tests {
		req, err := http.NewRequest("GET", "http://localhost:2380"+tt.path, nil)
		if err != nil {
			t.Fatalf("#%d: could not create request: %#v", i, err)
		}
		req.Header.Set("X-Etcd-Cluster-ID", "1")
		req.Header.Set("X-Server-Version", version.Version)
		req.Header.Set("X-Raft-To", "2")
		wterm := "1"
		req.Header.Set("X-Raft-Term", wterm)

		peer := newFakePeer()
		peerGetter := &fakePeerGetter{peers: map[types.ID]Peer{types.ID(1): peer}}
		h := newStreamHandler(peerGetter, &fakeRaft{}, types.ID(2), types.ID(1))

		rw := httptest.NewRecorder()
		go h.ServeHTTP(rw, req)

		var conn *outgoingConn
		select {
		case conn = <-peer.connc:
		case <-time.After(time.Second):
			t.Fatalf("#%d: failed to attach outgoingConn", i)
		}
		if g := rw.Header().Get("X-Server-Version"); g != version.Version {
			t.Errorf("#%d: X-Server-Version = %s, want %s", i, g, version.Version)
		}
		if conn.t != tt.wtype {
			t.Errorf("#%d: type = %s, want %s", i, conn.t, tt.wtype)
		}
		if conn.termStr != wterm {
			t.Errorf("#%d: term = %s, want %s", i, conn.termStr, wterm)
		}
		conn.Close()
	}
}

func TestServeRaftStreamPrefixBad(t *testing.T) {
	removedID := uint64(5)
	tests := []struct {
		method    string
		path      string
		clusterID string
		remote    string

		wcode int
	}{
		// bad method
		{
			"PUT",
			RaftStreamPrefix + "/message/1",
			"1",
			"1",
			http.StatusMethodNotAllowed,
		},
		// bad method
		{
			"POST",
			RaftStreamPrefix + "/message/1",
			"1",
			"1",
			http.StatusMethodNotAllowed,
		},
		// bad method
		{
			"DELETE",
			RaftStreamPrefix + "/message/1",
			"1",
			"1",
			http.StatusMethodNotAllowed,
		},
		// bad path
		{
			"GET",
			RaftStreamPrefix + "/strange/1",
			"1",
			"1",
			http.StatusNotFound,
		},
		// bad path
		{
			"GET",
			RaftStreamPrefix + "/strange",
			"1",
			"1",
			http.StatusNotFound,
		},
		// non-existant peer
		{
			"GET",
			RaftStreamPrefix + "/message/2",
			"1",
			"1",
			http.StatusNotFound,
		},
		// removed peer
		{
			"GET",
			RaftStreamPrefix + "/message/" + fmt.Sprint(removedID),
			"1",
			"1",
			http.StatusGone,
		},
		// wrong cluster ID
		{
			"GET",
			RaftStreamPrefix + "/message/1",
			"2",
			"1",
			http.StatusPreconditionFailed,
		},
		// wrong remote id
		{
			"GET",
			RaftStreamPrefix + "/message/1",
			"1",
			"2",
			http.StatusPreconditionFailed,
		},
	}
	for i, tt := range tests {
		req, err := http.NewRequest(tt.method, "http://localhost:2380"+tt.path, nil)
		if err != nil {
			t.Fatalf("#%d: could not create request: %#v", i, err)
		}
		req.Header.Set("X-Etcd-Cluster-ID", tt.clusterID)
		req.Header.Set("X-Server-Version", version.Version)
		req.Header.Set("X-Raft-To", tt.remote)
		rw := httptest.NewRecorder()
		peerGetter := &fakePeerGetter{peers: map[types.ID]Peer{types.ID(1): newFakePeer()}}
		r := &fakeRaft{removedID: removedID}
		h := newStreamHandler(peerGetter, r, types.ID(1), types.ID(1))
		h.ServeHTTP(rw, req)

		if rw.Code != tt.wcode {
			t.Errorf("#%d: code = %d, want %d", i, rw.Code, tt.wcode)
		}
	}
}

func TestCloseNotifier(t *testing.T) {
	c := newCloseNotifier()
	select {
	case <-c.closeNotify():
		t.Fatalf("received unexpected close notification")
	default:
	}
	c.Close()
	select {
	case <-c.closeNotify():
	default:
		t.Fatalf("failed to get close notification")
	}
}

// errReader implements io.Reader to facilitate a broken request.
type errReader struct{}

func (er *errReader) Read(_ []byte) (int, error) { return 0, errors.New("some error") }

type resWriterToError struct {
	code int
}

func (e *resWriterToError) Error() string                 { return "" }
func (e *resWriterToError) WriteTo(w http.ResponseWriter) { w.WriteHeader(e.code) }

type fakePeerGetter struct {
	peers map[types.ID]Peer
}

func (pg *fakePeerGetter) Get(id types.ID) Peer { return pg.peers[id] }

type fakePeer struct {
	msgs  []raftpb.Message
	urls  types.URLs
	term  uint64
	connc chan *outgoingConn
}

func newFakePeer() *fakePeer {
	return &fakePeer{
		connc: make(chan *outgoingConn, 1),
	}
}

func (pr *fakePeer) Send(m raftpb.Message)                 { pr.msgs = append(pr.msgs, m) }
func (pr *fakePeer) Update(urls types.URLs)                { pr.urls = urls }
func (pr *fakePeer) setTerm(term uint64)                   { pr.term = term }
func (pr *fakePeer) attachOutgoingConn(conn *outgoingConn) { pr.connc <- conn }
func (pr *fakePeer) activeSince() time.Time                { return time.Time{} }
func (pr *fakePeer) Stop()                                 {}
