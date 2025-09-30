package quackstore

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/OpenCHAMI/quack/quack"
)

// QuackStore implements the cistore.Store interface using Quack for persistence
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
		`CREATE TABLE IF NOT EXISTS groups (
			name TEXT PRIMARY KEY,
			data BLOB
		)`,
		`CREATE TABLE IF NOT EXISTS instances (
			node_name TEXT PRIMARY KEY,
			data BLOB
		)`,
		`CREATE TABLE IF NOT EXISTS cluster_defaults (
			id INTEGER PRIMARY KEY,
			data BLOB
		)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %q: %w", query, err)
		}
	}
	return nil
}

// GetGroups returns all groups
func (s *QuackStore) GetGroups() map[string]cistore.GroupData {
	groups := make(map[string]cistore.GroupData)

	rows, err := s.db.Query("SELECT name, data FROM groups")
	if err != nil {
		fmt.Printf("Error querying groups: %v\n", err)
		return groups
	}
	defer func() {
		_ = rows.Close() // Ignoring error on deferred Close
	}()

	for rows.Next() {
		var name string
		var data []byte
		if err := rows.Scan(&name, &data); err != nil {
			fmt.Printf("Error scanning row: %v\n", err)
			continue
		}

		fmt.Printf("Raw data from database for group %s: %s\n", name, string(data))

		var group cistore.GroupData
		if err := json.Unmarshal(data, &group); err != nil {
			fmt.Printf("Error unmarshaling group data: %v\n", err)
			continue
		}
		group.Name = name
		groups[name] = group
	}

	return groups
}

// AddGroupData adds a new group
func (s *QuackStore) AddGroupData(groupName string, groupData cistore.GroupData) error {
	// Check if group exists
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM groups WHERE name = ?)", groupName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check group existence: %w", err)
	}
	if exists {
		return fmt.Errorf("group '%s' not added as it already exists", groupName)
	}

	// Ensure name is set correctly
	groupData.Name = groupName

	data, err := json.Marshal(groupData)
	if err != nil {
		return fmt.Errorf("failed to marshal group data: %w", err)
	}

	fmt.Printf("Storing data for group %s: %s\n", groupName, string(data))

	_, err = s.db.Exec("INSERT INTO groups (name, data) VALUES (?, ?)", groupName, data)
	if err != nil {
		return fmt.Errorf("failed to insert group: %w", err)
	}

	return nil
}

// GetGroupData returns a specific group
func (s *QuackStore) GetGroupData(groupName string) (cistore.GroupData, error) {
	var data []byte
	err := s.db.QueryRow("SELECT data FROM groups WHERE name = ?", groupName).Scan(&data)
	if err == sql.ErrNoRows {
		return cistore.GroupData{}, fmt.Errorf("group (%s) not found", groupName)
	}
	if err != nil {
		return cistore.GroupData{}, fmt.Errorf("failed to query group: %w", err)
	}

	fmt.Printf("Raw data from database for group %s: %s\n", groupName, string(data))

	var group cistore.GroupData
	if err := json.Unmarshal(data, &group); err != nil {
		return cistore.GroupData{}, fmt.Errorf("failed to unmarshal group data: %w", err)
	}

	// Ensure name is set correctly
	group.Name = groupName
	return group, nil
}

// UpdateGroupData updates an existing group
func (s *QuackStore) UpdateGroupData(groupName string, groupData cistore.GroupData, create bool) error {
	// Ensure name is set correctly
	groupData.Name = groupName

	data, err := json.Marshal(groupData)
	if err != nil {
		return fmt.Errorf("failed to marshal group data: %w", err)
	}

	fmt.Printf("Storing data for group %s: %s\n", groupName, string(data))

	if create {
		_, err = s.db.Exec("INSERT OR REPLACE INTO groups (name, data) VALUES (?, ?)", groupName, data)
		if err != nil {
			return fmt.Errorf("failed to upsert group: %w", err)
		}
		return nil
	}

	result, err := s.db.Exec("UPDATE groups SET data = ? WHERE name = ?", data, groupName)
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("group (%s) not found", groupName)
	}

	return nil
}

// RemoveGroupData removes a group
func (s *QuackStore) RemoveGroupData(groupName string) error {
	result, err := s.db.Exec("DELETE FROM groups WHERE name = ?", groupName)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("group (%s) not found", groupName)
	}

	return nil
}

// GetInstanceInfo returns instance information for a node
func (s *QuackStore) GetInstanceInfo(nodeName string) (cistore.OpenCHAMIInstanceInfo, error) {
	var data []byte
	err := s.db.QueryRow("SELECT data FROM instances WHERE node_name = ?", nodeName).Scan(&data)
	if err == sql.ErrNoRows {
		// If not found, return a new instance with generated ID
		return cistore.OpenCHAMIInstanceInfo{
			InstanceID: generateInstanceId(),
		}, nil
	}
	if err != nil {
		return cistore.OpenCHAMIInstanceInfo{}, fmt.Errorf("failed to query instance: %w", err)
	}

	var info cistore.OpenCHAMIInstanceInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return cistore.OpenCHAMIInstanceInfo{}, fmt.Errorf("failed to unmarshal instance info: %w", err)
	}

	return info, nil
}

// SetInstanceInfo sets instance information for a node
func (s *QuackStore) SetInstanceInfo(nodeName string, instanceInfo cistore.OpenCHAMIInstanceInfo) error {
	// Get existing instance info to preserve instance ID if it exists
	var existingData []byte
	err := s.db.QueryRow("SELECT data FROM instances WHERE node_name = ?", nodeName).Scan(&existingData)
	if err == nil {
		var existingInfo cistore.OpenCHAMIInstanceInfo
		if err := json.Unmarshal(existingData, &existingInfo); err == nil {
			// Preserve existing instance ID
			instanceInfo.InstanceID = existingInfo.InstanceID
		}
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("failed to query existing instance: %w", err)
	} else if instanceInfo.InstanceID == "" {
		// Generate new instance ID if none exists
		instanceInfo.InstanceID = generateInstanceId()
	}

	data, err := json.Marshal(instanceInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal instance info: %w", err)
	}

	_, err = s.db.Exec("INSERT OR REPLACE INTO instances (node_name, data) VALUES (?, ?)", nodeName, data)
	if err != nil {
		return fmt.Errorf("failed to save instance info: %w", err)
	}

	return nil
}

// DeleteInstanceInfo deletes instance information for a node
func (s *QuackStore) DeleteInstanceInfo(nodeName string) error {
	result, err := s.db.Exec("DELETE FROM instances WHERE node_name = ?", nodeName)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("instance not found: %s", nodeName)
	}

	return nil
}

// GetClusterDefaults returns cluster defaults
func (s *QuackStore) GetClusterDefaults() (cistore.ClusterDefaults, error) {
	var data []byte
	err := s.db.QueryRow("SELECT data FROM cluster_defaults WHERE id = 1").Scan(&data)
	if err == sql.ErrNoRows {
		return cistore.ClusterDefaults{}, nil
	}
	if err != nil {
		return cistore.ClusterDefaults{}, fmt.Errorf("failed to query cluster defaults: %w", err)
	}

	var defaults cistore.ClusterDefaults
	if err := json.Unmarshal(data, &defaults); err != nil {
		return cistore.ClusterDefaults{}, fmt.Errorf("failed to unmarshal cluster defaults: %w", err)
	}

	return defaults, nil
}

// SetClusterDefaults sets cluster defaults
func (s *QuackStore) SetClusterDefaults(clusterDefaults cistore.ClusterDefaults) error {
	// Get existing defaults to merge with
	var existingData []byte
	err := s.db.QueryRow("SELECT data FROM cluster_defaults WHERE id = 1").Scan(&existingData)
	if err == nil {
		var existingDefaults cistore.ClusterDefaults
		if err := json.Unmarshal(existingData, &existingDefaults); err == nil {
			// Merge with existing defaults
			if clusterDefaults.ClusterName != "" {
				existingDefaults.ClusterName = clusterDefaults.ClusterName
			}
			if clusterDefaults.ShortName != "" {
				existingDefaults.ShortName = clusterDefaults.ShortName
			}
			if clusterDefaults.NidLength != 0 {
				existingDefaults.NidLength = clusterDefaults.NidLength
			}
			if clusterDefaults.BaseUrl != "" {
				existingDefaults.BaseUrl = clusterDefaults.BaseUrl
			}
			if clusterDefaults.AvailabilityZone != "" {
				existingDefaults.AvailabilityZone = clusterDefaults.AvailabilityZone
			}
			if clusterDefaults.Region != "" {
				existingDefaults.Region = clusterDefaults.Region
			}
			if clusterDefaults.CloudProvider != "" {
				existingDefaults.CloudProvider = clusterDefaults.CloudProvider
			}
			if len(clusterDefaults.PublicKeys) > 0 {
				existingDefaults.PublicKeys = clusterDefaults.PublicKeys
			}
			clusterDefaults = existingDefaults
		}
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("failed to query existing cluster defaults: %w", err)
	}

	data, err := json.Marshal(clusterDefaults)
	if err != nil {
		return fmt.Errorf("failed to marshal cluster defaults: %w", err)
	}

	_, err = s.db.Exec("INSERT OR REPLACE INTO cluster_defaults (id, data) VALUES (1, ?)", data)
	if err != nil {
		return fmt.Errorf("failed to save cluster defaults: %w", err)
	}

	return nil
}

// Close closes the Quack database connection
func (s *QuackStore) Close() error {
	return s.db.Close()
}

// generateInstanceID generates a unique instance ID in the format "i-XXXXXX",
// where "XXXXXX" is a random 6-digit hexadecimal string.
func generateInstanceId() string {
	randBytes := make([]byte, 3)
	_, _ = rand.Read(randBytes) // Read fills randBytes with cryptographically secure random bytes. It never returns an error, and always fills randBytes entirely
	return fmt.Sprintf("i-%x", randBytes)
}
