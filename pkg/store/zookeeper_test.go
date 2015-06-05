package store

import (
	"testing"
	"time"
)

func makeZkClient(t *testing.T) Store {
	client := "localhost:2181"

	kv, err := NewStore(
		ZK,
		[]string{client},
		&Config{
			ConnectionTimeout: 10 * time.Second,
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
	// testPutEphemeral(t, kv)
	testList(t, kv)
	testDeleteTree(t, kv)
}
