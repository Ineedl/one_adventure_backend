#!/usr/bin/env bash
set -euo pipefail

namespace=${KUBE_NAMESPACE:-one-adventure}
dry_run=false
while (($#)); do
  case "$1" in
    --dry-run) dry_run=true ;;
    --namespace)
      (($# >= 2)) || { echo 'error: --namespace requires a value' >&2; exit 2; }
      namespace=$2; shift ;;
    -h|--help)
      echo "Usage: $0 [--dry-run] [--namespace NAMESPACE]"
      echo 'Rolls Envoy to clear in-memory dynamic CDS/EDS state.'
      echo 'Preserves the Deployment, Service, envoy-config ConfigMap, and service registrations.'
      exit 0 ;;
    *) echo "error: unknown option: $1" >&2; exit 2 ;;
  esac
  shift
done
[[ "$namespace" =~ ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$ && ${#namespace} -le 63 ]] || {
  echo "error: invalid namespace: $namespace" >&2; exit 2;
}
command -v kubectl >/dev/null 2>&1 || { echo 'error: kubectl is required' >&2; exit 127; }
kubectl --request-timeout=30s get namespace "$namespace" >/dev/null
owner=$(kubectl --request-timeout=30s get namespace "$namespace" -o 'jsonpath={.metadata.labels.app\\.kubernetes\\.io/part-of}')
[[ "$owner" == one-adventure ]] || { echo 'error: namespace is not labeled as one-adventure' >&2; exit 2; }
kubectl --request-timeout=30s -n "$namespace" get deployment/envoy >/dev/null
if [[ "$dry_run" == true ]]; then
  echo "Would restart deployment/envoy in namespace $namespace."
  echo 'The Envoy pod will be recreated; envoy-config and service registrations are preserved.'
  exit 0
fi
kubectl --request-timeout=30s -n "$namespace" rollout restart deployment/envoy
kubectl --request-timeout=30s -n "$namespace" rollout status deployment/envoy --timeout=5m
echo 'Envoy restarted; waiting for xDS to repopulate dynamic configuration.'
