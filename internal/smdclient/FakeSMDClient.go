package smdclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	base "github.com/Cray-HPE/hms-base"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
)

type FakeSMDClient struct {
	clusterName     string
	components      map[string]base.Component
	groups          map[string][]string // map of group id to list of components
	rosetta_mapping []SMDRosettaStone
}

type SMDRosettaStone struct {
	ComponentID   string
	BootMAC       string
	BootIPAddress string
	NID           string
	Hostname      string
}

func NewFakeSMDClient(clusterName string, count int) *FakeSMDClient {
	client := &FakeSMDClient{}
	client.clusterName = clusterName
	component_map, rosetta, err := generateFakeComponents(count, "10.20.30.0/20")
	if err != nil {
		panic(err)
	}
	client.components = component_map
	client.rosetta_mapping = rosetta
	// create a group for each cabinet and add all the components in that cabinet to the group
	client.groups = make(map[string][]string)
	for _, c := range rosetta {
		cabinet := strings.Split(c.ComponentID, "c")[0]
		if _, ok := client.groups[cabinet]; !ok {
			client.groups[cabinet] = make([]string, 0)
		}
		client.groups[cabinet] = append(client.groups[cabinet], c.ComponentID)
	}
	// create a group that includes all nodes called compute
	client.groups["compute"] = make([]string, 0)
	for _, c := range rosetta {
		client.groups["compute"] = append(client.groups["compute"], c.ComponentID)
	}
	// remove 10% of the nodes from the compute group
	for i := 0; i < len(client.groups["compute"])/10; i++ {
		client.groups["compute"] = client.groups["compute"][:len(client.groups["compute"])-1]
	}
	// create an io group that includes 20% of the nodes from the compute group
	client.groups["io"] = make([]string, 0)
	for i := 0; i < len(client.groups["compute"])/5; i++ {
		client.groups["io"] = append(client.groups["io"], client.groups["compute"][i])
	}
	return client
}

func (f *FakeSMDClient) ClusterName() string {
	return f.clusterName
}

func (f *FakeSMDClient) IDfromMAC(mac string) (string, error) {
	for _, c := range f.rosetta_mapping {
		if c.BootMAC == mac {
			return c.ComponentID, nil
		}
	}
	return "", errors.New("not found")
}

func (f *FakeSMDClient) IDfromIP(ipaddr string) (string, error) {
	for _, c := range f.rosetta_mapping {
		if c.BootIPAddress == ipaddr {
			return c.ComponentID, nil
		}
	}
	return "", errors.New("not found")
}

func (f *FakeSMDClient) IPfromID(id string) (string, error) {
	for _, c := range f.rosetta_mapping {
		if c.ComponentID == id {
			return c.BootIPAddress, nil
		}
	}
	return "", errors.New("not found")
}

func (f *FakeSMDClient) MACfromID(id string) (string, error) {
	for _, c := range f.rosetta_mapping {
		if c.ComponentID == id {
			return c.BootMAC, nil
		}
	}
	return "", errors.New("not found")
}

func (f *FakeSMDClient) GroupMembership(id string) ([]string, error) {
	myGroups := make([]string, 0)
	for group, components := range f.groups {
		for _, c := range components {
			if c == id {
				myGroups = append(myGroups, group)
			}
		}
	}
	return myGroups, nil
}

func (f *FakeSMDClient) ComponentInformation(id string) (base.Component, error) {
	log.Debug().Msgf("FakeSMDClient: ComponentInformation(%s)", id)
	log.Debug().Msgf("FakeSMDClient: %d components from %s to %s", len(f.components), f.rosetta_mapping[0].ComponentID, f.rosetta_mapping[len(f.rosetta_mapping)-1].ComponentID)
	if c, ok := f.components[id]; ok {
		log.Debug().Msgf("FakeSMDClient: ComponentInformation(%s) found", id)
		groups, _ := f.GroupMembership(id)
		log.Debug().Msgf("FakeSMDClient: Groups for %s found: %v", id, groups)
		return c, nil
	}
	return base.Component{}, errors.New("component not found in fake SMD client")
}

func (f *FakeSMDClient) Summary() {
	fmt.Printf("FakeSMDClient: %d components from %s to %s\n", len(f.components), f.rosetta_mapping[0].ComponentID, f.rosetta_mapping[len(f.rosetta_mapping)-1].ComponentID)
	groupNames := make([]string, 0, len(f.groups))
	for groupName := range f.groups {
		groupNames = append(groupNames, groupName)
	}
	fmt.Printf("FakeSMDClient: %d groups [%v]\n", len(f.groups), groupNames)
}

func incrementIP(ip net.IP) net.IP {
	ip = ip.To4()
	if ip == nil {
		panic("invalid IPv4 address")
	}
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
	return ip
}

func incrementMAC(mac string) string {
	macParts := strings.Split(mac, ":")
	for i := len(macParts) - 1; i >= 0; i-- {
		part, err := strconv.ParseInt(macParts[i], 16, 0)
		if err != nil {
			panic(err)
		}
		part++
		if part <= 0xFF {
			macParts[i] = fmt.Sprintf("%02X", part)
			break
		} else {
			macParts[i] = "00"
		}
	}
	return strings.Join(macParts, ":")
}

func incrementXname(xname string) (string, error) {
	// XNames are in the format xNcYbXnZ where N is the cabinet number, Y is the chassis number, X is the BMC number, and Z is the node number
	// There are 4 nodes per BMC
	// There are 8 BMCs per chassis
	// There are 4 chassis per cabinet

	parts := strings.Split(xname, "c")
	if len(parts) != 2 {
		return "", errors.New("invalid xname format")
	}
	cabinetNumber, err := strconv.Atoi(parts[0][1:])
	if err != nil {
		return "", err
	}

	cabinetParts := strings.Split(parts[1], "b")
	if len(cabinetParts) != 2 {
		return "", errors.New("invalid xname format")
	}
	chassisNumber, err := strconv.Atoi(cabinetParts[0])
	if err != nil {
		return "", err
	}

	chassisParts := strings.Split(cabinetParts[1], "n")
	if len(chassisParts) != 2 {
		return "", errors.New("invalid xname format")
	}
	bmcNumber, err := strconv.Atoi(chassisParts[0])
	if err != nil {
		return "", err
	}

	nodeNumber, err := strconv.Atoi(chassisParts[1])
	if err != nil {
		return "", err
	}
	if nodeNumber > 3 {
		return "", errors.New("invalid node number")
	}

	// If we're at the last node, increment the BMC number
	if nodeNumber == 3 {
		nodeNumber = 0
		// If we're at the last BMC, increment the chassis number
		if bmcNumber == 7 {
			bmcNumber = 0
			// If we're at the last chassis, increment the cabinet number
			if chassisNumber == 3 {
				chassisNumber = 0
				cabinetNumber++
			} else {
				chassisNumber++
			}
		} else {
			bmcNumber++
		}
	} else {
		nodeNumber++
	}

	return fmt.Sprintf("x%dc%db%dn%d", cabinetNumber, chassisNumber, bmcNumber, nodeNumber), nil
}

func generateFakeComponents(numComponents int, cidr string) (map[string]base.Component, []SMDRosettaStone, error) {
	components := make(map[string]base.Component)
	rosettaMapping := make([]SMDRosettaStone, 0, numComponents)

	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, nil, err
	}

	var (
		mac   = "00:DE:AD:BE:EF:00"
		xname = "x3000c0b0n0"
	)

	for i := 1; i <= numComponents; i++ {
		mac = incrementMAC(mac)
		xname, _ = incrementXname(xname)
		component := base.Component{
			ID:      xname,
			Type:    "Node",
			NetType: "Ethernet",
			NID:     json.Number(fmt.Sprintf("%d", i)),
		}
		components[xname] = component

		ip = incrementIP(ip)
		if !ipNet.Contains(ip) {
			return nil, nil, fmt.Errorf("CIDR range exceeded")
		}

		rosettaMapping = append(rosettaMapping, SMDRosettaStone{
			ComponentID:   xname,
			BootMAC:       mac,
			BootIPAddress: ip.String(),
		})
	}

	return components, rosettaMapping, nil
}

func (f *FakeSMDClient) PopulateNodes() {
	// no-op
}

// ***** Simulated SMD Client functions.  Not part of the SMDClientInterface *****

// AddNodeToInventory adds a node to the inventory.  This is not part of the SMDClient Interface and only useful as part of the simulator
func (f *FakeSMDClient) AddNodeToInventory(node cistore.OpenCHAMIComponent) error {
	log.Debug().Msgf("FakeSMDClient: AddNodeToInventory(%s)", node.ID)
	// if the node already exists, return an error
	if _, ok := f.components[node.ID]; ok {
		return errors.New("node already exists")
	}
	// if the ip/mac is already in use, return an error
	for _, c := range f.rosetta_mapping {
		if c.BootMAC == node.MAC || c.BootIPAddress == node.IP {
			return errors.New("ip/mac already in use")
		}
	}
	f.components[node.ID] = node.Component
	f.rosetta_mapping = append(f.rosetta_mapping, SMDRosettaStone{
		ComponentID:   node.ID,
		BootMAC:       node.MAC,
		BootIPAddress: node.IP,
	})
	return nil
}

// AddNodeToGroups adds a node to the specified groups.  This is not part of the SMDClient Interface and only useful as part of the simulator
func (f *FakeSMDClient) AddNodeToGroups(id string, groups []string) error {
	log.Debug().Msgf("FakeSMDClient: AddNodeToGroups(%s, %v)", id, groups)
	for _, group := range groups {
		if _, ok := f.groups[group]; !ok {
			f.groups[group] = make([]string, 0)
		}
		f.groups[group] = append(f.groups[group], id)
	}
	return nil
}

func (f *FakeSMDClient) ListNodes() []cistore.OpenCHAMIComponent {
	nodes := make([]cistore.OpenCHAMIComponent, 0)
	for _, c := range f.rosetta_mapping {
		nodes = append(nodes, cistore.OpenCHAMIComponent{
			MAC:       c.BootMAC,
			IP:        c.BootIPAddress,
			Component: f.components[c.ComponentID],
		})
	}
	return nodes
}

func (f *FakeSMDClient) UpdateNode(node cistore.OpenCHAMIComponent) error {
	log.Debug().Msgf("FakeSMDClient: UpdateNode(%s)", node.ID)
	// if the node does not exist, return an error
	if _, ok := f.components[node.ID]; !ok {
		return errors.New("node does not exist")
	}
	// if the ip/mac is already in use, return an error
	for _, c := range f.rosetta_mapping {
		if c.BootMAC == node.MAC || c.BootIPAddress == node.IP {
			if c.ComponentID != node.ID {
				return errors.New("ip/mac already in use")
			}
		}
	}
	f.components[node.ID] = node.Component
	for i, c := range f.rosetta_mapping {
		if c.ComponentID == node.ID {
			if node.MAC != "" {
				f.rosetta_mapping[i].BootMAC = node.MAC
			}
			if node.IP != "" {
				f.rosetta_mapping[i].BootIPAddress = node.IP
			}
			break
		}
	}
	return nil
}
