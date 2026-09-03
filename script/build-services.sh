#!/usr/bin/env bash

# Build Linux/amd64 binaries consumed by each service's Dockerfile.
# Usage: script/build-services.sh [all|service ...]
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
project_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)

services=(gateway gate_server computing user item commerce server_manager order pay promotion xds)

usage() {
    echo "Usage: $0 [all|service ...]"
    echo "Services: ${services[*]}"
}

args=()
for arg in "$@"; do
    case "$arg" in
        -h|--help) usage; exit 0 ;;
        *) args+=("$arg") ;;
    esac
done

if ((${#args[@]} == 0)); then
    requested=("${services[@]}")
elif [[ "${args[0]}" == "all" ]]; then
    if ((${#args[@]} != 1)); then
        echo "error: 'all' cannot be combined with service names" >&2
        exit 2
    fi
    requested=("${services[@]}")
else
    requested=("${args[@]}")
fi

command -v gf >/dev/null 2>&1 || {
    echo "error: gf CLI is required (install GoFrame CLI first)" >&2
    exit 127
}

for service in "${requested[@]}"; do
    valid=false
    for known in "${services[@]}"; do
        [[ "$service" == "$known" ]] && valid=true && break
    done
    if [[ "$valid" != true ]]; then
        echo "error: unknown service '$service'" >&2
        usage >&2
        exit 2
    fi

    service_dir="$project_dir/$service"
    output="$service_dir/temp/linux_amd64/main"
    echo "==> Building $service (linux/amd64)"
    (
        cd "$service_dir"
        if [[ "$service" == xds ]]; then
            GOOS=linux GOARCH=amd64 go build -o temp/linux_amd64/main ./cmd/xds
        else
            gf build -a amd64 -s linux -p temp
        fi
    )
    [[ -x "$output" ]] || chmod +x "$output"
    [[ -f "$output" ]] || {
        echo "error: build output not found: $output" >&2
        exit 1
    }
    echo "    output: $output"
done

echo "All requested services built successfully."
