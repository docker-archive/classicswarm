package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func makeEtcdClient(t *testing.T) Store {
	client := "localhost:4001"

	kv, err := NewStore(
		ETCD,
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

func TestEtcdStore(t *testing.T) {
	kv := makeEtcdClient(t)

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

func TestCreateDirectory(t *testing.T) {
	kv := makeEtcdClient(t)

	etcd := kv.(*Etcd)

	dir := "create/me/please"

	// Create the directory
	err := etcd.createDirectory(normalize(dir))
	assert.NoError(t, err)

	// The directory must exist
	pair, err := kv.Get(dir)
	assert.NoError(t, err)
	if assert.NotNil(t, pair) {
		assert.NotNil(t, pair.Value)
	}
	assert.Equal(t, pair.Value, []byte(""))
	assert.NotEqual(t, pair.LastIndex, 0)
}
