#!/bin/bash

echo "Generating proto files..."

OUTPUT_DIR="internal/proto"

mkdir -p $OUTPUT_DIR

protoc --proto_path=proto \
  --go_out=$OUTPUT_DIR --go_opt=paths=source_relative \
  --go-grpc_out=$OUTPUT_DIR --go-grpc_opt=paths=source_relative \
  proto/manege.proto \
  proto/manege_v2.proto
