# Home Pager

A minimal Kubernetes application dashboard that displays apps from Ingress annotations.

## Features

- Parse Kubernetes Ingress YAML to display application links
- Support for internal/external network modes
- Annotations-based configuration (`homepage.link/*`)
- Minimal, secure container (~5MB scratch-based image)
- Health and readiness endpoints (`/healthz`, `/readyz`)
- Prometheus-style metrics endpoint (`/metrics`)

## Container Image

```bash
docker pull ghcr.io/damacus/home-pager:latest
```

### Security

- **Scratch-based image** - No shell, no package manager, minimal attack surface
- **Non-root user** - Runs as UID 65534 (nobody)
- **Read-only filesystem** - No writable paths required
- **No capabilities** - All Linux capabilities dropped

## Helm Chart

### Installation

```bash
# Add the OCI registry
helm install home-pager oci://ghcr.io/damacus/charts/home-pager

# Or with custom values
helm install home-pager oci://ghcr.io/damacus/charts/home-pager \
  --set app-template.ingress.main.enabled=true \
  --set app-template.ingress.main.hosts[0].host=dashboard.example.com
```

### Configuration

The chart uses [bjw-s app-template](https://bjw-s-labs.github.io/helm-charts/docs/app-template/) as a dependency.

Key values:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `app-template.controllers.main.containers.main.image.repository` | Image repository | `ghcr.io/damacus/home-pager` |
| `app-template.controllers.main.containers.main.image.tag` | Image tag | `0.1.0` |
| `app-template.ingress.main.enabled` | Enable ingress | `false` |
| `app-template.ingress.main.className` | Ingress class | `""` |
| `app-template.ingress.main.hosts` | Ingress hosts | `[]` |

## Ingress Annotations

Configure your Kubernetes Ingresses with these annotations to appear in the dashboard:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-app
  annotations:
    homepage.link/enabled: "true"
    homepage.link/name: "My Application"
    homepage.link/icon: "🚀"
    homepage.link/description: "Application description"
    homepage.link/internal-host: "app.internal.local"
    homepage.link/external-host: "app.example.com"
```

## Development

### Environment

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP listen port | `8080` |
| `KUBERNETES_TIMEOUT` | Kubernetes API timeout (e.g. `10s` or seconds) | `10s` |

### Build locally

```bash
docker build -t home-pager:local .
docker run -p 8080:8080 home-pager:local
```

### Local Go commands

```bash
mise trust
mise install
cd server
mise exec -- go test ./...
mise exec -- go vet ./...
```

### Helm chart development

```bash
cd charts/home-pager
helm dependency update
helm template . --debug
```

## License

MIT
