package store

import (
	"testing"
	"time"
)

func makeConsulClient(t *testing.T) Store {
	client := "localhost:8500"

	kv, err := NewStore(
		CONSUL,
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

func TestConsulStore(t *testing.T) {
	kv := makeConsulClient(t)

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
