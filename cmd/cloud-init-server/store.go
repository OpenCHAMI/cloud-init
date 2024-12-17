package main

import (
	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
)

// ciStore is an interface for storing cloud-init entries
type ciStore interface {
	// groups API
	GetGroups() map[string]citypes.GroupData
	AddGroupData(groupName string, groupData citypes.GroupData) error
	GetGroupData(groupName string) (citypes.GroupData, error)
	UpdateGroupData(groupName string, groupData citypes.GroupData) error
	RemoveGroupData(groupName string) error
	// Extended Instance Information API
	GetInstanceInfo(nodeName string) (citypes.OpenCHAMIInstanceInfo, error)
	SetInstanceInfo(nodeName string, instanceInfo citypes.OpenCHAMIInstanceInfo) error
}
