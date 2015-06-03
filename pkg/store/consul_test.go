package store

import (
	"bytes"
	"testing"
	"time"
)

func makeConsulClient(t *testing.T) Store {
	client := "localhost:8500"

	// Initialize a new store with consul
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

func TestConsulPutGetDelete(t *testing.T) {
	kv := makeConsulClient(t)

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

func TestConsulCompareAndSwap(t *testing.T) {
	kv := makeConsulClient(t)

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

func TestConsulLockUnlock(t *testing.T) {
	t.Parallel()
	kv := makeConsulClient(t)

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

func TestConsulCreateEphemeralSession(t *testing.T) {
	kv := makeConsulClient(t)
	consul := kv.(*Consul)

	// createEphemeralSession should create a valid session
	session, err := consul.createEphemeralSession()
	if err != nil && session == "" {
		t.Fatalf("err: %v", err)
	}
}

func TestConsulPutEphemeral(t *testing.T) {
	kv := makeConsulClient(t)

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
	time.Sleep(4 * time.Second)

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

	// Clean keys
	if err := kv.Delete("ploup"); err != nil {
		t.Fatalf("err: %v", err)
	}
	if err := kv.Delete("plop"); err != nil {
		t.Fatalf("err: %v", err)
	}
}
