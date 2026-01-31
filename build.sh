#!/bin/bash

# Build script for Word Frequency Counter
# This script compiles the Go program for multiple platforms

echo "Building Word Frequency Counter..."
echo ""

# Build for current platform
echo "Building for current platform..."
go build -o wordcount ./cmd/wordcount
if [ $? -eq 0 ]; then
    echo "✓ Build successful: ./wordcount"
else
    echo "✗ Build failed"
    exit 1
fi

echo ""
echo "Build complete! Run the program with:"
echo "  ./wordcount -file testdata/sample.txt -segments 4"
echo ""
echo "For verbose output showing intermediate results:"
echo "  ./wordcount -file testdata/sample.txt -segments 4 -verbose"
