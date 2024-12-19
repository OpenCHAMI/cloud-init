#!/bin/bash

# Define the cloud-config content
COMPUTE_CLOUD_CONFIG_CONTENT=$(cat <<EOF
#cloud-config
merge_how:
 - name: list
   settings: [append]
 - name: dict
   settings: [no_replace, recurse_list]
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
COMPUTE_JSON_PAYLOAD=$(cat <<EOF
{
  "name": "compute",
  "description": "Compute nodes",
  "file": {
    "content": "$(echo "$COMPUTE_CLOUD_CONFIG_CONTENT" | base64 -w 0)",
    "encoding": "base64"
  }
}
EOF
)

# Define the all content
ALL_CLOUD_CONFIG_CONTENT=$(cat <<EOF
## template: jinja
#cloud-config
merge_how:
 - name: list
   settings: [append]
 - name: dict
   settings: [no_replace, recurse_list]

disable_root: false
ssh_pwauth: true

users:
  - name: root
    lock_passwd: false
    passwd: "$6$IIUMs.4puvwJ7Yv1$9H81v9GzvnMmzhdLGFPt0RYpph4oX/cXTsvadE4wXc57IKuLHlN5LhoEf7vkvGAHP7JKKFOJUpmgvLUGMCeKz/"
    ssh_authorized_keys:
      - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDhVqewcj1NwUW7TG9DRY0YnR7BNTovQ2IqiPGgwimdq5s5EU+X2K6JjUpirHayxX4S5KOfp7eFgV3fMo/Wdb+Kq7LJjDEWltLM0xtM8JeCnjVca0SKMBraYgqjtmOSipGQ+j7IiwJ+YGIeQA7/rs0o2V5RJThJKYeXgDkHrmwoGumIiGdVeoKAyw2CJS7V6/ISZX8JAsXrIBSfLTcEvQOaiFveOIRAsEhciB1bD4jOFv14iMwiQac36/EncVlxInkFbWznH3G0+uVapy/fKvbTAOZixLqUjEW5EQqmuOTowB6J/Lro8VV29s5r3wvfPp0dGltIGP3TyJvbWiOetPXZWGnvKfnJcZkJAfoXfIhX1L0y9nROSfGjzaXagIjdXFwvgyXq1MM2+QMjqa5oZdJFC0sWEI55qVmIgJ7IVtU1rWX/wsGQLdQ9MDL+s+FsOei4JIhRm5aS5JoIBJVVmavd50RfesTYGCmD8qAw07t5UaDs6AexYT0gsJwHc0Xy/Ak=
    shell: /bin/bash

write_files:
  - path: /etc/ssh/sshd_config
    content: |
      PermitRootLogin yes
      PasswordAuthentication yes
      PubkeyAuthentication yes
      AuthorizedKeysFile .ssh/authorized_keys

runcmd:
  - systemctl reload sshd

packages:
  - dmidecode
  - tpm2-tools
  - tpm2-abrmd
  - tpm2-tss

phone_home:
  post: [pub_key_rsa, pub_key_ecdsa, pub_key_ed25519, instance_id, hostname, fqdn]
  tries: 5
  url: http://192.168.13.17:27777/cloud-init/phone-home/{{ v1.instance_id }}/
EOF
)

# Define the JSON payload
ALL_JSON_PAYLOAD=$(cat <<EOF
{
  "name": "all",
  "description": "all nodes",
  "file": {
    "content": "$(echo "$ALL_CLOUD_CONFIG_CONTENT" | base64 -w 0)",
    "encoding": "base64"
  }
}
EOF
)

# Make the POST requests
curl -X PUT http://localhost:27777/cloud-init/admin/groups/compute \
     -H "Content-Type: application/json" \
     -d "$COMPUTE_JSON_PAYLOAD"
curl -X PUT http://localhost:27777/cloud-init/admin/groups/all \
     -H "Content-Type: application/json" \
     -d "$ALL_JSON_PAYLOAD"

curl -X PUT http://localhost:27777/cloud-init/admin/fake-sm/nodes/x3003c3b5n0 \
     -H "Content-Type: application/json" \
     -d '  {
    "ID": "x3003c3b5n0",
    "Type": "Node",
    "NID": 500,
    "NetType": "Ethernet",
    "mac": "00:DE:AD:BE:F0:F4",
    "ip": "10.20.31.244",
    "groups": ["x3003", "compute", "all" ]
  }'

curl -X POST http://localhost:27777/cloud-init/admin/cluster-defaults/ \
    -H "Content-Type: application/json" \
    -d '{
          "cloud_provider": "OpenCHAMI",
          "region": "annandale",
          "availability-zone": "annandale-basement-1",
          "cluster-name": "bikeshack",
          "base-url": "http://192.168.13.17:27777/cloud-init/",
	  "public-keys": [
	      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINsQK8VMZM/BtweT1X6rC+ropQLoFhZ/8wE/miFtWV4f"
	      ]
        }'
