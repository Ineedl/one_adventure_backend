#!/usr/bin/env bash
set -euo pipefail

# Reapply and recreate Kafka, MySQL, and Redis. PVCs are never deleted.
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
project_dir=$(CDPATH= cd -- "$script_dir/../.." && pwd)
: "${KUBE_NAMESPACE:=one-adventure}"
: "${ROLLOUT_TIMEOUT_SECONDS:=600}"
: "${PVC_TIMEOUT_SECONDS:=120}"
local_storage=true

usage() {
  echo "Usage: $0 [--no-local-storage]"
  echo "Applies Kafka, MySQL, and Redis resources, then recreates their StatefulSets."
  echo "PVCs and their data are preserved."
}
while (($#)); do
  case "$1" in
    --no-local-storage) local_storage=false ;;
    -h|--help) usage; exit 0 ;;
    *) echo "error: unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
  shift
done
[[ "$KUBE_NAMESPACE" =~ ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$ && ${#KUBE_NAMESPACE} -le 63 ]] || {
  echo "error: invalid namespace: $KUBE_NAMESPACE" >&2; exit 2;
}
command -v kubectl >/dev/null 2>&1 || { echo 'error: kubectl is required' >&2; exit 127; }
command -v python3 >/dev/null 2>&1 || { echo 'error: python3 is required' >&2; exit 127; }
python3 -c 'import yaml' >/dev/null 2>&1 || { echo 'error: PyYAML is required' >&2; exit 2; }

tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT
sed "s|__KUBE_NAMESPACE__|$KUBE_NAMESPACE|g" "$script_dir/middleware.yaml" > "$tmp_dir/middleware.yaml"
sed "s|__KUBE_NAMESPACE__|$KUBE_NAMESPACE|g" "$script_dir/stack.yaml" > "$tmp_dir/stack.yaml"

# Keep only PVCs, Services, and StatefulSets needed by this targeted operation.
python3 - "$tmp_dir/middleware.yaml" "$tmp_dir/stack.yaml" > "$tmp_dir/selected.yaml" <<'PY'
import sys
from pathlib import Path
import yaml

selected = {
    "kafka-data", "mysql-data", "redis-data", "mysql-init",
    "mysql", "mysql-headless", "kafka", "kafka-headless",
    "redis", "redis-headless",
}
docs = []
for filename in sys.argv[1:]:
    for doc in yaml.safe_load_all(Path(filename).read_text()):
        if doc and doc.get("metadata", {}).get("name") in selected:
            docs.append(doc)
print(yaml.safe_dump_all(docs, sort_keys=False), end="")
PY

echo "Applying Kafka/MySQL/Redis resources in namespace $KUBE_NAMESPACE"
kubectl -n "$KUBE_NAMESPACE" delete deployment/kafka deployment/mysql deployment/redis \
  --ignore-not-found --cascade=foreground --wait=true --timeout=5m
kubectl apply -f "$tmp_dir/selected.yaml"
kubectl -n "$KUBE_NAMESPACE" create configmap kafka-config \
  --from-file=server.properties="$project_dir/one_adventure/kafka/server.properties" \
  --dry-run=client -o yaml | kubectl apply -f -

if [[ "$local_storage" == true ]]; then
  STORAGE_CLAIMS=kafka-data,mysql-data,redis-data KUBE_NAMESPACE="$KUBE_NAMESPACE" \
    python3 "$script_dir/setup-local-storage.py"
fi
for claim in kafka-data mysql-data redis-data; do
  echo "Waiting for PVC $claim"
  kubectl -n "$KUBE_NAMESPACE" wait --for=jsonpath='{.status.phase}'=Bound \
    "pvc/$claim" --timeout="${PVC_TIMEOUT_SECONDS}s"
done

echo 'Recreating Kafka, MySQL, and Redis Pods'
kubectl -n "$KUBE_NAMESPACE" rollout restart statefulset/kafka statefulset/mysql statefulset/redis
kubectl -n "$KUBE_NAMESPACE" rollout status statefulset/kafka --timeout="${ROLLOUT_TIMEOUT_SECONDS}s"
kubectl -n "$KUBE_NAMESPACE" rollout status statefulset/mysql --timeout="${ROLLOUT_TIMEOUT_SECONDS}s"
kubectl -n "$KUBE_NAMESPACE" rollout status statefulset/redis --timeout="${ROLLOUT_TIMEOUT_SECONDS}s"
kubectl -n "$KUBE_NAMESPACE" get statefulset/kafka statefulset/mysql statefulset/redis pvc/kafka-data pvc/mysql-data pvc/redis-data -o wide
kubectl -n "$KUBE_NAMESPACE" get pods -l 'app in (kafka,mysql,redis)' -o wide
echo 'Kafka, MySQL, and Redis recreation completed.'
