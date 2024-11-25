package memstore

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/stretchr/testify/assert"
)

func TestMemStore_Get(t *testing.T) {
	store := NewMemStore()

	// Test case: No groups found
	_, err := store.Get("test-id", []string{})
	assert.EqualError(t, err, "no groups found from SMD")

	// Test case: Group exists in store
	store.groups["group1"] = citypes.Group{"data": citypes.GroupData{"key": "value"}}
	ci, err := store.Get("test-id", []string{"group1"})
	assert.NoError(t, err)
	assert.Equal(t, "test-id", ci.Name)
	assert.NotNil(t, ci.CIData.MetaData)
	assert.Contains(t, ci.CIData.MetaData["groups"].(map[string]citypes.GroupData), "group1")

	// Test case: Group does not exist in store
	ci, err = store.Get("test-id", []string{"group2"})
	assert.NoError(t, err)
	assert.Equal(t, "test-id", ci.Name)
	assert.NotNil(t, ci.CIData.MetaData)
	assert.NotContains(t, ci.CIData.MetaData["groups"].(map[string]citypes.GroupData), "group2")

	// Test case: Multiple groups exist in store
	store.groups["computes"] = citypes.Group{"data": citypes.GroupData{
		"os_version":   "rocky9",
		"cluster_name": "hill",
		"admin":        "groves",
	}}
	store.groups["row1"] = citypes.Group{"data": citypes.GroupData{
		"rack":              "rack1",
		"syslog_aggregator": "syslog1",
	}}
	ci, err = store.Get("test-id", []string{"computes", "row1"})
	assert.NoError(t, err)
	assert.Equal(t, "test-id", ci.Name)
	assert.NotNil(t, ci.CIData.MetaData)
	assert.Contains(t, ci.CIData.MetaData["groups"].(map[string]citypes.GroupData), "row1")
	assert.Contains(t, ci.CIData.MetaData["groups"].(map[string]citypes.GroupData), "computes")
	// Print the full cloud-init payload
	fmt.Printf("Cloud-init payload: %+v", ci)
}

func TestMemStoreAddGroupData(t *testing.T) {
	store := NewMemStore()

	// Test case: Add group data to cloud-init data
	err := store.AddGroupData("computes", citypes.GroupData{"data": citypes.GroupData{
		"os_version":   "rocky9",
		"cluster_name": "hill",
		"admin":        "groves",
	},
		"actions": citypes.GroupData{"write_files": []citypes.WriteFiles{
			{Path: "/etc/hello", Content: "OK COMPUTER"},
		}},
	})
	assert.NoError(t, err)
	store.AddGroupData("row1", citypes.GroupData{"data": citypes.GroupData{
		"rack":              "rack1",
		"syslog_aggregator": "syslog1",
	},
		"actions": citypes.GroupData{"write_files": []citypes.WriteFiles{
			{Path: "/etc/hello", Content: "hello world"},
			{Path: "/etc/hello2", Content: "hello world"},
		}},
	})
	assert.NoError(t, err)

	ci, err := store.Get("test-id", []string{"computes", "row1"})
	assert.NoError(t, err)
	assert.Equal(t, "test-id", ci.Name)
	assert.NotNil(t, ci.CIData.MetaData)
	assert.Contains(t, ci.CIData.MetaData["groups"].(map[string]citypes.GroupData), "row1")
	assert.Contains(t, ci.CIData.MetaData["groups"].(map[string]citypes.GroupData), "computes")
	assert.Contains(t, ci.CIData.UserData, "write_files")

	ciJSON, err := json.Marshal(ci)
	if err != nil {
		t.Logf("Cloud-init payload: %+v", ci)
		t.Fatalf("Failed to marshal cloud-init data to JSON: %v", err)
	}
	t.Logf("Cloud-init JSON payload: %s", ciJSON)

}
