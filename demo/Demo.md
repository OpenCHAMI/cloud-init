# Demo of new cloud-init behavior

Updates to the cloud-init service are designed to push the complexity of merging configurations into the cloud-init client rather than the cloud-init server.  The codepaths for reducing surprises in merge behavior are much better tested in the client which is open source and deployed on countless instances around the world.  Staying compliant with the [nocloud-net datasource](https://cloudinit.readthedocs.io/en/latest/reference/datasources/nocloud.html) client requires a short review of how it handles the order of requests and the order of processing.

## Running the service

There are a few additional flags and environment variables to be aware of with cloud-init.

### Cluster Name

Each instance of cloud-init is linked to a single SMD and operates for a single cluster.  Ideally, the cluster name should be available through the inventory system.  Until that is true, it will need to be specified on the commandline with the `-cluster-name` flag.

### Fake SMD
```
CLOUD_INIT_SMD_SIMULATOR=true dist/cloud-init_darwin_arm64_v8.0/cloud-init-server -cluster-name venado -insecure -impersonation=true
```

For development purposes, you can run the cloud-init server without an SMD instance to connect to by setting `CLOUD_INIT_SMD_SIMULATOR` to `true` in your environment.  This will create a set of nodes spread across a few cabinets and add them to various groups for testing.  This also adds a set of endpoints under `cloud-init/admin/fake-sm/` which allow additional testing nodes to be created as needed.

### Impersonation

Since the HTTP handlers within cloud-init use the ip address of the requesting node to determine what to send, we have a set of impersonation routes that are not enabled by default.  They sit under the `admin` subrouter and can be enabled at runtime with `-impersonation=true` on the commandline.

* `curl http://localhost:27777/cloud-init/admin/impersonation/x3000c1b1n1/meta-data`
* `curl http://localhost:27777/cloud-init/admin/impersonation/x3000c1b1n1/user-data`
* `curl http://localhost:27777/cloud-init/admin/impersonation/x3000c1b1n1/vendor-data`
* `curl http://localhost:27777/cloud-init/admin/impersonation/x3000c1b1n1/compute.yaml`

## Nocloud-net Datasource

The nocloud-net datasource is configured in the kernel commandline by adding a few parameters. `cloud-init=enabled ds=nocloud-net;s=http://192.0.0.1/cloud-init`.  This instructs the cloud-init client to query for configuration data after the network is established.  The order of the requests is fixed and if any of the configurations fail to load, further requests are not attempted.

1. `/meta-data` - a yaml document with configuration parameters.
1. `/user-data` - a document which can be any of the [user data formats](https://cloudinit.readthedocs.io/en/latest/explanation/format.html#cloud-config-data)
1. `/vendor-data` - another document which can be any of the same formats
1. `/network-config` - An optional document in one of two [network configuration formats](https://cloudinit.readthedocs.io/en/latest/reference/network-config.html#network-config).  This is only requested if configured to do so with a kernel parameter or through cloud-init configuration in the image. __NB__: __OpenCHAMI doesn't support delivering `network-config` via the cloud-init server today__

### User-data and Vendor-data

Both `user-data` and `vendor-data` are based on the same format.  The philosophy of cloud-init is that anything supplied by the user should override anything supplied by the vendor.  If there is a conflict on the server, user-data will always win.

In OpenCHAMI, we send a blank `user-data` today, preserving that for future integration with users.

```yaml
#cloud-config
```

For `vendor-data` we don't merge anything on the server side.  Instead we take advantage of the [include-file user-data format](https://cloudinit.readthedocs.io/en/latest/explanation/format.html#include-file) which allows us to create a separate yaml file on demand for each group a node is a part of.  Since the include-file supports jinja templating, our server can send the same response to all `/vendor-data` requests.

```yaml
#template: jinja
#include
{% for group_name in vendor_data.groups.keys() %}
https://{{ vendor_data.cloud_init_base_url }}/{{ group_name }}.yaml
{% endfor %}
```

After client-side templates are applied using data from the `meta-data`, a node that is part of three groups: `all`, `compute`, and `login` will effectively process the following file.

```yaml
#include
http://192.168.13.3:8080/all.yaml
http://192.168.13.3:8080/login.yaml
http://192.168.13.3:8080/compute.yaml
```


## Updated Group Handling

Nodes are assigned to groups in smd, but we store the user-data and any parameters in cloud-init.  The user-data scripts themselves can be submitted as base64 encoded files or as raw strings. Since cloud-init doesn't validate or interfere with the user-data files in any way, the full set of formats are available, including jinja2 templating.

### Simple Example with jinja

This sets the syslog aggregator based on the contents of `meta-data`.  The `data` element in the json is added to the meta-data payload under the group name so `vendor-data.groups.x3001.syslog_aggregator` is valid within jinja2 templating

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

### Complex base64 example

This example adds the OpenHPC repo an installs the slurm client.  Normally, this kind of thing should be handled in the image to improve boot speed.  Notice that the yaml is properly formatted and easy to read before being passed to `base64 -w 0`.

```bash
#!/bin/bash

# Define the cloud-config content
CLOUD_CONFIG_CONTENT=$(cat <<EOF
#cloud-config
package_update: true
package_upgrade: true
yum_repos:
  OpenHPC:
    baseurl: https://repos.openhpc.community/OpenHPC/3/EL_9/
    enabled: true
    gpgcheck: false
    name: OpenHPC
packages:
  - ohpc-slurm-client
EOF
)

# Define the JSON payload
JSON_PAYLOAD=$(cat <<EOF
{
  "name": "compute",
  "description": "Compute nodes",
  "file": {
    "content": "$(echo "$CLOUD_CONFIG_CONTENT" | base64 -w 0)",
    "encoding": "base64"
  }
}
EOF
)

# Make the POST request
curl -X POST http://localhost:27777/cloud-init/admin/groups/ \
     -H "Content-Type: application/json" \
     -d "$JSON_PAYLOAD"
```

## Impersonation Routes




