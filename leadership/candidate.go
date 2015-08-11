package leadership

import (
	"sync"

	"github.com/docker/libkv/store"
)

// Candidate runs the leader election algorithm asynchronously
type Candidate struct {
	client store.Store
	key    string
	node   string

	electedCh chan bool
	lock      sync.Mutex
	leader    bool
	stopCh    chan struct{}
	resignCh  chan bool
	errCh     chan error
}

// NewCandidate creates a new Candidate
func NewCandidate(client store.Store, key, node string) *Candidate {
	return &Candidate{
		client: client,
		key:    key,
		node:   node,

		leader:   false,
		resignCh: make(chan bool),
		stopCh:   make(chan struct{}),
	}
}

// IsLeader returns true if the candidate is currently a leader.
func (c *Candidate) IsLeader() bool {
	return c.leader
}

// RunForElection starts the leader election algorithm. Updates in status are
// pushed through the ElectedCh channel.
//
// ElectedCh is used to get a channel which delivers signals on
// acquiring or losing leadership. It sends true if we become
// the leader, and false if we lose it.
func (c *Candidate) RunForElection() (<-chan bool, <-chan error) {
	c.electedCh = make(chan bool)
	c.errCh = make(chan error)

	// Need a `SessionTTL` (keep-alive) and a stop channel.
	lock, err := c.client.NewLock(c.key, &store.LockOptions{Value: []byte(c.node)})
	if err != nil {
		c.errCh <- err
	} else {
		go c.campaign(lock)
	}

	return c.electedCh, c.errCh
}

// Stop running for election.
func (c *Candidate) Stop() {
	close(c.stopCh)
}

// Resign forces the candidate to step-down and try again.
// If the candidate is not a leader, it doesn't have any effect.
// Candidate will retry immediately to acquire the leadership. If no-one else
// took it, then the Candidate will end up being a leader again.
func (c *Candidate) Resign() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.leader {
		c.resignCh <- true
	}
}

func (c *Candidate) update(status bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.leader = status
	c.electedCh <- status
}

func (c *Candidate) campaign(lock store.Locker) {
	defer close(c.electedCh)
	defer close(c.errCh)

	for {
		// Start as a follower.
		c.update(false)

		lostCh, err := lock.Lock()
		if err != nil {
			c.errCh <- err
			return
		}

		// Hooray! We acquired the lock therefore we are the new leader.
		c.update(true)

		select {
		case <-c.resignCh:
			// We were asked to resign, give up the lock and go back
			// campaigning.
			lock.Unlock()
		case <-c.stopCh:
			// Give up the leadership and quit.
			if c.leader {
				lock.Unlock()
			}
			return
		case <-lostCh:
			// We lost the lock. Someone else is the leader, try again.
		}
	}
}
