package testing

import (
	"testing"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/stretchr/testify/assert"
)

// RunStoreTests runs a suite of tests against any implementation of cistore.Store
func RunStoreTests(t *testing.T, store cistore.Store, cleanup func()) {
	// Ensure cleanup is called in all cases
	defer func() {
		if cleanup != nil {
			cleanup()
		}
	}()

	// Run each test in a separate sub-test to ensure isolation
	t.Run("Group Operations", func(t *testing.T) {
		// Clear existing data
		groups := store.GetGroups()
		for name := range groups {
			_ = store.RemoveGroupData(name)
		}
		testGroupOperations(t, store)
	})

	t.Run("Instance Operations", func(t *testing.T) {
		// Clear existing data
		groups := store.GetGroups()
		for name := range groups {
			_ = store.RemoveGroupData(name)
		}
		testInstanceOperations(t, store)
	})

	t.Run("Cluster Defaults Operations", func(t *testing.T) {
		// Clear existing data
		groups := store.GetGroups()
		for name := range groups {
			_ = store.RemoveGroupData(name)
		}
		testClusterDefaultsOperations(t, store)
	})
}

func testGroupOperations(t *testing.T, store cistore.Store) {
	// Test data
	testGroup := cistore.GroupData{
		Name:        "test-group",
		Description: "Test group description",
		Data: map[string]interface{}{
			"key1": map[string]interface{}{
				"value": "value1",
			},
			"key2": map[string]interface{}{
				"value": "value2",
			},
		},
		File: cistore.CloudConfigFile{
			Content:  []byte("test content"),
			Name:     "test.yaml",
			Encoding: "plain",
		},
	}

	// Test AddGroupData
	t.Run("Add Group", func(t *testing.T) {
		err := store.AddGroupData(testGroup.Name, testGroup)
		assert.NoError(t, err)

		// Try to add the same group again (should fail)
		err = store.AddGroupData(testGroup.Name, testGroup)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	// Test GetGroupData
	t.Run("Get Group", func(t *testing.T) {
		group, err := store.GetGroupData(testGroup.Name)
		assert.NoError(t, err)
		assert.Equal(t, testGroup.Name, group.Name)
		assert.Equal(t, testGroup.Description, group.Description)
		assert.Equal(t, testGroup.Data, group.Data)
		assert.Equal(t, testGroup.File.Content, group.File.Content)
		assert.Equal(t, testGroup.File.Name, group.File.Name)
		assert.Equal(t, testGroup.File.Encoding, group.File.Encoding)

		// Test non-existent group
		_, err = store.GetGroupData("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	// Test UpdateGroupData
	t.Run("Update Group", func(t *testing.T) {
		updatedGroup := testGroup
		updatedGroup.Description = "Updated description"
		updatedGroup.Data["key1"] = map[string]interface{}{
			"value": "updated-value",
		}
		updatedGroup.File.Content = []byte("updated content")

		// Test update without create
		err := store.UpdateGroupData(testGroup.Name, updatedGroup, false)
		assert.NoError(t, err)

		// Verify update
		group, err := store.GetGroupData(testGroup.Name)
		assert.NoError(t, err)
		assert.Equal(t, updatedGroup.Description, group.Description)
		assert.Equal(t, updatedGroup.Data["key1"], group.Data["key1"])
		assert.Equal(t, updatedGroup.File.Content, group.File.Content)

		// Test update with create
		newGroup := cistore.GroupData{
			Name:        "new-group",
			Description: "New group",
			Data: map[string]interface{}{
				"key1": map[string]interface{}{
					"value": "new-value",
				},
			},
			File: cistore.CloudConfigFile{
				Content:  []byte("new content"),
				Name:     "new.yaml",
				Encoding: "plain",
			},
		}
		err = store.UpdateGroupData(newGroup.Name, newGroup, true)
		assert.NoError(t, err)

		// Verify new group
		group, err = store.GetGroupData(newGroup.Name)
		assert.NoError(t, err)
		assert.Equal(t, newGroup.Name, group.Name)
		assert.Equal(t, newGroup.Description, group.Description)
		assert.Equal(t, newGroup.Data, group.Data)
		assert.Equal(t, newGroup.File.Content, group.File.Content)
		assert.Equal(t, newGroup.File.Name, group.File.Name)
		assert.Equal(t, newGroup.File.Encoding, group.File.Encoding)
	})

	// Test RemoveGroupData
	t.Run("Remove Group", func(t *testing.T) {
		err := store.RemoveGroupData(testGroup.Name)
		assert.NoError(t, err)

		// Verify removal
		_, err = store.GetGroupData(testGroup.Name)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	// Test GetGroups
	t.Run("Get All Groups", func(t *testing.T) {
		groups := store.GetGroups()
		assert.NotNil(t, groups)
		assert.Contains(t, groups, "new-group")
	})
}

func testInstanceOperations(t *testing.T, store cistore.Store) {
	// Test data
	testInstance := cistore.OpenCHAMIInstanceInfo{
		ID:               "test-node",
		InstanceID:       "i-test123",
		LocalHostname:    "test-host",
		Hostname:         "test-host.example.com",
		ClusterName:      "test-cluster",
		Region:           "test-region",
		AvailabilityZone: "test-zone",
		CloudProvider:    "test-provider",
		InstanceType:     "test-type",
		CloudInitBaseURL: "http://test.example.com",
		PublicKeys:       []string{"ssh-rsa test-key"},
	}

	// Test SetInstanceInfo
	t.Run("Set Instance Info", func(t *testing.T) {
		err := store.SetInstanceInfo(testInstance.ID, testInstance)
		assert.NoError(t, err)
	})

	// Test GetInstanceInfo
	t.Run("Get Instance Info", func(t *testing.T) {
		info, err := store.GetInstanceInfo(testInstance.ID)
		assert.NoError(t, err)
		assert.Equal(t, testInstance.ID, info.ID)
		assert.Equal(t, testInstance.InstanceID, info.InstanceID)
		assert.Equal(t, testInstance.LocalHostname, info.LocalHostname)
		assert.Equal(t, testInstance.Hostname, info.Hostname)
		assert.Equal(t, testInstance.ClusterName, info.ClusterName)
		assert.Equal(t, testInstance.Region, info.Region)
		assert.Equal(t, testInstance.AvailabilityZone, info.AvailabilityZone)
		assert.Equal(t, testInstance.CloudProvider, info.CloudProvider)
		assert.Equal(t, testInstance.InstanceType, info.InstanceType)
		assert.Equal(t, testInstance.CloudInitBaseURL, info.CloudInitBaseURL)
		assert.Equal(t, testInstance.PublicKeys, info.PublicKeys)

		// Test non-existent instance
		info, err = store.GetInstanceInfo("non-existent")
		assert.NoError(t, err)
		assert.NotEmpty(t, info.InstanceID)
	})

	// Test DeleteInstanceInfo
	t.Run("Delete Instance Info", func(t *testing.T) {
		err := store.DeleteInstanceInfo(testInstance.ID)
		assert.NoError(t, err)

		// Verify deletion
		info, err := store.GetInstanceInfo(testInstance.ID)
		assert.NoError(t, err)
		assert.Empty(t, info.ID)
		assert.NotEmpty(t, info.InstanceID)
	})
}

func testClusterDefaultsOperations(t *testing.T, store cistore.Store) {
	// Test data
	testDefaults := cistore.ClusterDefaults{
		ClusterName:      "test-cluster",
		ShortName:        "test",
		NidLength:        3,
		BaseUrl:          "http://test.example.com",
		AvailabilityZone: "test-zone",
		Region:           "test-region",
		CloudProvider:    "test-provider",
		PublicKeys:       []string{"ssh-rsa test-key"},
	}

	// Test SetClusterDefaults
	t.Run("Set Cluster Defaults", func(t *testing.T) {
		err := store.SetClusterDefaults(testDefaults)
		assert.NoError(t, err)
	})

	// Test GetClusterDefaults
	t.Run("Get Cluster Defaults", func(t *testing.T) {
		defaults, err := store.GetClusterDefaults()
		assert.NoError(t, err)
		assert.Equal(t, testDefaults.ClusterName, defaults.ClusterName)
		assert.Equal(t, testDefaults.ShortName, defaults.ShortName)
		assert.Equal(t, testDefaults.NidLength, defaults.NidLength)
		assert.Equal(t, testDefaults.BaseUrl, defaults.BaseUrl)
		assert.Equal(t, testDefaults.AvailabilityZone, defaults.AvailabilityZone)
		assert.Equal(t, testDefaults.Region, defaults.Region)
		assert.Equal(t, testDefaults.CloudProvider, defaults.CloudProvider)
		assert.Equal(t, testDefaults.PublicKeys, defaults.PublicKeys)
	})

	// Test partial update
	t.Run("Update Cluster Defaults", func(t *testing.T) {
		partialDefaults := cistore.ClusterDefaults{
			ClusterName: "updated-cluster",
			ShortName:   "updated",
		}

		err := store.SetClusterDefaults(partialDefaults)
		assert.NoError(t, err)

		defaults, err := store.GetClusterDefaults()
		assert.NoError(t, err)
		assert.Equal(t, partialDefaults.ClusterName, defaults.ClusterName)
		assert.Equal(t, partialDefaults.ShortName, defaults.ShortName)
		assert.Equal(t, testDefaults.NidLength, defaults.NidLength)
		assert.Equal(t, testDefaults.BaseUrl, defaults.BaseUrl)
		assert.Equal(t, testDefaults.AvailabilityZone, defaults.AvailabilityZone)
		assert.Equal(t, testDefaults.Region, defaults.Region)
		assert.Equal(t, testDefaults.CloudProvider, defaults.CloudProvider)
		assert.Equal(t, testDefaults.PublicKeys, defaults.PublicKeys)
	})
}
