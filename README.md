# OpenCHAMI Cloud-Init Server

## Summary of Repo
The **OpenCHAMI cloud-init service** retrieves detailed inventory information from SMD and uses it to create cloud-init payloads customized for each node in an OpenCHAMI cluster.

## Table of Contents
1. [About / Introduction](#about--introduction)
2. [Build / Install](#build--install)
   - [Environment Variables](#environment-variables)
   - [Building Locally with GoReleaser](#building-locally-with-goreleaser)
3. [Running the Service](#running-the-service)
   - [Cluster Name](#cluster-name)
   - [Fake SMD Mode](#fake-smd-mode)
   - [Impersonation](#impersonation)
   - [Nocloud-net Datasource](#nocloud-net-datasource)
4. [Testing the Service](#testing-the-service)
   - [Basic Endpoint Testing](#basic-endpoint-testing)
   - [Meta-data](#meta-data)
   - [User-data](#user-data)
   - [Vendor-data](#vendor-data)
5. [Group Handling and Overrides](#group-handling-and-overrides)
   - [Updating Group Data with a Simple Jinja Example](#updating-group-data-with-a-simple-jinja-example)
   - [Complex Base64 Example](#complex-base64-example)
   - [Cluster Defaults and Instance Overrides](#cluster-defaults-and-instance-overrides)
     - [Set Cluster Defaults](#set-cluster-defaults)
     - [Override Instance Data](#override-instance-data)
6. [More Reading](#more-reading)

---

## About / Introduction
The **OpenCHAMI Cloud-Init Service** is designed to generate cloud-init configuration for nodes in an OpenCHAMI cluster. The new design pushes the complexity of merging configurations into the cloud-init client rather than the server. This README provides instructions based on the [Demo.md](https://github.com/OpenCHAMI/cloud-init/blob/main/demo/Demo.md) file for running and testing the service.

This service provides configuration data to cloud-init clients via the standard nocloud-net datasource. The service merges configuration from several sources:
- **SMD data** (or simulated data in development mode)
- **User-supplied JSON** (for custom configurations)
- **Cluster defaults and group overrides**

Cloud-init on nodes retrieves data in a fixed order:
1. `/meta-data` – YAML document with system configuration.
2. `/user-data` - a document which can be any of the [user data formats](https://cloudinit.readthedocs.io/en/latest/explanation/format.html#cloud-config-data)
3. `/vendor-data` – Vendor-supplied configuration via include-file mechanisms.
4. `/network-config` – An optional document in one of two [network configuration formats](https://cloudinit.readthedocs.io/en/latest/reference/network-config.html#network-config).  This is only requested if configured to do so with a kernel parameter or through cloud-init configuration in the image. __NB__: __OpenCHAMI doesn't support delivering `network-config` via the cloud-init server today__


---

## Build / Install

This project uses **[GoReleaser](https://goreleaser.com/)** for building and releasing, embedding additional metadata such as commit info, build time, and version. Below is a brief overview for local builds.

### Environment Variables
To include detailed metadata in your builds, set the following:

- **GIT_STATE**: `clean` if your repo is clean, `dirty` if uncommitted changes exist  
- **BUILD_HOST**: Hostname of the build machine  
- **GO_VERSION**: Version of Go used (for consistent versioning info)  
- **BUILD_USER**: Username of the person/system performing the build  

```bash
export GIT_STATE=$(if git diff-index --quiet HEAD --; then echo 'clean'; else echo 'dirty'; fi)
export BUILD_HOST=$(hostname)
export GO_VERSION=$(go version | awk '{print $3}')
export BUILD_USER=$(whoami)
```

### Building Locally with GoReleaser
1. [Install GoReleaser](https://goreleaser.com/install/) following their documentation.  
2. Run in snapshot mode to build locally without releasing:

   ```bash
   goreleaser release --snapshot --clean --single-target
   ```
3. Check the `dist/` directory for compiled binaries, which will include the injected metadata.

> [!NOTE]
> If you encounter errors, ensure your GoReleaser version matches the one used in the [Release Action](.github/workflows/Release.yml).

---

## Running the Service

### Cluster Name

Each instance of cloud-init is linked to a single SMD and operates for a single cluster. Until the cluster name is automatically available via your inventory system, you must specify it on the command line using the `-cluster-name` flag.

_Example:_
```bash
-cluster-name venado
```

### Fake SMD Mode

For development purposes, you can run the cloud-init server without connecting to a real SMD instance. By setting the environment variable `CLOUD_INIT_SMD_SIMULATOR` to `true`, the service will generate a set of simulated nodes.

**Example command:**
```bash
CLOUD_INIT_SMD_SIMULATOR=true dist/cloud-init_darwin_arm64_v8.0/cloud-init-server -cluster-name venado -insecure -impersonation=true
```

### Impersonation

By default, the service determines what configuration to return based on the IP address of the requesting node. For testing, impersonation routes can be enabled with the `-impersonation=true` flag.

**Sample commands:**
```bash
curl http://localhost:27777/cloud-init/admin/impersonation/x3000c1b1n1/meta-data
```

### Nocloud-net Datasource

```bash
cloud-init=enabled ds=nocloud-net;s=http://192.0.0.1/cloud-init
```

---
## Testing the Service

The following testing steps (adapted from Demo.md) help you verify that the service is functioning correctly.

### Basic Endpoint Testing

#### Start the Service in Fake SMD Mode (if desired):

**Example:**

```bash
CLOUD_INIT_SMD_SIMULATOR=true dist/cloud-init_darwin_arm64_v8.0/cloud-init-server -cluster-name venado -insecure -impersonation=true
```

#### Query the Standard Endpoints:

#### Meta-data:

```bash
curl http://localhost:27777/cloud-init/meta-data
```

You should see a YAML document with instance information (e.g., instance-id, cluster-name, etc.).

#### User-data:

```bash
curl http://localhost:27777/cloud-init/user-data
```

For now, this returns a blank cloud-config document:

```yaml
#cloud-config
```

#### Vendor-data:

```bash
curl http://localhost:27777/cloud-init/vendor-data
```

Vendor-data typically includes include-file directives pointing to group-specific YAML files:

```yaml
#include
http://192.168.13.3:8080/all.yaml
http://192.168.13.3:8080/login.yaml
http://192.168.13.3:8080/compute.yaml
```

## Group Handling and Overrides

The service supports advanced configuration through group handling and instance overrides.

### Updating Group Data with a Simple Jinja Example

This example sets a syslog aggregator via jinja templating. The group data is stored under the group name and then used in the vendor-data file.

```bash
curl -X POST http://localhost:27777/cloud-init/admin/groups/ \
    -H "Content-Type: application/json" \
    -d '{
        "name": "x3001",
        "description": "Cabinet x3001",
        "data": {
            "syslog_aggregator": "192.168.0.1"
        },
        "file": {
            "content": "#template: jinja\n#cloud-config\nrsyslog:\n  remotes: {x3001: {{ vendor_data.groups[\"x3001\"].syslog_aggregator }}}\n  service_reload_command: auto\n",
            "encoding": "plain"
        }
    }'
```

### Complex Base64 Example

To add more sophisticated vendor-data (for example, installing the slurm client), you can encode a complete cloud-config in base64. (See the script in Demo.md for a complete example.)

### Cluster Defaults and Instance Overrides

#### Set Cluster Defaults:

```bash
curl -X POST http://localhost:27777/cloud-init/admin/cluster-defaults/ \
    -H "Content-Type: application/json" \
    -d '{
        "cloud-provider": "openchami",
        "region": "us-west-2",
        "availability-zone": "us-west-2a",
        "cluster-name": "venado",
        "public-keys": [
            "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEArV2...",
            "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEArV3..."
        ]
    }'
```

#### Override Instance Data:

```bash
curl -X PUT http://localhost:27777/cloud-init/admin/instance-info/x3000c1b1n1 \
    -H "Content-Type: application/json" \
    -d '{
        "local-hostname": "compute-1",
        "instance-type": "t2.micro"
    }'
```
---

## More Reading

- [Official cloud-init documentation](https://cloud-init.io/)  
- [OpenCHAMI TPM-manager service](https://github.com/OpenCHAMI/TPM-manager)  
- [GoReleaser Documentation](https://goreleaser.com/)  
- [SMD Documentation](https://github.com/OpenCHAMI/smd)  
