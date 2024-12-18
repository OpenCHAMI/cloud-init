package memstore

import (
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/rs/zerolog/log"
)

type MemStore struct {
	Groups               map[string]cistore.GroupData `json:"groups,omitempty"`
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

func (m *MemStore) GetGroups() map[string]cistore.GroupData {
	m.GroupsMutex.RLock()
	defer m.GroupsMutex.RUnlock()
	return m.Groups
}

func (m *MemStore) AddGroupData(groupName string, newGroupData cistore.GroupData) error {
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
func (m *MemStore) GetGroupData(groupName string) (cistore.GroupData, error) {
	m.GroupsMutex.RLock()
	defer m.GroupsMutex.RUnlock()
	group, ok := m.Groups[groupName]
	if ok {
		return group, nil
	} else {
		return cistore.GroupData{}, fmt.Errorf("group (%s) not found in memstore", groupName)
	}

}

// UpdateGroupData is similar to AddGroupData but only works if the group exists
func (m *MemStore) UpdateGroupData(groupName string, groupData cistore.GroupData) error {
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

func (m *MemStore) GetInstanceInfo(nodeName string) (cistore.OpenCHAMIInstanceInfo, error) {
	m.InstancesMutex.RLock()
	defer m.InstancesMutex.RUnlock()
	if _, ok := m.Instances[nodeName]; !ok {
		m.Instances[nodeName] = cistore.OpenCHAMIInstanceInfo{
			InstanceID: generateInstanceId(),
		}
	}
	return m.Instances[nodeName], nil
}

func (m *MemStore) SetInstanceInfo(nodeName string, instanceInfo cistore.OpenCHAMIInstanceInfo) error {
	m.InstancesMutex.RLock()
	defer m.InstancesMutex.RUnlock()
	if _, ok := m.Instances[nodeName]; !ok {
		// This is a creation operation
		if instanceInfo.InstanceID == "" {
			instanceInfo.InstanceID = generateInstanceId()
		}
		m.Instances[nodeName] = instanceInfo
	} else {
		// This is an update operation.  We need to keep the instance ID the same.
		instanceInfo.InstanceID = m.Instances[nodeName].InstanceID
		m.Instances[nodeName] = instanceInfo
	}
	return nil
}

func (m *MemStore) DeleteInstanceInfo(nodeName string) error {
	m.InstancesMutex.RLock()
	defer m.InstancesMutex.RUnlock()
	delete(m.Instances, nodeName)
	return nil
}

func (m *MemStore) GetClusterDefaults() (cistore.ClusterDefaults, error) {
	m.ClusterDefaultsMutex.RLock()
	defer m.ClusterDefaultsMutex.RUnlock()
	return m.ClusterDefaults, nil
}

func (m *MemStore) SetClusterDefaults(clusterDefaults cistore.ClusterDefaults) error {
	m.ClusterDefaultsMutex.Lock()
	defer m.ClusterDefaultsMutex.Unlock()
	cd := m.ClusterDefaults
	if clusterDefaults.ClusterName != "" {
		log.Debug().Msgf("Setting ClusterName to %s", clusterDefaults.ClusterName)
		cd.ClusterName = clusterDefaults.ClusterName
	}
	if clusterDefaults.BaseUrl != "" {
		log.Debug().Msgf("Setting BaseUrl to %s", clusterDefaults.BaseUrl)
		cd.BaseUrl = clusterDefaults.BaseUrl
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
	m.ClusterDefaults = cd
	return nil
}

func generateInstanceId() string {
	// in the future, we might want to map the instance-id to an xname or something else.
	return generateUniqueID("i")

}

func generateUniqueID(prefix string) string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%s-%x", prefix, b)
}
