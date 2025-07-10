
set -e  # Exit on error

echo "Copying lastest protobufs to SDK..."


# Step 1: Copy required protobufs
echo "Copying protobuf packages..."
rm -rf internal/protobufs
mkdir -p internal/protobufs
cp -r ../libs/protobufs/gateway internal/protobufs/
cp -r ../libs/protobufs/shared internal/protobufs/

# Step 2: Update imports in SDK files
echo "Updating imports in SDK files..."
for file in *.go; do
    if [ -f "$file" ]; then
        sed -i.bak 's|github.com/clavataai/monorail/libs/protobufs/|github.com/clavataai/gosdk/internal/protobufs/|g' "$file"
        rm -f "$file.bak"
    fi
done

# Step 3: Update imports in copied protobuf files
echo "Updating imports in protobuf files..."
find internal/protobufs -name "*.go" -type f | while read file; do
    sed -i.bak 's|github.com/clavataai/monorail/libs/protobufs/|github.com/clavataai/gosdk/internal/protobufs/|g' "$file"
    rm -f "$file.bak"
done