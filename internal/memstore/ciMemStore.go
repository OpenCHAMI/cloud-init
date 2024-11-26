package memstore

import (
	"errors"
	"fmt"
	"log"

	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/samber/lo"
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
	list   map[string]citypes.CI
	groups map[string]citypes.Group
}

func NewMemStore() *MemStore {
	var (
		list   = make(map[string]citypes.CI)
		groups = make(map[string]citypes.Group)
	)
	return &MemStore{list, groups}
}

// Add creates new cloud-init data and stores in MemStore. If the data already
// exists, an error will be returned
func (m MemStore) Add(name string, ci citypes.CI) error {
	curr := m.list[name]

	// add user data if no current data
	if ci.CIData.UserData != nil {
		if curr.CIData.UserData == nil {
			curr.CIData.UserData = ci.CIData.UserData
		} else {
			return ErrUserDataExists
		}
	}

	if ci.CIData.MetaData != nil {
		if curr.CIData.MetaData == nil {
			curr.CIData.MetaData = ci.CIData.MetaData
		} else {
			return ErrMetaDataExists
		}
	}

	if ci.CIData.VendorData != nil {
		if curr.CIData.VendorData == nil {
			curr.CIData.VendorData = ci.CIData.VendorData
		} else {
			return ErrVendorDataExists
		}
	}

	m.list[name] = curr
	return nil
}

// Get retrieves data stored in MemStore and returns it or an error
func (m *MemStore) Get(id string, groupLabels []string) (citypes.CI, error) {

	fmt.Printf("groups: %v\n", groupLabels)

	if len(groupLabels) == 0 {
		return citypes.CI{}, errors.New("no groups found from SMD")
	}
	// If there's cloud-init data associated with the ID, we should return it.
	ci, ok := m.list[id]
	if ok {
		return ci, nil
	}

	// At this point, we can be sure that we are generating UserData and MetaData based on the groups
	ci = citypes.CI{
		Name: id,
		CIData: citypes.CIData{
			UserData: map[string]any{},
			MetaData: map[string]any{"groups": map[string]citypes.Group{}},
		},
	}

	// add matching group data stored with groups API to metadata
	for _, groupLabel := range groupLabels {
		// check if the group is stored with label from SMD
		group, ok := m.groups[groupLabel]
		if ok {
			// check if we already have a "groups" section in metadata
			if groups, ok := ci.CIData.MetaData["groups"].(map[string]citypes.GroupData); ok {
				// found "groups" so add the new group + it's data
				groups[groupLabel] = group["data"]
			} else {
				// did not find "groups", so add it with current group data
				ci.CIData.MetaData["groups"] = map[string]citypes.GroupData{
					groupLabel: group["data"],
				}
			}

			// In user data, we cannot store things as groups.  We store the write_files and runcmd lists directly.
			// check if we already have a "write_files" section in user data
			if writeFiles, ok := ci.CIData.UserData["write_files"].([]citypes.WriteFiles); ok {
				// found the "write_files" section, so add the new group's write_files
				if actions, ok := group["actions"]; ok {

					if groupWriteFiles, ok := actions["write_files"].([]citypes.WriteFiles); ok {
						for _, wf := range groupWriteFiles {
							wf.Group = groupLabel
							writeFiles = append(writeFiles, wf)
						}

						ci.CIData.UserData["write_files"] = writeFiles
					}
				}
			} else {
				// did not find "write_files", so add it with current group's write_files
				if actions, ok := group["actions"]; ok {

					if groupWriteFiles, ok := actions["write_files"].([]citypes.WriteFiles); ok {

						ci.CIData.UserData["write_files"] = groupWriteFiles
					}
				}
			}
		} else {
			// we didn't find the group in the memstore with the label, so
			// go on to the next one
			log.Printf("failed to get '%s' from groups", groupLabel)
			continue
		}
	}

	fmt.Printf("ci: %v\n", ci)
	return ci, nil
}

// Merge combines cloud-init data from MemStore with new citypes.CI
func (m MemStore) Merge(name string, newData citypes.CI) (citypes.CI, error) {
	ci := new(citypes.CI)
	if v, ok := m.list[name]; ok {
		ci.CIData.UserData = lo.Assign(newData.CIData.UserData, v.CIData.UserData)
		ci.CIData.VendorData = lo.Assign(newData.CIData.VendorData, v.CIData.VendorData)
		ci.CIData.MetaData = lo.Assign(newData.CIData.MetaData, v.CIData.MetaData)
	}
	return *ci, nil
}

func (m MemStore) List() (map[string]citypes.CI, error) {
	return m.list, nil
}

func (m MemStore) Update(name string, ci citypes.CI) error {

	if _, ok := m.list[name]; ok {
		curr := m.list[name]
		if ci.CIData.UserData != nil {
			curr.CIData.UserData = ci.CIData.UserData
		}
		if ci.CIData.MetaData != nil {
			curr.CIData.MetaData = ci.CIData.MetaData
		}
		if ci.CIData.VendorData != nil {
			curr.CIData.VendorData = ci.CIData.VendorData
		}
		m.list[name] = curr
		return nil
	}
	return ErrResourceNotFound
}

func (m MemStore) Remove(name string) error {
	delete(m.list, name)
	return nil
}

func (m MemStore) GetGroups() map[string]citypes.Group {
	return m.groups
}

/*
AddGroupData adds a new group with it's associated data specified by the user.
The key/value information "data" is included in the metadata.  The "actions" are stored in the user data.

Example:

AddGroup("x3000", data)

	{
		"data": {
			"syslog_aggregator": "192.168.0.1"
		},
		"actions": {
			"write_files": [
				{ "path": "/etc/hello", "content": "hello world" },
				{ "path": "/etc/hello2", "content": "hello world" }
			]
		}
	}
*/
func (m MemStore) AddGroupData(groupName string, newGroupData citypes.GroupData) error {
	var (
	// node      citypes.CI
	// groupData citypes.GroupData
	)

	// do nothing if no data found from the request
	if len(newGroupData) <= 0 {
		fmt.Printf("no data")
		return ErrEmptyRequestBody
	}

	// get CI data and check if groups IDENTIFIER exists (creates if not)
	_, ok := m.groups[groupName]
	if ok {
		// found group so return error
		return ErrGroupDataExists
	} else {
		// does not exist, so create and update
		m.groups[groupName] = citypes.Group{}

		if data, ok := newGroupData["data"].(citypes.GroupData); ok {
			m.groups[groupName]["data"] = data
		}
		if actions, ok := newGroupData["actions"].(citypes.GroupData); ok {
			m.groups[groupName]["actions"] = actions
		}
	}
	return nil
}

// GetGroupData returns the value of a specific group
func (m MemStore) GetGroupData(groupName string) (citypes.GroupData, error) {
	var (
	// node      citypes.CI
	// groupData citypes.GroupData
	)

	group, ok := m.groups[groupName]
	if ok {
		return group["data"], nil
	} else {
		return nil, ErrResourceNotFound
	}

}

// UpdateGroupData is similar to AddGroupData but only works if the group exists
func (m MemStore) UpdateGroupData(groupName string, groupData citypes.GroupData) error {
	var (
	// node citypes.CI
	)

	// do nothing if no data found
	if len(groupData) <= 0 {
		return ErrEmptyRequestBody
	}

	group, ok := m.groups[groupName]
	if ok {
		group["data"] = groupData
		m.groups[groupName] = group
	} else {
		return ErrResourceNotFound
	}
	return nil
}

func (m MemStore) RemoveGroupData(name string) error {
	delete(m.groups, name)
	return nil
}
