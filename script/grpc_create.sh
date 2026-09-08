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

# ws_gateway.proto
protoc \
    -I . \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    proto/ws_gateway/ws_gateway.proto

# order.proto and pay.proto
for proto in proto/order/order.proto proto/pay/pay.proto; do
protoc \
    -I . \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    "$proto"
done
