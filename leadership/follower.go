package leadership

import "github.com/docker/swarm/pkg/store"

// Follower can folow an election in real-time and push notifications whenever
// there is a change in leadership.
type Follower struct {
	LeaderCh chan string

	client store.Store
	key    string

	stopCh chan struct{}
}

// NewFollower creates a new follower.
func NewFollower(client store.Store, key string) *Follower {
	return &Follower{
		LeaderCh: make(chan string),
		client:   client,
		key:      key,
		stopCh:   make(chan struct{}),
	}
}

// FollowElection starts monitoring the election. The current leader is updated
// in real-time and pushed through `LeaderCh`.
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
	defer close(f.LeaderCh)

	// FIXME: We should pass `RequireConsistent: true` to Consul.
	ch, err := f.client.Watch(f.key, f.stopCh)
	if err != nil {
		return
	}

	prev := ""
	for kv := range ch {
		curr := string(kv.Value)
		if curr == prev {
			continue
		}
		prev = curr
		f.LeaderCh <- string(curr)
	}
}
