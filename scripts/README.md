# Universal GitHub Go K8s Deployment Scripts

These scripts enable **one-command, zero-interruption deployment** for any GitHub Go project to Kubernetes.

## Quick Start

```bash
# 1. Clone this repository or copy scripts to your project
cp -r scripts/ /path/to/your/go-project/

# 2. Run one-command deployment
cd /path/to/your/go-project
./scripts/deploy.sh \
  --repo OWNER/REPO \
  --namespace your-namespace \
  --domain your-domain.example.com \
  --kubeconfig /path/to/kubeconfig \
  --token github_pat_...
```

## What It Does

1. **Preflight Check**: Validates cluster, collects all required info
2. **Generate Manifests**: Creates K8s deployment, service, ingress
3. **Generate CI**: Creates GitHub Actions workflow
4. **Setup GitHub**: Sets all secrets and variables
5. **Push**: Commits and pushes, triggering CI/CD

## Scripts Overview

| Script | Purpose | Input | Output |
|--------|---------|-------|--------|
| `preflight.sh` | Collect cluster info, validate project | CLI args | JSON config |
| `generate_manifests.sh` | Generate K8s YAML files | JSON config | k8s/app/*.yaml |
| `generate_ci.sh` | Generate GitHub Actions workflow | JSON config | .github/workflows/ci.yml |
| `setup_github.py` | Set GitHub secrets/variables | JSON config | Configured repo |
| `deploy.sh` | Orchestrate all steps | CLI args | Deployed app |

## Required User Input

Only these 5 items are required:

```yaml
repo: OWNER/REPO           # GitHub repository
namespace: your-namespace  # K8s namespace
domain: app.example.com    # Application domain
kubeconfig: /path/to/kubeconfig  # K8s config
token: github_pat_...      # GitHub token with actions:write
```

## Optional Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `--registry` | `ghcr.io` | Container registry |
| `--deploy-strategy` | `apply` | `apply` or `replace` |
| `--skip-smoke-test` | `false` | Skip smoke test in CI |

## Features

- **Zero Interruption**: One command, no manual steps
- **Auto-Detection**: Automatically detects Traefik IP, TLS secrets, ingress patterns
- **Validation**: Pre-validates Go code, Dockerfile, K8s manifests
- **Batch Operations**: All GitHub secrets/variables set in one batch
- **Error Prevention**: All known pitfalls from skill are pre-handled

## Examples

### Basic Deployment

```bash
./scripts/deploy.sh \
  --repo fys05/go-test1 \
  --namespace abm-production \
  --domain omp-go-test1.steedgrace.com \
  --kubeconfig /tmp/kubeconfig.yaml \
  --token github_pat_...
```

### Custom Registry

```bash
./scripts/deploy.sh \
  --repo fys05/go-test1 \
  --namespace abm-production \
  --domain omp-go-test1.steedgrace.com \
  --kubeconfig /tmp/kubeconfig.yaml \
  --token github_pat_... \
  --registry harbor.example.com
```

### Skip Smoke Test

```bash
./scripts/deploy.sh \
  --repo fys05/go-test1 \
  --namespace abm-production \
  --domain omp-go-test1.steedgrace.com \
  --kubeconfig /tmp/kubeconfig.yaml \
  --token github_pat_... \
  --skip-smoke-test
```

### Force Replace Strategy

```bash
./scripts/deploy.sh \
  --repo fys05/go-test1 \
  --namespace abm-production \
  --domain omp-go-test1.steedgrace.com \
  --kubeconfig /tmp/kubeconfig.yaml \
  --token github_pat_... \
  --deploy-strategy replace
```

## Troubleshooting

### Common Issues

| Issue | Solution |
|-------|----------|
| `kubectl not found` | Install kubectl or use Docker: `docker run --rm -v $(pwd):/work -w /work bitnami/kubectl` |
| `docker not found` | Install Docker or use rootless mode |
| `go.mod not found` | Run in Go project root directory |
| `Tests failed` | Fix Go tests before deployment |
| `Namespace not found` | Create namespace: `kubectl create ns $NAMESPACE` |

### Debug Mode

```bash
# Run with debug output
bash -x ./scripts/deploy.sh ...
```

## Integration with Existing Projects

If you already have K8s manifests or CI configuration:

```bash
# Only run preflight and generate config
./scripts/preflight.sh \
  --repo OWNER/REPO \
  --namespace NS \
  --domain DOMAIN \
  --kubeconfig PATH > config.json

# Use generated config with your own templates
```

## License

MIT
