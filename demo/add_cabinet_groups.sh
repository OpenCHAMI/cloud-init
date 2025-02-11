#!/bin/bash

curl -X POST http://localhost:27777/cloud-init/admin/groups/ \
     -H "Content-Type: application/json" \
     -d '{
          "name": "x3000",
          "description": "Cabinet x3000",
          "meta-data": {
            "syslog_aggregator": "192.168.0.1"
          },
          "file": {
            "content": "#cloud-config\nrsyslog:\n  remotes: {x3000: \"192.168.0.5\"}\nservice_reload_command: auto\n",
            "encoding": "plain"
          }
        }'

curl -X POST http://localhost:27777/cloud-init/admin/groups/ \
    -H "Content-Type: application/json" \
    -d '{
        "name": "x3001",
        "description": "Cabinet x3001",
        "meta-data": {
          "syslog_aggregator": "192.168.0.1"
          },
        "file": {
            "content": "## template: jinja\n#cloud-config\nrsyslog:\n  remotes: {x3001: {{ vendor_data.groups[\"x3002\"].syslog_aggregator }}}\n  service_reload_command: auto\n",
            "encoding": "plain"
        }
    }'



curl -X POST http://localhost:27777/cloud-init/admin/groups/ \
    -H "Content-Type: application/json" \
    -d '{
        "name": "x3002",
        "description": "Cabinet x3002",
        "file": {
            "content": "## template: jinja\n#cloud-config\nrsyslog:\n  remotes: {x3002: {{ vendor_data.groups[\"x3002\"].syslog_aggregator }}}\n  service_reload_command: auto\n",
            "encoding": "plain"
        }
    }'

curl -X POST http://localhost:27777/cloud-init/admin/groups/ \
     -H "Content-Type: application/json" \
     -d '{
          "name": "x3003",
          "description": "Cabinet x3003",
          "file": {
            "content": "#cloud-config\nrsyslog:\n  remotes: {x3003: \"192.168.0.5\"}\nservice_reload_command: auto\n",
            "encoding": "plain"
          }
        }'
    