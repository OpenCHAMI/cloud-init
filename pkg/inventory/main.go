package inventory

import (
	"sync"

	base "github.com/Cray-HPE/hms-base"
	"github.com/OpenCHAMI/smd/v2/pkg/sm"
	"github.com/rs/zerolog/log"
)

type ComponentMapping struct {
	ComponentID   string
	BootMAC       string
	BootIPAddress string
}

// Implements the SMDClientInterface
type LocalInventory struct {
	// Like SMD, we store comonents, Group Membership, Ethernet Interfaces, and BMC information
	components         map[string]base.Component
	componentMutex     *sync.RWMutex
	groupMembership    map[string][]string
	groupMutex         *sync.RWMutex
	ethernetInterfaces []sm.CompEthInterfaceV2
	ethernetMutex      *sync.RWMutex
}

// IDfromMAC returns the ID of the node with the given MAC address
func (l *LocalInventory) IDfromMAC(mac string) (string, error) {
	return "", nil
}

// IDfromIP returns the ID of the node with the given IP address
func (l *LocalInventory) IDfromIP(ipaddr string) (string, error) {
	return "", nil
}

// GroupMembership returns the group labels for the node with the given ID
func (l *LocalInventory) GroupMembership(id string) ([]string, error) {
	return nil, nil
}

// ComponentInformation returns the component information for the node with the given ID
func (l *LocalInventory) ComponentInformation(id string) (base.Component, error) {
	return base.Component{}, nil
}

// AddNodeToInventory adds a node to the inventory
func (l *LocalInventory) AddNodeToInventory(node base.Component) {
}

// NewLocalInventory creates a new LocalInventory object
func NewLocalInventory() *LocalInventory {
	log.Info().Msg("Creating new LocalInventory object")
	return &LocalInventory{
		components:         make(map[string]base.Component),
		groupMembership:    make(map[string][]string),
		ethernetInterfaces: make([]sm.CompEthInterfaceV2, 0),
		componentMutex:     &sync.RWMutex{},
		groupMutex:         &sync.RWMutex{},
		ethernetMutex:      &sync.RWMutex{},
	}
}
