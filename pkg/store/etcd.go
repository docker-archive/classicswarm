package store

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"time"

	etcd "github.com/coreos/go-etcd/etcd"
)

// Etcd embeds the client
type Etcd struct {
	client       *etcd.Client
	ephemeralTTL time.Duration
}

// InitializeEtcd creates a new Etcd client given
// a list of endpoints and optional tls config
func InitializeEtcd(addrs []string, options *Config) (Store, error) {
	s := &Etcd{}

	entries := createEndpoints(addrs, "http")
	s.client = etcd.NewClient(entries)

	// Set options
	if options != nil {
		if options.TLS != nil {
			s.setTLS(options.TLS)
		}
		if options.ConnectionTimeout != 0 {
			s.setTimeout(options.ConnectionTimeout)
		}
		if options.EphemeralTTL != 0 {
			s.setEphemeralTTL(options.EphemeralTTL)
		}
	}

	return s, nil
}

// SetTLS sets the tls configuration given the path
// of certificate files
func (s *Etcd) setTLS(tls *tls.Config) {
	// Change to https scheme
	var addrs []string
	entries := s.client.GetCluster()
	for _, entry := range entries {
		addrs = append(addrs, strings.Replace(entry, "http", "https", -1))
	}
	s.client.SetCluster(addrs)

	// Set transport
	t := http.Transport{
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second, // default timeout
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     tls,
	}
	s.client.SetTransport(&t)
}

// SetTimeout sets the timeout used for connecting to the store
func (s *Etcd) setTimeout(time time.Duration) {
	s.client.SetDialTimeout(time)
}

// SetHeartbeat sets the heartbeat value to notify we are alive
func (s *Etcd) setEphemeralTTL(time time.Duration) {
	s.ephemeralTTL = time
}

// Create the entire path for a directory that does not exist
func (s *Etcd) createDirectory(path string) error {
	if _, err := s.client.CreateDir(normalize(path), 10); err != nil {
		if etcdError, ok := err.(*etcd.EtcdError); ok {
			if etcdError.ErrorCode != 105 { // Skip key already exists
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

// Get the value at "key", returns the last modified index
// to use in conjunction to CAS calls
func (s *Etcd) Get(key string) (*KVPair, error) {
	result, err := s.client.Get(normalize(key), false, false)
	if err != nil {
		if etcdError, ok := err.(*etcd.EtcdError); ok {
			// Not a Directory or Not a file
			if etcdError.ErrorCode == 102 || etcdError.ErrorCode == 104 {
				return nil, ErrKeyNotFound
			}
		}
		return nil, err
	}
	return &KVPair{result.Node.Key, []byte(result.Node.Value), result.Node.ModifiedIndex}, nil
}

// Put a value at "key"
func (s *Etcd) Put(key string, value []byte, opts *WriteOptions) error {

	// Default TTL = 0 means no expiration
	var ttl uint64
	if opts != nil && opts.Ephemeral {
		ttl = uint64(s.ephemeralTTL.Seconds())
	}

	if _, err := s.client.Set(key, string(value), ttl); err != nil {
		if etcdError, ok := err.(*etcd.EtcdError); ok {
			if etcdError.ErrorCode == 104 { // Not a directory
				// Remove the last element (the actual key) and set the prefix as a dir
				err = s.createDirectory(getDirectory(key))
				if _, err := s.client.Set(key, string(value), ttl); err != nil {
					return err
				}
			}
		}
		return err
	}
	return nil
}

// Delete a value at "key"
func (s *Etcd) Delete(key string) error {
	if _, err := s.client.Delete(normalize(key), false); err != nil {
		return err
	}
	return nil
}

// Exists checks if the key exists inside the store
func (s *Etcd) Exists(key string) (bool, error) {
	entry, err := s.Get(key)
	if err != nil {
		if err == ErrKeyNotFound || entry.Value == nil {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Watch changes on a key.
// Returns a channel that will receive changes or an error.
// Upon creating a watch, the current value will be sent to the channel.
// Providing a non-nil stopCh can be used to stop watching.
func (s *Etcd) Watch(key string, stopCh <-chan struct{}) (<-chan *KVPair, error) {
	key = normalize(key)

	// Get the current value
	current, err := s.Get(key)
	if err != nil {
		return nil, err
	}

	// Start an etcd watch.
	// Note: etcd will send the current value through the channel.
	etcdWatchCh := make(chan *etcd.Response)
	etcdStopCh := make(chan bool)
	go s.client.Watch(key, 0, false, etcdWatchCh, etcdStopCh)

	// Adapter goroutine: The goal here is to convert wathever format etcd is
	// using into our interface.
	watchCh := make(chan *KVPair)
	go func() {
		defer close(watchCh)

		// Push the current value through the channel.
		watchCh <- current

		for {
			select {
			case result := <-etcdWatchCh:
				watchCh <- &KVPair{
					result.Node.Key,
					[]byte(result.Node.Value),
					result.Node.ModifiedIndex,
				}
			case <-stopCh:
				etcdStopCh <- true
				return
			}
		}
	}()
	return watchCh, nil
}

// WatchTree watches changes on a "directory"
// Returns a channel that will receive changes or an error.
// Upon creating a watch, the current value will be sent to the channel.
// Providing a non-nil stopCh can be used to stop watching.
func (s *Etcd) WatchTree(prefix string, stopCh <-chan struct{}) (<-chan []*KVPair, error) {
	prefix = normalize(prefix)

	// Get the current value
	current, err := s.List(prefix)
	if err != nil {
		return nil, err
	}

	// Start an etcd watch.
	etcdWatchCh := make(chan *etcd.Response)
	etcdStopCh := make(chan bool)
	go s.client.Watch(prefix, 0, true, etcdWatchCh, etcdStopCh)

	// Adapter goroutine: The goal here is to convert wathever format etcd is
	// using into our interface.
	watchCh := make(chan []*KVPair)
	go func() {
		defer close(watchCh)

		// Push the current value through the channel.
		watchCh <- current

		for {
			select {
			case <-etcdWatchCh:
				// FIXME: We should probably use the value pushed by the channel.
				// However, .Node.Nodes seems to be empty.
				if list, err := s.List(prefix); err == nil {
					watchCh <- list
				}
			case <-stopCh:
				etcdStopCh <- true
				return
			}
		}
	}()
	return watchCh, nil
}

// AtomicPut put a value at "key" if the key has not been
// modified in the meantime, throws an error if this is the case
func (s *Etcd) AtomicPut(key string, value []byte, previous *KVPair, options *WriteOptions) (bool, error) {
	_, err := s.client.CompareAndSwap(normalize(key), string(value), 0, "", previous.LastIndex)
	if err != nil {
		return false, err
	}
	return true, nil
}

// AtomicDelete deletes a value at "key" if the key has not
// been modified in the meantime, throws an error if this is the case
func (s *Etcd) AtomicDelete(key string, previous *KVPair) (bool, error) {
	_, err := s.client.CompareAndDelete(normalize(key), "", previous.LastIndex)
	if err != nil {
		return false, err
	}
	return true, nil
}

// List the content of a given prefix
func (s *Etcd) List(prefix string) ([]*KVPair, error) {
	resp, err := s.client.Get(normalize(prefix), true, true)
	if err != nil {
		return nil, err
	}
	kv := []*KVPair{}
	for _, n := range resp.Node.Nodes {
		kv = append(kv, &KVPair{n.Key, []byte(n.Value), n.ModifiedIndex})
	}
	return kv, nil
}

// DeleteTree deletes a range of keys based on prefix
func (s *Etcd) DeleteTree(prefix string) error {
	if _, err := s.client.Delete(normalize(prefix), true); err != nil {
		return err
	}
	return nil
}

// NewLock returns a handle to a lock struct which can be used to acquire and
// release the mutex.
func (s *Etcd) NewLock(key string, options *LockOptions) (Locker, error) {
	return nil, ErrNotImplemented
}
