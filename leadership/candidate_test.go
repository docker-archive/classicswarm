package leadership

import (
	"testing"

	kv "github.com/docker/swarm/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCandidate(t *testing.T) {
	store, err := kv.NewStore("mock", []string{}, nil)
	assert.NoError(t, err)

	mockStore := store.(*kv.Mock)
	mockLock := &kv.MockLock{}
	mockStore.On("NewLock", "test_key", mock.Anything).Return(mockLock, nil)

	// Lock and unlock always succeeds.
	lostCh := make(chan struct{})
	var mockLostCh <-chan struct{} = lostCh
	mockLock.On("Lock").Return(mockLostCh, nil)
	mockLock.On("Unlock").Return(nil)

	candidate := NewCandidate(store, "test_key", "test_node")
	candidate.RunForElection()
	electedCh := candidate.ElectedCh()

	// Should issue a false upon start, no matter what.
	assert.False(t, candidate.IsLeader())
	assert.False(t, <-electedCh)

	// Since the lock always succeeeds, we should get elected.
	assert.True(t, <-electedCh)
	assert.True(t, candidate.IsLeader())

	// Signaling a lost lock should get us de-elected...
	close(lostCh)
	assert.False(t, <-electedCh)

	// And we should attempt to get re-elected again.
	assert.True(t, <-electedCh)

	// When we resign, unlock will get called, we'll be notified of the
	// de-election and we'll try to get the lock again.
	go candidate.Resign()
	assert.False(t, <-electedCh)
	assert.True(t, <-electedCh)

	// After stopping the candidate, the ElectedCh should be closed.
	candidate.Stop()
	select {
	case <-electedCh:
		assert.True(t, false) // we should not get here.
	default:
		assert.True(t, true)
	}

	mockStore.AssertExpectations(t)
}
