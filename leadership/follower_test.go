package leadership

import (
	"testing"

	kv "github.com/docker/swarm/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFollower(t *testing.T) {
	store, err := kv.NewStore("mock", []string{}, nil)
	assert.NoError(t, err)

	mockStore := store.(*kv.Mock)

	kvCh := make(chan *kv.KVPair)
	var mockKVCh <-chan *kv.KVPair = kvCh
	mockStore.On("Watch", "test_key", mock.Anything).Return(mockKVCh, nil)

	follower := NewFollower(store, "test_key")
	follower.FollowElection()
	leaderCh := follower.LeaderCh()

	// Simulate leader updates
	go func() {
		kvCh <- &kv.KVPair{Key: "test_key", Value: []byte("leader1")}
		kvCh <- &kv.KVPair{Key: "test_key", Value: []byte("leader1")}
		kvCh <- &kv.KVPair{Key: "test_key", Value: []byte("leader2")}
		kvCh <- &kv.KVPair{Key: "test_key", Value: []byte("leader1")}
	}()

	// We shouldn't see duplicate events.
	assert.Equal(t, <-leaderCh, "leader1")
	assert.Equal(t, follower.Leader(), "leader1")
	assert.Equal(t, <-leaderCh, "leader2")
	assert.Equal(t, <-leaderCh, "leader1")

	// Once stopped, iteration over the leader channel should stop.
	follower.Stop()
	close(kvCh)
	assert.Equal(t, "", <-leaderCh)

	mockStore.AssertExpectations(t)
}
