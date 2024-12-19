#!/bin/bash

curl -X POST http://localhost:27777/cloud-init/admin/fake-sm/nodes \
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



