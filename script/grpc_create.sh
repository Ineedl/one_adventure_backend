#!/bin/bash

# Keep the original relative protoc paths independent of the caller's cwd.
cd "$(dirname "$0")/../rpc" || exit 1

# computing.proto
protoc \
    -I . \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    proto/computing/computing.proto

# user.proto
protoc \
    -I . \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    proto/user/user.proto

# item.proto
protoc \
    -I . \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    proto/item/item.proto

# server_manager.proto
protoc \
    -I . \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    proto/server_manager/server_manager.proto
