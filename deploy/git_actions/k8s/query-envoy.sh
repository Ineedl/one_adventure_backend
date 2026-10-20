#!/usr/bin/env bash
set -euo pipefail
namespace=${KUBE_NAMESPACE:-one-adventure}
command_name=${1:-summary}
tmp_dir=''; forward_pid=''
usage(){ echo "Usage: $0 [summary|configmap|clusters|endpoints|listeners|config|stats]"; }
case "$command_name" in -h|--help|help) usage; exit 0;; summary|configmap|clusters|endpoints|listeners|config|stats);; *) echo "error: unknown command: $command_name" >&2; usage >&2; exit 2;; esac
[[ "$namespace" =~ ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$ && ${#namespace} -le 63 ]] || { echo "error: invalid namespace: $namespace" >&2; exit 2; }
command -v kubectl >/dev/null || { echo 'error: kubectl is required' >&2; exit 127; }
if [[ "$command_name" == configmap ]]; then
  kubectl --request-timeout=30s -n "$namespace" get configmap envoy-config -o jsonpath='{.data.envoy\.yaml}'
  exit 0
fi
tmp_dir=$(mktemp -d)
cleanup(){ trap - EXIT INT TERM; [[ -n "$forward_pid" ]] && kill "$forward_pid" 2>/dev/null || true; [[ -n "$forward_pid" ]] && wait "$forward_pid" 2>/dev/null || true; rm -rf "$tmp_dir"; }
trap cleanup EXIT INT TERM
kubectl --request-timeout=30s -n "$namespace" get service/envoy >/dev/null
kubectl --request-timeout=30s -n "$namespace" port-forward --address 127.0.0.1 service/envoy 19901:9901 >"$tmp_dir/forward.log" 2>&1 & forward_pid=$!
ready=false
for _ in {1..30}; do
  kill -0 "$forward_pid" 2>/dev/null || break
  if curl --fail --silent http://127.0.0.1:19901/ready >/dev/null 2>&1; then ready=true; break; fi
  sleep 1
done
[[ "$ready" == true ]] || { cat "$tmp_dir/forward.log" >&2; echo 'error: Envoy Admin API did not become ready' >&2; exit 1; }
request(){ curl --fail --silent --show-error "http://127.0.0.1:19901$1"; }
pretty(){ command -v jq >/dev/null 2>&1 && jq . || cat; }
case "$command_name" in
 summary) echo '== Envoy ConfigMap =='; kubectl -n "$namespace" get configmap envoy-config -o jsonpath='{.data.envoy\.yaml}'; echo; echo '== Envoy readiness =='; request /ready; echo; echo '== Envoy clusters =='; request '/clusters?format=json' | pretty;;
 clusters) request '/clusters?format=json' | pretty;;
 endpoints) request '/config_dump?resource=dynamic_endpoint_configs' | pretty;;
 listeners) request '/listeners?format=json' | pretty;;
 config) request /config_dump | pretty;;
 stats) request '/stats?format=json' | pretty;;
esac
