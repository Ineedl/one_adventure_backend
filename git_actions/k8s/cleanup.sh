#!/usr/bin/env bash
set -euo pipefail

namespace=${KUBE_NAMESPACE:-one-adventure}
dry_run=false
positional=()
while (($#)); do
  case "$1" in
    --dry-run) dry_run=true ;;
    -h|--help)
      echo "Usage: $0 [--dry-run] [NAMESPACE]"
      echo 'Removes all project application/middleware Deployments, Services and configuration.'
      echo 'Preserves the namespace, PVCs and middleware-auth Secret for redeployment.'
      echo 'NAMESPACE defaults to KUBE_NAMESPACE or one-adventure. No confirmation prompt.'
      echo '--dry-run previews the exact resources without deleting them.'
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
if [[ -z "$existing" ]]; then
  echo "Namespace does not exist; nothing to clean up."
  exit 0
fi
owner=$(kubectl get namespace "$namespace" -o 'jsonpath={.metadata.labels.app\.kubernetes\.io/part-of}')
if [[ "$owner" != one-adventure ]]; then
  echo "error: namespace must have label app.kubernetes.io/part-of=one-adventure" >&2
  exit 2
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
if [[ "$dry_run" == true ]]; then
  printf 'Would delete: %s\n' "${resources[@]}"
  echo 'Preserving namespace, PVCs and middleware-auth Secret.'
  exit 0
fi
# Deleting Deployments also garbage-collects their ReplicaSets and Pods.
kubectl -n "$namespace" delete "${resources[@]}" --ignore-not-found --wait=true --timeout=5m
echo 'Cleanup complete. Namespace, PVCs and middleware-auth Secret were preserved.'
