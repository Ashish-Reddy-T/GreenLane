#!/bin/bash
# Generate protobuf code for Go services

set -e

echo "📝 Generating protobuf code..."

# Generate for ingestion service
protoc --go_out=services/ingestion --go_opt=paths=source_relative \
    --go-grpc_out=services/ingestion --go-grpc_opt=paths=source_relative \
    proto/fleet.proto

# Generate for CLI
protoc --go_out=cli --go_opt=paths=source_relative \
    --go-grpc_out=cli --go-grpc_opt=paths=source_relative \
    proto/fleet.proto

echo "✅ Protobuf generation complete!"
