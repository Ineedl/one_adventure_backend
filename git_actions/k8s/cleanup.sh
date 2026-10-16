#!/usr/bin/env bash
set -euo pipefail

namespace=${KUBE_NAMESPACE:-one-adventure}
dry_run=false
delete_data=false
positional=()
while (($#)); do
  case "$1" in
    --dry-run) dry_run=true ;;
    --delete-data) delete_data=true ;;
    -h|--help)
      echo "Usage: $0 [--dry-run] [--delete-data] [NAMESPACE]"
      echo 'Removes all project application/middleware Deployments, Services and configuration.'
      echo 'By default preserves the namespace, PVCs and middleware-auth Secret for redeployment.'
      echo 'NAMESPACE defaults to KUBE_NAMESPACE or one-adventure. No confirmation prompt.'
      echo '--dry-run previews the exact resources without deleting them.'
      echo '--delete-data also deletes MySQL/Redis PVCs, project static PVs and middleware-auth Secret; node directories remain.'
      exit 0
      ;;
    --*) echo "error: unknown option: $1" >&2; exit 2 ;;
    *) positional+=("$1") ;;
  esac
  shift
done
if ((${#positional[@]} > 1)); then
  echo "error: expected at most one namespace" >&2
  exit 2
fi
if ((${#positional[@]} == 1)); then namespace=${positional[0]}; fi
if [[ ! "$namespace" =~ ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$ || ${#namespace} -gt 63 ]]; then
  echo "error: invalid namespace: $namespace" >&2
  exit 2
fi
case "$namespace" in
  default|kube-*) echo "error: refusing to clean system namespace: $namespace" >&2; exit 2 ;;
esac
command -v kubectl >/dev/null 2>&1 || { echo "error: kubectl is required" >&2; exit 127; }
context=$(kubectl config current-context)
echo "Kubernetes context: $context"
echo "Project namespace: $namespace"
existing=$(kubectl get namespace "$namespace" --ignore-not-found -o name)
if [[ -z "$existing" && "$delete_data" == false ]]; then
  echo "Namespace does not exist; nothing to clean up."
  exit 0
fi
if [[ -n "$existing" ]]; then
owner=$(kubectl get namespace "$namespace" -o 'jsonpath={.metadata.labels.app\.kubernetes\.io/part-of}')
if [[ "$owner" != one-adventure ]]; then
  echo "error: namespace must have label app.kubernetes.io/part-of=one-adventure" >&2
  exit 2
fi
fi
applications=(gateway gate-server computing user item commerce ws-gateway order pay promotion xds)
services=("${applications[@]}" etcd kafka envoy mysql redis)
resources=()
for service in "${services[@]}"; do
  resources+=("deployment/$service" "service/$service")
done
for service in "${applications[@]}"; do
  resources+=("secret/$service-config")
done
resources+=(secret/dockerhub-auth configmap/envoy-config configmap/kafka-config configmap/mysql-init)
data_resources=(persistentvolumeclaim/mysql-data persistentvolumeclaim/redis-data secret/middleware-auth)
# PVs are cluster-scoped and may survive deletion of the namespace.
static_volumes=()
if [[ "$delete_data" == true ]]; then
  command -v python3 >/dev/null || { echo 'error: python3 is required for PV ownership checks' >&2; exit 127; }
  for claim in mysql-data redis-data; do
    pv_name="$namespace-$claim"
    pv_json=$(kubectl --request-timeout=30s get pv "$pv_name" --ignore-not-found -o json)
    [[ -n "$pv_json" ]] || continue
    python3 -c '
import json, sys
v=json.load(sys.stdin); ns, claim=sys.argv[1:]
spec=v.get("spec", {}); ref=spec.get("claimRef", {})
valid=(v.get("metadata", {}).get("labels", {}).get("app.kubernetes.io/part-of")=="one-adventure"
       and (ref.get("namespace"), ref.get("name"))==(ns, claim)
       and spec.get("hostPath", {}).get("path")==f"/var/lib/one-adventure/{ns}/{claim}")
if not valid: sys.exit("error: refusing to delete PV with mismatched project ownership or data path")
' "$namespace" "$claim" <<<"$pv_json"
    static_volumes+=("persistentvolume/$pv_name")
  done
fi
if [[ "$dry_run" == true ]]; then
  printf 'Would delete: %s\n' "${resources[@]}"
  if [[ "$delete_data" == true ]]; then
    printf 'Would delete: %s\n' "${data_resources[@]}"
    if ((${#static_volumes[@]})); then printf 'Would delete: %s\n' "${static_volumes[@]}"; fi
    echo 'Node directories remain; retained MySQL data still needs its original password.'
  else
    echo 'Preserving namespace, PVCs and middleware-auth Secret.'
  fi
  exit 0
fi
# Deleting Deployments also garbage-collects their ReplicaSets and Pods.
kubectl -n "$namespace" delete "${resources[@]}" --ignore-not-found --wait=true --timeout=5m
if [[ "$delete_data" == true ]]; then
  echo 'Deleting MySQL/Redis PVCs and middleware-auth Secret. Stored data may be permanently lost.'
  kubectl -n "$namespace" delete "${data_resources[@]}" --ignore-not-found --wait=true --timeout=5m
  if ((${#static_volumes[@]})); then
    kubectl delete "${static_volumes[@]}" --ignore-not-found --wait=true --timeout=5m
  fi
  echo 'Cleanup complete: PVCs, project static PVs and credentials removed. Namespace and node directories preserved.'
  echo 'Retained MySQL files still use their original password; removing PV objects does not reset the database.'
else
  echo 'Cleanup complete. Namespace, PVCs and middleware-auth Secret were preserved.'
fi
