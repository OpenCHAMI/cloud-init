package bss

import (
	"errors"
	"sync"
)

// Store defines the interface for storing and retrieving boot parameters
type Store interface {
	// Set stores boot parameters with the given ID
	Set(id string, params *BootParams) error
	// Get retrieves boot parameters by ID
	Get(id string) (*BootParams, error)
	// Update updates existing boot parameters and increments version
	Update(id string, params *BootParams) error
}

// MemoryStore is an in-memory implementation of the Store interface
type MemoryStore struct {
	mu     sync.RWMutex
	params map[string]*BootParams
}

// NewMemoryStore creates a new MemoryStore
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		params: make(map[string]*BootParams),
	}
}

// Set stores boot parameters with the given ID
func (s *MemoryStore) Set(id string, params *BootParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.params[id]; exists {
		return errors.New("boot parameters with this ID already exist")
	}

	// Set initial version
	params.Version = 1
	s.params[id] = params
	return nil
}

// Get retrieves boot parameters by ID
func (s *MemoryStore) Get(id string) (*BootParams, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	params, exists := s.params[id]
	if !exists {
		return nil, errors.New("boot parameters not found")
	}

	return params, nil
}

// Update updates existing boot parameters and increments version
func (s *MemoryStore) Update(id string, params *BootParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.params[id]
	if !exists {
		return errors.New("boot parameters not found")
	}

	// Preserve the ID and increment version
	params.Version = existing.Version + 1
	s.params[id] = params
	return nil
}
