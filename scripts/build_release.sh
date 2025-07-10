#!/bin/bash

# Script to prepare the SDK for release

set -e  # Exit on error

echo "Preparing SDK for release..."

# Step 1-3: Copy required protobufs using the copy_protos.sh script
./scripts/copy_protos.sh

# Step 4: Run go mod tidy
echo "Running go mod tidy..."
go mod tidy

# Step 5: Vendor dependencies
echo "Vendoring dependencies..."
go mod vendor

# Step 6: Verify the build
echo "Verifying build..."
go build -mod=vendor ./...

# Step 7: Run tests with vendored dependencies
echo "Running tests with vendored dependencies..."
go test -mod=vendor ./...

echo ""
echo "✅ Release preparation complete!"
echo ""
echo "The SDK is now ready to be published with:"
echo "  - Internal protobuf files (no external dependency)"
echo "  - Vendored dependencies"
echo ""
echo "Users can now 'go get github.com/clavataai/gosdk' without needing access to your monorepo." 