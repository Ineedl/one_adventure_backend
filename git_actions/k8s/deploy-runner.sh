#!/usr/bin/env bash
set -euo pipefail

# GitHub Actions entry point. This file is kept separate from deploy.sh so the
# original manual deployment script remains unchanged.
workspace=${GITHUB_WORKSPACE:-$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)}
export GITHUB_WORKSPACE="$workspace"
exec bash "$workspace/git_actions/k8s/deploy-actions.sh" "$@"
