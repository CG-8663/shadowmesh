#!/bin/bash
# Quick test suite for ShadowMesh
set -e

echo "Running quick tests..."
go test ./shared/crypto -short
go test ./shared/protocol -short  
go test ./test/integration -short
echo "All tests passed!"
