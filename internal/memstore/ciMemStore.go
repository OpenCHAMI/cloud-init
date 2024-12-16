package memstore

import (
	"fmt"

	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/rs/zerolog/log"
)

var (
	ErrEmptyRequestBody = fmt.Errorf("no data found in request body")
	ErrResourceNotFound = fmt.Errorf("resource not found")
)

type MemStore struct {
	Groups map[string]citypes.GroupData `json:"groups,omitempty"`
}

func NewMemStore() *MemStore {
	var (
		groups = make(map[string]citypes.GroupData)
	)
	return &MemStore{groups}
}

func (m MemStore) GetGroups() map[string]citypes.GroupData {
	return m.Groups
}

func (m MemStore) AddGroupData(groupName string, newGroupData citypes.GroupData) error {

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
func (m MemStore) GetGroupData(groupName string) (citypes.GroupData, error) {

	group, ok := m.Groups[groupName]
	if ok {
		return group, nil
	} else {
		return citypes.GroupData{}, fmt.Errorf("group (%s) not found in memstore", groupName)
	}

}

// UpdateGroupData is similar to AddGroupData but only works if the group exists
func (m MemStore) UpdateGroupData(groupName string, groupData citypes.GroupData) error {

	_, ok := m.Groups[groupName]
	if ok {
		m.Groups[groupName] = groupData
	} else {
		return ErrResourceNotFound
	}
	return nil
}

func (m MemStore) RemoveGroupData(name string) error {
	delete(m.Groups, name)
	return nil
}
