#!/bin/bash
# deploy.sh - Universal one-command deployment for GitHub Go K8s projects
# Usage: ./deploy.sh --repo OWNER/REPO --namespace NS --domain DOMAIN --kubeconfig PATH

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[DEPLOY]${NC} $1"; }
success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --repo) REPO="$2"; shift 2 ;;
        --namespace) NAMESPACE="$2"; shift 2 ;;
        --domain) DOMAIN="$2"; shift 2 ;;
        --kubeconfig) KUBECONFIG="$2"; shift 2 ;;
        --registry) REGISTRY="$2"; shift 2 ;;
        --deploy-strategy) DEPLOY_STRATEGY="$2"; shift 2 ;;
        --skip-smoke-test) SKIP_SMOKE_TEST=true; shift ;;
        --token) GITHUB_TOKEN="$2"; shift 2 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# Validate required args
for arg in REPO NAMESPACE DOMAIN KUBECONFIG GITHUB_TOKEN; do
    if [[ -z "${!arg:-}" ]]; then
        echo "Error: --${arg,,} is required"
        exit 1
    fi
done

export KUBECONFIG GITHUB_TOKEN

log "Starting one-command deployment for $REPO"

# Step 1: Run preflight checks
log "Step 1/5: Running preflight checks..."
"$SCRIPT_DIR/preflight.sh" \
    --repo "$REPO" \
    --namespace "$NAMESPACE" \
    --domain "$DOMAIN" \
    --kubeconfig "$KUBECONFIG" \
    ${REGISTRY:+--registry "$REGISTRY"} \
    ${DEPLOY_STRATEGY:+--deploy-strategy "$DEPLOY_STRATEGY"} \
    ${SKIP_SMOKE_TEST:+--skip-smoke-test} \
    > /tmp/preflight-config.json

if [[ ! -f /tmp/preflight-config.json ]]; then
    echo "Error: Preflight check failed"
    exit 1
fi

success "Preflight checks passed"

# Step 2: Generate K8s manifests
log "Step 2/5: Generating K8s manifests..."
"$SCRIPT_DIR/generate_manifests.sh" /tmp/preflight-config.json

success "K8s manifests generated"

# Step 3: Generate CI/CD workflow
log "Step 3/5: Generating CI/CD workflow..."
"$SCRIPT_DIR/generate_ci.sh" /tmp/preflight-config.json

success "CI/CD workflow generated"

# Step 4: Setup GitHub secrets/variables
log "Step 4/5: Setting up GitHub..."
export CONFIG_FILE=/tmp/preflight-config.json
"$SCRIPT_DIR/setup_github.py" "$REPO"

success "GitHub setup complete"

# Step 5: Commit and push
log "Step 5/5: Committing and pushing..."
git add -A
git commit -m "feat: automated CI/CD setup for $DOMAIN" || true
git push origin main

success "Deployment triggered! CI/CD pipeline running."
log "Monitor at: https://github.com/$REPO/actions"
