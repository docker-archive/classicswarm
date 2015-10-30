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
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/coreos/etcd/etcdserver/stats"
	"github.com/coreos/etcd/pkg/testutil"
	"github.com/coreos/etcd/pkg/types"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/coreos/etcd/version"
)

// TestPipelineSend tests that pipeline could send data using roundtripper
// and increase success count in stats.
func TestPipelineSend(t *testing.T) {
	tr := &roundTripperRecorder{}
	picker := mustNewURLPicker(t, []string{"http://localhost:2380"})
	fs := &stats.FollowerStats{}
	p := newPipeline(tr, picker, types.ID(2), types.ID(1), types.ID(1), newPeerStatus(types.ID(1)), fs, &fakeRaft{}, nil)

	p.msgc <- raftpb.Message{Type: raftpb.MsgApp}
	testutil.WaitSchedule()
	p.stop()

	if tr.Request() == nil {
		t.Errorf("sender fails to post the data")
	}
	fs.Lock()
	defer fs.Unlock()
	if fs.Counts.Success != 1 {
		t.Errorf("success = %d, want 1", fs.Counts.Success)
	}
}

func TestPipelineExceedMaximumServing(t *testing.T) {
	tr := newRoundTripperBlocker()
	picker := mustNewURLPicker(t, []string{"http://localhost:2380"})
	fs := &stats.FollowerStats{}
	p := newPipeline(tr, picker, types.ID(2), types.ID(1), types.ID(1), newPeerStatus(types.ID(1)), fs, &fakeRaft{}, nil)

	// keep the sender busy and make the buffer full
	// nothing can go out as we block the sender
	testutil.WaitSchedule()
	for i := 0; i < connPerPipeline+pipelineBufSize; i++ {
		select {
		case p.msgc <- raftpb.Message{}:
		default:
			t.Errorf("failed to send out message")
		}
		// force the sender to grab data
		testutil.WaitSchedule()
	}

	// try to send a data when we are sure the buffer is full
	select {
	case p.msgc <- raftpb.Message{}:
		t.Errorf("unexpected message sendout")
	default:
	}

	// unblock the senders and force them to send out the data
	tr.unblock()
	testutil.WaitSchedule()

	// It could send new data after previous ones succeed
	select {
	case p.msgc <- raftpb.Message{}:
	default:
		t.Errorf("failed to send out message")
	}
	p.stop()
}

// TestPipelineSendFailed tests that when send func meets the post error,
// it increases fail count in stats.
func TestPipelineSendFailed(t *testing.T) {
	picker := mustNewURLPicker(t, []string{"http://localhost:2380"})
	fs := &stats.FollowerStats{}
	p := newPipeline(newRespRoundTripper(0, errors.New("blah")), picker, types.ID(2), types.ID(1), types.ID(1), newPeerStatus(types.ID(1)), fs, &fakeRaft{}, nil)

	p.msgc <- raftpb.Message{Type: raftpb.MsgApp}
	testutil.WaitSchedule()
	p.stop()

	fs.Lock()
	defer fs.Unlock()
	if fs.Counts.Fail != 1 {
		t.Errorf("fail = %d, want 1", fs.Counts.Fail)
	}
}

func TestPipelinePost(t *testing.T) {
	tr := &roundTripperRecorder{}
	picker := mustNewURLPicker(t, []string{"http://localhost:2380"})
	p := newPipeline(tr, picker, types.ID(2), types.ID(1), types.ID(1), newPeerStatus(types.ID(1)), nil, &fakeRaft{}, nil)
	if err := p.post([]byte("some data")); err != nil {
		t.Fatalf("unexpect post error: %v", err)
	}
	p.stop()

	if g := tr.Request().Method; g != "POST" {
		t.Errorf("method = %s, want %s", g, "POST")
	}
	if g := tr.Request().URL.String(); g != "http://localhost:2380/raft" {
		t.Errorf("url = %s, want %s", g, "http://localhost:2380/raft")
	}
	if g := tr.Request().Header.Get("Content-Type"); g != "application/protobuf" {
		t.Errorf("content type = %s, want %s", g, "application/protobuf")
	}
	if g := tr.Request().Header.Get("X-Server-Version"); g != version.Version {
		t.Errorf("version = %s, want %s", g, version.Version)
	}
	if g := tr.Request().Header.Get("X-Min-Cluster-Version"); g != version.MinClusterVersion {
		t.Errorf("min version = %s, want %s", g, version.MinClusterVersion)
	}
	if g := tr.Request().Header.Get("X-Etcd-Cluster-ID"); g != "1" {
		t.Errorf("cluster id = %s, want %s", g, "1")
	}
	b, err := ioutil.ReadAll(tr.Request().Body)
	if err != nil {
		t.Fatalf("unexpected ReadAll error: %v", err)
	}
	if string(b) != "some data" {
		t.Errorf("body = %s, want %s", b, "some data")
	}
}

func TestPipelinePostBad(t *testing.T) {
	tests := []struct {
		u    string
		code int
		err  error
	}{
		// RoundTrip returns error
		{"http://localhost:2380", 0, errors.New("blah")},
		// unexpected response status code
		{"http://localhost:2380", http.StatusOK, nil},
		{"http://localhost:2380", http.StatusCreated, nil},
	}
	for i, tt := range tests {
		picker := mustNewURLPicker(t, []string{tt.u})
		p := newPipeline(newRespRoundTripper(tt.code, tt.err), picker, types.ID(2), types.ID(1), types.ID(1), newPeerStatus(types.ID(1)), nil, &fakeRaft{}, make(chan error))
		err := p.post([]byte("some data"))
		p.stop()

		if err == nil {
			t.Errorf("#%d: err = nil, want not nil", i)
		}
	}
}

func TestPipelinePostErrorc(t *testing.T) {
	tests := []struct {
		u    string
		code int
		err  error
	}{
		{"http://localhost:2380", http.StatusForbidden, nil},
	}
	for i, tt := range tests {
		picker := mustNewURLPicker(t, []string{tt.u})
		errorc := make(chan error, 1)
		p := newPipeline(newRespRoundTripper(tt.code, tt.err), picker, types.ID(2), types.ID(1), types.ID(1), newPeerStatus(types.ID(1)), nil, &fakeRaft{}, errorc)
		p.post([]byte("some data"))
		p.stop()
		select {
		case <-errorc:
		default:
			t.Fatalf("#%d: cannot receive from errorc", i)
		}
	}
}

func TestStopBlockedPipeline(t *testing.T) {
	picker := mustNewURLPicker(t, []string{"http://localhost:2380"})
	p := newPipeline(newRoundTripperBlocker(), picker, types.ID(2), types.ID(1), types.ID(1), newPeerStatus(types.ID(1)), nil, &fakeRaft{}, nil)
	// send many messages that most of them will be blocked in buffer
	for i := 0; i < connPerPipeline*10; i++ {
		p.msgc <- raftpb.Message{}
	}

	done := make(chan struct{})
	go func() {
		p.stop()
		done <- struct{}{}
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("failed to stop pipeline in 1s")
	}
}

type roundTripperBlocker struct {
	unblockc chan struct{}
	mu       sync.Mutex
	cancel   map[*http.Request]chan struct{}
}

func newRoundTripperBlocker() *roundTripperBlocker {
	return &roundTripperBlocker{
		unblockc: make(chan struct{}),
		cancel:   make(map[*http.Request]chan struct{}),
	}
}
func (t *roundTripperBlocker) unblock() {
	close(t.unblockc)
}
func (t *roundTripperBlocker) CancelRequest(req *http.Request) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if c, ok := t.cancel[req]; ok {
		c <- struct{}{}
		delete(t.cancel, req)
	}
}

type respRoundTripper struct {
	code   int
	header http.Header
	err    error
}

func newRespRoundTripper(code int, err error) *respRoundTripper {
	return &respRoundTripper{code: code, err: err}
}
func (t *respRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: t.code, Header: t.header, Body: &nopReadCloser{}}, t.err
}

type roundTripperRecorder struct {
	req *http.Request
	sync.Mutex
}

func (t *roundTripperRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	t.Lock()
	defer t.Unlock()
	t.req = req
	return &http.Response{StatusCode: http.StatusNoContent, Body: &nopReadCloser{}}, nil
}
func (t *roundTripperRecorder) Request() *http.Request {
	t.Lock()
	defer t.Unlock()
	return t.req
}

type nopReadCloser struct{}

func (n *nopReadCloser) Read(p []byte) (int, error) { return 0, io.EOF }
func (n *nopReadCloser) Close() error               { return nil }
