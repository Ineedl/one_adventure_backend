#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
project_dir=$(CDPATH= cd -- "$script_dir/../.." && pwd)
: "${IMAGE_REPOSITORY:=${REGISTRY:-}}"
: "${IMAGE_TAG:=${GITHUB_SHA:-$(git -C "$project_dir" rev-parse HEAD)}}"
: "${KUBE_NAMESPACE:=one-adventure}"
deploy_only=false
docker_config_file=${DOCKER_CONFIG_FILE:-}
xds_image_tag=${XDS_IMAGE_TAG:-}
positional=()
while (($#)); do
  arg=$1
  case "$arg" in
    --deploy-only) deploy_only=true ;;
    --xds-tag)
      if (($# < 2)) || [[ -z "$2" || "$2" == --* ]]; then
        echo "error: --xds-tag requires an image version" >&2
        exit 2
      fi
      xds_image_tag=$2
      shift
      ;;
    --docker-config)
      if (($# < 2)) || [[ -z "$2" || "$2" == --* ]]; then
        echo "error: --docker-config requires a file path" >&2
        exit 2
      fi
      docker_config_file=$2
      shift
      ;;
    -h|--help)
      echo "Usage: $0 [--deploy-only] [--docker-config FILE] [--xds-tag TAG] IMAGE_REPOSITORY [IMAGE_TAG]"
      echo "--deploy-only skips binary builds, image builds, and image pushes."
      echo '--docker-config FILE imports Docker auth for Kubernetes image pulls (e.g. "$HOME/.docker/config.json").'
      echo '--xds-tag TAG overrides the xds version for image pushes and deployment (also XDS_IMAGE_TAG).'
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
xds_image_tag=${xds_image_tag:-$IMAGE_TAG}
for tag in "$IMAGE_TAG" "$xds_image_tag"; do
  [[ "$tag" =~ ^[a-zA-Z0-9_][a-zA-Z0-9_.-]*$ && ${#tag} -le 116 ]] || {
    echo "error: invalid image version: $tag" >&2
    exit 2
  }
done
command -v kubectl >/dev/null 2>&1 || { echo "error: kubectl is required" >&2; exit 127; }
command -v python3 >/dev/null 2>&1 || { echo "error: python3 is required" >&2; exit 127; }
python3 -c 'import yaml' || { echo "error: install PyYAML (python3 -m pip install PyYAML)" >&2; exit 2; }
middleware_secret='{}'
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
  "$project_dir/script/build-services.sh" all
  docker compose -f "$project_dir/one_adventure/docker-compose.yml" build "${services[@]}"
fi
for service in "${services[@]}"; do
  image_tag="$service-$IMAGE_TAG"
  if [[ "$service" == xds ]]; then image_tag="xds-$xds_image_tag"; fi
  docker tag "one-adventure/$service:latest" "$IMAGE_REPOSITORY:$image_tag"
  docker push "$IMAGE_REPOSITORY:$image_tag"
done
fi
render_dir=$(mktemp -d)
trap 'rm -rf "$render_dir"' EXIT
python3 "$script_dir/render-config.py" "$CONFIG_ROOT" "$render_dir" "${services[@]}"
cp "$script_dir"/*.yaml "$render_dir"/
for file in "$render_dir"/*.yaml; do
  sed -i "s|__IMAGE_REPOSITORY__|$IMAGE_REPOSITORY|g; s|__IMAGE_TAG__|$IMAGE_TAG|g; s|__XDS_IMAGE_TAG__|$xds_image_tag|g; s|__KUBE_NAMESPACE__|$KUBE_NAMESPACE|g; s|__HOST_GATEWAY_IP__|$HOST_GATEWAY_IP|g" "$file"
done
kubectl apply -f "$render_dir/namespace.yaml"
kubectl -n "$KUBE_NAMESPACE" create secret generic middleware-auth \
  --from-file=mysql-root-password="$render_dir/mysql-root-password" \
  --from-file=redis-password="$render_dir/redis-password" \
  --dry-run=client -o yaml | kubectl apply -f -
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
kubectl -n "$KUBE_NAMESPACE" create configmap envoy-config \
  --from-file=envoy.yaml="$project_dir/one_adventure/envoy.yaml" \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl -n "$KUBE_NAMESPACE" create configmap kafka-config \
  --from-file=server.properties="$project_dir/one_adventure/kafka/server.properties" \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f "$render_dir/middleware.yaml"
for middleware in mysql redis; do
  kubectl -n "$KUBE_NAMESPACE" rollout status "deployment/$middleware" --timeout=10m
done
kubectl apply -f "$render_dir/stack.yaml"
kubectl apply -f "$render_dir/applications.yaml"
for service in "${services[@]}"; do
  config_hash=$(cat "$render_dir/$service-config.sha256")
  kubectl -n "$KUBE_NAMESPACE" patch deployment "$service" --type=strategic \
    -p "{\"spec\":{\"template\":{\"metadata\":{\"annotations\":{\"checksum/config\":\"$config_hash\"}}}}}"
done
if [[ -n "$docker_config_file" ]]; then
  for service in "${services[@]}"; do
    kubectl -n "$KUBE_NAMESPACE" patch deployment "$service" --type=strategic \
      -p '{"spec":{"template":{"spec":{"imagePullSecrets":[{"name":"dockerhub-auth"}]}}}}'
  done
fi
deployments=$(kubectl -n "$KUBE_NAMESPACE" get deployments -o name)
for deployment in $deployments; do
  kubectl -n "$KUBE_NAMESPACE" rollout status "$deployment" --timeout=5m
done
