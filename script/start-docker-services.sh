#!/usr/bin/env bash

# Start or recreate services with Docker Compose.
# Usage: script/start-docker-services.sh [start|restart] [all|service ...]
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
project_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
compose_file="$project_dir/one_adventure/docker-compose.yml"

application_services=(gateway gate_server computing user item commerce server_manager order pay promotion xds)
middleware_services=(nginx envoy etcd kafka tempo loki prometheus grafana)

usage() {
    echo "Usage: $0 [start|restart] [all|service ...]"
    echo "Applications: ${application_services[*]}"
    echo "Middleware:   ${middleware_services[*]}"
    echo "No action defaults to 'start'; no service defaults to all applications."
    echo "The 'nginx' alias operates on the load balancer and all three proxy nodes."
}

command -v docker >/dev/null 2>&1 || {
    echo "error: docker is required" >&2
    exit 127
}
docker compose version >/dev/null 2>&1 || {
    echo "error: Docker Compose plugin is required" >&2
    exit 127
}

action=start
if [[ "${1:-}" == "start" || "${1:-}" == "restart" ]]; then
    action=$1
    shift
fi

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" || "${1:-}" == "help" ]]; then
    usage
    exit 0
fi

if (($# == 0)) || [[ "${1:-}" == "all" ]]; then
    if (($# > 1)); then
        echo "error: 'all' cannot be combined with service names" >&2
        exit 2
    fi
    compose_services=(xds gateway gate-server computing user item commerce server-manager order pay promotion nginx-lb)
else
    compose_services=()
    start_nginx=false
    for service in "$@"; do
        case "$service" in
            gate_server) compose_services+=("gate-server") ;;
            server_manager) compose_services+=("server-manager") ;;
            nginx)
                compose_services+=("nginx-lb" "nginx-node-1" "nginx-node-2" "nginx-node-3")
                ;;
            gateway)
                compose_services+=("gateway")
                start_nginx=true
                ;;
            computing|user|item|commerce|order|pay|promotion|xds|envoy|etcd|kafka|tempo|loki|prometheus|grafana)
                compose_services+=("$service")
                ;;
            *)
                echo "error: unknown service '$service'" >&2
                usage >&2
                exit 2
                ;;
        esac
    done
    if [[ "$start_nginx" == true ]]; then
        compose_services+=("nginx-lb")
    fi
fi

if [[ "$action" == restart ]]; then
    echo "==> Recreating: ${compose_services[*]}"
    docker compose -f "$compose_file" up -d --build --force-recreate "${compose_services[@]}"
else
    echo "==> Building images and starting: ${compose_services[*]}"
    docker compose -f "$compose_file" up -d --build "${compose_services[@]}"
fi
docker compose -f "$compose_file" ps "${compose_services[@]}"
