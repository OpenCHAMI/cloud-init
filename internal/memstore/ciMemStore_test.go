package memstore

import (
	"encoding/json"
	"testing"

	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/stretchr/testify/assert"
)

func TestMemStoreAddGroupData(t *testing.T) {
	store := NewMemStore()

	// Test case: Add group data to cloud-init data
	err := store.AddGroupData("computes", citypes.GroupData{
		Data: citypes.MetaDataKV{
			"os_version":   "rocky9",
			"cluster_name": "hill",
			"admin":        "groves",
		},
	})
	assert.NoError(t, err)
	store.AddGroupData("row1", citypes.GroupData{
		Data: citypes.MetaDataKV{
			"rack":              "rack1",
			"syslog_aggregator": "syslog1",
		},
	})
	assert.NoError(t, err)

	ci, err := store.GetCIData("test-id", []string{"computes", "row1"})
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
