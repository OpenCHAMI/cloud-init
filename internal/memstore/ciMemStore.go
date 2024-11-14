package memstore

import (
	"errors"
	"fmt"

	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/samber/lo"
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
func (m MemStore) Get(id string, sm *smdclient.SMDClient) (citypes.CI, error) {
	var (
		groupLabels []string
		// ci_merged  citypes.CI
		err error
	)

	// fetch group name/labels from SMD
	groupLabels, err = sm.GroupMembership(id)
	fmt.Printf("groups: %v\n", groupLabels)

	// check that we actually got something back with no errors
	if err != nil {
		return citypes.CI{}, err
	} else if len(groupLabels) == 0 {
		return citypes.CI{}, errors.New("no groups found from SMD")
	} else {
		// make sure we already have cloud-init data and create it if it doesn't exist
		ci, ok := m.list[id]
		if !ok {
			ci = citypes.CI{
				Name: id,
				CIData: citypes.CIData{
					MetaData: map[string]any{"groups": map[string]citypes.Group{}},
				},
			}
		}

		// add matching group data stored with groups API to metadata
		for _, groupLabel := range groupLabels {
			// check if the group is stored with label from SMD
			group, ok := m.groups[groupLabel]
			if ok {
				// check if there's already metadata
				if ci.CIData.MetaData != nil {
					// check if we already have a "groups" section
					if groups, ok := ci.CIData.MetaData["groups"].(map[string]citypes.Group); ok {
						// found "groups" so add the new group + it's data
						groups[groupLabel] = group
					} else {
						// did not find "groups", so add it with current group data
						ci.CIData.MetaData["groups"] = map[string]citypes.Group{
							groupLabel: group,
						}
					}
				} else {
					// no metadata found, so create it with current group data here
					ci.CIData.MetaData = map[string]any{
						"groups": map[string]citypes.Group{
							groupLabel: group,
						},
					}
				}
			} else {
				// we didn't find the group in the memstore with the label, so
				// go on to the next one
				fmt.Printf("failed to get '%s' from groups", groupLabel)
				continue
			}
		}
		m.list[id] = ci
	}
	return m.list[id], nil
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
	return NotFoundErr
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
	var (
	// node      citypes.CI
	// groupData citypes.GroupData
	)

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
	var (
	// node citypes.CI
	)

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
