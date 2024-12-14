package memstore

import (
	"errors"
	"fmt"

	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
)

var (
	ErrEmptyRequestBody = errors.New("no data found in request body")
	ErrResourceNotFound = errors.New("resource not found")
	ErrGroupDataExists  = errors.New("citypes.GroupData exists for this entry")
	ErrUserDataExists   = errors.New("user data exists for this entry")
	ErrVendorDataExists = errors.New("vendor data exists for this entry")
	ErrMetaDataExists   = errors.New("metadata exists for this entry")
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

/*
AddGroupData adds a new group with it's associated data specified by the user.
The key/value information "data" is included in the metadata.  The "actions" are stored in the user data.

Example:

AddGroup("x3000", data)

		{
			"name": "x3000",
			"data": {
				"syslog_aggregator": "192.168.0.1"
			},
			"file": {
	           "contents": "#template: jinja\n#cloud-config\nrsyslog:\n  remotes: {rack5: 10.0.4.1, {{ meta-data.system_name }}: 192.168.1.1}\n  service_reload_command: auto\n",
			}
		}
*/
func (m MemStore) AddGroupData(groupName string, newGroupData citypes.GroupData) error {

	// get CI data and check if groups IDENTIFIER exists (creates if not)
	_, ok := m.Groups[groupName]
	if ok {
		// found group so return error
		return ErrGroupDataExists
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
