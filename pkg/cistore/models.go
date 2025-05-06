package cistore

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"

	base "github.com/Cray-HPE/hms-base"
)

type GroupData struct {
	Name        string                 `json:"name" example:"compute" description:"Group name"`
	Description string                 `json:"description,omitempty" example:"The compute group" description:"A short description of the group"`
	Data        map[string]interface{} `json:"meta-data,omitempty" description:"json map of a string (key) to a struct (value) representing group meta-data"`
	File        CloudConfigFile        `json:"file,omitempty" description:"Cloud-Init configuration for group"`
}

func (g *GroupData) ParseFromJSON(body []byte) error {
	// Parse the JSON
	if err := json.Unmarshal(body, g); err != nil {
		return err
	}

	// Perform validation
	if g.Name == "" {
		return errors.New("name is required")
	}

	return nil
}

type OpenCHAMIComponent struct {
	base.Component
	MAC  string `json:"mac"`            // MAC address of the inteface used to boot the component
	IP   string `json:"ip"`             // IP address of the interface used to boot the component
	WGIP string `json:"wgip,omitempty"` // Wireguard IP address of the interface used for cloud-init
}

type OpenCHAMIInstanceInfo struct {
	ID               string   `json:"id" example:"x3000c1b1n1" description:"Node unique identifier, on systems that support xnames, this will be an xname which includes location information"`
	InstanceID       string   `json:"instance-id" yaml:"instance-id"`
	LocalHostname    string   `json:"local-hostname,omitempty" yaml:"local-hostname" example:"compute-1" description:"Node-specific hostname"`
	Hostname         string   `json:"hostname,omitempty" yaml:"hostname"`
	ClusterName      string   `json:"cluster-name,omitempty" yaml:"cluster-name" example:"demo" description:"Long name of entire cluster, used as a human-readable identifier and is used in the cluster's FQDN"`
	Region           string   `json:"region,omitempty" yaml:"region"`
	AvailabilityZone string   `json:"availability-zone,omitempty" yaml:"availability-zone"`
	CloudProvider    string   `json:"cloud-provider,omitempty" yaml:"cloud-provider"`
	InstanceType     string   `json:"instance-type,omitempty" yaml:"instance-type"`
	CloudInitBaseURL string   `json:"cloud-init-base-url,omitempty" yaml:"cloud-init-base-url"`
	PublicKeys       []string `json:"public-keys,omitempty" yaml:"public-keys,omitempty" example:"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMLtQNuzGcMDatF+YVMMkuxbX2c5v2OxWftBhEVfFb+U user1@demo-head,ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIB4vVRvkzmGE5PyWX2fuzJEgEfET4PRLHXCnD1uFZ8ZL user2@demo-head"`
}

// ClusterDefaults represents the possible meta-data that can be set as default
// values for a cluster.
type ClusterDefaults struct {
	CloudProvider    string   `json:"cloud_provider,omitempty" yaml:"cloud-provider,omitempty"`
	Region           string   `json:"region,omitempty" yaml:"region,omitempty"`
	AvailabilityZone string   `json:"availability-zone,omitempty" yaml:"availability-zone,omitempty"`
	ClusterName      string   `json:"cluster-name,omitempty" yaml:"cluster-name,omitempty" example:"demo" description:"Long name of entire cluster, used as a human-readable identifier and is used in the cluster's FQDN"`
	PublicKeys       []string `json:"public-keys,omitempty" yaml:"public-keys,omitempty" example:"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMLtQNuzGcMDatF+YVMMkuxbX2c5v2OxWftBhEVfFb+U user1@demo-head,ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIB4vVRvkzmGE5PyWX2fuzJEgEfET4PRLHXCnD1uFZ8ZL user2@demo-head"`
	BaseUrl          string   `json:"base-url,omitempty" yaml:"base-url,omitempty" example:"http://demo.openchami.cluster:8081/cloud-init"`
	BootSubnet       string   `json:"boot-subnet,omitempty" yaml:"boot-subnet,omitempty"`
	WGSubnet         string   `json:"wg-subnet,omitempty" yaml:"wg-subnet,omitempty"`
	ShortName        string   `json:"short-name,omitempty" yaml:"short-name,omitempty" example:"nid" description:"Shortened name of cluster; this string is prepended to padded NID and set as node hostname if hostname is not set for node"`
	NidLength        int      `json:"nid-length,omitempty" yaml:"nid-length,omitempty" example:"3" description:"Width of digits for node ID"`
}

type CloudConfigFile struct {
	Content  []byte `json:"content" yaml:"content" swaggertype:"string" example:"IyMgdGVtcGxhdGU6IGppbmphCiNjbG91ZC1jb25maWcKbWVyZ2VfaG93OgotIG5hbWU6IGxpc3QKICBzZXR0aW5nczogW2FwcGVuZF0KLSBuYW1lOiBkaWN0CiAgc2V0dGluZ3M6IFtub19yZXBsYWNlLCByZWN1cnNlX2xpc3RdCnVzZXJzOgogIC0gbmFtZTogcm9vdAogICAgc3NoX2F1dGhvcml6ZWRfa2V5czoge3sgZHMubWV0YV9kYXRhLmluc3RhbmNlX2RhdGEudjEucHVibGljX2tleXMgfX0KZGlzYWJsZV9yb290OiBmYWxzZQo=" description:"Cloud-Init configuration content whose encoding depends on the value of 'encoding'"`
	Name     string `json:"filename" yaml:"filename"`
	Encoding string `json:"encoding,omitempty" yaml:"encoding,omitempty" enums:"base64,plain"`
}

// UnmarshalJSON implements json.Unmarshaler
func (f *CloudConfigFile) UnmarshalJSON(data []byte) error {
	// Use an auxiliary struct so that:
	//
	// 1. json.Unmarshal doesn't recurse forever and overflow the stack.
	// 2. json.Unmarshal doesn't try to base64-decode "content" in the data
	//    before assigning the bytes to f.Content. Content is unmarshalled
	//    as a string instead of bytes in order to prevent this. After
	//    unmarshalling, the string is converted back to bytes and assigned
	//    to f.Content.
	type Alias CloudConfigFile
	aux := &struct {
		Content string `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(f),
	}

	// Unmarshal into the helper struct
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Handle encoding
	switch aux.Encoding {
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(aux.Content)
		if err != nil {
			return fmt.Errorf("failed to decode base64 content: %w", err)
		}
		f.Content = decoded
	case "plain":
		f.Content = []byte(aux.Content)
	default:
		return fmt.Errorf("unsupported encoding: %s", aux.Encoding)
	}

	return nil
}

// Custom YAML unmarshaller for CloudConfigFile. This is needed because the yaml
// library cannot unmarshal string to []byte like json can, so it needs to be
// told how to do so.
func (f *CloudConfigFile) UnmarshalYAML(n *yaml.Node) error {
	// Use an auxiliary struct so that:
	//
	// 1. json.Unmarshal doesn't recurse forever and overflow the stack.
	// 2. json.Unmarshal doesn't try to base64-decode "content" in the data
	//    before assigning the bytes to f.Content. Content is unmarshalled
	//    as a string instead of bytes in order to prevent this. After
	//    unmarshalling, the string is converted back to bytes and assigned
	//    to f.Content.
	//
	// Interestingly, yaml.Unmarshal will not unmarshal into aux's pointer
	// to f, so we have to use json.Unmarshal.
	type Alias CloudConfigFile
	aux := &struct {
		Content string `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(f),
	}

	// Decode YAML document (n) into aux struct, using a map as an
	// intermediary (since n.Decode() cannot decode into a []byte). We have
	// to use JSON for the unmarshalling (and, by consequence, the
	// marshalling) so that f will get written via aux's pointer to it. For
	// some reason, the yaml unmarshaller will not do that.
	//
	// 1. Decode YAML document (n) into map.
	// 2. JSON marshal map into bytes.
	// 3. JSON unmarshal bytes into aux struct.
	// 4. Set f.Content to byte-ified aux.Content.
	var mAux map[string]interface{}
	if err := n.Decode(&mAux); err != nil {
		return err
	}
	t, err := json.Marshal(mAux)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(t, &aux); err != nil {
		return err
	}
	f.Content = []byte(aux.Content)

	return nil
}

// Custom JSON marshaler for CloudConfigFile
func (f CloudConfigFile) MarshalJSON() ([]byte, error) {
	// Use an auxiliary struct so that:
	//
	// 1. json.Marshal doesn't recurse forever and overflow the stack.
	// 2. json.Marshal doesn't try to base64-encode f.Content. f.Content is
	//    converted from bytes to a string and then assigned to aux.Content
	//    to prevent this. Then, aux gets marshalled instead of f.
	type Alias CloudConfigFile
	aux := &struct {
		Content string `json:"content"`
		Alias
	}{
		Alias: (Alias)(f),
	}

	return json.Marshal(aux)
}

// Custom YAML marshaler for CloudConfigFile. This is needed because the yaml
// library will not marshal []byte into string like json will, so it needs to be
// told how to do so.
func (f CloudConfigFile) MarshalYAML() (interface{}, error) {
	// Use an auxiliary struct so that:
	//
	// 1. yaml.Marshal doesn't recurse forever and overflow the stack.
	// 2. yaml.Marshal doesn't try to base64-encode f.Content. f.Content is
	//    converted from bytes to a string and then assigned to aux.Content
	//    to prevent this. Then, aux gets marshalled instead of f.
	//
	// aux is set to the values of f, but has its own string Content, set to
	// the stringified f.Content.
	type Alias CloudConfigFile
	aux := &struct {
		Content string `yaml:"content"`
		Alias
	}{
		Content: string(f.Content),
		Alias:   (Alias)(f),
	}

	// Convert aux into map, which has an "alias" key containing the
	// "content" that will be set to the actual string value. The map that
	// is mapped to "alias" is what is returned.
	//
	// 1. YAML marshal aux to bytes.
	// 2. YAML unmarshal bytes into map.
	// 3. Set content in "alias" to aux.Content.
	// 4. Return "alias" map.
	t, err := yaml.Marshal(aux)
	if err != nil {
		return nil, err
	}
	var nMap map[string]interface{}
	if err := yaml.Unmarshal(t, &nMap); err != nil {
		return nil, err
	}
	switch c := (nMap["alias"]).(type) {
	case map[string]interface{}:
		c["content"] = aux.Content
	default:
		return nil, fmt.Errorf("cloud config file in map is unknown type: wanted=(map[string]interface{}) actual=(%v)", c)
	}

	return nMap["alias"], nil
}
