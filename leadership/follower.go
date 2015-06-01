package leadership

import "github.com/docker/swarm/pkg/store"

// Follower can folow an election in real-time and push notifications whenever
// there is a change in leadership.
type Follower struct {
	client store.Store
	key    string

	leader   string
	leaderCh chan string
	stopCh   chan struct{}
}

// NewFollower creates a new follower.
func NewFollower(client store.Store, key string) *Follower {
	return &Follower{
		client:   client,
		key:      key,
		leaderCh: make(chan string),
		stopCh:   make(chan struct{}),
	}
}

// LeaderCh is used to get a channel which delivers the currently elected
// leader.
func (f *Follower) LeaderCh() <-chan string {
	return f.leaderCh
}

// Leader returns the current leader.
func (f *Follower) Leader() string {
	return f.leader
}

// FollowElection starts monitoring the election.
func (f *Follower) FollowElection() error {
	ch, err := f.client.Watch(f.key, f.stopCh)
	if err != nil {
		return err
	}

	go f.follow(ch)

	return nil
}

// Stop stops monitoring an election.
func (f *Follower) Stop() {
	close(f.stopCh)
}

func (f *Follower) follow(<-chan *store.KVPair) {
	defer close(f.leaderCh)

	// FIXME: We should pass `RequireConsistent: true` to Consul.
	ch, err := f.client.Watch(f.key, f.stopCh)
	if err != nil {
		return
	}

	f.leader = ""
	for kv := range ch {
		curr := string(kv.Value)
		if curr == f.leader {
			continue
		}
		f.leader = curr
		f.leaderCh <- f.leader
	}
}
