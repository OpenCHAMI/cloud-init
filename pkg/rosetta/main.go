package rosetta

import (
	"errors"
)

// IDMapping holds the IDs for a specific entity
type IDMapping struct {
	XName      string
	InstanceID string
	NodeID     string
}

// IDMapper provides lookup maps for IDs
type IDMapper struct {
	xnameToID      map[string]IDMapping
	instanceIDToID map[string]IDMapping
	nodeIDToID     map[string]IDMapping
}

// NewIDMapper creates a new IDMapper
func NewIDMapper() *IDMapper {
	return &IDMapper{
		xnameToID:      make(map[string]IDMapping),
		instanceIDToID: make(map[string]IDMapping),
		nodeIDToID:     make(map[string]IDMapping),
	}
}

// AddMapping adds a new ID mapping
func (m *IDMapper) AddMapping(xname, instanceID, nodeID string) error {
	if _, exists := m.xnameToID[xname]; exists {
		return errors.New("xname already exists")
	}
	if _, exists := m.instanceIDToID[instanceID]; exists {
		return errors.New("instanceID already exists")
	}
	if _, exists := m.nodeIDToID[nodeID]; exists {
		return errors.New("nodeID already exists")
	}

	mapping := IDMapping{
		XName:      xname,
		InstanceID: instanceID,
		NodeID:     nodeID,
	}

	m.xnameToID[xname] = mapping
	m.instanceIDToID[instanceID] = mapping
	m.nodeIDToID[nodeID] = mapping

	return nil
}

// GetByXName retrieves the ID mapping by xname
func (m *IDMapper) GetByXName(xname string) (IDMapping, error) {
	if mapping, exists := m.xnameToID[xname]; exists {
		return mapping, nil
	}
	return IDMapping{}, errors.New("xname not found")
}

// GetByInstanceID retrieves the ID mapping by instanceID
func (m *IDMapper) GetByInstanceID(instanceID string) (IDMapping, error) {
	if mapping, exists := m.instanceIDToID[instanceID]; exists {
		return mapping, nil
	}
	return IDMapping{}, errors.New("instanceID not found")
}

// GetByNodeID retrieves the ID mapping by nodeID
func (m *IDMapper) GetByNodeID(nodeID string) (IDMapping, error) {
	if mapping, exists := m.nodeIDToID[nodeID]; exists {
		return mapping, nil
	}
	return IDMapping{}, errors.New("nodeID not found")
}
