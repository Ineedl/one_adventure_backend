#!/usr/bin/env bash

# 清理 Envoy 进程内缓存的 CDS/EDS 配置，并重新启动 Envoy。
# 注意：此脚本不会删除 etcd 中的微服务注册信息；服务重新启动后会由 xDS 再次下发。
set -euo pipefail

compose_file="$(cd "$(dirname "$0")" && pwd)/docker-compose.yml"
admin_url="${ENVOY_ADMIN_URL:-http://127.0.0.1:9901}"

echo "检查 Envoy Admin 接口: ${admin_url}"
if ! curl --fail --silent --show-error "${admin_url}/server_info" >/dev/null; then
  echo "Envoy Admin 接口不可用，停止清理。" >&2
  exit 1
fi

echo "停止当前 Envoy，清理内存中的动态注册信息..."
curl --fail --silent --show-error -X POST "${admin_url}/quitquitquit" >/dev/null

echo "重新创建 Envoy 容器..."
docker compose -f "${compose_file}" up -d --force-recreate envoy

echo "Envoy 已重启，等待 xDS 重新下发配置..."
echo "查看当前集群: curl -s ${admin_url}/clusters"
