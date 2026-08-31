#!/bin/bash

# SPDX-FileCopyrightText: 2026 Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
#
# SPDX-License-Identifier: MIT

curl -X PUT http://localhost:27777/cloud-init/admin/instance-info/x3000c1b1n1 \
    -H "Content-Type: application/json" \
    -d '{
        "local-hostname": "compute-1",
        "instance-type": "t2.micro"
    }'
