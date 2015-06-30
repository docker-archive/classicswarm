// Package libkv provides a Go native library to store metadata.
//
// The goal of libkv is to abstract common store operations for multiple
// Key/Value backends and offer the same experience no matter which one of the
// backend you want to use.
//
// For example, you can use it to store your metadata or for service discovery to
// register machines and endpoints inside your cluster.
//
// As of now, `libkv` offers support for `Consul`, `Etcd` and `Zookeeper`.
//
// ## Example of usage
//
// ### Create a new store and use Put/Get
//
//
//     package main
//
//     import (
//     	"fmt"
//     	"time"
//
//     	"github.com/docker/libkv"
//     	"github.com/docker/libkv/store"
//     	log "github.com/Sirupsen/logrus"
//     )
//
//     func main() {
//     	client := "localhost:8500"
//
//     	// Initialize a new store with consul
//     	kv, err := libkv.NewStore(
//     		store.CONSUL, // or "consul"
//     		[]string{client},
//     		&store.Config{
//     			ConnectionTimeout: 10*time.Second,
//     		},
//     	)
//     	if err != nil {
//     		log.Fatal("Cannot create store consul")
//     	}
//
//     	key := "foo"
//     	err = kv.Put(key, []byte("bar"), nil)
//     	if err != nil {
//     		log.Error("Error trying to put value at key `", key, "`")
//     	}
//
//     	pair, err := kv.Get(key)
//     	if err != nil {
//     		log.Error("Error trying accessing value at key `", key, "`")
//     	}
//
//     	log.Info("value: ", string(pair.Value))
//     }
//
// ##Copyright and license
//
// Code and documentation copyright 2015 Docker, inc. Code released under the
// Apache 2.0 license. Docs released under Creative commons.
//
package libkv

import (
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/consul"
	"github.com/docker/libkv/store/etcd"
	"github.com/docker/libkv/store/zookeeper"
)

// Initialize creates a new Store object, initializing the client
type Initialize func(addrs []string, options *store.Config) (store.Store, error)

var (
	// Backend initializers
	initializers = map[store.Backend]Initialize{
		store.CONSUL: consul.New,
		store.ETCD:   etcd.New,
		store.ZK:     zookeeper.New,
	}
)

// NewStore creates a an instance of store
func NewStore(backend store.Backend, addrs []string, options *store.Config) (store.Store, error) {
	if init, exists := initializers[backend]; exists {
		return init(addrs, options)
	}

	return nil, store.ErrNotSupported
}
