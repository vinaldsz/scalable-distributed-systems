package store

import "sync"

// Entry holds a value and its logical version number.
type Entry struct {
	Value   string `json:"value"`
	Version int    `json:"version"`
}

// KVStore is a thread-safe in-memory key-value store with versioning.
type KVStore struct {
	mu   sync.RWMutex
	data map[string]Entry
}

func New() *KVStore {
	return &KVStore{data: make(map[string]Entry)}
}

// Get returns the entry for key, or (Entry{}, false) if absent.
func (s *KVStore) Get(key string) (Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	return e, ok
}

// Set writes a value. If version == 0 (unspecified), it auto-increments.
// Returns the version that was stored.
func (s *KVStore) Set(key, value string, version int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if version == 0 {
		if existing, ok := s.data[key]; ok {
			version = existing.Version + 1
		} else {
			version = 1
		}
	}
	s.data[key] = Entry{Value: value, Version: version}
	return version
}
