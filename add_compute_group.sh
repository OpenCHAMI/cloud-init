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
    "filename": "cloud-config.yaml",
    "encoding": "base64"
  }
}
EOF
)

# Make the POST request
curl -X POST http://localhost:27777/cloud-init/admin/groups/ \
     -H "Content-Type: application/json" \
     -d "$JSON_PAYLOAD"

