package store

import (
	"crypto/tls"
	"errors"
	"time"
)

var (
	// ErrNotSupported is exported
	ErrNotSupported = errors.New("Backend storage not supported yet, please choose another one")
	// ErrNotImplemented is exported
	ErrNotImplemented = errors.New("Call not implemented in current backend")
	// ErrNotReachable is exported
	ErrNotReachable = errors.New("Api not reachable")
	// ErrCannotLock is exported
	ErrCannotLock = errors.New("Error acquiring the lock")
	// ErrWatchDoesNotExist is exported
	ErrWatchDoesNotExist = errors.New("No watch found for specified key")
	// ErrKeyModified is exported
	ErrKeyModified = errors.New("Unable to complete atomic operation, key modified")
	// ErrKeyNotFound is exported
	ErrKeyNotFound = errors.New("Key not found in store")
)

// KV represents the different supported K/V
type KV string

const (
	// CONSUL is exported
	CONSUL KV = "consul"
	// ETCD is exported
	ETCD = "etcd"
	// ZOOKEEPER is exported
	ZOOKEEPER = "zookeeper"
)

// TLSConfig takes a built tls.Config object
func TLSConfig(tls *tls.Config) interface{} {
	return tls
}

// Timeout takes a timout for client Initialization
func Timeout(time time.Duration) interface{} {
	return time
}
