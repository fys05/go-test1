#!/bin/bash
# preflight.sh - Universal preflight checker for GitHub Go K8s deployments
# Collects all required information in one pass

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Default values
DEFAULT_REGISTRY="ghcr.io"
DEFAULT_DEPLOY_STRATEGY="apply"

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --repo) REPO="$2"; shift 2 ;;
            --namespace) NAMESPACE="$2"; shift 2 ;;
            --domain) DOMAIN="$2"; shift 2 ;;
            --kubeconfig) KUBECONFIG="$2"; shift 2 ;;
            --registry) REGISTRY="$2"; shift 2 ;;
            --deploy-strategy) DEPLOY_STRATEGY="$2"; shift 2 ;;
            --skip-smoke-test) SKIP_SMOKE_TEST=true; shift ;;
            --help) show_help; exit 0 ;;
            *) log_error "Unknown option: $1"; show_help; exit 1 ;;
        esac
    done
    
    # Set defaults
    REGISTRY=${REGISTRY:-$DEFAULT_REGISTRY}
    DEPLOY_STRATEGY=${DEPLOY_STRATEGY:-$DEFAULT_DEPLOY_STRATEGY}
    SKIP_SMOKE_TEST=${SKIP_SMOKE_TEST:-false}
    
    # Validate required args
    local missing=()
    [[ -z "${REPO:-}" ]] && missing+=("--repo")
    [[ -z "${NAMESPACE:-}" ]] && missing+=("--namespace")
    [[ -z "${DOMAIN:-}" ]] && missing+=("--domain")
    [[ -z "${KUBECONFIG:-}" ]] && missing+=("--kubeconfig")
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "Missing required arguments: ${missing[*]}"
        show_help
        exit 1
    fi
}

show_help() {
    cat << EOF
Usage: $0 [OPTIONS]

Universal preflight checker for GitHub Go K8s deployments.

Required:
  --repo OWNER/REPO          GitHub repository (e.g., fys05/go-test1)
  --namespace NS             Kubernetes namespace
  --domain DOMAIN            Application domain
  --kubeconfig PATH          Path to kubeconfig file

Optional:
  --registry REGISTRY        Container registry (default: ghcr.io)
  --deploy-strategy STRATEGY Deploy strategy: apply|replace (default: apply)
  --skip-smoke-test          Skip smoke test in CI
  --help                     Show this help

Output: JSON with all collected configuration
EOF
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    local tools=("kubectl" "docker" "git" "curl" "jq" "openssl")
    for tool in "${tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "$tool is required but not installed"
            exit 1
        fi
    done
    
    log_info "All prerequisites satisfied"
}

# Validate kubeconfig and cluster connection
validate_cluster() {
    log_info "Validating cluster connection..."
    
    export KUBECONFIG="$KUBECONFIG"
    
    # Check if kubeconfig exists
    if [[ ! -f "$KUBECONFIG" ]]; then
        log_error "Kubeconfig not found: $KUBECONFIG"
        exit 1
    fi
    
    # Test cluster connection
    if ! kubectl get nodes &> /dev/null; then
        log_error "Cannot connect to cluster. Check kubeconfig and network."
        exit 1
    fi
    
    # Check namespace
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_error "Namespace '$NAMESPACE' not found"
        exit 1
    fi
    
    log_info "Cluster connection validated"
}

# Extract cluster information
extract_cluster_info() {
    log_info "Extracting cluster information..."
    
    # API Server
    API_SERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
    
    # CA Certificate SHA256
    CA_DATA=$(kubectl config view --raw --minify -o jsonpath='{.clusters[0].cluster.certificate-authority-data}')
    if [[ -n "$CA_DATA" ]]; then
        CA_SHA256=$(echo "$CA_DATA" | base64 -d | openssl x509 -outform DER 2>/dev/null | sha256sum | cut -d' ' -f1)
    else
        CA_FILE=$(kubectl config view --raw --minify -o jsonpath='{.clusters[0].cluster.certificate-authority}')
        if [[ -n "$CA_FILE" && -f "$CA_FILE" ]]; then
            CA_SHA256=$(openssl x509 -in "$CA_FILE" -outform DER 2>/dev/null | sha256sum | cut -d' ' -f1)
        else
            log_error "Cannot extract CA certificate"
            exit 1
        fi
    fi
    
    # Traefik LB IP
    TRAEFIK_IP=$(kubectl get svc traefik -n kube-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
    if [[ -z "$TRAEFIK_IP" ]]; then
        # Try alternative ingress controllers
        TRAEFIK_IP=$(kubectl get svc -n kube-system -o jsonpath='{.items[?(@.spec.type=="LoadBalancer")].status.loadBalancer.ingress[0].ip}' 2>/dev/null | head -1 || echo "")
    fi
    
    # TLS Secret from existing ingress
    TLS_SECRET=$(kubectl get ingress -n "$NAMESPACE" -o jsonpath='{.items[0].spec.tls[0].secretName}' 2>/dev/null || echo "")
    if [[ -z "$TLS_SECRET" ]]; then
        # Try to find any TLS secret in namespace
        TLS_SECRET=$(kubectl get secrets -n "$NAMESPACE" --field-selector type=kubernetes.io/tls -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    fi
    
    # Image pull secret
    IMAGE_PULL_SECRET=$(kubectl get secrets -n "$NAMESPACE" --field-selector type=kubernetes.io/dockerconfigjson -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    
    # Ingress annotations pattern
    INGRESS_ANNOTATIONS=$(kubectl get ingress -n "$NAMESPACE" -o jsonpath='{.items[0].metadata.annotations}' 2>/dev/null | jq -c '.' || echo "{}")
    
    log_info "Cluster information extracted"
}

# Validate GitHub repository
validate_github() {
    log_info "Validating GitHub repository..."
    
    # Check if repo is accessible
    if ! git ls-remote "https://github.com/${REPO}.git" &> /dev/null; then
        log_error "Cannot access repository: $REPO"
        exit 1
    fi
    
    log_info "GitHub repository validated"
}

# Validate Go project
validate_go_project() {
    log_info "Validating Go project..."
    
    # Check if it's a Go project
    if [[ ! -f "go.mod" ]]; then
        log_error "go.mod not found. Not a Go project."
        exit 1
    fi
    
    # Check Go version
    GO_VERSION=$(grep -oP '^go \K[0-9.]+' go.mod || echo "1.22")
    log_info "Detected Go version: $GO_VERSION"
    
    # Validate go.mod
    if ! docker run --rm -v "$(pwd):/app" -w /app "golang:${GO_VERSION}-alpine" go mod verify &> /dev/null; then
        log_warn "go mod verify failed, running go mod tidy..."
        docker run --rm -v "$(pwd):/app" -w /app "golang:${GO_VERSION}-alpine" go mod tidy
    fi
    
    # Check gofmt
    local unformatted
    unformatted=$(docker run --rm -v "$(pwd):/app" -w /app "golang:${GO_VERSION}-alpine" gofmt -l . 2>/dev/null || echo "")
    if [[ -n "$unformatted" ]]; then
        log_warn "Unformatted files detected, running gofmt..."
        docker run --rm -v "$(pwd):/app" -w /app "golang:${GO_VERSION}-alpine" gofmt -w .
    fi
    
    # Run tests
    log_info "Running tests..."
    if ! docker run --rm -v "$(pwd):/app" -w /app "golang:${GO_VERSION}-alpine" go test ./... &> /dev/null; then
        log_error "Tests failed"
        exit 1
    fi
    
    log_info "Go project validated"
}

# Generate configuration
generate_config() {
    log_info "Generating configuration..."
    
    # Extract repo name
    REPO_NAME=$(basename "$REPO")
    APP_NAME=$(echo "$REPO_NAME" | tr '[:upper:]' '[:lower:]')
    
    # Build image name
    IMAGE_NAME="${REGISTRY}/${REPO}"
    
    # Output JSON configuration
    cat << EOF
{
  "repo": "$REPO",
  "repo_name": "$REPO_NAME",
  "app_name": "$APP_NAME",
  "namespace": "$NAMESPACE",
  "domain": "$DOMAIN",
  "registry": "$REGISTRY",
  "image_name": "$IMAGE_NAME",
  "deploy_strategy": "$DEPLOY_STRATEGY",
  "skip_smoke_test": $SKIP_SMOKE_TEST,
  "cluster": {
    "api_server": "$API_SERVER",
    "ca_sha256": "$CA_SHA256",
    "traefik_ip": "$TRAEFIK_IP",
    "tls_secret": "$TLS_SECRET",
    "image_pull_secret": "$IMAGE_PULL_SECRET"
  },
  "go": {
    "version": "$GO_VERSION"
  },
  "ingress": {
    "annotations": $INGRESS_ANNOTATIONS
  }
}
EOF
}

# Main execution
main() {
    parse_args "$@"
    check_prerequisites
    validate_cluster
    extract_cluster_info
    validate_github
    validate_go_project
    generate_config
}

main "$@"
