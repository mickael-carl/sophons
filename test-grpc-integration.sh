#!/bin/bash

# Test script for gRPC executer integration
# This script tests both in-process and gRPC execution modes

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Cleanup function
cleanup() {
    if [ ! -z "$GRPC_PID" ]; then
        echo "Stopping gRPC executer (PID: $GRPC_PID)..."
        kill $GRPC_PID 2>/dev/null || true
        wait $GRPC_PID 2>/dev/null || true
    fi
    rm -rf /tmp/test-playbook-dir
}

trap cleanup EXIT

echo "=== Testing gRPC Executer Integration ==="
echo ""

# Create test directory structure
mkdir -p /tmp/test-playbook-dir/roles
cat > /tmp/test-playbook-dir/playbook.yaml <<'EOF'
- hosts: all
  tasks:
    - name: Create a test file
      ansible.builtin.command:
        cmd: "touch /tmp/grpc-test-file"
        creates: "/tmp/grpc-test-file"

    - name: Echo test message
      ansible.builtin.command:
        cmd: "echo 'gRPC test successful'"
EOF

echo "1. Testing in-process execution (default mode)..."
rm -f /tmp/grpc-test-file
./bin/executer-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/x86_64/' | sed 's/aarch64/arm64/') \
    /tmp/test-playbook-dir/playbook.yaml

if [ -f /tmp/grpc-test-file ]; then
    echo -e "${GREEN}✓ In-process execution successful${NC}"
else
    echo -e "${RED}✗ In-process execution failed${NC}"
    exit 1
fi

echo ""
echo "2. Starting gRPC executer on port 50051..."
./bin/grpc-executer-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/x86_64/' | sed 's/aarch64/arm64/') \
    -port 50051 > /tmp/grpc-executer.log 2>&1 &
GRPC_PID=$!

echo "   gRPC executer started (PID: $GRPC_PID)"
sleep 2  # Give the server time to start

echo ""
echo "3. Testing gRPC execution (--grpc-executer flag)..."
rm -f /tmp/grpc-test-file
./bin/executer-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/x86_64/' | sed 's/aarch64/arm64/') \
    --grpc-executer localhost:50051 \
    /tmp/test-playbook-dir/playbook.yaml

if [ -f /tmp/grpc-test-file ]; then
    echo -e "${GREEN}✓ gRPC execution successful${NC}"
else
    echo -e "${RED}✗ gRPC execution failed${NC}"
    echo "gRPC executer log:"
    cat /tmp/grpc-executer.log
    exit 1
fi

echo ""
echo -e "${GREEN}=== All tests passed! ===${NC}"
echo ""
echo "Usage examples:"
echo "  # In-process execution (default):"
echo "  ./bin/executer-darwin-arm64 playbook.yaml"
echo ""
echo "  # gRPC execution:"
echo "  ./bin/grpc-executer-darwin-arm64 -port 50051 &"
echo "  ./bin/executer-darwin-arm64 --grpc-executer localhost:50051 playbook.yaml"
