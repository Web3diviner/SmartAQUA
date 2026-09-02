#!/bin/bash
# Build script for Smart Fish Feeder SIL Simulation

echo "========================================="
echo "Building SIL Simulation..."
echo "========================================="

# Create build directory
mkdir -p build
cd build

# Run CMake
cmake ..

# Build
make

# Check if build succeeded
if [ $? -eq 0 ]; then
    echo ""
    echo "========================================="
    echo "Build successful!"
    echo "========================================="
    echo ""
    echo "Run simulation with: ./build/fishfeeder_sim"
    echo ""
else
    echo ""
    echo "========================================="
    echo "Build failed!"
    echo "========================================="
    exit 1
fi
