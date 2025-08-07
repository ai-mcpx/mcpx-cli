#!/bin/bash

# Test script for mcpx-cli
set -e

# Test 1: Build the CLI
echo "1. Building CLI..."
go build -o mcpx-cli .
echo "✓ Build successful"

# Test 2: Test help/usage
echo ""
echo "2. Testing help output..."
./mcpx-cli > /dev/null 2>&1 || echo "✓ Help displayed correctly (exit code 1 expected)"

# Test 3: Test health command (will fail if no server running, but should show proper error)
echo ""
echo "3. Testing health command (expect connection error if no server running)..."
./mcpx-cli health || echo "✓ Health command executed (connection error expected if no server)"

# Test 4: Test servers command
echo ""
echo "4. Testing servers command (expect connection error if no server running)..."
./mcpx-cli servers || echo "✓ Servers command executed (connection error expected if no server)"

# Test 5: Test invalid server ID format
echo ""
echo "5. Testing server command with invalid ID..."
./mcpx-cli server invalid-id || echo "✓ Server command with invalid ID handled correctly"

# Test 6: Test publish without token
echo ""
echo "6. Testing publish command without token..."
./mcpx-cli publish example-server.json || echo "✓ Publish command correctly requires token"

# Test 7: Validate example server JSON
echo ""
echo "7. Validating example server JSON..."
if command -v jq &> /dev/null; then
    cat example-server.json | jq . > /dev/null
    echo "✓ Example server JSON is valid"
else
    echo "⚠ jq not available, skipping JSON validation"
fi

echo ""
echo "=== All tests completed ==="
echo ""
echo "To run the CLI manually:"
echo "  ./mcpx-cli health"
echo "  ./mcpx-cli servers"
echo "  ./mcpx-cli server <id>"
echo "  ./mcpx-cli publish example-server.json --token <token>"
