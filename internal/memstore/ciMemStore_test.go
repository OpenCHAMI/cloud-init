package memstore

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	storetesting "github.com/OpenCHAMI/cloud-init/pkg/cistore/testing"
)

func TestMemStore(t *testing.T) {
	// Create a new MemStore instance
	store := NewMemStore()

	// Create a cleanup function that will be called in all cases
	cleanup := func() {
		// No cleanup needed for MemStore as it's in-memory
	}

	// Run the standard test suite
	storetesting.RunStoreTests(t, store, cleanup)
}

func TestNewMemStoreFromPath(t *testing.T) {
	testDir, err := os.MkdirTemp("", "cimemstore")
	require.NoError(t, err)
	invalidDir, err := os.MkdirTemp("", "cimemstore")
	require.NoError(t, err)

	defer os.RemoveAll(testDir)
	defer os.RemoveAll(invalidDir)

	_, err = NewMemStoreFromPath(testDir)
	require.Error(t, err)
	require.ErrorContains(t, err, fmt.Sprintf("error opening %s", filepath.Join(testDir, groupsFile)))

	for _, file := range []string{groupsFile, instancesFile, defaultsFile} {
		err := os.WriteFile(filepath.Join(invalidDir, file), []byte(testInvalidFile), 0666)
		require.NoError(t, err)
	}

	_, err = NewMemStoreFromPath(invalidDir)
	require.Error(t, err)
	require.ErrorContains(t, err, fmt.Sprintf("error unmarshaling %s", groupsFile))

	err = os.WriteFile(filepath.Join(testDir, groupsFile), []byte(testGroupsFile), 0666)
	err = os.WriteFile(filepath.Join(testDir, instancesFile), []byte(testInstancesFile), 0666)
	err = os.WriteFile(filepath.Join(testDir, defaultsFile), []byte(testDefaultsFile), 0666)
	require.NoError(t, err)

	store, err := NewMemStoreFromPath(testDir)
	require.NoError(t, err)
	require.Len(t, store.Groups, 3)
	require.Len(t, store.Instances, 0)
	require.Equal(t, "http://test.example/", store.ClusterDefaults.BaseUrl)
	require.Equal(t, "Login nodes", store.Groups["login"].Description)

}

// Instances and groups follow similar map[string]Type structs. The files must be present, but may be empty if no
// such resources are loaded, so instances is empty to confirm this works.

const (
	testGroupsFile = `allnodes:
  name: allnodes
  description: All nodes
  file:
    content: I2Nsb3VkLWNvbmZpZwpzc2hfYXV0aG9yaXplZF9rZXlzOgotICJzc2gtcnNhIEFBQUFCM056YUMxeWMyRUFBQUFEQVFBQkFBQUJnUUNHejVpSjFGRjVBWFA1eGVWbE9EdldlRGpiblllL25KdDkwS21ySlhwL3FveFd4RTc5WVRwWlhlWlVPeTVDdXZWME9ObG5nK3crS0lheDZrTkFsMWVkUTQyQ2hZMUFBdXdDNWlFb0Y3VmZuS25ndWhJTS9YakxidWtwbGp4NU5SeUg0L1VoMmJ6RzhXNVNiM3ptQWRUNitYMlVkcTBqMFF0RWtWaEFVbnUycXdLZGdZdWJnS0JEWENZWDdqWXdEaWxNM0pvdE1IcFJ4WS8wZEt2QW81VE45VUo0ZGJaWGIxMldaWlBTWWgxeVJDNXB1SnJLMURFQ2lZMzRmKysxWkhyYXB5TlZnODBmN09KSWJxRVNrMkNNMk5jeXNLdU03dkRxMVdLam1QM0p2WTFvdXppbUllcVFadjg1Qis3UWlpZkMxS2JJMHM1dGZNWlh6akxBUWhYT2FzaTczT3oxQzlsN21SRFVtSnoraVFSWm83WXp5NnNXaUUrdVlhK0hCa2docnpSeTNKaEZjUTdaVWhGVllQVE4xNFZEbDhpY1R5RWY5c0lBUUJmb3VLY3orSEloS2RLMkVaR0ppOGlBaUFOTnNwUmNOQWRMajExaEpvSDFOM2kyeER2L0JvK25lWjhlT05pelZBZlF5U245eGVGWC9FSG1uR2lGak1EOVVncz0gZm9vQGV4YW1wbGUiCi0gInNzaC1yc2EgQUFBQUIzTnphQzF5YzJFQUFBQURBUUFCQUFBQmdRQzAvd0tDSHVqZXQyYmU2dWFLSEZ3MXk5TVVBUWZSYnZ3eUQyN3Bab3VQSmxkcThEYVlROExTMkdkbEhmTDYxRVp0Y0p0Mno3ZWZPWkV6YXVqWFlKTk9VZ1Q2YU9vdFZpZ0tZMnhPVmM3RmxVYXdyd2RtTlR0RGsrMXBXT0dadHZJU3g0cXU0NExrNzlXMzZTeGF3aTdheXovNGpOQy9TSFQyTmRqSEF6L3YzY0ZiN3k3R0pmNjQzL2pic0hCOVRWcllsaXY0S0VnRnBHNkdQcUdtanJCY3kxWXJYN1JZem53V2lYaVFrZlpSVUpLbUl1a2pnenAyZlllT28wVWNJT0lZcGs3RGI1TnlSQXNMWkxtWU5sdy9ZWC9xWnN6dkNvYkEzeUtlaUNBYWlFUmtxcFVnNE5Cd2xSMzBCY1RtandUMWNwY256am4zTHN4MUx4akc2RlYreHJTYkxhd3djcFlWeG5iMkVuWkpYbFFOZzdqSmZSc3ZoNEp1ZjlUUWZONS9IWlBvV0huS0pjZFVLaXoyTmtXckZjUE9sVHAvVCs4VzExakp6MjYwY3UxQURucW5EbWNUaVV2SXF2WjBJMGU1amhaay9oMnJ6UDNpSWlUQzhkdEgzMmY0OXZIcGFBbXhTamZZNzV4YnpueDM3NGtaYkY4N2krdkFsNWRFV1JNPSBiYXJAZXhhbXBsZSIK
    encoding: base64
login:
  name: login
  description: Login nodes
  file:
    content: I2Nsb3VkLWNvbmZpZwpwYWNrYWdlczoKLSBmb3J0dW5lLW1vZAp3cml0ZV9maWxlczoKLSBlbmNvZGluZzogYjY0CiAgY29udGVudDogImFHVnNiRzhnYkc5bmFXNEsiCiAgb3duZXI6IG11bmdlOm11bmdlCiAgcGF0aDogL2V0Yy9oZWxsbwogIHBlcm1pc3Npb25zOiAnMDYwMCcK
    encoding: base64
compute:
  name: compute
  description: Compute nodes
  file:
    content: I2Nsb3VkLWNvbmZpZwpwYWNrYWdlczoKLSBjb3dzYXkKd3JpdGVfZmlsZXM6Ci0gZW5jb2Rpbmc6IGI2NAogIGNvbnRlbnQ6ICJhR1ZzYkc4Z1kyOXRjSFYwWlFvPSIKICBvd25lcjogcm9vdDpyb290CiAgcGF0aDogL2V0Yy9jb21wdXRlX2hlbGxvCiAgcGVybWlzc2lvbnM6ICcwNjAwJwo=
    encoding: base64
`
	testDefaultsFile = `cloud-provider: openchami
region: us-west-2
availability-zone: us-west-2a
cluster-name: venado
base-url: http://test.example/
`
	testInstancesFile = ``

	testInvalidFile = `this is not yaml`
)
