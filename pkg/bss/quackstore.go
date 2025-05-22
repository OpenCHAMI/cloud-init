package bss

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/OpenCHAMI/quack/quack"
)

// QuackStore implements the Store interface using Quack for persistence
type QuackStore struct {
	db *sql.DB
}

// NewQuackStore creates a new QuackStore instance
func NewQuackStore(dbPath string) (*QuackStore, error) {
	storage, err := quack.NewDuckDBStorage(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Quack database: %w", err)
	}

	store := &QuackStore{
		db: storage.DB(),
	}

	// Initialize database schema
	if err := store.initializeSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initializeSchema creates the necessary tables if they don't exist
func (s *QuackStore) initializeSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS boot_params (
			id TEXT PRIMARY KEY,
			current_version INTEGER,
			default_version INTEGER,
			versions BLOB
		)`,
		`CREATE TABLE IF NOT EXISTS v1_boot_params (
			xname TEXT PRIMARY KEY,
			data BLOB
		)`,
		`CREATE TABLE IF NOT EXISTS group_templates (
			group_name TEXT PRIMARY KEY,
			param_id TEXT,
			version INTEGER,
			FOREIGN KEY(param_id) REFERENCES boot_params(id)
		)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %q: %w", query, err)
		}
	}
	return nil
}

// Set stores boot parameters with the given ID
func (s *QuackStore) Set(id string, params *BootParams) error {
	// Check if boot parameters exist
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM boot_params WHERE id = ?)", id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check boot parameters existence: %w", err)
	}
	if exists {
		return fmt.Errorf("boot parameters with ID '%s' already exist", id)
	}

	// Initialize versioned boot parameters
	versioned := &VersionedBootParams{
		CurrentVersion: 1,
		DefaultVersion: 1,
		Versions:       []BootParams{*params},
	}

	// Set initial version
	versioned.Versions[0].Version = 1

	// Marshal the versioned boot parameters
	data, err := json.Marshal(versioned)
	if err != nil {
		return fmt.Errorf("failed to marshal boot parameters: %w", err)
	}

	// Store in database
	_, err = s.db.Exec("INSERT INTO boot_params (id, current_version, default_version, versions) VALUES (?, ?, ?, ?)",
		id, versioned.CurrentVersion, versioned.DefaultVersion, data)
	if err != nil {
		return fmt.Errorf("failed to insert boot parameters: %w", err)
	}

	return nil
}

// Get retrieves the latest version of boot parameters by ID
func (s *QuackStore) Get(id string) (*BootParams, error) {
	var currentVersion int
	var data []byte
	err := s.db.QueryRow("SELECT current_version, versions FROM boot_params WHERE id = ?", id).Scan(&currentVersion, &data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("boot parameters not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query boot parameters: %w", err)
	}

	var versioned VersionedBootParams
	if err := json.Unmarshal(data, &versioned); err != nil {
		return nil, fmt.Errorf("failed to unmarshal boot parameters: %w", err)
	}

	if len(versioned.Versions) == 0 {
		return nil, fmt.Errorf("no versions available")
	}

	// Return a copy of the current version
	current := versioned.Versions[currentVersion-1]
	return &current, nil
}

// GetVersion retrieves a specific version of boot parameters by ID
func (s *QuackStore) GetVersion(id string, version int) (*BootParams, error) {
	var data []byte
	err := s.db.QueryRow("SELECT versions FROM boot_params WHERE id = ?", id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("boot parameters not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query boot parameters: %w", err)
	}

	var versioned VersionedBootParams
	if err := json.Unmarshal(data, &versioned); err != nil {
		return nil, fmt.Errorf("failed to unmarshal boot parameters: %w", err)
	}

	if version < 1 || version > len(versioned.Versions) {
		return nil, fmt.Errorf("invalid version number")
	}

	// Return a copy of the requested version
	params := versioned.Versions[version-1]
	return &params, nil
}

// GetDefault retrieves the default version of boot parameters by ID
func (s *QuackStore) GetDefault(id string) (*BootParams, error) {
	var defaultVersion int
	var data []byte
	err := s.db.QueryRow("SELECT default_version, versions FROM boot_params WHERE id = ?", id).Scan(&defaultVersion, &data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("boot parameters not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query boot parameters: %w", err)
	}

	var versioned VersionedBootParams
	if err := json.Unmarshal(data, &versioned); err != nil {
		return nil, fmt.Errorf("failed to unmarshal boot parameters: %w", err)
	}

	if defaultVersion < 1 || defaultVersion > len(versioned.Versions) {
		return nil, fmt.Errorf("invalid default version")
	}

	// Return a copy of the default version
	params := versioned.Versions[defaultVersion-1]
	return &params, nil
}

// Update stores a new version of existing boot parameters
func (s *QuackStore) Update(id string, params *BootParams) error {
	var currentVersion int
	var data []byte
	err := s.db.QueryRow("SELECT current_version, versions FROM boot_params WHERE id = ?", id).Scan(&currentVersion, &data)
	if err == sql.ErrNoRows {
		return fmt.Errorf("boot parameters not found")
	}
	if err != nil {
		return fmt.Errorf("failed to query boot parameters: %w", err)
	}

	var versioned VersionedBootParams
	if err := json.Unmarshal(data, &versioned); err != nil {
		return fmt.Errorf("failed to unmarshal boot parameters: %w", err)
	}

	// Create new version
	newVersion := *params
	newVersion.Version = len(versioned.Versions) + 1
	versioned.Versions = append(versioned.Versions, newVersion)
	versioned.CurrentVersion = newVersion.Version

	// Marshal the updated versioned boot parameters
	updatedData, err := json.Marshal(versioned)
	if err != nil {
		return fmt.Errorf("failed to marshal boot parameters: %w", err)
	}

	// Update in database
	_, err = s.db.Exec("UPDATE boot_params SET current_version = ?, versions = ? WHERE id = ?",
		versioned.CurrentVersion, updatedData, id)
	if err != nil {
		return fmt.Errorf("failed to update boot parameters: %w", err)
	}

	return nil
}

// SetDefault sets the default version for a boot parameter set
func (s *QuackStore) SetDefault(id string, version int) error {
	var data []byte
	err := s.db.QueryRow("SELECT versions FROM boot_params WHERE id = ?", id).Scan(&data)
	if err == sql.ErrNoRows {
		return fmt.Errorf("boot parameters not found")
	}
	if err != nil {
		return fmt.Errorf("failed to query boot parameters: %w", err)
	}

	var versioned VersionedBootParams
	if err := json.Unmarshal(data, &versioned); err != nil {
		return fmt.Errorf("failed to unmarshal boot parameters: %w", err)
	}

	if version < 1 || version > len(versioned.Versions) {
		return fmt.Errorf("invalid version number")
	}

	// Update default version in database
	_, err = s.db.Exec("UPDATE boot_params SET default_version = ? WHERE id = ?", version, id)
	if err != nil {
		return fmt.Errorf("failed to update default version: %w", err)
	}

	return nil
}

// Delete deletes all versions of boot parameters by ID
func (s *QuackStore) Delete(id string) error {
	// Start a transaction to ensure atomicity
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete from boot_params table
	_, err = tx.Exec("DELETE FROM boot_params WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete boot parameters: %w", err)
	}

	// Delete from group_templates table
	_, err = tx.Exec("DELETE FROM group_templates WHERE param_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete group templates: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetV1 retrieves the latest version of V1 boot parameters by xname
func (s *QuackStore) GetV1(xname string) (*BootParamsV1, error) {
	var data []byte
	err := s.db.QueryRow("SELECT data FROM v1_boot_params WHERE xname = ?", xname).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("boot parameters not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query boot parameters: %w", err)
	}

	var params BootParamsV1
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, fmt.Errorf("failed to unmarshal boot parameters: %w", err)
	}

	return &params, nil
}

// SetV1 stores V1 boot parameters by xname
func (s *QuackStore) SetV1(xname string, params *BootParamsV1) error {
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal boot parameters: %w", err)
	}

	_, err = s.db.Exec("INSERT OR REPLACE INTO v1_boot_params (xname, data) VALUES (?, ?)", xname, data)
	if err != nil {
		return fmt.Errorf("failed to store boot parameters: %w", err)
	}

	return nil
}

// AssignTemplateToGroup assigns a versioned template to a group
func (s *QuackStore) AssignTemplateToGroup(paramId, group string, version int) error {
	// Check if boot parameters exist
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM boot_params WHERE id = ?)", paramId).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check boot parameters existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("boot parameters not found")
	}

	// If version is 0, get the default version
	if version == 0 {
		err = s.db.QueryRow("SELECT default_version FROM boot_params WHERE id = ?", paramId).Scan(&version)
		if err != nil {
			return fmt.Errorf("failed to get default version: %w", err)
		}
	}

	// Store the template assignment
	_, err = s.db.Exec("INSERT OR REPLACE INTO group_templates (group_name, param_id, version) VALUES (?, ?, ?)",
		group, paramId, version)
	if err != nil {
		return fmt.Errorf("failed to assign template to group: %w", err)
	}

	return nil
}

// GetTemplateForGroup retrieves the template for a group
func (s *QuackStore) GetTemplateForGroup(group string) (*BootParams, error) {
	var paramId string
	var version int
	err := s.db.QueryRow("SELECT param_id, version FROM group_templates WHERE group_name = ?", group).Scan(&paramId, &version)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query template: %w", err)
	}

	return s.GetVersion(paramId, version)
}

// Close closes the database connection
func (s *QuackStore) Close() error {
	return s.db.Close()
}
