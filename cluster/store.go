package cluster

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

// A simple key<->Container store.
type Store struct {
	RootDir    string
	containers map[string]*Container

	sync.RWMutex
}

func NewStore(rootdir string) *Store {
	return &Store{
		RootDir:    rootdir,
		containers: make(map[string]*Container),
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
			return fmt.Errorf("invalid file extension for filename %s (%s)", file, extension)
		}

		// Load the object back.
		container, err := s.load(path.Join(s.RootDir, file))
		if err != nil {
			return err
		}

		// Extract the key.
		key := file[0 : len(file)-len(extension)]
		if len(key) == 0 {
			return fmt.Errorf("invalid filename %s", file)
		}

		// Store it back.
		s.containers[key] = container
	}
	return nil
}

func (s *Store) load(file string) (*Container, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("unable to load %s: %v", file, err)
	}
	container := &Container{}
	if err := json.Unmarshal(data, container); err != nil {
		return nil, err
	}
	return container, nil
}

// Retrieves an object from the store keyed by `key`.
func (s *Store) Get(key string) (*Container, error) {
	s.RLock()
	defer s.RUnlock()

	if value, ok := s.containers[key]; ok {
		return value, nil
	}
	return nil, ErrNotFound
}

// Return all objects of the store.
func (s *Store) All() []*Container {
	s.RLock()
	defer s.RUnlock()

	states := make([]*Container, len(s.containers))
	i := 0
	for _, state := range s.containers {
		states[i] = state
		i = i + 1
	}
	return states
}

func (s *Store) set(key string, value *Container) error {
	data, err := json.MarshalIndent(value, "", "    ")
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(s.path(key), data, 0600); err != nil {
		return err
	}

	s.containers[key] = value
	return nil
}

// Add a new object on the store. `key` must be unique.
func (s *Store) Add(key string, value *Container) error {
	s.Lock()
	defer s.Unlock()

	if _, exists := s.containers[key]; exists {
		return ErrAlreadyExists
	}

	return s.set(key, value)
}

// Replaces an already existing object from the store.
func (s *Store) Replace(key string, value *Container) error {
	s.Lock()
	defer s.Unlock()

	if _, exists := s.containers[key]; !exists {
		return ErrNotFound
	}

	return s.set(key, value)
}

// Remove `key` from the store.
func (s *Store) Remove(key string) error {
	s.Lock()
	defer s.Unlock()

	if _, exists := s.containers[key]; !exists {
		return ErrNotFound
	}

	if err := os.Remove(s.path(key)); err != nil {
		return err
	}

	delete(s.containers, key)
	return nil
}
