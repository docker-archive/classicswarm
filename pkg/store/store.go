package store

import (
	"time"

	log "github.com/Sirupsen/logrus"
)

// Backend represents a KV Store Backend
type Backend string

const (
	// CONSUL backend
	CONSUL Backend = "consul"
	// ETCD backend
	ETCD = "etcd"
	// ZK backend
	ZK = "zk"
)

// WatchCallback is used for watch methods on keys
// and is triggered on key change
type WatchCallback func(entries ...*KVEntry)

// Initialize creates a new Store object, initializing the client
type Initialize func(addrs []string, options *Config) (Store, error)

// Store represents the backend K/V storage
// Each store should support every call listed
// here. Or it couldn't be implemented as a K/V
// backend for libkv
type Store interface {
	// Put a value at the specified key
	Put(key string, value []byte) error

	// Get a value given its key
	Get(key string) (*KVEntry, error)

	// Delete the value at the specified key
	Delete(key string) error

	// Verify if a Key exists in the store
	Exists(key string) (bool, error)

	// Watch changes on a key
	Watch(key string, heartbeat time.Duration, callback WatchCallback) error

	// Cancel watch key
	CancelWatch(key string) error

	// CreateLock for a given key.
	// The returned Locker is not held and must be acquired with `.Lock`.
	// value is optional.
	CreateLock(key string, value []byte) (Locker, error)

	// Get range of keys based on prefix
	List(prefix string) ([]*KVEntry, error)

	// Delete range of keys based on prefix
	DeleteTree(prefix string) error

	// Watch key namespaces
	WatchRange(prefix string, filter string, heartbeat time.Duration, callback WatchCallback) error

	// Cancel watch key range
	CancelWatchRange(prefix string) error

	// Atomic operation on a single value
	AtomicPut(key string, value []byte, previous *KVEntry) (bool, error)

	// Atomic delete of a single value
	AtomicDelete(key string, previous *KVEntry) (bool, error)
}

// KVEntry represents {Key, Value, Lastindex} tuple
type KVEntry struct {
	Key       string
	Value     []byte
	LastIndex uint64
}

// Locker provides locking mechanism on top of the store.
// Similar to `sync.Lock` except it may return errors.
type Locker interface {
	Lock() (<-chan struct{}, error)
	Unlock() error
}

var (
	// Backend initializers
	initializers = map[Backend]Initialize{
		CONSUL: InitializeConsul,
		ETCD:   InitializeEtcd,
		ZK:     InitializeZookeeper,
	}
)

// CreateStore creates a an instance of store
func CreateStore(backend Backend, addrs []string, options *Config) (Store, error) {
	if init, exists := initializers[backend]; exists {
		log.WithFields(log.Fields{"backend": backend}).Debug("Initializing store service")
		return init(addrs, options)
	}

	return nil, ErrNotSupported
}
