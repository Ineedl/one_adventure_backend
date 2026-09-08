#!/usr/bin/env bash

# Query promotion stock keys from Redis without blocking the server.
# Usage: script/query-promotion-redis.sh [pattern]
set -euo pipefail

host="${REDIS_HOST:-127.0.0.1}"
port="${REDIS_PORT:-6379}"
password="${REDIS_PASSWORD:-123456}"
pattern="${1:-promotion:*}"

usage() {
    echo "Usage: $0 [pattern]"
    echo "Environment: REDIS_HOST, REDIS_PORT, REDIS_PASSWORD"
    echo "Default pattern: promotion:*"
}

if [[ "$pattern" == "-h" || "$pattern" == "--help" ]]; then
    usage
    exit 0
fi

command -v redis-cli >/dev/null 2>&1 || {
    echo "error: redis-cli is required" >&2
    exit 127
}

redis_args=(--no-auth-warning -h "$host" -p "$port")
if [[ -n "$password" ]]; then
    redis_args+=(-a "$password")
fi

echo "Redis: ${host}:${port}"
echo "Pattern: ${pattern}"
echo

count=0
while IFS= read -r key; do
    [[ -n "$key" ]] || continue
    count=$((count + 1))
    echo "==> $key"
    echo "TTL: $(redis-cli "${redis_args[@]}" TTL "$key")s"
    echo "Value:"
    redis-cli "${redis_args[@]}" --raw HGETALL "$key" | sed 'N;s/\n/: /' || true
    echo
done < <(redis-cli "${redis_args[@]}" --scan --pattern "$pattern")

echo "Found $count key(s)."
