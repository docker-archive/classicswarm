package store

import (
	"crypto/tls"
	"net/http"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	api "github.com/hashicorp/consul/api"
)

const (
	// DefaultWatchWaitTime is how long we block for at a time to check if the
	// watched key has changed.  This affects the minimum time it takes to
	// cancel a watch.
	DefaultWatchWaitTime = 15 * time.Second
)

// Consul embeds the client and watches
type Consul struct {
	config           *api.Config
	client           *api.Client
	ephemeralSession string
}

type consulLock struct {
	lock *api.Lock
}

// InitializeConsul creates a new Consul client given
// a list of endpoints and optional tls config
func InitializeConsul(endpoints []string, options *Config) (Store, error) {
	s := &Consul{}

	// Create Consul client
	config := api.DefaultConfig()
	s.config = config
	config.HttpClient = http.DefaultClient
	config.Address = endpoints[0]
	config.Scheme = "http"

	// Set options
	if options != nil {
		if options.TLS != nil {
			s.setTLS(options.TLS)
		}
		if options.ConnectionTimeout != 0 {
			s.setTimeout(options.ConnectionTimeout)
		}
	}

	// Creates a new client
	client, err := api.NewClient(config)
	if err != nil {
		log.Errorf("Couldn't initialize consul client..")
		return nil, err
	}
	s.client = client

	// Create global ephemeral keys session
	entry := &api.SessionEntry{
		Behavior: api.SessionBehaviorDelete,
		TTL:      time.Duration(EphemeralTTL * time.Second).String(),
	}
	session, _, err := s.client.Session().Create(entry, nil)
	if err != nil {
		return nil, err
	}
	s.ephemeralSession = session

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
func (s *Consul) Get(key string) (*KVPair, error) {
	pair, meta, err := s.client.KV().Get(s.normalize(key), nil)
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, ErrKeyNotFound
	}
	return &KVPair{pair.Key, pair.Value, meta.LastIndex}, nil
}

// Put a value at "key"
func (s *Consul) Put(key string, value []byte, opts *WriteOptions) error {
	p := &api.KVPair{
		Key:   s.normalize(key),
		Value: value,
	}

	if opts != nil && opts.Ephemeral {
		p.Session = s.ephemeralSession

		// Create lock option with the
		// EphemeralSession
		lockOpts := &api.LockOptions{
			Key:     key,
			Session: s.ephemeralSession,
		}

		// Lock and ignore if lock is held
		// It's just a placeholder for the
		// ephemeral behavior
		lock, _ := s.client.LockOpts(lockOpts)
		if lock != nil {
			lock.Lock(nil)
		}

		// Try to renew the session
		_, _, err := s.client.Session().Renew(p.Session, nil)
		if err != nil {
			return err
		}
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

// List the content of a given prefix
func (s *Consul) List(prefix string) ([]*KVPair, error) {
	pairs, _, err := s.client.KV().List(s.normalize(prefix), nil)
	if err != nil {
		return nil, err
	}
	if len(pairs) == 0 {
		return nil, ErrKeyNotFound
	}
	kv := []*KVPair{}
	for _, pair := range pairs {
		if pair.Key == prefix {
			continue
		}
		kv = append(kv, &KVPair{pair.Key, pair.Value, pair.ModifyIndex})
	}
	return kv, nil
}

// DeleteTree deletes a range of keys based on prefix
func (s *Consul) DeleteTree(prefix string) error {
	_, err := s.client.KV().DeleteTree(s.normalize(prefix), nil)
	return err
}

// Watch changes on a key.
// Returns a channel that will receive changes or an error.
// Upon creating a watch, the current value will be sent to the channel.
// Providing a non-nil stopCh can be used to stop watching.
func (s *Consul) Watch(key string, stopCh <-chan struct{}) (<-chan *KVPair, error) {
	key = s.normalize(key)
	kv := s.client.KV()
	watchCh := make(chan *KVPair)

	go func() {
		defer close(watchCh)

		// Use a wait time in order to check if we should quit from time to
		// time.
		opts := &api.QueryOptions{WaitTime: DefaultWatchWaitTime}
		for {
			// Check if we should quit
			select {
			case <-stopCh:
				return
			default:
			}
			pair, meta, err := kv.Get(key, opts)
			if err != nil {
				log.Errorf("consul: %v", err)
				return
			}
			// If LastIndex didn't change then it means `Get` returned because
			// of the WaitTime and the key didn't change.
			if opts.WaitIndex == meta.LastIndex {
				continue
			}
			opts.WaitIndex = meta.LastIndex
			// FIXME: What happens when a key is deleted?
			if pair != nil {
				watchCh <- &KVPair{pair.Key, pair.Value, pair.ModifyIndex}
			}
		}
	}()

	return watchCh, nil
}

// WatchTree watches changes on a "directory"
// Returns a channel that will receive changes or an error.
// Upon creating a watch, the current value will be sent to the channel.
// Providing a non-nil stopCh can be used to stop watching.
func (s *Consul) WatchTree(prefix string, stopCh <-chan struct{}) (<-chan []*KVPair, error) {
	prefix = s.normalize(prefix)
	kv := s.client.KV()
	watchCh := make(chan []*KVPair)

	go func() {
		defer close(watchCh)

		// Use a wait time in order to check if we should quit from time to
		// time.
		opts := &api.QueryOptions{WaitTime: DefaultWatchWaitTime}
		for {
			// Check if we should quit
			select {
			case <-stopCh:
				return
			default:
			}

			pairs, meta, err := kv.List(prefix, opts)
			if err != nil {
				log.Errorf("consul: %v", err)
				return
			}
			// If LastIndex didn't change then it means `Get` returned because
			// of the WaitTime and the key didn't change.
			if opts.WaitIndex == meta.LastIndex {
				continue
			}
			opts.WaitIndex = meta.LastIndex
			kv := []*KVPair{}
			for _, pair := range pairs {
				if pair.Key == prefix {
					continue
				}
				kv = append(kv, &KVPair{pair.Key, pair.Value, pair.ModifyIndex})
			}
			watchCh <- kv
		}
	}()

	return watchCh, nil
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
func (s *Consul) AtomicPut(key string, value []byte, previous *KVPair, options *WriteOptions) (bool, error) {
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
func (s *Consul) AtomicDelete(key string, previous *KVPair) (bool, error) {
	p := &api.KVPair{Key: s.normalize(key), ModifyIndex: previous.LastIndex}
	if work, _, err := s.client.KV().DeleteCAS(p, nil); err != nil {
		return false, err
	} else if !work {
		return false, ErrKeyModified
	}
	return true, nil
}
