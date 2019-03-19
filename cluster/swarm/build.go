package swarm

import (
	"sync"
	"time"

	"github.com/docker/swarm/scheduler/node"
	"github.com/pkg/errors"
)

// waitRefCounter allows multiple waiting routines to exit however they'd like
// while still cleaning up correctly.
type waitRefCounter struct {
	done  chan struct{}
	count int
}

func newBuildSyncer() *buildSyncer {
	return &buildSyncer{
		nodeByBuildID:   map[string]*node.Node{},
		nodeBySessionID: map[string]*node.Node{},
		queueSession:    map[string]chan struct{}{},
		queueBuild:      map[string]*waitRefCounter{},
	}
}

type buildSyncer struct {
	mu              sync.Mutex
	nodeByBuildID   map[string]*node.Node
	nodeBySessionID map[string]*node.Node
	// queueSession is a map to channels, because if we try to call
	// waitSessionNode more than once we'll error out.
	queueSession map[string]chan struct{}
	// queueBuild has a map of waitRefCounter, so in the odd case where
	// multiple outstanding waits happen simultaneously, we can handle it.
	queueBuild map[string]*waitRefCounter
}

func (b *buildSyncer) waitSessionNode(sessionID string, timeout time.Duration) (*node.Node, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if n, ok := b.nodeBySessionID[sessionID]; ok {
		return n, nil
	}

	ch, ok := b.queueSession[sessionID]
	// we should return an error if we try to wait twice on the same
	// session
	if ok {
		return nil, errors.Errorf("already waiting on node for session %v", sessionID)
	}
	ch = make(chan struct{})
	b.queueSession[sessionID] = ch
	// defer removing this session wait from the queue. this will happen before
	// we release the lock.
	defer func() {
		delete(b.queueSession, sessionID)
	}()

	// we need to release the lock so other operations can proceed. we will
	// reacquire it before exiting
	b.mu.Unlock()
	select {
	case <-time.After(timeout):
		b.mu.Lock()
		return nil, errors.Errorf("timeout waiting for build to start")
	case <-ch:
		b.mu.Lock()
		if n, ok := b.nodeBySessionID[sessionID]; ok {
			return n, nil
		}
		return nil, errors.Errorf("build closed")
	}
}

func (b *buildSyncer) waitBuildNode(buildID string, timeout time.Duration) (*node.Node, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if n, ok := b.nodeByBuildID[buildID]; ok {
		return n, nil
	}

	// try to get the wait for the buildID. If one doesn't exist, create it
	// now.
	wait, ok := b.queueBuild[buildID]
	if !ok {
		wait = &waitRefCounter{
			done:  make(chan struct{}),
			count: 0,
		}
		b.queueBuild[buildID] = wait
	}

	wait.count += 1
	ch := wait.done

	// this defer handles cleaning up the queueBuild map. it will execute
	// before the lock is released.
	defer func() {
		wait.count -= 1
		// if there are no more waiters, then remove the map entry
		if wait.count == 0 {
			delete(b.queueBuild, buildID)
		}
	}()

	b.mu.Unlock()
	select {
	case <-time.After(timeout):
		b.mu.Lock()
		return nil, errors.Errorf("timeout waiting for build to start")
	case <-ch:
		b.mu.Lock()
		if n, ok := b.nodeByBuildID[buildID]; ok {
			return n, nil
		}
		return nil, errors.Errorf("build closed")
	}
}

func (b *buildSyncer) startBuild(sessionID, buildID string, node *node.Node) (*node.Node, func(), error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	_, ok := b.nodeByBuildID[buildID]
	if ok {
		return nil, nil, errors.Errorf("build already started")
	}

	n, ok := b.nodeBySessionID[buildID]
	if ok {
		node = n
	} else {
		b.nodeBySessionID[sessionID] = node
	}

	b.nodeByBuildID[buildID] = node

	// if there are routines waiting on a node, close the channel to signal to
	// them that a node has been selected.
	if wait, ok := b.queueBuild[buildID]; ok {
		close(wait.done)
	}
	if ch, ok := b.queueSession[sessionID]; ok {
		close(ch)
	}

	return node, func() {
		// delay cleanup to support shared long running sessions
		time.AfterFunc(time.Hour, func() {
			b.mu.Lock()
			delete(b.nodeBySessionID, sessionID)
			delete(b.nodeByBuildID, buildID)
			b.mu.Unlock()
		})
	}, nil
}
