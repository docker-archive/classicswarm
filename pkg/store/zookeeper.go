package store

import (
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	zk "github.com/samuel/go-zookeeper/zk"
)

// Zookeeper embeds the zookeeper client
// and list of watches
type Zookeeper struct {
	timeout time.Duration
	client  *zk.Conn
	watches map[string]<-chan zk.Event
}

type zookeeperLock struct {
	lock *zk.Lock
}

// InitializeZookeeper creates a new Zookeeper client
// given a list of endpoints and optional tls config
func InitializeZookeeper(endpoints []string, options *Config) (Store, error) {
	s := &Zookeeper{}
	s.watches = make(map[string]<-chan zk.Event)
	s.timeout = 5 * time.Second // default timeout

	if options.Timeout != 0 {
		s.setTimeout(options.Timeout)
	}

	conn, _, err := zk.Connect(endpoints, s.timeout)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	s.client = conn
	return s, nil
}

// SetTimeout sets the timout for connecting to Zookeeper
func (s *Zookeeper) setTimeout(time time.Duration) {
	s.timeout = time
}

// Get the value at "key", returns the last modified index
// to use in conjunction to CAS calls
func (s *Zookeeper) Get(key string) (*KVEntry, error) {
	resp, meta, err := s.client.Get(normalize(key))
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, ErrKeyNotFound
	}
	return &KVEntry{key, resp, uint64(meta.Mzxid)}, nil
}

// Create the entire path for a directory that does not exist
func (s *Zookeeper) createFullpath(path []string) error {
	for i := 1; i <= len(path); i++ {
		newpath := "/" + strings.Join(path[:i], "/")
		_, err := s.client.Create(newpath, []byte{1}, 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			// Skip if node already exists
			if err != zk.ErrNodeExists {
				return err
			}
		}
	}
	return nil
}

// Put a value at "key"
func (s *Zookeeper) Put(key string, value []byte) error {
	fkey := normalize(key)
	exists, err := s.Exists(key)
	if err != nil {
		return err
	}
	if !exists {
		s.createFullpath(splitKey(key))
	}
	_, err = s.client.Set(fkey, value, -1)
	return err
}

// Delete a value at "key"
func (s *Zookeeper) Delete(key string) error {
	err := s.client.Delete(normalize(key), -1)
	return err
}

// Exists checks if the key exists inside the store
func (s *Zookeeper) Exists(key string) (bool, error) {
	exists, _, err := s.client.Exists(normalize(key))
	if err != nil {
		return false, err
	}
	return exists, nil
}

// Watch a single key for modifications
func (s *Zookeeper) Watch(key string, _ time.Duration, callback WatchCallback) error {
	fkey := normalize(key)
	_, _, eventChan, err := s.client.GetW(fkey)
	if err != nil {
		return err
	}

	// Create a new Watch entry with eventChan
	s.watches[fkey] = eventChan

	for e := range eventChan {
		if e.Type == zk.EventNodeChildrenChanged {
			log.WithField("name", "zk").Debug("Discovery watch triggered")
			entry, err := s.Get(key)
			if err == nil {
				callback(entry)
			}
		}
	}

	return nil
}

// CancelWatch cancels a watch, sends a signal to the appropriate
// stop channel
func (s *Zookeeper) CancelWatch(key string) error {
	key = normalize(key)
	if _, ok := s.watches[key]; !ok {
		log.Error("Chan does not exist for key: ", key)
		return ErrWatchDoesNotExist
	}
	// Just remove the entry on watches key
	s.watches[key] = nil
	return nil
}

// GetRange gets a range of values at "directory"
func (s *Zookeeper) List(prefix string) ([]*KVEntry, error) {
	prefix = normalize(prefix)
	entries, stat, err := s.client.Children(prefix)
	if err != nil {
		log.Error("Cannot fetch range of keys beginning with prefix: ", prefix)
		return nil, err
	}
	kv := []*KVEntry{}
	for _, item := range entries {
		kv = append(kv, &KVEntry{prefix, []byte(item), uint64(stat.Mzxid)})
	}
	return kv, err
}

// DeleteRange deletes a range of values at "directory"
func (s *Zookeeper) DeleteTree(prefix string) error {
	err := s.client.Delete(normalize(prefix), -1)
	return err
}

// WatchRange triggers a watch on a range of values at "directory"
func (s *Zookeeper) WatchTree(prefix string, filter string, _ time.Duration, callback WatchCallback) error {
	fprefix := normalize(prefix)
	_, _, eventChan, err := s.client.ChildrenW(fprefix)
	if err != nil {
		return err
	}

	// Create a new Watch entry with eventChan
	s.watches[fprefix] = eventChan

	for e := range eventChan {
		if e.Type == zk.EventNodeChildrenChanged {
			log.WithField("name", "zk").Debug("Discovery watch triggered")
			kvi, err := s.List(prefix)
			if err == nil {
				callback(kvi...)
			}
		}
	}

	return nil
}

// CancelWatchRange stops the watch on the range of values, sends
// a signal to the appropriate stop channel
func (s *Zookeeper) CancelWatchRange(prefix string) error {
	return s.CancelWatch(prefix)
}

// AtomicPut put a value at "key" if the key has not been
// modified in the meantime, throws an error if this is the case
func (s *Zookeeper) AtomicPut(key string, value []byte, previous *KVEntry) (bool, error) {
	// Use index of Set method to implement CAS
	return false, ErrNotImplemented
}

// AtomicDelete deletes a value at "key" if the key has not
// been modified in the meantime, throws an error if this is the case
func (s *Zookeeper) AtomicDelete(key string, previous *KVEntry) (bool, error) {
	return false, ErrNotImplemented
}

// CreateLock returns a handle to a lock struct which can be used
// to acquire and release the mutex.
func (s *Zookeeper) CreateLock(key string, value []byte) (Locker, error) {
	// FIXME: `value` is not being used since there is no support in zk.NewLock().
	return &zookeeperLock{
		lock: zk.NewLock(s.client, normalize(key), zk.WorldACL(zk.PermAll)),
	}, nil
}

// Lock attempts to acquire the lock and blocks while doing so.
// Returns a channel that is closed if our lock is lost or an error.
func (l *zookeeperLock) Lock() (<-chan struct{}, error) {
	if err := l.lock.Lock(); err != nil {
		return nil, err
	}
	return make(<-chan struct{}), nil
}

// Unlock released the lock. It is an error to call this
// if the lock is not currently held.
func (l *zookeeperLock) Unlock() error {
	return l.lock.Unlock()
}
