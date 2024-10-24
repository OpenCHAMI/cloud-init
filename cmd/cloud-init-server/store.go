package main

import (
	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
)

// ciStore is an interface for storing cloud-init entries

type ciStore interface {
	Add(name string, ci citypes.CI) error
	Get(name string, sm *smdclient.SMDClient) (citypes.CI, error)
	List() (map[string]citypes.CI, error)
	Update(name string, ci citypes.CI) error
	Remove(name string) error

	// metadata group API
	AddGroups(name string, groupData map[string]any) error
	GetGroups(name string) (map[string]any, error)
	UpdateGroups(name string, groupData map[string]any) error
	RemoveGroups(name string) error
}
