#!/usr/bin/env bash
set -euo pipefail

namespace=${KUBE_NAMESPACE:-one-adventure}
mysql_port=${MYSQL_LOCAL_PORT:-13306}
redis_port=${REDIS_LOCAL_PORT:-16379}
mysql_pid=''
redis_pid=''
tmp_dir=''

usage() {
  echo "Usage: $0 [--namespace NAMESPACE] [--mysql-port PORT] [--redis-port PORT]"
  echo "Forwards 127.0.0.1:MYSQL_PORT to mysql:3306 and 127.0.0.1:REDIS_PORT to redis:6379."
  echo "Defaults: namespace=KUBE_NAMESPACE or one-adventure, mysql-port=13306, redis-port=16379."
  echo "The forwarding stays active until Ctrl-C and then is cleaned up automatically."
}

valid_port() { [[ "$1" =~ ^[0-9]+$ && "$1" -ge 1 && "$1" -le 65535 ]]; }
while (($#)); do
  case "$1" in
    --namespace)
      (($# >= 2)) || { echo 'error: --namespace requires a value' >&2; exit 2; }
      namespace=$2; shift ;;
    --mysql-port)
      (($# >= 2)) || { echo 'error: --mysql-port requires a value' >&2; exit 2; }
      mysql_port=$2; shift ;;
    --redis-port)
      (($# >= 2)) || { echo 'error: --redis-port requires a value' >&2; exit 2; }
      redis_port=$2; shift ;;
    -h|--help) usage; exit 0 ;;
    --*) echo "error: unknown option: $1" >&2; exit 2 ;;
    *) echo "error: unexpected argument: $1" >&2; exit 2 ;;
  esac
  shift
done
[[ "$namespace" =~ ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$ && ${#namespace} -le 63 ]] || {
  echo "error: invalid namespace: $namespace" >&2; exit 2;
}
valid_port "$mysql_port" || { echo "error: invalid MySQL local port: $mysql_port" >&2; exit 2; }
valid_port "$redis_port" || { echo "error: invalid Redis local port: $redis_port" >&2; exit 2; }
[[ "$mysql_port" != "$redis_port" ]] || { echo 'error: MySQL and Redis local ports must differ' >&2; exit 2; }
command -v kubectl >/dev/null 2>&1 || { echo 'error: kubectl is required' >&2; exit 127; }

cleanup() {
  trap - EXIT INT TERM
  [[ -n "$mysql_pid" ]] && kill "$mysql_pid" 2>/dev/null || true
  [[ -n "$redis_pid" ]] && kill "$redis_pid" 2>/dev/null || true
  [[ -n "$mysql_pid" ]] && wait "$mysql_pid" 2>/dev/null || true
  [[ -n "$redis_pid" ]] && wait "$redis_pid" 2>/dev/null || true
  [[ -n "$tmp_dir" ]] && rm -rf "$tmp_dir"
}
trap cleanup EXIT INT TERM

tmp_dir=$(mktemp -d)
kubectl -n "$namespace" get service/mysql >/dev/null
kubectl -n "$namespace" get service/redis >/dev/null

kubectl -n "$namespace" port-forward --address 127.0.0.1 "service/mysql" "$mysql_port:3306" >"$tmp_dir/mysql.log" 2>&1 &
mysql_pid=$!
kubectl -n "$namespace" port-forward --address 127.0.0.1 "service/redis" "$redis_port:6379" >"$tmp_dir/redis.log" 2>&1 &
redis_pid=$!

for attempt in {1..30}; do
  mysql_ready=false
  redis_ready=false
  kill -0 "$mysql_pid" 2>/dev/null && grep -q 'Forwarding from 127.0.0.1:' "$tmp_dir/mysql.log" && mysql_ready=true
  kill -0 "$redis_pid" 2>/dev/null && grep -q 'Forwarding from 127.0.0.1:' "$tmp_dir/redis.log" && redis_ready=true
  [[ "$mysql_ready" == true && "$redis_ready" == true ]] && break
  sleep 1
done
if [[ "$mysql_ready" != true || "$redis_ready" != true ]]; then
  echo 'error: failed to start port forwarding' >&2
  cat "$tmp_dir/mysql.log" "$tmp_dir/redis.log" >&2
  exit 1
fi

echo "MySQL: 127.0.0.1:$mysql_port -> $namespace/mysql:3306"
echo "Redis: 127.0.0.1:$redis_port -> $namespace/redis:6379"
echo 'Port forwarding is active. Press Ctrl-C to stop.'
while kill -0 "$mysql_pid" 2>/dev/null && kill -0 "$redis_pid" 2>/dev/null; do
  sleep 2
done
echo 'Port forwarding stopped.'
