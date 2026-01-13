#!/bin/bash

# Simple build script for QR File Transfer tool
# Builds for current platform only

set -e

echo "Building QR File Transfer tools for current platform..."

# Clean previous builds
rm -rf bin/
mkdir -p bin/

# Build for current platform
echo "Building sender..."
go build -o bin/qrtransfer-sender ./cmd/sender

echo "Building receiver..."
go build -o bin/qrtransfer-receiver ./cmd/receiver

echo "Build complete! Executables are in the 'bin/' directory."
echo ""
echo "Usage:"
echo "  ./bin/qrtransfer-sender      # Run sender"
echo "  ./bin/qrtransfer-receiver    # Run receiver"
echo ""
echo "Example workflow:"
echo "1. On remote machine: ./bin/qrtransfer-sender"
echo "2. Select file and start transfer"
echo "3. On local machine: ./bin/qrtransfer-receiver" 
echo "4. Start capture and save file"