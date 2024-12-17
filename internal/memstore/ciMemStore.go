package memstore

import (
	"fmt"
	"sync"

	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/rs/zerolog/log"
)

type MemStore struct {
	Groups         map[string]citypes.GroupData `json:"groups,omitempty"`
	GroupsMutex    sync.RWMutex
	Instances      map[string]citypes.OpenCHAMIInstanceInfo
	InstancesMutex sync.RWMutex
}

func NewMemStore() *MemStore {
	return &MemStore{
		Groups:         make(map[string]citypes.GroupData),
		GroupsMutex:    sync.RWMutex{},
		Instances:      make(map[string]citypes.OpenCHAMIInstanceInfo),
		InstancesMutex: sync.RWMutex{},
	}
}

func (m *MemStore) GetGroups() map[string]citypes.GroupData {
	m.GroupsMutex.RLock()
	defer m.GroupsMutex.RUnlock()
	return m.Groups
}

func (m *MemStore) AddGroupData(groupName string, newGroupData citypes.GroupData) error {
	m.GroupsMutex.RLock()
	defer m.GroupsMutex.RUnlock()
	// get CI data and check if groups IDENTIFIER exists (creates if not)
	_, ok := m.Groups[groupName]
	if ok {
		// found group so return error without changing anything
		log.Error().Msgf("group '%s' not added as it already exists", groupName)
		return fmt.Errorf("group '%s' not added as it already exists", groupName)
	} else {
		// does not exist, so create and update
		m.Groups[groupName] = newGroupData

	}
	return nil
}

// GetGroupData returns the value of a specific group
func (m *MemStore) GetGroupData(groupName string) (citypes.GroupData, error) {
	m.GroupsMutex.RLock()
	defer m.GroupsMutex.RUnlock()
	group, ok := m.Groups[groupName]
	if ok {
		return group, nil
	} else {
		return citypes.GroupData{}, fmt.Errorf("group (%s) not found in memstore", groupName)
	}

}

// UpdateGroupData is similar to AddGroupData but only works if the group exists
func (m *MemStore) UpdateGroupData(groupName string, groupData citypes.GroupData) error {
	m.GroupsMutex.RLock()
	defer m.GroupsMutex.RUnlock()

	_, ok := m.Groups[groupName]
	if ok {
		m.Groups[groupName] = groupData
	} else {
		return fmt.Errorf("group (%s) not found", groupName)
	}
	return nil
}

func (m *MemStore) RemoveGroupData(name string) error {
	m.GroupsMutex.RLock()
	defer m.GroupsMutex.RUnlock()
	delete(m.Groups, name)
	return nil
}

func (m *MemStore) GetInstanceInfo(nodeName string) (citypes.OpenCHAMIInstanceInfo, error) {
	m.InstancesMutex.RLock()
	defer m.InstancesMutex.RUnlock()
	info, ok := m.Instances[nodeName]
	if ok {
		return info, nil
	} else {
		return citypes.OpenCHAMIInstanceInfo{}, fmt.Errorf("instance (%s) not found in memstore", nodeName)
	}
}

func (m *MemStore) SetInstanceInfo(nodeName string, instanceInfo citypes.OpenCHAMIInstanceInfo) error {
	m.InstancesMutex.RLock()
	defer m.InstancesMutex.RUnlock()
	m.Instances[nodeName] = instanceInfo
	return nil
}
