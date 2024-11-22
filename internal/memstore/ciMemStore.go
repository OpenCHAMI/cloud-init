package memstore

import (
	"errors"
	"fmt"

	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
)

var (
	EmptyRequest = errors.New("No data found in request body.")
	NotFoundErr  = errors.New("Not found.")
	ExistingErr  = errors.New("citypes.GroupData exists for this entry. Update instead.")
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
			return ExistingErr
		}
	}

	if ci.CIData.MetaData != nil {
		if curr.CIData.MetaData == nil {
			curr.CIData.MetaData = ci.CIData.MetaData
		} else {
			return ExistingErr
		}
	}

	if ci.CIData.VendorData != nil {
		if curr.CIData.VendorData == nil {
			curr.CIData.VendorData = ci.CIData.VendorData
		} else {
			return ExistingErr
		}
	}

	m.list[name] = curr
	return nil
}

// Get retrieves data stored in MemStore and returns it or an error
func (m MemStore) Get(id string) (citypes.CI, error) {
	var (
		ci citypes.CI
		ok bool
	)

	// fetch group name/labels from SMD
	ci, ok = m.list[id]
	if ok {
		return ci, nil
	}
	return ci, NotFoundErr
}

func (m MemStore) List() (map[string]citypes.CI, error) {
	return m.list, nil
}

func (m MemStore) Update(name string, ci citypes.CI) error {
	var (
		existing citypes.CI
		ok       bool
	)
	// check if we already have existing data
	existing, ok = m.list[name]
	if ok {
		// update user data if we have existing data and new data is supplied
		if ci.CIData.UserData != nil {
			existing.CIData.UserData = ci.CIData.UserData
		}

		// NOTE: I think we technically would not want to allow this given the
		// how the new API works.
		//
		// update meta data if we have existing data and new data is supplied
		if ci.CIData.MetaData != nil {
			existing.CIData.MetaData = ci.CIData.MetaData
		}

		// update vendor data if we have existing data and new data is supplied
		if ci.CIData.VendorData != nil {
			existing.CIData.VendorData = ci.CIData.VendorData
		}
		m.list[name] = existing
	} else {
		// set the IDENTIFIER and add all of the new data if existing not found
		ci.Name = name
		m.list[name] = ci
	}
	return nil
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
The group data is included in the metadata with making a request for cloud-init
data.

Example:

AddGroup("x3000", data)

	{
		"meta-data": {
			"groups": { // POST request should only include this data
				"x3000": {
					"data": {
						"syslog_aggregator": "192.168.0.1"
					}
				},
				"canary-123": {
					"data": {
						"hello": "world"
					}
				}
			}
		}
	}
*/
func (m MemStore) AddGroupData(groupName string, newGroupData citypes.GroupData) error {
	// do nothing if no data found from the request
	if len(newGroupData) <= 0 {
		fmt.Printf("no data")
		return EmptyRequest
	}

	// get CI data and check if groups IDENTIFIER exists (creates if not)
	_, ok := m.groups[groupName]
	if ok {
		// found group so return error
		return ExistingErr
	} else {
		// does not exist, so create and update
		m.groups[groupName] = citypes.Group{"data": newGroupData}
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
		return nil, NotFoundErr
	}

}

// UpdateGroupData is similar to AddGroupData but only works if the group exists
func (m MemStore) UpdateGroupData(groupName string, groupData citypes.GroupData) error {

	// do nothing if no data found
	if len(groupData) <= 0 {
		return EmptyRequest
	}

	group, ok := m.groups[groupName]
	if ok {
		group["data"] = groupData
		m.groups[groupName] = group
	} else {
		return NotFoundErr
	}
	return nil
}

func (m MemStore) RemoveGroupData(name string) error {
	delete(m.groups, name)
	return nil
}

func getGroupsFromMetadata(ci citypes.CI) citypes.GroupData {
	_, ok := ci.CIData.MetaData["groups"]
	if ok {
		return ci.CIData.MetaData["groups"].(citypes.GroupData)
	}
	return nil
}

func setGroupsInMetadata(ci citypes.CI, data citypes.GroupData) {
	ci.CIData.MetaData["groups"] = data
}
