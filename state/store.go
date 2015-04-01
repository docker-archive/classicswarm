package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"

	log "github.com/Sirupsen/logrus"
)

var (
	// ErrNotFound is exported
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists is exported
	ErrAlreadyExists = errors.New("already exists")
	// ErrInvalidKey is exported
	ErrInvalidKey = errors.New("invalid key")
)

// Store is a simple key<->RequestedState store.
type Store struct {
	RootDir string
	values  map[string]*RequestedState

	sync.RWMutex
}

// NewStore is exported
func NewStore(rootdir string) *Store {
	return &Store{
		RootDir: rootdir,
		values:  make(map[string]*RequestedState),
	}
}

// Initialize must be called before performing any operation on the store. It
// will attempt to restore the data from disk.
func (s *Store) Initialize() error {
	s.Lock()
	defer s.Unlock()

	if err := os.MkdirAll(s.RootDir, 0700); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := s.restore(); err != nil {
		return err
	}

	return nil
}

func (s *Store) path(key string) string {
	return path.Join(s.RootDir, key+".json")
}

func (s *Store) restore() error {
	files, err := ioutil.ReadDir(s.RootDir)
	if err != nil {
		return err
	}
	for _, fileinfo := range files {
		file := fileinfo.Name()

		// Verify the file extension.
		extension := filepath.Ext(file)
		if extension != ".json" {
			log.Errorf("invalid file extension for filename %s (%s)", file, extension)
			continue
		}

		// Load the object back.
		value, err := s.load(path.Join(s.RootDir, file))
		if err != nil {
			log.Errorf(err.Error())
			continue
		}

		// Extract the key.
		key := file[0 : len(file)-len(extension)]
		if len(key) == 0 {
			log.Errorf("invalid filename %s", file)
			continue
		}

		// Store it back.
		s.values[key] = value
	}
	return nil
}

func (s *Store) load(file string) (*RequestedState, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("unable to load %s: %v", file, err)
	}
	value := &RequestedState{}
	if err := json.Unmarshal(data, value); err != nil {
		return nil, err
	}
	return value, nil
}

// Get an object from the store keyed by `key`.
func (s *Store) Get(key string) (*RequestedState, error) {
	s.RLock()
	defer s.RUnlock()

	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return nil, ErrNotFound
}

// All objects of the store are returned.
func (s *Store) All() []*RequestedState {
	s.RLock()
	defer s.RUnlock()

	states := make([]*RequestedState, 0, len(s.values))
	for _, state := range s.values {
		states = append(states, state)
	}
	return states
}

func (s *Store) set(key string, value *RequestedState) error {
	if len(key) == 0 {
		return ErrInvalidKey
	}

	data, err := json.MarshalIndent(value, "", "    ")
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(s.path(key), data, 0600); err != nil {
		return err
	}

	s.values[key] = value
	return nil
}

// Add a new object on the store. `key` must be unique.
func (s *Store) Add(key string, value *RequestedState) error {
	s.Lock()
	defer s.Unlock()

	if _, exists := s.values[key]; exists {
		return ErrAlreadyExists
	}

	return s.set(key, value)
}

// Replace an already existing object from the store.
func (s *Store) Replace(key string, value *RequestedState) error {
	s.Lock()
	defer s.Unlock()

	if _, exists := s.values[key]; !exists {
		return ErrNotFound
	}

	return s.set(key, value)
}

// Remove `key` from the store.
func (s *Store) Remove(key string) error {
	s.Lock()
	defer s.Unlock()

	if _, exists := s.values[key]; !exists {
		return ErrNotFound
	}

	if err := os.Remove(s.path(key)); err != nil {
		return err
	}

	delete(s.values, key)
	return nil
}
