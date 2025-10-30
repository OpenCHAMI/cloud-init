#!/bin/sh
set -e -o pipefail

# As configured in systemd, we expect to inherit the "ochami_ci_url" cmdline
# parameter as an env var. Exit if this is not the case.
if [ -z "${ochami_wg_ip}" ];
then
    echo "ERROR: Failed to find the 'ochami_wg_url' environment variable."
    echo "It should be specified on the kernel cmdline, and will be inherited from there."
    if [ -f "/etc/cloud/cloud.cfg.d/ochami.cfg" ];
    then
        echo "Removing ochami-specific cloud-config; cloud-init will use other defaults"
        rm /etc/cloud/cloud.cfg.d/ochami.cfg
    else
        echo "Not writing ochami-specific cloud-config; cloud-init will use other defaults"
    fi
    exit 0
fi
echo "Found OpenCHAMI cloud-init URL '${ochami_wg_ip}'"
echo "!!!!Starting pre cloud-init config!!!!"

echo "Using userspace WireGuard via wireguard-go"
if ! command -v wireguard-go >/dev/null 2>&1; then
    echo "ERROR: wireguard-go is not installed or not in PATH."
    echo "Please install wireguard-go or ensure it is available, then retry."
    exit 1
fi

echo "Generating Wireguard keys"
wg genkey | tee /etc/wireguard/private.key | wg pubkey > /etc/wireguard/public.key

echo "Making Request to configure wireguard tunnel"
PUBLIC_KEY=$(cat /etc/wireguard/public.key)
PAYLOAD="{ \"public_key\": \"${PUBLIC_KEY}\" }"
WG_PAYLOAD=$(curl -s -X POST -d "${PAYLOAD}" http://${ochami_wg_ip}:27777/cloud-init/wg-init)

echo $WG_PAYLOAD | jq

CLIENT_IP=$(echo $WG_PAYLOAD | jq -r '."client-vpn-ip"')
SERVER_IP=$(echo $WG_PAYLOAD | jq -r '."server-ip"' | awk -F'/' '{print $1}')
SERVER_PORT=$(echo $WG_PAYLOAD | jq -r '."server-port"')
SERVER_KEY=$(echo $WG_PAYLOAD | jq -r '."server-public-key"')

echo "Setting up local userspace wireguard interface"
echo "Starting wireguard-go for wg0"
# Start wireguard-go in background and wait for readiness
wireguard-go wg0 >/var/log/wireguard-go.log 2>&1 &
WGGO_PID=$!
READY=0
for i in $(seq 1 30); do
    if wg show wg0 public-key >/dev/null 2>&1; then
        READY=1
        break
    fi
    sleep 0.1
done
if [ "$READY" -ne 1 ]; then
    echo "WARNING: wg0 interface did not report ready in time; continuing anyway"
fi
echo "Adding ip address ${CLIENT_IP}/32"
ip address add dev wg0 ${CLIENT_IP}/32
echo "Setting the private key"
wg set wg0 private-key /etc/wireguard/private.key
echo "Bringing up the wg0 link"
ip link set wg0 up
echo "Setting up the peer with the server"
wg set wg0 peer ${SERVER_KEY} allowed-ips ${SERVER_IP}/32 endpoint ${ochami_wg_ip}:$SERVER_PORT
rm /etc/wireguard/private.key
rm /etc/wireguard/public.key