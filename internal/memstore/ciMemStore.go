package memstore

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"

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
	Groups map[string]citypes.GroupData `json:"groups,omitempty"`
}

func NewMemStore() *MemStore {
	var (
		list   = make(map[string]citypes.CI)
		groups = make(map[string]citypes.GroupData)
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
			MetaData: map[string]any{"groups": make(map[string]citypes.MetaDataKV)}, // groups is a map of group name to list of key/value pairs
		},
	}

	log.Info().Msgf("groups: %v", groupLabels)

	// add matching group data stored with groups API to metadata
	for _, groupLabel := range groupLabels {
		log.Debug().Msgf("groupLabel: %s", groupLabel)
		// check if the group is stored locally with the label obtained from SMD
		groupData, ok := m.Groups[groupLabel]
		if !ok {
			// we didn't find the group in the memstore with the label, so
			// go on to the next one
			log.Debug().Msgf("failed to get '%s' from groups", groupLabel)
		} else {
			// found the group, so add it to the metadata
			log.Debug().Msgf("found group '%s' in groups", groupLabel)
			log.Debug().Msgf("groupData.Name: %v", groupData.Name)
			log.Debug().Msgf("groupData.Data: %v", groupData.Data)
			log.Debug().Msgf("groupData.Actions: %v", groupData.Actions)
			groups := ci.CIData.MetaData["groups"].(map[string]citypes.MetaDataKV)
			groups[groupLabel] = groupData.Data
			log.Debug().Msgf("Adding groups to MetaData: %v", groups)
			ci.CIData.MetaData["groups"] = groups
		}

		// In user data, we cannot store things as groups.  We store the write_files lists directly.
		// First establish that the write_files section exists in the groupData from our storage
		if _, ok := groupData.Actions["write_files"]; ok {
			log.Debug().Msgf("write_files found in group '%s'", groupLabel)
			log.Debug().Msgf("groupData.Actions[\"write_files\"]: %v", groupData.Actions["write_files"])

			// Then ensure that the write_files section exists in the user data
			if _, ok := ci.CIData.UserData["write_files"].([]citypes.WriteFiles); !ok {
				// write_files does not exist, so add it with current group's write_files
				ci.CIData.UserData["write_files"] = []citypes.WriteFiles{}
			}

			// Make sure that the the write files are in the correct format
			switch groupData.Actions["write_files"].(type) {
			default:
				log.Fatal().Msg("Unexpected type for write file. Check your submitted data.")
			case []interface{}:
			}

			// Now iterate through the entries and add them to the UserData
			for _, wf := range groupData.Actions["write_files"].([]interface{}) {
				writeFilesEntry := citypes.WriteFiles{
					Path:    wf.(map[string]interface{})["path"].(string),
					Content: wf.(map[string]interface{})["content"].(string),
					Group:   groupLabel,
				}
				log.Debug().Msgf("writeFilesEntry: %v", writeFilesEntry)
				writeFilesEntry.Group = groupLabel
				ci.CIData.UserData["write_files"] = append(ci.CIData.UserData["write_files"].([]citypes.WriteFiles), writeFilesEntry)
			}
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

func (m MemStore) GetGroups() map[string]citypes.GroupData {
	return m.Groups
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
		return citypes.GroupData{}, ErrResourceNotFound
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

func (m MemStore) GetClusterData(id string) (citypes.ClusterData, error) {
	return citypes.ClusterData{}, nil
}
