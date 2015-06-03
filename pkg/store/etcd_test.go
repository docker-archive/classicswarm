package store

import (
	"bytes"
	"testing"
	"time"
)

func makeEtcdClient(t *testing.T) Store {
	client := "localhost:4001"

	// Initialize a new store with consul
	kv, err := NewStore(
		ETCD,
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

func TestEtcdPutGetDelete(t *testing.T) {
	kv := makeEtcdClient(t)

	key := "foo"
	value := []byte("bar")

	// Put the key
	if err := kv.Put(key, value, nil); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Get should return the value and an incremented index
	pair, err := kv.Get(key)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if pair.Value == nil {
		t.Fatalf("expected value: %#v", value)
	}
	if !bytes.Equal(pair.Value, value) {
		t.Fatalf("unexpected value: %#v", pair.Value)
	}
	if pair.LastIndex == 0 {
		t.Fatalf("unexpected index: %#v", pair.LastIndex)
	}

	// Delete the key
	if err := kv.Delete(key); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Get should fail
	pair, err = kv.Get(key)
	if err == nil {
		t.Fatalf("err: %v", err)
	}
	if pair != nil {
		t.Fatalf("unexpected value: %#v", pair)
	}
}

func TestEtcdCompareAndSwap(t *testing.T) {
	kv := makeEtcdClient(t)

	key := "hello"
	value := []byte("world")

	// Put the key
	if err := kv.Put(key, value, nil); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Get should return the value and an incremented index
	pair, err := kv.Get(key)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if pair.Value == nil {
		t.Fatalf("expected value: %#v", value)
	}
	if !bytes.Equal(pair.Value, value) {
		t.Fatalf("unexpected value: %#v", pair.Value)
	}
	if pair.LastIndex == 0 {
		t.Fatalf("unexpected index: %#v", pair.LastIndex)
	}

	// This CAS should succeed
	success, _, err := kv.AtomicPut("hello", []byte("WORLD"), pair, nil)
	if err == nil && success == false {
		t.Fatalf("err: %v", err)
	}

	// This CAS should fail
	pair.LastIndex = 0
	success, _, err = kv.AtomicPut("hello", []byte("WORLDWORLD"), pair, nil)
	if err == nil && success {
		t.Fatalf("unexpected CAS success: %#v", success)
	}
}

func TestEtcdLockUnlock(t *testing.T) {
	t.Parallel()
	kv := makeEtcdClient(t)

	key := "foo"
	value := []byte("bar")

	// We should be able to create a new lock on key
	lock, err := kv.NewLock(key, &LockOptions{Value: value})
	if err != nil {
		t.Fatalf("cannot initialize lock: %v", err)
	}

	// Lock should successfully succeed or block
	lockChan, err := lock.Lock()
	if err != nil && lockChan == nil {
		t.Fatalf("unexpected lock failure: %v", err)
	}

	// Get should work
	pair, err := kv.Get(key)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if pair.Value == nil {
		t.Fatalf("expected value: %#v", value)
	}
	if !bytes.Equal(pair.Value, value) {
		t.Fatalf("unexpected value: %#v", pair.Value)
	}
	if pair.LastIndex == 0 {
		t.Fatalf("unexpected index: %#v", pair.LastIndex)
	}

	// Unlock should succeed
	if err := lock.Unlock(); err != nil {
		t.Fatalf("unexpected release failure: %v", err)
	}

	// Get should work
	pair, err = kv.Get(key)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if pair.Value == nil {
		t.Fatalf("expected value: %#v", value)
	}
	if !bytes.Equal(pair.Value, value) {
		t.Fatalf("unexpected value: %#v", pair.Value)
	}
	if pair.LastIndex == 0 {
		t.Fatalf("unexpected index: %#v", pair.LastIndex)
	}
}

func TestEtcdPutEphemeral(t *testing.T) {
	kv := makeEtcdClient(t)

	firstKey := "foo"
	firstValue := []byte("foo")

	// Put the first key with the Ephemeral flag
	if err := kv.Put(firstKey, firstValue, &WriteOptions{Ephemeral: true}); err != nil {
		t.Fatalf("err: %v", err)
	}

	secondKey := "bar"
	secondValue := []byte("bar")

	// Put a second key with the Ephemeral flag
	if err := kv.Put(secondKey, secondValue, &WriteOptions{Ephemeral: true}); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Get on firstKey should work
	pair, err := kv.Get(firstKey)
	if err != nil && pair == nil {
		t.Fatalf("err: %v", err)
	}

	// Get on secondKey should work
	pair, err = kv.Get(secondKey)
	if err != nil && pair == nil {
		t.Fatalf("err: %v", err)
	}

	// Let the session expire
	time.Sleep(3 * time.Second)

	// Get on firstKey shouldn't work
	pair, err = kv.Get(firstKey)
	if err == nil && pair != nil {
		t.Fatalf("unexpected success of Get for: %v", firstKey)
	}

	// Get on secondKey shouldn't work
	pair, err = kv.Get(secondKey)
	if err == nil && pair != nil {
		t.Fatalf("unexpected success of Get for: %v", secondKey)
	}
}
