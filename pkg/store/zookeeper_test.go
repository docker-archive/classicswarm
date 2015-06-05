package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func makeZkClient(t *testing.T) Store {
	client := "localhost:2181"

	kv, err := NewStore(
		ZK,
		[]string{client},
		&Config{
			ConnectionTimeout: 3 * time.Second,
			EphemeralTTL:      2 * time.Second,
		},
	)
	if err != nil {
		t.Fatalf("cannot create store: %v", err)
	}

	return kv
}

func TestZkStore(t *testing.T) {
	kv := makeZkClient(t)

	testPutGetDelete(t, kv)
	testWatch(t, kv)
	testWatchTree(t, kv)
	testAtomicPut(t, kv)
	testAtomicDelete(t, kv)
	testLockUnlock(t, kv)
	testPutEphemeral(t, kv)
	testList(t, kv)
	testDeleteTree(t, kv)
}

func TestCreateFullPath(t *testing.T) {
	kv := makeZkClient(t)

	zk := kv.(*Zookeeper)

	dir := "create/me/please"

	// Create the directory
	err := zk.createFullpath([]string{"create", "me", "please"}, false)
	assert.NoError(t, err)

	// The directory must exist
	pair, err := kv.Get(dir)
	assert.NoError(t, err)
	if assert.NotNil(t, pair) {
		assert.NotNil(t, pair.Value)
	}
	assert.Equal(t, pair.Value, []byte{1})
	assert.NotEqual(t, pair.LastIndex, 0)
}
