#!/bin/bash

curl -X POST http://localhost:27777/cloud-init/admin/cluster-defaults/ \
    -H "Content-Type: application/json" \
    -d '{
          "cloud_provider": "OpenCHAMI",
          "region": "lanl",
          "availability-zone": "lanl-yellow",
          "cluster-name": "venado",
          "base-url": "http://127.0.0.1:27777/cloud-init/",
	  "public-keys": [
	      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINsQK8VMZM/BtweT1X6rC+ropQLoFhZ/8wE/miFtWV4f"
	      ]
        }'
