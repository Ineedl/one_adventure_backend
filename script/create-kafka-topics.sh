#!/usr/bin/env bash

# Create all Kafka topics used by the project. Safe to run repeatedly.
# Usage: script/create-kafka-topics.sh [--recreate --yes]
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
project_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
compose_file="$project_dir/one_adventure/docker-compose.yml"
kafka_container="${KAFKA_CONTAINER:-kafka}"
bootstrap_server="${KAFKA_BOOTSTRAP_SERVER:-kafka:9092}"
partitions="${KAFKA_PARTITIONS:-3}"
replication_factor="${KAFKA_REPLICATION_FACTOR:-1}"

topics=(
    promotion_order_create
    promotion_order_compensate
    pay_report
    refund_report
)

recreate=false
confirm=false
for arg in "$@"; do
    case "$arg" in
        --recreate) recreate=true ;;
        --yes) confirm=true ;;
        -h|--help)
            echo "Usage: $0 [--recreate --yes]"
            echo "Creates: ${topics[*]}"
            echo "--recreate deletes and recreates project topics (requires --yes)."
            exit 0
            ;;
        *) echo "error: unknown option '$arg'" >&2; exit 2 ;;
    esac
done

if [[ "$recreate" == true && "$confirm" != true ]]; then
    echo "error: --recreate requires --yes because existing messages and offsets will be deleted" >&2
    exit 2
fi

if [[ "$recreate" == true ]]; then
    echo "Deleting project topics..."
    for topic in "${topics[@]}"; do
        docker compose -f "$compose_file" exec -T "$kafka_container" \
            /opt/kafka/bin/kafka-topics.sh \
            --bootstrap-server "$bootstrap_server" \
            --delete --topic "$topic" >/dev/null 2>&1 || true
        echo "deleted or absent: $topic"
    done
    sleep 2
fi

if [[ "$recreate" == true ]]; then
    echo "Recreating topics..."
else
    echo "Creating topics..."
fi

command -v docker >/dev/null 2>&1 || {
    echo "error: docker is required" >&2
    exit 127
}

echo "Waiting for Kafka at ${bootstrap_server}..."
for attempt in $(seq 1 30); do
    if docker compose -f "$compose_file" exec -T "$kafka_container" \
        /opt/kafka/bin/kafka-topics.sh --bootstrap-server "$bootstrap_server" --list >/dev/null 2>&1; then
        break
    fi
    if [[ "$attempt" == 30 ]]; then
        echo "error: Kafka did not become ready" >&2
        exit 1
    fi
    sleep 2
done

for topic in "${topics[@]}"; do
    docker compose -f "$compose_file" exec -T "$kafka_container" \
        /opt/kafka/bin/kafka-topics.sh \
        --bootstrap-server "$bootstrap_server" \
        --create --if-not-exists \
        --topic "$topic" \
        --partitions "$partitions" \
        --replication-factor "$replication_factor"
    echo "created or already exists: $topic"
done

echo "Kafka topics are ready."
