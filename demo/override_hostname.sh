#!/bin/bash

# SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
#
# SPDX-License-Identifier: MIT

set -euo pipefail

: "${CLOUD_INIT_ADMIN_TOKEN:?Set CLOUD_INIT_ADMIN_TOKEN to an administrative JWT}"
CLOUD_INIT_URL="${CLOUD_INIT_URL:-http://localhost:27777/cloud-init}"

curl --fail-with-body -X PUT "${CLOUD_INIT_URL%/}/admin/instance-info/x3000c1b1n1" \
    -H "Authorization: Bearer ${CLOUD_INIT_ADMIN_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{
        "local-hostname": "compute-1",
        "instance-type": "t2.micro"
    }'
