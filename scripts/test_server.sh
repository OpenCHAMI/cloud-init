#!/bin/bash

# Exit on any error
set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to print test results
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ $2${NC}"
    else
        echo -e "${RED}✗ $2${NC}"
        exit 1
    fi
}

# Start the server in the background
echo "Starting netboot server..."
./netboot-server serve --cluster-name test-cluster --fake-smd --storage-backend mem &
SERVER_PID=$!

# Wait for server to start
sleep 2

# Cleanup function to kill the server when the script exits
cleanup() {
    echo "Stopping server..."
    kill $SERVER_PID
}
trap cleanup EXIT

# Test creating boot parameters
echo -e "\nTesting boot parameters API..."

# Create boot parameters
RESPONSE=$(http POST http://localhost:8080/boot/v1/bootparams \
    kernel=/boot/vmlinuz \
    initrd=/boot/initrd.img \
    params="console=ttyS0 root=/dev/sda1" \
    --json \
    --check-status \
    --body)

# Extract the ID from the Location header
ID=$(echo $RESPONSE | jq -r '.id')
if [ -z "$ID" ]; then
    print_result 1 "Failed to create boot parameters"
else
    print_result 0 "Created boot parameters with ID: $ID"
fi

# Get boot parameters
RESPONSE=$(http GET http://localhost:8080/boot/v1/bootparams/$ID --check-status --body)
KERNEL=$(echo $RESPONSE | jq -r '.kernel')
INITRD=$(echo $RESPONSE | jq -r '.initrd')
PARAMS=$(echo $RESPONSE | jq -r '.params')

if [ "$KERNEL" = "/boot/vmlinuz" ] && [ "$INITRD" = "/boot/initrd.img" ] && [ "$PARAMS" = "console=ttyS0 root=/dev/sda1" ]; then
    print_result 0 "Retrieved boot parameters match created parameters"
else
    print_result 1 "Retrieved boot parameters do not match created parameters"
fi

# Update boot parameters
RESPONSE=$(http PUT http://localhost:8080/boot/v1/bootparams/$ID \
    kernel=/boot/vmlinuz-new \
    initrd=/boot/initrd.img \
    params="console=ttyS0 root=/dev/sda1" \
    --json \
    --check-status \
    --body)

# Verify the update
RESPONSE=$(http GET http://localhost:8080/boot/v1/bootparams/$ID --check-status --body)
KERNEL=$(echo $RESPONSE | jq -r '.kernel')

if [ "$KERNEL" = "/boot/vmlinuz-new" ]; then
    print_result 0 "Updated boot parameters successfully"
else
    print_result 1 "Failed to update boot parameters"
fi

# Test boot script generation
echo -e "\nTesting boot script generation..."

RESPONSE=$(http GET "http://localhost:8080/boot/v1/bootscript?node=$ID" --check-status --body)
if echo "$RESPONSE" | jq -e . >/dev/null 2>&1; then
    print_result 0 "Generated boot script successfully"
else
    print_result 1 "Failed to generate boot script"
fi

echo -e "\nAll tests completed successfully!" 