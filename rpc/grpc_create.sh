#!/bin/bash

# gateway
protoc \
    -I . \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    proto/gateway/gateway.proto

# ping.proto and computing.proto
protoc \
    -I . \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    proto/ping/ping.proto \
    proto/computing/computing.proto
