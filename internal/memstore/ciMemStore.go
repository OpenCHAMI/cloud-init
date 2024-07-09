package memstore

import (
	"errors"
	"log"

	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/samber/lo"
)

var (
	NotFoundErr      = errors.New("not found")
	ExistingEntryErr = errors.New("Data exists for this entry. Update instead")
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

func (m MemStore) Add(name string, ci citypes.CI) error {
	curr := m.list[name]

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

func (m MemStore) Get(name string, sm *smdclient.SMDClient) (citypes.CI, error) {

	ci_merged := new(citypes.CI)

	id, err := sm.IDfromMAC(name)
	if err != nil {
		log.Print(err)
	} else {
		log.Printf("xname %s with mac %s found\n", id, name)
	}

	gl, err := sm.GroupMembership(id)
	if err != nil {
		log.Print(err)
	} else if len(gl) > 0 {
		log.Printf("xname %s is a member of these groups: %s\n", id, gl)

		for g := 0; g < len(gl); g++ {
			if val, ok := m.list[gl[g]]; ok {
				ci_merged.CIData.UserData = lo.Assign(ci_merged.CIData.UserData, val.CIData.UserData)
				ci_merged.CIData.VendorData = lo.Assign(ci_merged.CIData.VendorData, val.CIData.VendorData)
				ci_merged.CIData.MetaData = lo.Assign(ci_merged.CIData.MetaData, val.CIData.MetaData)
			}
		}
	} else {
		log.Printf("ID %s is not a member of any groups\n", name)
	}

	if val, ok := m.list[id]; ok {
		ci_merged.CIData.UserData = lo.Assign(ci_merged.CIData.UserData, val.CIData.UserData)
		ci_merged.CIData.VendorData = lo.Assign(ci_merged.CIData.VendorData, val.CIData.VendorData)
		ci_merged.CIData.MetaData = lo.Assign(ci_merged.CIData.MetaData, val.CIData.MetaData)
	} else {
		log.Printf("ID %s has no specific configuration\n", id)
	}

	if len(ci_merged.CIData.UserData) == 0 &&
		len(ci_merged.CIData.VendorData) == 0 &&
		len(ci_merged.CIData.MetaData) == 0 {
		return citypes.CI{}, NotFoundErr
	} else {
		return *ci_merged, nil
	}
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
