package bss

import (
	"errors"
	"sync"
)

// VersionedBootParams represents a versioned set of boot parameters
type VersionedBootParams struct {
	CurrentVersion int          `json:"current_version"` // Current active version
	DefaultVersion int          `json:"default_version"` // Default version to use
	Versions       []BootParams `json:"versions"`        // History of all versions
}

// Store defines the interface for storing and retrieving boot parameters
type Store interface {
	// Set stores boot parameters with the given ID and creates a new version
	Set(id string, params *BootParams) error
	// Get retrieves the latest version of boot parameters by ID
	Get(id string) (*BootParams, error)
	// GetV1 retrieves the latest version of V1 boot parameters by xname
	GetV1(xname string) (*BootParamsV1, error)
	// SetV1 stores V1 boot parameters by xname
	SetV1(xname string, params *BootParamsV1) error
	// GetVersion retrieves a specific version of boot parameters by ID
	GetVersion(id string, version int) (*BootParams, error)
	// GetDefault retrieves the default version of boot parameters by ID
	GetDefault(id string) (*BootParams, error)
	// Update stores a new version of existing boot parameters
	Update(id string, params *BootParams) error
	// SetDefault sets the default version for a boot parameter set
	SetDefault(id string, version int) error
	// Delete deletes all versions of boot parameters by ID
	Delete(id string) error
	// AssignTemplateToGroup assigns a versioned template to a group. Passing 0 for version will assign the latest version.
	AssignTemplateToGroup(paramId, group string, version int) error
	// GetTemplateForGroup retrieves the template for a group
	GetTemplateForGroup(group string) (*BootParams, error)
}

// MemoryStore is an in-memory implementation of the Store interface
type MemoryStore struct {
	mu     sync.RWMutex
	params map[string]*VersionedBootParams
	v1     map[string]*BootParamsV1 // maps an xname to a boot params v1 struct
	v2     map[string]struct {
		BootParams *VersionedBootParams
		Version    int
	}
}

// NewMemoryStore creates a new MemoryStore
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		params: make(map[string]*VersionedBootParams),
		v1:     make(map[string]*BootParamsV1),
	}
}

// Set stores boot parameters with the given ID
func (s *MemoryStore) Set(id string, params *BootParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.params[id]; exists {
		return errors.New("boot parameters with this ID already exist")
	}

	// Initialize versioned boot parameters
	versioned := &VersionedBootParams{
		CurrentVersion: 1,
		DefaultVersion: 1,
		Versions:       []BootParams{*params},
	}

	// Set initial version
	versioned.Versions[0].Version = 1
	s.params[id] = versioned
	return nil
}

// Get retrieves the latest version of boot parameters by ID
func (s *MemoryStore) Get(id string) (*BootParams, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	versioned, exists := s.params[id]
	if !exists {
		return nil, errors.New("boot parameters not found")
	}

	if len(versioned.Versions) == 0 {
		return nil, errors.New("no versions available")
	}

	// Return a copy of the current version
	current := versioned.Versions[versioned.CurrentVersion-1]
	return &current, nil
}

// GetVersion retrieves a specific version of boot parameters by ID
func (s *MemoryStore) GetVersion(id string, version int) (*BootParams, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	versioned, exists := s.params[id]
	if !exists {
		return nil, errors.New("boot parameters not found")
	}

	if version < 1 || version > len(versioned.Versions) {
		return nil, errors.New("invalid version number")
	}

	// Return a copy of the requested version
	params := versioned.Versions[version-1]
	return &params, nil
}

// GetDefault retrieves the default version of boot parameters by ID
func (s *MemoryStore) GetDefault(id string) (*BootParams, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	versioned, exists := s.params[id]
	if !exists {
		return nil, errors.New("boot parameters not found")
	}

	if versioned.DefaultVersion < 1 || versioned.DefaultVersion > len(versioned.Versions) {
		return nil, errors.New("invalid default version")
	}

	// Return a copy of the default version
	params := versioned.Versions[versioned.DefaultVersion-1]
	return &params, nil
}

// Update updates existing boot parameters and creates a new version
func (s *MemoryStore) Update(id string, params *BootParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	versioned, exists := s.params[id]
	if !exists {
		return errors.New("boot parameters not found")
	}

	// Create new version
	newVersion := *params
	newVersion.Version = len(versioned.Versions) + 1
	versioned.Versions = append(versioned.Versions, newVersion)
	versioned.CurrentVersion = newVersion.Version

	return nil
}

// SetDefault sets the default version for a boot parameter set
func (s *MemoryStore) SetDefault(id string, version int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	versioned, exists := s.params[id]
	if !exists {
		return errors.New("boot parameters not found")
	}

	if version < 1 || version > len(versioned.Versions) {
		return errors.New("invalid version number")
	}

	versioned.DefaultVersion = version
	return nil
}

// Delete deletes all versions of boot parameters by ID
func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.params, id)
	return nil
}

// GetV1 retrieves the latest version of V1 boot parameters by xname
func (s *MemoryStore) GetV1(xname string) (*BootParamsV1, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	params, exists := s.v1[xname]
	if !exists {
		return nil, errors.New("boot parameters not found")
	}

	return params, nil
}

// SetV1 stores V1 boot parameters by xname
func (s *MemoryStore) SetV1(xname string, params *BootParamsV1) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.v1[xname] = params
	return nil
}

// AssignTemplateToGroup assigns a versioned template to a group. Passing 0 for version will assign the latest version.
func (s *MemoryStore) AssignTemplateToGroup(paramId, group string, version int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if version == 0 {
		version = s.params[paramId].DefaultVersion
	}

	s.v2[group] = struct {
		BootParams *VersionedBootParams
		Version    int
	}{
		BootParams: s.params[paramId],
		Version:    version,
	}
	return nil
}

// GetTemplateForGroup retrieves the template for a group
func (s *MemoryStore) GetTemplateForGroup(group string) (*BootParams, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	template, exists := s.v2[group]
	if !exists {
		return nil, errors.New("template not found")
	}

	return &template.BootParams.Versions[template.Version], nil
}
