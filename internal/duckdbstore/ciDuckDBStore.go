package duckdbstore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	_ "github.com/marcboeker/go-duckdb"
)

type DuckDBStore struct {
	db *sql.DB
	mu sync.RWMutex
}

func NewDuckDBStore(dsn string) (*DuckDBStore, error) {
	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, err
	}
	return &DuckDBStore{db: db}, nil
}

func (d *DuckDBStore) GetGroups() (map[string]cistore.GroupData, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.db.Query("SELECT name, description, data, file, versions FROM groups")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close() // Ignoring error on deferred Close
	}()

	groups := make(map[string]cistore.GroupData)
	for rows.Next() {
		var group cistore.GroupData
		var data, file, versions []byte
		if err := rows.Scan(&group.Name, &group.Description, &data, &file, &versions); err != nil {
			continue
		}
		err := json.Unmarshal(data, &group.Data)
		if err != nil {
			return nil, err
		}
		group.File.Content = file
		if err = json.Unmarshal(versions, &group.Versions); err != nil {
			return nil, err
		}
		groups[group.Name] = group
	}
	return groups, nil
}

func (d *DuckDBStore) AddGroupData(groupName string, groupData cistore.GroupData) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	data, _ := json.Marshal(groupData.Data)         // Ignoring error because Data is always serializable
	versions, _ := json.Marshal(groupData.Versions) // Ignoring error because Versions is always serializable
	_, err := d.db.Exec("INSERT INTO groups (name, description, data, file, versions) VALUES (?, ?, ?, ?, ?)",
		groupName, groupData.Description, data, groupData.File.Content, versions)
	return err
}

func (d *DuckDBStore) GetGroupData(groupName string) (cistore.GroupData, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var group cistore.GroupData
	var data, file, versions []byte
	err := d.db.QueryRow("SELECT name, description, data, file, versions FROM groups WHERE name = ?", groupName).
		Scan(&group.Name, &group.Description, &data, &file, &versions)
	if err != nil {
		return group, err
	}
	err = json.Unmarshal(data, &group.Data)
	if err != nil {
		return group, err
	}
	group.File.Content = file
	err = json.Unmarshal(versions, &group.Versions)
	if err != nil {
		return group, err
	}
	return group, nil
}

func (d *DuckDBStore) UpdateGroupData(groupName string, groupData cistore.GroupData, create bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	data, _ := json.Marshal(groupData.Data)         // Ignoring error because Data is always serializable
	versions, _ := json.Marshal(groupData.Versions) // Ignoring error because Versions is always serializable
	if create {
		_, err := d.db.Exec("INSERT INTO groups (name, description, data, file, versions) VALUES (?, ?, ?, ?, ?)",
			groupName, groupData.Description, data, groupData.File.Content, versions)
		return err
	}
	_, err := d.db.Exec("UPDATE groups SET description = ?, data = ?, file = ?, versions = ? WHERE name = ?",
		groupData.Description, data, groupData.File.Content, versions, groupName)
	return err
}

func (d *DuckDBStore) RemoveGroupData(groupName string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec("DELETE FROM groups WHERE name = ?", groupName)
	return err
}

func (d *DuckDBStore) GetInstanceInfo(nodeName string) (cistore.OpenCHAMIInstanceInfo, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var instance cistore.OpenCHAMIInstanceInfo
	err := d.db.QueryRow("SELECT id, instance_id, local_hostname, hostname, cluster_name, region, availability_zone, cloud_provider, instance_type, cloud_init_base_url, public_keys FROM instances WHERE id = ?", nodeName).
		Scan(&instance.ID, &instance.InstanceID, &instance.LocalHostname, &instance.Hostname, &instance.ClusterName, &instance.Region, &instance.AvailabilityZone, &instance.CloudProvider, &instance.InstanceType, &instance.CloudInitBaseURL, &instance.PublicKeys)
	return instance, err
}

func (d *DuckDBStore) SetInstanceInfo(nodeName string, instanceInfo cistore.OpenCHAMIInstanceInfo) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	publicKeys, _ := json.Marshal(instanceInfo.PublicKeys) // Not checking error because PublicKeys is always serializable
	_, err := d.db.Exec("INSERT INTO instances (id, instance_id, local_hostname, hostname, cluster_name, region, availability_zone, cloud_provider, instance_type, cloud_init_base_url, public_keys) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET instance_id = ?, local_hostname = ?, hostname = ?, cluster_name = ?, region = ?, availability_zone = ?, cloud_provider = ?, instance_type = ?, cloud_init_base_url = ?, public_keys = ?",
		nodeName, instanceInfo.InstanceID, instanceInfo.LocalHostname, instanceInfo.Hostname, instanceInfo.ClusterName, instanceInfo.Region, instanceInfo.AvailabilityZone, instanceInfo.CloudProvider, instanceInfo.InstanceType, instanceInfo.CloudInitBaseURL, publicKeys,
		instanceInfo.InstanceID, instanceInfo.LocalHostname, instanceInfo.Hostname, instanceInfo.ClusterName, instanceInfo.Region, instanceInfo.AvailabilityZone, instanceInfo.CloudProvider, instanceInfo.InstanceType, instanceInfo.CloudInitBaseURL, publicKeys)
	return err
}

func (d *DuckDBStore) DeleteInstanceInfo(nodeName string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec("DELETE FROM instances WHERE id = ?", nodeName)
	return err
}

func (d *DuckDBStore) GetClusterDefaults() (cistore.ClusterDefaults, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var defaults cistore.ClusterDefaults
	err := d.db.QueryRow("SELECT cloud_provider, region, availability_zone, cluster_name, public_keys, base_url, boot_subnet, wg_subnet, short_name, nid_length FROM cluster_defaults").
		Scan(&defaults.CloudProvider, &defaults.Region, &defaults.AvailabilityZone, &defaults.ClusterName, &defaults.PublicKeys, &defaults.BaseUrl, &defaults.BootSubnet, &defaults.WGSubnet, &defaults.ShortName, &defaults.NidLength)
	return defaults, err
}

func (d *DuckDBStore) SetClusterDefaults(clusterDefaults cistore.ClusterDefaults) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	publicKeys, _ := json.Marshal(clusterDefaults.PublicKeys) // Not checking error on Marshall because PublicKeys is always serializable
	_, err := d.db.Exec("INSERT INTO cluster_defaults (cloud_provider, region, availability_zone, cluster_name, public_keys, base_url, boot_subnet, wg_subnet, short_name, nid_length) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT DO UPDATE SET cloud_provider = ?, region = ?, availability_zone = ?, cluster_name = ?, public_keys = ?, base_url = ?, boot_subnet = ?, wg_subnet = ?, short_name = ?, nid_length = ?",
		clusterDefaults.CloudProvider, clusterDefaults.Region, clusterDefaults.AvailabilityZone, clusterDefaults.ClusterName, publicKeys, clusterDefaults.BaseUrl, clusterDefaults.BootSubnet, clusterDefaults.WGSubnet, clusterDefaults.ShortName, clusterDefaults.NidLength,
		clusterDefaults.CloudProvider, clusterDefaults.Region, clusterDefaults.AvailabilityZone, clusterDefaults.ClusterName, publicKeys, clusterDefaults.BaseUrl, clusterDefaults.BootSubnet, clusterDefaults.WGSubnet, clusterDefaults.ShortName, clusterDefaults.NidLength)
	return err
}

func (d *DuckDBStore) SerializeToParquet(filePath string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	tables := []string{"groups", "instances", "cluster_defaults"}
	for _, table := range tables {
		_, err := d.db.Exec(fmt.Sprintf("COPY (SELECT * FROM %s) TO '%s/%s.parquet' (FORMAT 'parquet')", table, filePath, table))
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DuckDBStore) LoadFromParquet(filePath string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	tables := []string{"groups", "instances", "cluster_defaults"}
	for _, table := range tables {
		_, err := d.db.Exec(fmt.Sprintf("COPY %s FROM '%s/%s.parquet' (FORMAT 'parquet')", table, filePath, table))
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DuckDBStore) ApplyMigrations(migrations []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, migration := range migrations {
		if _, err := d.db.Exec(migration); err != nil {
			return err
		}
	}
	return nil
}
