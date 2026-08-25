//go:build stress

package memstore

import (
	"fmt"
	"sync"
	"testing"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/stretchr/testify/require"
)

func TestStressMemStoreDefensiveCopies10K(t *testing.T) {
	store := NewMemStore()
	seed := cistore.GroupData{
		Name: "compute",
		Data: map[string]any{
			"nested": map[string]any{"key": "value"},
			"list":   []any{"a", map[string]any{"b": "c"}},
		},
		File:     cistore.CloudConfigFile{Content: []byte("#cloud-config")},
		Versions: map[string]string{"v1": "one"},
	}
	require.NoError(t, store.AddGroupData(seed.Name, seed))

	const operationCount = 10_000
	var ready sync.WaitGroup
	var start sync.WaitGroup
	var done sync.WaitGroup
	ready.Add(operationCount)
	start.Add(1)
	done.Add(operationCount)

	for i := range operationCount {
		go func() {
			defer done.Done()
			ready.Done()
			start.Wait()

			if i%10 == 0 {
				name := fmt.Sprintf("group-%d", i)
				_ = store.UpdateGroupData(name, cistore.GroupData{Name: name, Data: map[string]any{"index": i}}, true)
				return
			}

			groups := store.GetGroups()
			if group, found := groups["compute"]; found {
				mutateGroupData(group)
			}
			group, err := store.GetGroupData("compute")
			if err == nil {
				mutateGroupData(group)
			}
		}()
	}

	ready.Wait()
	start.Done()
	done.Wait()

	fresh, err := store.GetGroupData("compute")
	require.NoError(t, err)
	require.Equal(t, "value", fresh.Data["nested"].(map[string]any)["key"])
	require.Equal(t, "c", fresh.Data["list"].([]any)[1].(map[string]any)["b"])
	require.Equal(t, []byte("#cloud-config"), fresh.File.Content)
	require.Equal(t, "one", fresh.Versions["v1"])
}
