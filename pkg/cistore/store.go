package cistore

// ciStore is an interface for storing cloud-init entries
type Store interface {
	// groups API
	GetGroups() map[string]GroupData
	AddGroupData(groupName string, groupData GroupData) error
	GetGroupData(groupName string) (GroupData, error)
	UpdateGroupData(groupName string, groupData GroupData) error
	RemoveGroupData(groupName string) error
	// Extended Instance Information API
	GetInstanceInfo(nodeName string) (OpenCHAMIInstanceInfo, error)
	SetInstanceInfo(nodeName string, instanceInfo OpenCHAMIInstanceInfo) error
	DeleteInstanceInfo(nodeName string) error
	// Cluster Defaults
	GetClusterDefaults() (ClusterDefaults, error)
	SetClusterDefaults(clusterDefaults ClusterDefaults) error
}
