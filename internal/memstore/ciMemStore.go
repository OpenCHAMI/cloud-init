// SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package memstore

import (
	"crypto/rand"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"sigs.k8s.io/yaml"

	"github.com/openchami/cloud-init/pkg/cistore"
)

type MemStore struct {
	Groups               map[string]cistore.GroupData `json:"groups,omitempty" yaml:"groups,omitempty"`
	GroupsMutex          sync.RWMutex
	Instances            map[string]cistore.OpenCHAMIInstanceInfo
	InstancesMutex       sync.RWMutex
	ClusterDefaults      cistore.ClusterDefaults
	ClusterDefaultsMutex sync.RWMutex
}

func NewMemStore() *MemStore {
	return &MemStore{
		Groups:               make(map[string]cistore.GroupData),
		GroupsMutex:          sync.RWMutex{},
		Instances:            make(map[string]cistore.OpenCHAMIInstanceInfo),
		InstancesMutex:       sync.RWMutex{},
		ClusterDefaults:      cistore.ClusterDefaults{},
		ClusterDefaultsMutex: sync.RWMutex{},
	}
}

const (
	groupsFile    = "groups.yaml"
	instancesFile = "instances.yaml"
	defaultsFile  = "clusterdefaults.yaml"
)

func NewMemStoreFromPath(path string) (*MemStore, error) {
	store := NewMemStore()

	groupsPath := filepath.Join(path, groupsFile)
	instancesPath := filepath.Join(path, instancesFile)
	defaultsPath := filepath.Join(path, defaultsFile)

	groups, err := os.ReadFile(groupsPath)
	if err != nil {
		return nil, fmt.Errorf("error opening %q: %w", groupsPath, err)
	}
	err = yaml.Unmarshal(groups, &store.Groups)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling %q: %w", groupsPath, err)
	}

	instances, err := os.ReadFile(instancesPath)
	if err != nil {
		return nil, fmt.Errorf("error opening %q: %w", instancesPath, err)
	}
	err = yaml.Unmarshal(instances, &store.Instances)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling %q: %w", instancesPath, err)
	}

	defaults, err := os.ReadFile(defaultsPath)
	if err != nil {
		return nil, fmt.Errorf("error opening %q: %w", defaultsPath, err)
	}
	err = yaml.Unmarshal(defaults, &store.ClusterDefaults)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling %q: %w", defaultsPath, err)
	}

	return store, err
}

func (m *MemStore) GetGroups() map[string]cistore.GroupData {
	m.GroupsMutex.RLock()
	defer m.GroupsMutex.RUnlock()
	groups := make(map[string]cistore.GroupData, len(m.Groups))
	for groupName, groupData := range m.Groups {
		groups[groupName] = cloneGroupData(groupData)
	}
	return groups
}

func (m *MemStore) AddGroupData(groupName string, newGroupData cistore.GroupData) error {
	m.GroupsMutex.Lock()
	defer m.GroupsMutex.Unlock()
	// get CI data and check if groups IDENTIFIER exists (creates if not)
	_, ok := m.Groups[groupName]
	if ok {
		// found group so return error without changing anything
		log.Error().Msgf("group '%s' not added as it already exists", groupName)
		return fmt.Errorf("group '%s' not added as it already exists", groupName)
	} else {
		// does not exist, so create and update
		m.Groups[groupName] = cloneGroupData(newGroupData)

	}
	return nil
}

// GetGroupData returns the value of a specific group
func (m *MemStore) GetGroupData(groupName string) (cistore.GroupData, error) {
	m.GroupsMutex.RLock()
	defer m.GroupsMutex.RUnlock()
	group, ok := m.Groups[groupName]
	if ok {
		return cloneGroupData(group), nil
	} else {
		return cistore.GroupData{}, fmt.Errorf("group (%s) not found in memstore", groupName)
	}

}

// UpdateGroupData is similar to AddGroupData but only works if the group exists
func (m *MemStore) UpdateGroupData(groupName string, groupData cistore.GroupData, create bool) error {
	m.GroupsMutex.Lock()
	defer m.GroupsMutex.Unlock()
	if create {
		m.Groups[groupName] = cloneGroupData(groupData)
		return nil
	}

	_, ok := m.Groups[groupName]
	if ok {
		m.Groups[groupName] = cloneGroupData(groupData)
	} else {
		return fmt.Errorf("group (%s) not found", groupName)
	}
	return nil
}

func (m *MemStore) RemoveGroupData(name string) error {
	m.GroupsMutex.Lock()
	defer m.GroupsMutex.Unlock()
	delete(m.Groups, name)
	return nil
}

func (m *MemStore) GetInstanceInfo(nodeName string) (cistore.OpenCHAMIInstanceInfo, error) {
	m.InstancesMutex.Lock()
	defer m.InstancesMutex.Unlock()
	if _, ok := m.Instances[nodeName]; !ok {
		m.Instances[nodeName] = cistore.OpenCHAMIInstanceInfo{
			InstanceID: generateInstanceId(),
		}
	}
	return cloneInstanceInfo(m.Instances[nodeName]), nil
}

func (m *MemStore) SetInstanceInfo(nodeName string, instanceInfo cistore.OpenCHAMIInstanceInfo) error {
	m.InstancesMutex.Lock()
	defer m.InstancesMutex.Unlock()
	if _, ok := m.Instances[nodeName]; !ok {
		// This is a creation operation
		if instanceInfo.InstanceID == "" {
			instanceInfo.InstanceID = generateInstanceId()
		}
		m.Instances[nodeName] = cloneInstanceInfo(instanceInfo)
	} else {
		// This is an update operation.  We need to keep the instance ID the same.
		instanceInfo.InstanceID = m.Instances[nodeName].InstanceID
		m.Instances[nodeName] = cloneInstanceInfo(instanceInfo)
	}
	return nil
}

func (m *MemStore) DeleteInstanceInfo(nodeName string) error {
	m.InstancesMutex.Lock()
	defer m.InstancesMutex.Unlock()
	delete(m.Instances, nodeName)
	return nil
}

func (m *MemStore) GetClusterDefaults() (cistore.ClusterDefaults, error) {
	m.ClusterDefaultsMutex.RLock()
	defer m.ClusterDefaultsMutex.RUnlock()
	return cloneClusterDefaults(m.ClusterDefaults), nil
}

func (m *MemStore) SetClusterDefaults(clusterDefaults cistore.ClusterDefaults) error {
	m.ClusterDefaultsMutex.Lock()
	defer m.ClusterDefaultsMutex.Unlock()
	cd := m.ClusterDefaults
	if clusterDefaults.ClusterName != "" {
		log.Debug().Msgf("Setting ClusterName to %s", clusterDefaults.ClusterName)
		cd.ClusterName = clusterDefaults.ClusterName
	}
	if clusterDefaults.ShortName != "" {
		log.Debug().Msgf("Setting ShortName to %s", clusterDefaults.ShortName)
		cd.ShortName = clusterDefaults.ShortName
	}
	if clusterDefaults.NidLength != 0 {
		log.Debug().Msgf("Setting NidLength to %v", clusterDefaults.NidLength)
		cd.NidLength = clusterDefaults.NidLength
	}
	if clusterDefaults.BaseUrl != "" {
		log.Debug().Msgf("Setting BaseUrl to %s", clusterDefaults.BaseUrl)
		cd.BaseUrl = strings.TrimRight(clusterDefaults.BaseUrl, "/")
	}
	if clusterDefaults.AvailabilityZone != "" {
		log.Debug().Msgf("Setting Availability Zone to %s", clusterDefaults.AvailabilityZone)
		cd.AvailabilityZone = clusterDefaults.AvailabilityZone
	}
	if clusterDefaults.Region != "" {
		log.Debug().Msgf("Setting Region to %s", clusterDefaults.Region)
		cd.Region = clusterDefaults.Region
	}
	if clusterDefaults.CloudProvider != "" {
		log.Debug().Msgf("Setting Cloud Provider to %s", clusterDefaults.CloudProvider)
		cd.CloudProvider = clusterDefaults.CloudProvider
	}
	if len(clusterDefaults.PublicKeys) > 0 {
		log.Debug().Msgf("Setting Public Keys to %v", clusterDefaults.PublicKeys)
		cd.PublicKeys = clusterDefaults.PublicKeys
	}
	m.ClusterDefaults = cloneClusterDefaults(cd)
	return nil
}

func cloneGroupData(groupData cistore.GroupData) cistore.GroupData {
	groupData.Data = cloneAnyMap(groupData.Data)
	groupData.File.Content = slices.Clone(groupData.File.Content)
	groupData.Versions = maps.Clone(groupData.Versions)
	return groupData
}

func cloneInstanceInfo(instanceInfo cistore.OpenCHAMIInstanceInfo) cistore.OpenCHAMIInstanceInfo {
	instanceInfo.PublicKeys = slices.Clone(instanceInfo.PublicKeys)
	return instanceInfo
}

func cloneClusterDefaults(clusterDefaults cistore.ClusterDefaults) cistore.ClusterDefaults {
	clusterDefaults.PublicKeys = slices.Clone(clusterDefaults.PublicKeys)
	return clusterDefaults
}

func cloneAnyMap(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	clone := make(map[string]any, len(source))
	for key, value := range source {
		clone[key] = cloneAny(value)
	}
	return clone
}

func cloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneAnyMap(typed)
	case []any:
		clone := make([]any, len(typed))
		for i, item := range typed {
			clone[i] = cloneAny(item)
		}
		return clone
	default:
		return value
	}
}

func generateInstanceId() string {
	// in the future, we might want to map the instance-id to an xname or something else.
	return generateUniqueID("i")

}

func generateUniqueID(prefix string) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b) // Read fills b with cryptographically secure random bytes. It never returns an error, and always fills b entirely
	return fmt.Sprintf("%s-%x", prefix, b)
}
