#!/bin/bash

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