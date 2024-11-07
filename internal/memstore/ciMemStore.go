package memstore

import (
	"errors"
	"fmt"
	"log"

	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/samber/lo"
)

var (
	EmptyRequest     = errors.New("No data found in request body.")
	NotFoundErr      = errors.New("Not found.")
	ExistingEntryErr = errors.New("citypes.GroupData exists for this entry. Update instead.")
)

type MemStore struct {
	list map[string]citypes.CI
}

func NewMemStore() *MemStore {
	list := make(map[string]citypes.CI)
	return &MemStore{
		list,
	}
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
			return ExistingEntryErr
		}
	}

	if ci.CIData.MetaData != nil {
		if curr.CIData.MetaData == nil {
			curr.CIData.MetaData = ci.CIData.MetaData
		} else {
			return ExistingEntryErr
		}
	}

	if ci.CIData.VendorData != nil {
		if curr.CIData.VendorData == nil {
			curr.CIData.VendorData = ci.CIData.VendorData
		} else {
			return ExistingEntryErr
		}
	}

	m.list[name] = curr
	return nil
}

// Get retrieves data stored in MemStore and returns it or an error
func (m MemStore) Get(name string, sm *smdclient.SMDClient) (citypes.CI, error) {

	ci_merged := new(citypes.CI)

	gl, err := sm.GroupMembership(name)
	if err != nil {
		log.Print(err)
	} else if len(gl) > 0 {
		log.Printf("Node %s is a member of these groups: %s\n", name, gl)

		for g := 0; g < len(gl); g++ {
			if val, ok := m.list[gl[g]]; ok {
				ci_merged.CIData.UserData = lo.Assign(ci_merged.CIData.UserData, val.CIData.UserData)
				ci_merged.CIData.VendorData = lo.Assign(ci_merged.CIData.VendorData, val.CIData.VendorData)
				ci_merged.CIData.MetaData = lo.Assign(ci_merged.CIData.MetaData, val.CIData.MetaData)
			}
		}
	} else {
		log.Printf("Node %s is not a member of any groups\n", name)
	}

	if val, ok := m.list[name]; ok {
		ci_merged.CIData.UserData = lo.Assign(ci_merged.CIData.UserData, val.CIData.UserData)
		ci_merged.CIData.VendorData = lo.Assign(ci_merged.CIData.VendorData, val.CIData.VendorData)
		ci_merged.CIData.MetaData = lo.Assign(ci_merged.CIData.MetaData, val.CIData.MetaData)
	} else {
		log.Printf("Node %s has no specific configuration\n", name)
	}

	if len(ci_merged.CIData.UserData) == 0 &&
		len(ci_merged.CIData.VendorData) == 0 &&
		len(ci_merged.CIData.MetaData) == 0 {
		return citypes.CI{}, NotFoundErr
	} else {
		return *ci_merged, nil
	}
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

/*
AddGroup adds a new group with it's associated meta-data.

Example:

	data := map[string]any{
		"x3000": map[string]any {
			"syslog_aggregator": "192.168.0.1",
		},
		"canary-123": map[string]any {
			"syslog_aggregator": "127.0.0.1",
		}
	}

AddGroup("compute", "x3000", data)

	{
		"name": "compute",
		...
		"meta-data": {
			"groups": { // POST request should only include this data
	        	"x3000": {
	            	"syslog_aggregator": "192.168.0.1"
	          	},
	          	"canary-123": {
	                "hello": "world"
	          	}
	        }
		}
	}
*/
func (m MemStore) AddGroups(newGroupData citypes.GroupData) error {
	var (
		node              citypes.CI
		existingGroupData citypes.GroupData
	)

	// do nothing if no data found
	if len(newGroupData) <= 0 {
		return EmptyRequest
	}

	// get CI data and add groups to metadata if not exists
	node, ok := m.list[citypes.GROUP_IDENTIFIER]
	if ok {
		// check if data already exists, otherwise add
		if node.CIData.MetaData != nil {
			return ExistingEntryErr
		} else {
			// check if there's already a 'groups' property
			existingGroupData = getGroupsFromMetadata(node)
			if existingGroupData != nil {
				setGroupsInMetadata(node, newGroupData)
			} else {
				// groups already exist so return error
				return ExistingEntryErr
			}
		}
	} else {
		// groups not found in metadata so create everything
		node = citypes.CI{
			CIData: citypes.CIData{
				MetaData: citypes.GroupData{
					"groups": newGroupData,
				},
			},
		}
	}
	m.list[citypes.GROUP_IDENTIFIER] = node
	return nil
}

// GetGroup returns the 'meta-data.groups' found with the provided 'name' argument.
// Returns an error if the data is not found.
func (m MemStore) GetGroups() (citypes.GroupData, error) {
	var (
		node      citypes.CI
		groupData citypes.GroupData
	)
	node, ok := m.list[citypes.GROUP_IDENTIFIER]
	if ok {
		groupData = getGroupsFromMetadata(node)
		if groupData != nil {
			return groupData, nil
		} else {
			// no group data found so return error
			return nil, NotFoundErr
		}
	}
	// no node, component, or xname found so return error
	return nil, NotFoundErr
}

// UpdateGroups updates the data for an existing 'meta-data.groups'. An error is
// returned if it is not found.
func (m MemStore) UpdateGroups(newGroupData citypes.GroupData) error {
	var (
		node              citypes.CI
		existingGroupData citypes.GroupData
	)

	// do nothing if no data found
	if len(newGroupData) <= 0 {
		return EmptyRequest
	}

	// get CI data and update groups whether it exists or not
	node, ok := m.list[citypes.GROUP_IDENTIFIER]
	if ok {
		// get existing group data and update
		existingGroupData = getGroupsFromMetadata(node)
		if existingGroupData != nil {
			setGroupsInMetadata(node, newGroupData)
			return nil
		}
		// no groups found so return not found error
		return NotFoundErr
	} else {
		// no node, component, or xname found which is required to update
		return NotFoundErr
	}
}

func (m MemStore) RemoveGroups() error {
	var (
		node                   citypes.CI
		removeGroupsInMetadata = func(ci citypes.CI) {
			delete(ci.CIData.MetaData, citypes.GROUP_IDENTIFIER)
		}
	)

	node, ok := m.list[citypes.GROUP_IDENTIFIER]
	if ok {
		removeGroupsInMetadata(node)
		return nil
	}
	return NotFoundErr
}

// AddGroupData creates a new key-value for a group is it does not exists.
//
// NOTE: For this method, it makes sense to create the `meta-data` automatically
func (m MemStore) AddGroupData(groupName string, newGroupData citypes.GroupData) error {
	var (
		node      citypes.CI
		groupData citypes.GroupData
	)

	// do nothing if no data found from the request
	if len(newGroupData) <= 0 {
		fmt.Printf("no data")
		return EmptyRequest
	}

	// get CI data and check if groups IDENTIFIER exists (creates if not)
	node, ok := m.list[citypes.GROUP_IDENTIFIER]
	if ok {
		// check if metadata already exists (create if not)
		if node.CIData.MetaData == nil {
			node.CIData.MetaData = map[string]any{
				"groups": map[string]any{
					groupName: newGroupData,
				},
			}
		} else {
			// check if group already exists (create if not)
			groupData = getGroupsFromMetadata(node)
			if groupData == nil {
				setGroupsInMetadata(node, citypes.GroupData{groupName: newGroupData})
			} else {
				// check for key in group already exists
				_, ok = groupData[groupName]
				if ok {
					// fail here since we don't want to overwrite
					return ExistingEntryErr
				} else {
					groupData[groupName] = newGroupData
				}
				// add new group data to metadata
				node.CIData.MetaData["groups"] = groupData
			}
		}
	} else {
		// no node data found so create a default one
		node = citypes.CI{
			CIData: citypes.CIData{
				MetaData: citypes.GroupData{
					"groups": map[string]any{groupName: newGroupData},
				},
			},
		}
		// update the node's name with the identifier
		node.Name = citypes.GROUP_IDENTIFIER
	}

	// finally, update the CIData after making changes
	m.list[citypes.GROUP_IDENTIFIER] = node
	return nil
}

// GetGroupData returns the value of a specific group
func (m MemStore) GetGroupData(groupName string) (citypes.GroupData, error) {
	var (
		node      citypes.CI
		groupData citypes.GroupData
	)

	// get the node info with default group identitifer
	node, ok := m.list[citypes.GROUP_IDENTIFIER]
	if ok {
		// get the groups object from the metadata
		groupData = getGroupsFromMetadata(node)
		if groupData != nil {
			// finally, get the specific key from the group
			key, ok := groupData[groupName].(map[string]any)
			if ok {
				return key, nil
			}
			return nil, NotFoundErr
		} else {
			return nil, NotFoundErr
		}
	} else {
		return nil, NotFoundErr
	}

}

// UpdateGroupData is similar to AddGroupData but only works if the group exists
func (m MemStore) UpdateGroupData(groupName string, value citypes.GroupData) error {
	var (
		node      citypes.CI
		groupData citypes.GroupData
	)

	// do nothing if no data found
	if len(value) <= 0 {
		return EmptyRequest
	}

	// get CI data and group if exists (creates CI data if it doesn't)
	node, ok := m.list[citypes.GROUP_IDENTIFIER]
	if ok {
		// check if metadata already exists, and create if not
		if node.CIData.MetaData == nil {
			node.CIData.MetaData = map[string]any{
				"groups": map[string]any{
					groupName: value,
				},
			}
		}

		// check if group exists and create if there isn't
		groupData = getGroupsFromMetadata(node)
		if groupData == nil {
			setGroupsInMetadata(node, citypes.GroupData{groupName: value})
		}

		// check for key in group and only create if it doesn't exist
		_, ok = groupData[groupName]
		if ok {
			groupData[groupName] = value
			return nil
		} else {
			return NotFoundErr
		}

	} else {
		// no node data found so
		return NotFoundErr
	}

	return NotFoundErr
}

func (m MemStore) RemoveGroupData(name string) error {
	var (
		node                      citypes.CI
		groupData                 citypes.GroupData
		removeGroupDataInMetadata = func(ci citypes.CI, name string) {
			if ci.CIData.MetaData != nil {
				groupData = getGroupsFromMetadata(node)
				if groupData != nil {
					delete(groupData, name)
				}
			}
		}
	)

	node, ok := m.list[citypes.GROUP_IDENTIFIER]
	if ok {
		removeGroupDataInMetadata(node, name)
		return nil
	}
	return NotFoundErr
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
