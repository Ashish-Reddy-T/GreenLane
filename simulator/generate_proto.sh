#!/bin/bash
# Generate Python protobuf code for the simulator

set -e

echo "📝 Generating Python protobuf code..."

python3 -m grpc_tools.protoc \
    -I../proto \
    --python_out=. \
    --grpc_python_out=. \
    ../proto/fleet.proto

echo "✅ Python protobuf generation complete!"
echo "Generated: fleet_pb2.py, fleet_pb2_grpc.py"
