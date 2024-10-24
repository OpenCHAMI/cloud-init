package memstore

import (
	"errors"
	"log"

	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/samber/lo"
)

var (
	NotFoundErr      = errors.New("Not found.")
	ExistingEntryErr = errors.New("Data exists for this entry. Update instead.")
)

type (
	Data      = map[string]any
	GroupData = map[string]Data
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
func (m MemStore) AddGroups(name string, newGroupData Data) error {
	var (
		node              citypes.CI
		existingGroupData Data
	)

	// get CI data and add groups to metadata if not exists
	node, ok := m.list[name]
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
		// no node, component, or xname found which is required for groups
		return NotFoundErr
	}
	return nil
}

// GetGroup returns the 'meta-data.groups' found with the provided 'name' argument.
// Returns an error if the data is not found.
func (m MemStore) GetGroups(name string) (Data, error) {
	var (
		node      citypes.CI
		groupData Data
	)
	node, ok := m.list[name]
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
func (m MemStore) UpdateGroups(name string, newGroupData Data) error {
	var (
		node              citypes.CI
		existingGroupData Data
	)

	// get CI data and update groups whether it exists or not
	node, ok := m.list[name]
	if ok {
		existingGroupData = getGroupsFromMetadata(node)
		if existingGroupData != nil {
			setGroupsInMetadata(node, newGroupData)
			return nil
		}
		// no groups found so return not found error
		return NotFoundErr
	}
	// no node, component, or xname found which is required to update
	return NotFoundErr
}

func (m MemStore) RemoveGroups(name string) error {
	var (
		node                   citypes.CI
		removeGroupsInMetadata = func(ci citypes.CI, name string) {
			delete(ci.CIData.MetaData, name)
		}
	)

	node, ok := m.list[name]
	if ok {
		removeGroupsInMetadata(node, name)
	}
	return NotFoundErr
}

func getGroupsFromMetadata(ci citypes.CI) Data {
	_, ok := ci.CIData.MetaData["groups"]
	if ok {
		return ci.CIData.MetaData["groups"].(Data)
	}
	return nil
}

func setGroupsInMetadata(ci citypes.CI, data Data) {
	ci.CIData.MetaData["groups"] = data
}
