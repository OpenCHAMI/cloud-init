#!/bin/bash

# SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
#
# SPDX-License-Identifier: MIT

set -euo pipefail

: "${CLOUD_INIT_ADMIN_TOKEN:?Set CLOUD_INIT_ADMIN_TOKEN to an administrative JWT}"
CLOUD_INIT_URL="${CLOUD_INIT_URL:-http://localhost:27777/cloud-init}"

# Define the cloud-config content
CLOUD_CONFIG_CONTENT=$(cat <<'EOF'
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
    "content": "$(echo "$CLOUD_CONFIG_CONTENT" | base64 | tr -d '\n')",
    "encoding": "base64"
  }
}
EOF
)

# Make the POST request
curl --fail-with-body -X POST "${CLOUD_INIT_URL%/}/admin/groups/" \
    -H "Authorization: Bearer ${CLOUD_INIT_ADMIN_TOKEN}" \
     -H "Content-Type: application/json" \
     -d "$JSON_PAYLOAD"
