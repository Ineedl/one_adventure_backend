#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
project_dir=${GITHUB_WORKSPACE:-$(CDPATH= cd -- "$script_dir/../.." && pwd)}
: "${IMAGE_REPOSITORY:=${REGISTRY:-}}"
: "${IMAGE_TAG:=${GITHUB_SHA:-$(git -C "$project_dir" rev-parse HEAD)}}"
: "${KUBE_NAMESPACE:=one-adventure}"
: "${ROLLOUT_TIMEOUT_SECONDS:=300}"
: "${PVC_TIMEOUT_SECONDS:=120}"
for limit in "$ROLLOUT_TIMEOUT_SECONDS" "$PVC_TIMEOUT_SECONDS"; do
  [[ "$limit" =~ ^[1-9][0-9]{0,4}$ ]] || { echo 'error: deployment timeouts must be positive seconds (1-99999)' >&2; exit 2; }
done
command -v timeout >/dev/null || { echo 'error: GNU timeout is required' >&2; exit 127; }
# Bound each API request as well as the complete watch command.
kubectl() { command kubectl --request-timeout=30s "$@"; }
diagnose_deployment() {
  local service=$1
  timeout 20s kubectl --request-timeout=10s -n "$KUBE_NAMESPACE" get pods,pvc -o wide || true
  timeout 20s kubectl --request-timeout=10s -n "$KUBE_NAMESPACE" describe pods -l "app=$service" || true
  timeout 20s kubectl --request-timeout=10s -n "$KUBE_NAMESPACE" get events --field-selector type=Warning --sort-by=.lastTimestamp || true
}
wait_deployment() {
  local service=$1
  echo "Waiting up to ${ROLLOUT_TIMEOUT_SECONDS}s for $service"
  if ! timeout --kill-after=5s "$((ROLLOUT_TIMEOUT_SECONDS + 10))s" kubectl --request-timeout=30s -n "$KUBE_NAMESPACE" rollout status "deployment/$service" --timeout="${ROLLOUT_TIMEOUT_SECONDS}s"; then
    echo "error: deployment/$service failed to become ready within ${ROLLOUT_TIMEOUT_SECONDS}s" >&2
    diagnose_deployment "$service"
    exit 1
  fi
}
# Actions-only copy; keep deploy.sh available for the original manual flow.
selected_service=${DEPLOY_SERVICE:-all}
selected_service=${selected_service//_/-}
case "$selected_service" in
  all|gateway|gate-server|computing|user|item|commerce|ws-gateway|order|pay|promotion|xds) ;;
  *) echo "error: unknown service: $selected_service" >&2; exit 2 ;;
esac
deploy_only=false
local_storage=false
docker_config_file=${DOCKER_CONFIG_FILE:-}
positional=()
while (($#)); do
  arg=$1
  case "$arg" in
    --deploy-only) deploy_only=true ;;
    --local-storage) local_storage=true ;;
    --docker-config)
      if (($# < 2)) || [[ -z "$2" || "$2" == --* ]]; then
        echo "error: --docker-config requires a file path" >&2
        exit 2
      fi
      docker_config_file=$2
      shift
      ;;
    -h|--help)
      echo "Usage: $0 [--deploy-only] [--local-storage] [--docker-config FILE] IMAGE_REPOSITORY [IMAGE_TAG]"
      echo '--local-storage creates retained node-local PVs for pending MySQL/Redis PVCs (single node, or STORAGE_NODE_NAME).'
      echo "--deploy-only skips binary builds, image builds, and image pushes."
      echo '--docker-config FILE imports Docker auth for Kubernetes image pulls (e.g. "$HOME/.docker/config.json").'
      echo 'MySQL/Redis passwords are generated automatically and reused from the cluster; environment variables override them.'
      echo 'Python 3 with PyYAML is required to render cluster configuration.'
      exit 0
      ;;
    --*) echo "error: unknown option: $arg" >&2; exit 2 ;;
    *) positional+=("$arg") ;;
  esac
  shift
done
if [[ -n "$docker_config_file" && ( ! -f "$docker_config_file" || ! -r "$docker_config_file" ) ]]; then
  echo "error: Docker config file is not a readable regular file: $docker_config_file" >&2
  exit 2
fi
if ((${#positional[@]} > 2)); then
  echo "error: expected IMAGE_REPOSITORY and optional IMAGE_TAG" >&2
  exit 2
fi
set -- "${positional[@]}"
if [[ -n "${1:-}" ]]; then IMAGE_REPOSITORY=$1; fi
IMAGE_REPOSITORY=${IMAGE_REPOSITORY%/}
[[ -n "$IMAGE_REPOSITORY" ]] || { echo "usage: $0 IMAGE_REPOSITORY [IMAGE_TAG]" >&2; exit 2; }
if [[ -n "${2:-}" ]]; then IMAGE_TAG=$2; fi
for tag in "$IMAGE_TAG"; do
  [[ "$tag" =~ ^[a-zA-Z0-9_][a-zA-Z0-9_.-]*$ && ${#tag} -le 116 ]] || {
    echo "error: invalid image version: $tag" >&2
    exit 2
  }
done
command -v kubectl >/dev/null 2>&1 || { echo "error: kubectl is required" >&2; exit 127; }
command -v python3 >/dev/null 2>&1 || { echo "error: python3 is required" >&2; exit 127; }
python3 -c 'import yaml' || { echo "error: install PyYAML (python3 -m pip install PyYAML)" >&2; exit 2; }
middleware_secret='{}'
if [[ "$selected_service" != all ]]; then
  # Partial releases must use the credentials already used by middleware.
  middleware_secret=$(kubectl -n "$KUBE_NAMESPACE" get secret middleware-auth -o json)
  python3 -c 'import json,sys; d=json.load(sys.stdin).get("data", {}); sys.exit(0 if d.get("mysql-root-password") and d.get("redis-password") else "error: deploy all first to initialize middleware credentials")' <<<"$middleware_secret"
  unset MYSQL_ROOT_PASSWORD REDIS_PASSWORD
fi
if [[ -z "${MYSQL_ROOT_PASSWORD:-}" || -z "${REDIS_PASSWORD:-}" ]]; then
  middleware_secret=$(kubectl -n "$KUBE_NAMESPACE" get secret middleware-auth --ignore-not-found -o json)
  middleware_secret=${middleware_secret:-'{}'}
fi
default_password() {
  python3 -c '
import base64, json, secrets, sys
value = (json.load(sys.stdin).get("data") or {}).get(sys.argv[1])
print(base64.b64decode(value).decode() if value else secrets.token_hex(24), end="")
' "$1" <<<"$middleware_secret"
}
: "${MYSQL_ROOT_PASSWORD:=$(default_password mysql-root-password)}"
: "${REDIS_PASSWORD:=$(default_password redis-password)}"
unset middleware_secret
# GoFrame's MySQL link format uses these characters as separators.
[[ "$MYSQL_ROOT_PASSWORD" =~ ^[a-zA-Z0-9_.-]+$ ]] || {
  echo "error: MYSQL_ROOT_PASSWORD may contain only letters, digits, underscores, dots and hyphens" >&2
  exit 2
}
export MYSQL_ROOT_PASSWORD REDIS_PASSWORD
services=(gateway gate-server computing user item commerce ws-gateway order pay promotion xds)
if [[ "$selected_service" != all ]]; then services=("$selected_service"); fi
echo "Deploying services: ${services[*]} (version: $IMAGE_TAG)"
# Remaining host.docker.internal references (such as tracing) use the node IP.
: "${HOST_GATEWAY_IP:=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')}"
: "${CONFIG_ROOT:=$project_dir}"
[[ -n "$HOST_GATEWAY_IP" ]] || { echo "error: HOST_GATEWAY_IP is required" >&2; exit 2; }
for service in "${services[@]}"; do
  config_dir="$CONFIG_ROOT/${service//-/_}/manifest/config-docker"
  [[ -d "$config_dir" ]] || { echo "error: missing configuration directory: $config_dir" >&2; exit 2; }
done
if [[ "$deploy_only" == false ]]; then
command -v docker >/dev/null 2>&1 || { echo "error: docker is required" >&2; exit 127; }
if [[ "${SKIP_BUILD:-0}" != 1 ]]; then
  build_services=()
  for service in "${services[@]}"; do build_services+=("${service//-/_}"); done
  bash "$project_dir/script/build-services.sh" "${build_services[@]}"
  docker compose -f "$project_dir/one_adventure/docker-compose.yml" build "${services[@]}"
fi
for service in "${services[@]}"; do
  image_tag="$service-$IMAGE_TAG"
  docker tag "one-adventure/$service:latest" "$IMAGE_REPOSITORY:$image_tag"
  docker push "$IMAGE_REPOSITORY:$image_tag"
done
fi
render_dir=$(mktemp -d)
trap 'rm -rf "$render_dir"' EXIT
python3 "$script_dir/render-config.py" "$CONFIG_ROOT" "$render_dir" "${services[@]}"
cp "$script_dir"/*.yaml "$render_dir"/
for file in "$render_dir"/*.yaml; do
  sed -i "s|__IMAGE_REPOSITORY__|$IMAGE_REPOSITORY|g; s|__IMAGE_TAG__|$IMAGE_TAG|g; s|__XDS_IMAGE_TAG__|$IMAGE_TAG|g; s|__KUBE_NAMESPACE__|$KUBE_NAMESPACE|g; s|__HOST_GATEWAY_IP__|$HOST_GATEWAY_IP|g" "$file"
done
if [[ "$selected_service" == all ]]; then
kubectl apply -f "$render_dir/namespace.yaml"
kubectl -n "$KUBE_NAMESPACE" create secret generic middleware-auth \
  --from-file=mysql-root-password="$render_dir/mysql-root-password" \
  --from-file=redis-password="$render_dir/redis-password" \
  --dry-run=client -o yaml | kubectl apply -f -
fi
if [[ -n "$docker_config_file" ]]; then
  kubectl -n "$KUBE_NAMESPACE" create secret generic dockerhub-auth \
    --type=kubernetes.io/dockerconfigjson \
    --from-file=".dockerconfigjson=$docker_config_file" \
    --dry-run=client -o yaml | kubectl apply -f -
fi
for service in "${services[@]}"; do
  config_dir="$render_dir/config/$service"
  kubectl -n "$KUBE_NAMESPACE" create secret generic "$service-config" \
    --from-file="$config_dir" --dry-run=client -o yaml | kubectl apply -f -
done
if [[ "$selected_service" == all ]]; then
kubectl -n "$KUBE_NAMESPACE" create configmap envoy-config \
  --from-file=envoy.yaml="$project_dir/one_adventure/envoy.yaml" \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl -n "$KUBE_NAMESPACE" create configmap kafka-config \
  --from-file=server.properties="$project_dir/one_adventure/kafka/server.properties" \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f "$render_dir/middleware.yaml"
if [[ "$local_storage" == true ]]; then
  KUBE_NAMESPACE="$KUBE_NAMESPACE" python3 "$script_dir/setup-local-storage.py"
fi
for middleware in mysql redis; do
  if ! timeout --kill-after=5s "$((PVC_TIMEOUT_SECONDS + 10))s" kubectl --request-timeout=30s -n "$KUBE_NAMESPACE" wait --for=jsonpath='{.status.phase}'=Bound "pvc/$middleware-data" --timeout="${PVC_TIMEOUT_SECONDS}s"; then
    echo "error: PVC $middleware-data did not bind within ${PVC_TIMEOUT_SECONDS}s" >&2
    timeout 20s kubectl --request-timeout=10s -n "$KUBE_NAMESPACE" describe pvc "$middleware-data" || true
    diagnose_deployment "$middleware"
    exit 1
  fi
  wait_deployment "$middleware"
done
fi
# Apply only selected application resources for a partial release. Shared
# middleware and other applications keep their existing versions and config.
DOCKER_CONFIG_FILE="$docker_config_file" python3 - "$render_dir" "${services[@]}" <<'FILTER'
from pathlib import Path
import os
import sys
import uuid
import yaml
root = Path(sys.argv[1])
selected = set(sys.argv[2:])
# One token per invocation also refreshes images when a tag is reused.
release_id = uuid.uuid4().hex
documents = []
for name in ("stack.yaml", "applications.yaml"):
    for doc in yaml.safe_load_all((root / name).read_text()):
        if not doc:
            continue
        if doc.get("metadata", {}).get("name") not in selected:
            if len(selected) != 11:
                continue
        if doc.get("kind") == "Deployment" and doc["metadata"]["name"] in selected:
            service = doc["metadata"]["name"]
            annotations = doc["spec"]["template"].setdefault("metadata", {}).setdefault("annotations", {})
            annotations["checksum/config"] = (root / f"{service}-config.sha256").read_text().strip()
            annotations["one-adventure/deployment-id"] = release_id
            if os.environ.get("DOCKER_CONFIG_FILE"):
                doc["spec"]["template"]["spec"]["imagePullSecrets"] = [{"name": "dockerhub-auth"}]
            for container in doc["spec"]["template"]["spec"]["containers"]:
                container["imagePullPolicy"] = "Always"
        documents.append(doc)
(root / "selected.yaml").write_text(yaml.safe_dump_all(documents, sort_keys=False))
FILTER
kubectl apply -f "$render_dir/selected.yaml"
# The complete Pod template is applied once; only observe the rollout here.
for service in "${services[@]}"; do
  wait_deployment "$service"
done
