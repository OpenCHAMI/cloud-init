package memstore

import (
	"encoding/json"
	"testing"

	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/stretchr/testify/assert"
)

func TestMemStore_Get(t *testing.T) {
	store := NewMemStore()

	t.Run("No groups found", func(t *testing.T) {
		_, err := store.Get("test-id", []string{})
		assert.EqualError(t, err, "no groups found from SMD")
	})

	t.Run("Group exists in store", func(t *testing.T) {
		group1Data := citypes.GroupData{
			Name: "group1",
			Data: []citypes.MetaDataKV{
				{"key": "value"},
			},
		}
		store.groups = make(map[string]citypes.GroupData)
		store.groups["group1"] = group1Data
		ci, err := store.Get("test-id", []string{"group1"})
		assert.NoError(t, err)
		assert.Equal(t, "test-id", ci.Name)
		assert.NotNil(t, ci.CIData.MetaData)
		assert.Contains(t, ci.CIData.MetaData["groups"], "group1")
	})

	t.Run("Group does not exist in store", func(t *testing.T) {
		ci, err := store.Get("test-id", []string{"group2"})
		assert.NoError(t, err)
		assert.Equal(t, "test-id", ci.Name)
		assert.NotNil(t, ci.CIData.MetaData)
		assert.NotContains(t, ci.CIData.MetaData["groups"], "group2")
	})

	t.Run("Multiple groups exist in store", func(t *testing.T) {
		store.groups["computes"] = citypes.GroupData{
			Data: []citypes.MetaDataKV{
				{"os_version": "rocky9"},
				{"cluster_name": "hill"},
				{"admin": "groves"},
			}}
		store.groups["row1"] = citypes.GroupData{
			Data: []citypes.MetaDataKV{
				{"rack": "rack1"},
				{"syslog_aggregator": "syslog1"},
			}}
		ci, err := store.Get("test-id", []string{"computes", "row1"})
		assert.NoError(t, err)
		assert.Equal(t, "test-id", ci.Name)
		assert.NotNil(t, ci.CIData.MetaData)
		assert.Contains(t, ci.CIData.MetaData["groups"].(map[string][]citypes.MetaDataKV), "row1")
		assert.Contains(t, ci.CIData.MetaData["groups"].(map[string][]citypes.MetaDataKV), "computes")
		ciJSON, err := json.Marshal(ci)
		if err != nil {
			t.Logf("Cloud-init payload: %+v", ci)
			t.Fatalf("Failed to marshal cloud-init data to JSON: %v", err)
		}
		t.Logf("Cloud-init JSON payload: %s", ciJSON)
	})
}

func TestMemStoreAddGroupData(t *testing.T) {
	store := NewMemStore()

	// Test case: Add group data to cloud-init data
	err := store.AddGroupData("computes", citypes.GroupData{
		Data: []citypes.MetaDataKV{
			{"os_version": "rocky9"},
			{"cluster_name": "hill"},
			{"admin": "groves"},
		},
		Actions: map[string]any{
			"write_files": []citypes.WriteFiles{
				{Path: "/etc/hello", Content: "OK COMPUTER"},
			}},
	})
	assert.NoError(t, err)
	store.AddGroupData("row1", citypes.GroupData{
		Data: []citypes.MetaDataKV{
			{"rack": "rack1"},
			{"syslog_aggregator": "syslog1"},
		},
		Actions: map[string]any{
			"write_files": []citypes.WriteFiles{
				{Path: "/etc/hello", Content: "hello world"},
				{Path: "/etc/hello2", Content: "hello world"},
			}},
	})
	assert.NoError(t, err)

	ci, err := store.Get("test-id", []string{"computes", "row1"})
	assert.NoError(t, err)
	assert.Equal(t, "test-id", ci.Name)
	assert.NotNil(t, ci.CIData.MetaData)
	assert.Contains(t, ci.CIData.MetaData["groups"].(map[string][]citypes.MetaDataKV), "row1")
	assert.Contains(t, ci.CIData.MetaData["groups"].(map[string][]citypes.MetaDataKV), "computes")
	assert.Contains(t, ci.CIData.UserData, "write_files")

	ciJSON, err := json.Marshal(ci)
	if err != nil {
		t.Logf("Cloud-init payload: %+v", ci)
		t.Fatalf("Failed to marshal cloud-init data to JSON: %v", err)
	}
	t.Logf("Cloud-init JSON payload: %s", ciJSON)

}
