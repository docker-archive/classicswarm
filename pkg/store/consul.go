package store

import (
	"crypto/tls"
	"net/http"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	api "github.com/hashicorp/consul/api"
)

// Consul embeds the client and watches
type Consul struct {
	config  *api.Config
	client  *api.Client
	watches map[string]*Watch
}

type consulLock struct {
	lock *api.Lock
}

// Watch embeds the event channel and the
// refresh interval
type Watch struct {
	LastIndex uint64
	Interval  time.Duration
}

// InitializeConsul creates a new Consul client given
// a list of endpoints and optional tls config
func InitializeConsul(endpoints []string, options *Config) (Store, error) {
	s := &Consul{}
	s.watches = make(map[string]*Watch)

	// Create Consul client
	config := api.DefaultConfig()
	s.config = config
	config.HttpClient = http.DefaultClient
	config.Address = endpoints[0]
	config.Scheme = "http"

	if options.TLS != nil {
		s.setTLS(options.TLS)
	}

	if options.Timeout != 0 {
		s.setTimeout(options.Timeout)
	}

	// Creates a new client
	client, err := api.NewClient(config)
	if err != nil {
		log.Errorf("Couldn't initialize consul client..")
		return nil, err
	}
	s.client = client

	return s, nil
}

// SetTLS sets Consul TLS options
func (s *Consul) setTLS(tls *tls.Config) {
	s.config.HttpClient.Transport = &http.Transport{
		TLSClientConfig: tls,
	}
	s.config.Scheme = "https"
}

// SetTimeout sets the timout for connecting to Consul
func (s *Consul) setTimeout(time time.Duration) {
	s.config.WaitTime = time
}

// Normalize the key for usage in Consul
func (s *Consul) normalize(key string) string {
	key = normalize(key)
	return strings.TrimPrefix(key, "/")
}

// Get the value at "key", returns the last modified index
// to use in conjunction to CAS calls
func (s *Consul) Get(key string) (*KVEntry, error) {
	pair, meta, err := s.client.KV().Get(s.normalize(key), nil)
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, ErrKeyNotFound
	}
	return &KVEntry{pair.Key, pair.Value, meta.LastIndex}, nil
}

// Put a value at "key"
func (s *Consul) Put(key string, value []byte) error {
	p := &api.KVPair{Key: s.normalize(key), Value: value}
	if s.client == nil {
		log.Error("Error initializing client")
	}
	_, err := s.client.KV().Put(p, nil)
	return err
}

// Delete a value at "key"
func (s *Consul) Delete(key string) error {
	_, err := s.client.KV().Delete(s.normalize(key), nil)
	return err
}

// Exists checks that the key exists inside the store
func (s *Consul) Exists(key string) (bool, error) {
	_, err := s.Get(key)
	if err != nil && err == ErrKeyNotFound {
		return false, err
	}
	return true, nil
}

// GetRange gets a range of values at "directory"
func (s *Consul) List(prefix string) ([]*KVEntry, error) {
	pairs, _, err := s.client.KV().List(s.normalize(prefix), nil)
	if err != nil {
		return nil, err
	}
	if len(pairs) == 0 {
		return nil, ErrKeyNotFound
	}
	kv := []*KVEntry{}
	for _, pair := range pairs {
		if pair.Key == prefix {
			continue
		}
		kv = append(kv, &KVEntry{pair.Key, pair.Value, pair.ModifyIndex})
	}
	return kv, nil
}

// DeleteRange deletes a range of values at "directory"
func (s *Consul) DeleteRange(prefix string) error {
	_, err := s.client.KV().DeleteTree(s.normalize(prefix), nil)
	return err
}

// Watch a single key for modifications
func (s *Consul) Watch(key string, heartbeat time.Duration, callback WatchCallback) error {
	fkey := s.normalize(key)

	// We get the last index first
	_, meta, err := s.client.KV().Get(fkey, nil)
	if err != nil {
		return err
	}

	// Add watch to map
	s.watches[fkey] = &Watch{LastIndex: meta.LastIndex, Interval: heartbeat}
	eventChan := s.waitForChange(fkey)

	for _ = range eventChan {
		log.WithField("name", "consul").Debug("Key watch triggered")
		entry, err := s.Get(key)
		if err != nil {
			log.Error("Cannot refresh the key: ", fkey, ", cancelling watch")
			s.watches[fkey] = nil
			return err
		}
		callback(entry)
	}

	return nil
}

// CancelWatch cancels a watch, sends a signal to the appropriate
// stop channel
func (s *Consul) CancelWatch(key string) error {
	key = s.normalize(key)
	if _, ok := s.watches[key]; !ok {
		log.Error("Chan does not exist for key: ", key)
		return ErrWatchDoesNotExist
	}
	s.watches[key] = nil
	return nil
}

// Internal function to check if a key has changed
func (s *Consul) waitForChange(key string) <-chan uint64 {
	ch := make(chan uint64)
	kv := s.client.KV()
	go func() {
		for {
			watch, ok := s.watches[key]
			if !ok {
				log.Error("Cannot access last index for key: ", key, " closing channel")
				break
			}
			option := &api.QueryOptions{
				WaitIndex: watch.LastIndex,
				WaitTime:  watch.Interval,
			}
			_, meta, err := kv.List(key, option)
			if err != nil {
				log.WithField("name", "consul").Errorf("Discovery error: %v", err)
				break
			}
			watch.LastIndex = meta.LastIndex
			ch <- watch.LastIndex
		}
		close(ch)
	}()
	return ch
}

// WatchRange triggers a watch on a range of values at "directory"
func (s *Consul) WatchRange(prefix string, filter string, heartbeat time.Duration, callback WatchCallback) error {
	fprefix := s.normalize(prefix)

	// We get the last index first
	_, meta, err := s.client.KV().Get(prefix, nil)
	if err != nil {
		return err
	}

	// Add watch to map
	s.watches[fprefix] = &Watch{LastIndex: meta.LastIndex, Interval: heartbeat}
	eventChan := s.waitForChange(fprefix)

	for _ = range eventChan {
		log.WithField("name", "consul").Debug("Key watch triggered")
		kvi, err := s.List(prefix)
		if err != nil {
			log.Error("Cannot refresh keys with prefix: ", fprefix, ", cancelling watch")
			s.watches[fprefix] = nil
			return err
		}
		callback(kvi...)
	}

	return nil
}

// CancelWatchRange stops the watch on the range of values, sends
// a signal to the appropriate stop channel
func (s *Consul) CancelWatchRange(prefix string) error {
	return s.CancelWatch(prefix)
}

// CreateLock returns a handle to a lock struct which can be used
// to acquire and release the mutex.
func (s *Consul) CreateLock(key string, value []byte) (Locker, error) {
	l, err := s.client.LockOpts(&api.LockOptions{
		Key:   s.normalize(key),
		Value: value,
	})
	if err != nil {
		return nil, err
	}
	return &consulLock{lock: l}, nil
}

// Lock attempts to acquire the lock and blocks while doing so.
// Returns a channel that is closed if our lock is lost or an error.
func (l *consulLock) Lock() (<-chan struct{}, error) {
	return l.lock.Lock(nil)
}

// Unlock released the lock. It is an error to call this
// if the lock is not currently held.
func (l *consulLock) Unlock() error {
	return l.lock.Unlock()
}

// AtomicPut put a value at "key" if the key has not been
// modified in the meantime, throws an error if this is the case
func (s *Consul) AtomicPut(key string, value []byte, previous *KVEntry) (bool, error) {
	p := &api.KVPair{Key: s.normalize(key), Value: value, ModifyIndex: previous.LastIndex}
	if work, _, err := s.client.KV().CAS(p, nil); err != nil {
		return false, err
	} else if !work {
		return false, ErrKeyModified
	}
	return true, nil
}

// AtomicDelete deletes a value at "key" if the key has not
// been modified in the meantime, throws an error if this is the case
func (s *Consul) AtomicDelete(key string, previous *KVEntry) (bool, error) {
	p := &api.KVPair{Key: s.normalize(key), ModifyIndex: previous.LastIndex}
	if work, _, err := s.client.KV().DeleteCAS(p, nil); err != nil {
		return false, err
	} else if !work {
		return false, ErrKeyModified
	}
	return true, nil
}
