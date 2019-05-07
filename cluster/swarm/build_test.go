package swarm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/docker/swarm/scheduler/node"
)

const (
	sessionID = "someSession"
	buildID   = "someBuild"
)

var (
	expectedNode = &node.Node{ID: "someNode"}
)

// TestBuildSyncerWaitSessionNode tests the main, error-free path of
// (*buildSyncer).waitSessionNode
func TestBuildSyncerWaitSessionNode(t *testing.T) {
	b := newBuildSyncer()

	// we need to create some variables to capture in the goroutine below
	var (
		actualNode *node.Node
		waitErr    error
	)

	// start this wait
	done := ensureRunsAsync(func() {
		actualNode, waitErr = b.waitSessionNode(sessionID, 1*time.Second)
	})

	defer overrideCleanupDelay()()

	// now, call startBuild to notify the waiting routine
	retNode, cleanup, err := b.startBuild(sessionID, buildID, expectedNode)
	assert.NoError(t, err)

	// now, verify waitSessionNode has returned
	<-done

	assert.NoError(t, waitErr)
	assert.NotNil(t, actualNode)
	assert.Equal(t, actualNode, retNode)

	// now, we'll do a bit of whitebox testing... does everything successfully
	// clean up?

	// first, make sure there is stuff to clean up
	assert.Contains(t, b.nodeBySessionID, sessionID)
	assert.Contains(t, b.nodeByBuildID, buildID)

	// now check that the channel was cleaned up successfully
	assert.Empty(t, b.queueSession)

	// call the cleanup function and make sure the node by ID entries are
	// removed
	cleanup()
	time.Sleep(2 * cleanupDelay)

	b.mu.Lock()
	assert.NotContains(t, b.nodeBySessionID, sessionID)
	assert.NotContains(t, b.nodeByBuildID, buildID)
	b.mu.Unlock()
}

// TestBuildSyncerWaitSessionNodeTimeout tests that waitSessionNode behaves
// correctly when the timeout is reached.
func TestBuildSyncerWaitSessionNodeTimeout(t *testing.T) {
	b := newBuildSyncer()

	var (
		actualNode *node.Node
		waitErr    error
	)

	// see TestBuildSyncerWaitSessionNode for an explanation of this if needed
	done := ensureRunsAsync(func() {
		actualNode, waitErr = b.waitSessionNode(sessionID, 1*time.Second)
	})

	select {
	case <-done:
		assert.Error(t, waitErr, "timeout waiting for build to start")
		assert.Nil(t, actualNode)
	case <-time.After(2 * time.Second):
		t.Error("waitSessionNode did not time out and return in time")
	}

	// verify that the channel we were using to wait has been cleaned up
	assert.Empty(t, b.queueSession)
}

// TestBuildSyncerWaitSessionNodeSecondCallError tests that calling
// waitSessionNode a second time on the same sessionID returns an error
func TestBuildSyncerWaitSessionNodeSecondCallError(t *testing.T) {
	b := newBuildSyncer()

	// we're not gonna take the return channel -- we don't need it
	ensureRunsAsync(func() {
		// in this test, we don't actually care about the return value of this
		// call
		b.waitSessionNode(sessionID, 1*time.Second)
	})

	// now, try calling waitSessionNode again. we'll not use a goroutine, and
	// in the failure case we'll just wait for test timeout
	n, err := b.waitSessionNode(sessionID, 1*time.Second)
	assert.Error(t, err, "already waiting on node for session someSession")
	assert.Nil(t, n)
}

// TestBuildSyncerWaitBuildNode tests the main, error-free path of
// waitBuildNode when there is only 1 waiter
func TestBuildSyncerWaitBuildNode(t *testing.T) {
	b := newBuildSyncer()

	var (
		actualNode *node.Node
		waitErr    error
	)
	done := ensureRunsAsync(func() {
		actualNode, waitErr = b.waitBuildNode(buildID, 1*time.Second)
	})

	defer overrideCleanupDelay()()

	retNode, cleanup, err := b.startBuild(sessionID, buildID, expectedNode)
	assert.NoError(t, err)

	<-done

	assert.NoError(t, waitErr)
	assert.NotNil(t, actualNode)
	assert.Equal(t, actualNode, retNode)

	assert.Contains(t, b.nodeBySessionID, sessionID)
	assert.Contains(t, b.nodeByBuildID, buildID)

	assert.Empty(t, b.queueBuild)

	cleanup()
	time.Sleep(2 * cleanupDelay)

	b.mu.Lock()
	assert.NotContains(t, b.nodeBySessionID, sessionID)
	assert.NotContains(t, b.nodeByBuildID, buildID)
	b.mu.Unlock()
}

// TestBuildSyncerWaitBuildNodeMulti tests the main, error-free path of
// waitBuild node when there are multiple waiting routines.
func TestBuildSyncerWaitBuildNodeMulti(t *testing.T) {
	b := newBuildSyncer()
	// routineCount controls, definitively, the total number of simultaneous
	// calls we make to waitBuildNode.
	routineCount := 4

	nodeResults := make([]*node.Node, routineCount)
	errorResults := make([]error, routineCount)
	doneResults := make([]<-chan struct{}, routineCount)
	for i := 0; i < routineCount; i++ {
		// capture the variable i. you need to do this when dealing with
		// goroutines and for loops
		i := i
		doneResults[i] = ensureRunsAsync(func() {
			nodeResults[i], errorResults[i] = b.waitBuildNode(buildID, 2*time.Second)
		})
	}

	defer overrideCleanupDelay()()

	retNode, cleanup, err := b.startBuild(sessionID, buildID, expectedNode)
	assert.NoError(t, err)

	defer cleanup()

	for i := 0; i < routineCount; i++ {
		<-doneResults[i]
		assert.NoError(t, errorResults[i])
		assert.NotNil(t, nodeResults[i])
		assert.Equal(t, nodeResults[i], retNode)
	}

	assert.Empty(t, b.queueBuild)
}

// TestBuildSyncerWaitBuildNodeTimeout tests that waitBuildNode behaves
// correctly when a waiter times out. This ensures, additionally, that if one
// wait times out, other routines can still complete
func TestBuildSyncerWaitBuildNodeTimeout(t *testing.T) {
	b := newBuildSyncer()

	var (
		timeoutNode, completesNode *node.Node
		timeoutErr, completesErr   error
	)
	doneTimesOut := ensureRunsAsync(func() {
		timeoutNode, timeoutErr = b.waitBuildNode(buildID, 1*time.Second)
	})

	// this one will complete, so we'll set a longer timeout that we don't hit
	doneCompletes := ensureRunsAsync(func() {
		completesNode, completesErr = b.waitBuildNode(buildID, 5*time.Second)
	})

	select {
	case <-doneTimesOut:
		assert.Error(t, timeoutErr, "timeout waiting for build to start")
		assert.Nil(t, timeoutNode)
	case <-time.After(2 * time.Second):
		t.Error("waitBuildNode did not time out and return in time")
	}

	defer overrideCleanupDelay()()

	// now call startBuild to cause the 2nd waiter to return
	retNode, cleanup, err := b.startBuild(sessionID, buildID, expectedNode)
	assert.NoError(t, err)

	<-doneCompletes
	assert.NoError(t, completesErr)
	assert.NotNil(t, completesNode)
	assert.Equal(t, completesNode, retNode)

	cleanup()
	time.Sleep(2 * cleanupDelay)

	b.mu.Lock()
	assert.Empty(t, b.nodeByBuildID)
	assert.Empty(t, b.nodeBySessionID)
	b.mu.Unlock()
}

// ensureRunsAsync is a function that takes a closure, starts it in a goroutine,
// blocks until it starts, and then return a channel to be closed when the
// routine exits
func ensureRunsAsync(fn func()) <-chan struct{} {
	// testing blocking code like this is always funky. in order to make sure
	// this code gets scheduled and run, we'll need to use some channels. this
	// first channel will get closed right before the function gets called.  by
	// blocking unconditionally on it in the parent routine, we will guarantee
	// that the goroutine gets scheduled to run
	started := make(chan struct{})
	// this second channel is closed after b.waitSessionNode returns, so we can
	// ensure that it has successfully exited (and avoid possible races on the
	// variables)
	done := make(chan struct{})
	go func() {
		close(started)
		fn()
		close(done)
	}()

	// now, we block on started. this routine will be totally blocked, so the
	// test can't proceed until the Go scheduler has given the above goroutine
	// a chance to run
	<-started
	return done
}

// override cleanupDelay for testing
func overrideCleanupDelay() func() {
	restore := cleanupDelay
	cleanupDelay = 100 * time.Millisecond
	return func() {
		cleanupDelay = restore
	}
}
