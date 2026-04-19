---
title: How to configure environments and secrets
description: Use cascading YAML files, environment variables, .env files, and secret resolvers to configure a Piko application per environment.
nav:
  sidebar:
    section: "how-to"
    subsection: "deployment"
    order: 20
---

# How to configure environments and secrets

Piko loads configuration from layered sources so the same binary can run in development, staging, and production without per-environment builds. This guide covers the layering rules, `.env` parsing, and the built-in secret resolvers for AWS, GCP, Azure, HashiCorp Vault, and Kubernetes. See the [`config.json` reference](../../reference/config-json-schema.md) for every field and the [secrets API reference](../../reference/secrets-api.md) for the Go side.

## Configuration file layout

Piko looks for up to four YAML files at the project root:

```
piko.yaml           # base; committed
piko-{env}.yaml     # environment override; committed
piko.local.yaml     # personal override; gitignored
.env                # dotenv file; gitignored
```

`{env}` is the current environment name, set by the `--env` flag or the `PIKO_ENV` environment variable. It defaults to `dev`. Piko skips files that do not exist.

## Loading order

Configuration resolves in this order, with later sources overriding earlier ones:

1. Programmatic defaults (any `piko.WithX` calls passed to `piko.New`).
2. Struct-tag defaults baked into Piko.
3. `piko.yaml` (base file).
4. `piko-{env}.yaml` (environment file).
5. `piko.local.yaml` (local overrides).
6. `.env` file.
7. Environment variables prefixed with `PIKO_`.
8. Command-line flags.
9. Secret resolvers (applied last, resolve `aws-secret:`, `gcp-secret:`, and similar placeholders).

Any field that you set programmatically through `piko.With*` options wins over every other source. The order matches [`internal/config/doc.go`](https://github.com/piko-sh/piko/blob/master/internal/config/doc.go).

## Per-environment overrides

**`piko-dev.yaml`**:

```yaml
network:
  autoNextPort: true
build:
  watchMode: true
logger:
  level: debug
```

**`piko-prod.yaml`**:

```yaml
network:
  publicDomain: myapp.example.com
  forceHttps: true
  autoNextPort: false
build:
  watchMode: false
logger:
  level: info
```

**`piko.local.yaml`** (each developer's personal overrides):

```yaml
network:
  port: "3001"
logger:
  level: trace
```

Select the environment at runtime:

```bash
./app prod --env=staging
# or
PIKO_ENV=staging ./app prod
```

## Environment variables

Every YAML field has a matching environment variable with the `PIKO_` prefix and snake-case capitalisation. Common ones:

| Variable | Overrides |
|---|---|
| `PIKO_PORT` | `network.port` |
| `PIKO_AUTO_PORT` | `network.autoNextPort` |
| `PIKO_FORCE_HTTPS` | `network.forceHttps` |
| `PIKO_ENV` | Environment name. |
| `PIKO_LOG_LEVEL` | `logger.level` (also accepts numeric values). |
| `PIKO_HEALTH_PROBE_PORT` | `healthProbe.port` |
| `PIKO_HEALTH_PROBE_BIND_ADDRESS` | `healthProbe.bindAddress` |
| `PIKO_DATABASE_DRIVER` | `database.driver` |
| `PIKO_DATABASE_POSTGRES_URL` | `database.postgres.url` |

Set them on the shell, in a container environment file, or through your orchestrator.

## `.env` file

Piko parses a `.env` file in the working directory at startup. You do not need to source it manually.

```bash
PIKO_PORT=3000
DATABASE_URL=postgres://dev:dev@localhost/myapp_dev
SMTP_PASSWORD=dev_password

# Variable expansion is supported
API_BASE=https://api.example.com
API_USERS=${API_BASE}/users

# Quoting rules: double quotes expand, single quotes do not
MESSAGE="Line 1\nLine 2"
LITERAL='$NOT_EXPANDED'

# Inline comment on unquoted values
PORT=8080 # this is ignored
```

- `${VAR}` always expands in double-quoted and unquoted values.
- `$VAR` expands only for multi-character names and recognised shell names (`PATH`, `HOME`, `USER`, `LANG`, `TERM`).
- Undefined variables expand to the empty string.
- Lines beginning with `#` are comments. Inline comments only work on unquoted values.
- `export VAR=value` works too, for shell compatibility.

Add `.env` to `.gitignore`. Commit a `.env.example` that lists the expected variables with placeholder values.

## Secret resolvers

Secret resolvers fetch values from external services at configuration-load time. Register each resolver you want to use, then reference the secret by prefix in any YAML field or environment variable.

Every resolver lives in its own sub-package and exposes a `Register()` function. Import and call it from your `cmd/main/main.go` before `piko.New(...)`.

### AWS Secrets Manager

```go
import "piko.sh/piko/wdk/config/config_resolver_aws"

func main() {
    ctx := context.Background()
    config_resolver_aws.Register(ctx)
    // ...
}
```

```yaml
database:
  postgres:
    url: "aws-secret:myapp/database-url"
security:
  encryptionKey: "aws-secret:myapp/secrets#encryption_key"
```

The `#key` suffix extracts one field from a JSON secret. The resolver uses the default AWS credential chain. Grant the role `secretsmanager:GetSecretValue`.
### Google Cloud Secret Manager
```go
import "piko.sh/piko/wdk/config/config_resolver_gcp"

config_resolver_gcp.Register(ctx)
```

```yaml
database:
  postgres:
    url: "gcp-secret:projects/my-project/secrets/database-url/versions/latest"
```
The value must be the full resource name of the secret version. The resolver authenticates through the [Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials) chain. The service account needs the `secretmanager.secretAccessor` role.
### Azure Key Vault
```go
import "piko.sh/piko/wdk/config/config_resolver_azure"

config_resolver_azure.Register()
```

```yaml
database:
  postgres:
    url: "azure-kv:myapp-vault/database-url"
```

Format: `azure-kv:vault-name/secret-name[#json-key]`. Uses Azure SDK's `DefaultAzureCredential` chain (Managed Identity, CLI, environment).

### HashiCorp Vault

```go
import "piko.sh/piko/wdk/config/config_resolver_vault"

config_resolver_vault.Register()
```

```yaml
database:
  postgres:
    url: "vault:secret/myapp#database_url"
```

Format: `vault:mount/path#key`. Designed for Vault's KV v2 engine. Do not include the `data/` prefix, because the resolver adds it. Configure with `VAULT_ADDR`, `VAULT_TOKEN`, and optionally `VAULT_NAMESPACE`.

### Kubernetes secrets

```go
import "piko.sh/piko/wdk/config/config_resolver_kubernetes"

config_resolver_kubernetes.Register()
```

```yaml
database:
  postgres:
    url: "kubernetes-secret:myapp-secrets#DATABASE_URL"
security:
  encryptionKey: "kubernetes-secret:production/app-secrets#ENCRYPTION_KEY"
```

Format: `kubernetes-secret:[namespace/]secret-name#key`. When running in-cluster, the namespace defaults to the pod's namespace. Out-of-cluster, the resolver reads `~/.kube/config` or the `KUBECONFIG` environment variable.

### Built-in placeholder prefixes

Three placeholders do not require a resolver registration:

| Prefix | Behaviour |
|---|---|
| `env:NAME` | Reads from the environment variable `NAME`. |
| `file:/path/to/file` | Reads the file contents (trim trailing newline). Use for Docker secrets. |
| `base64:...` | Decodes a base64 string. |

## Command-line flags

Flags win over every file source. They are useful for CI overrides and quick local tweaks.

```bash
./app prod --port=9000 --forceHttps=true --env=staging
```

Supported forms: `--flag=value` and `--flag value`. Common flags:

| Flag | Field |
|---|---|
| `--env` | Current environment name. |
| `--port` | `network.port` |
| `--baseDir` | Project root. |
| `--watch` | `build.watchMode` |
| `--forceHttps` | `network.forceHttps` |
| `--configFile` | Base config file (default `piko.yaml`). |

## What to exclude from version control

```gitignore
.env
.env.*
!.env.example
piko.local.yaml
*.pem
*.key
```

Commit `.env.example` with variable names and placeholder values so teammates know which variables to set.

## See also

- [How to production build](production-build.md).
- [How to TLS](tls.md).
- [Secrets API reference](../../reference/secrets-api.md).
- [`config.json` reference](../../reference/config-json-schema.md).
- [Bootstrap options reference](../../reference/bootstrap-options.md).
