#!/usr/bin/env bash

# Query the Envoy Admin API without changing runtime state.
# Usage: script/query-envoy.sh [summary|clusters|endpoints|listeners|config|stats]
set -euo pipefail

admin_url="${ENVOY_ADMIN_URL:-http://127.0.0.1:9901}"
command_name="${1:-summary}"

usage() {
    echo "Usage: $0 [summary|clusters|endpoints|listeners|config|stats]"
    echo "Environment: ENVOY_ADMIN_URL (default: http://127.0.0.1:9901)"
}

command -v curl >/dev/null 2>&1 || {
    echo "error: curl is required" >&2
    exit 127
}

if [[ "$command_name" == "-h" || "$command_name" == "--help" || "$command_name" == "help" ]]; then
    usage
    exit 0
fi

request() {
    curl --fail --silent --show-error "${admin_url}$1"
}

pretty_json() {
    if command -v jq >/dev/null 2>&1; then
        jq .
    else
        cat
    fi
}

if ! request "/ready" >/dev/null; then
    echo "error: Envoy Admin API is unavailable at ${admin_url}" >&2
    echo "hint: start Envoy or set ENVOY_ADMIN_URL" >&2
    exit 1
fi

case "$command_name" in
    summary)
        echo "== Envoy status =="
        request "/ready"
        echo
        echo "== Dynamic clusters and endpoints =="
        request "/clusters?format=json" | pretty_json
        ;;
    clusters)
        request "/clusters?format=json" | pretty_json
        ;;
    endpoints)
        request "/config_dump?resource=dynamic_endpoint_configs" | pretty_json
        ;;
    listeners)
        request "/listeners?format=json" | pretty_json
        ;;
    config)
        request "/config_dump" | pretty_json
        ;;
    stats)
        request "/stats?format=json" | pretty_json
        ;;
    *)
        echo "error: unknown command '$command_name'" >&2
        usage >&2
        exit 2
        ;;
esac
