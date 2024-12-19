#!/bin/bash

curl -X PUT http://localhost:27777/cloud-init/admin/instance-info/x3000c1b1n1 \
    -H "Content-Type: application/json" \
    -d '{
        "local-hostname": "compute-1",
        "instance-type": "t2.micro"
    }'