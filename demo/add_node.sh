#!/bin/bash

# SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
#
# SPDX-License-Identifier: MIT

set -euo pipefail

: "${CLOUD_INIT_ADMIN_TOKEN:?Set CLOUD_INIT_ADMIN_TOKEN to an administrative JWT}"
CLOUD_INIT_URL="${CLOUD_INIT_URL:-http://localhost:27777/cloud-init}"

curl --fail-with-body -X POST "${CLOUD_INIT_URL%/}/admin/fake-sm/nodes" \
    -H "Authorization: Bearer ${CLOUD_INIT_ADMIN_TOKEN}" \
     -H "Content-Type: application/json" \
     -d '{
           "ID": "x4000c1b1n1",
           "Role": "Compute",
           "Type": "Node",
           "NID": 501,
           "MAC": "52:54:00:7f:d5:e4",
           "Groups": ["x4000","all","compute"],
           "IP": "192.168.100.21"
         }'
