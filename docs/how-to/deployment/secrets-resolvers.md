---
title: How to resolve secrets from a custom source
description: Use the wdk/config resolver kit (or your own) to expand placeholder strings into concrete values before you pass them to Piko.
nav:
  sidebar:
    section: "how-to"
    subsection: "deployment"
    order: 25
---

# How to resolve secrets from a custom source

Piko itself does not load any configuration files or environment variables (apart from `PIKO_LOG_LEVEL` for the bootstrap logger). Every value reaches the framework through a `With*` option you call in `func main`. To keep secrets out of the binary and out of version control, expand placeholder strings yourself before passing them to the relevant option.

Piko ships an optional, user-facing utility at `piko.sh/piko/wdk/config` that handles the placeholder pattern. It includes built-in resolvers for common stores (`env:`, `file:`, `base64:`, AWS Secrets Manager, GCP Secret Manager, Azure Key Vault, Kubernetes Secrets, HashiCorp Vault). You can also write your own resolver - the `Resolver` interface has nothing to do with the framework's internals. See [configuration philosophy](../../explanation/about-configuration.md) for the bigger picture.

## The placeholder pattern

A placeholder is a string with a recognised prefix. Each prefix maps to a resolver. The built-in resolvers shipped under `wdk/config` are:

```
env:DB_PASSWORD
file:/run/secrets/db_password
base64:aGVsbG8=
vault:secret/data/app#db_password
aws-secret:prod/app-db#password
azure-kv:my-vault/db-password
gcp-secret:projects/my-project/secrets/db-password/versions/latest
kubernetes-secret:default/app-secrets#db_password
```

Your application config holds these strings. At startup you ask the resolver kit to resolve them, then pass the concrete values to Piko.

## Use the built-in `wdk/config` kit

The kit exposes a `Load` convenience function that takes a struct pointer and a `LoaderOptions` value. Pass any extra resolvers you need on `LoaderOptions.Resolvers`, then forward the resolved values to the appropriate `With*` option:

```go
package main

import (
    "context"
    "log"
    "os"

    "piko.sh/piko"
    "piko.sh/piko/wdk/config"
    "piko.sh/piko/wdk/config/config_resolver_vault"
)

type AppConfig struct {
    DatabaseURL string
    CSRFSecret  string
}

func main() {
    vaultResolver, err := config_resolver_vault.NewResolver()
    if err != nil {
        log.Fatalf("creating vault resolver: %v", err)
    }

    cfg := AppConfig{
        DatabaseURL: "vault:secret/data/app#database_url",
        CSRFSecret:  "env:CSRF_SECRET",
    }

    ctx := context.Background()
    if _, err := config.Load(ctx, &cfg, config.LoaderOptions{
        Resolvers: []config.Resolver{vaultResolver},
    }); err != nil {
        log.Fatalf("resolving config: %v", err)
    }

    ssr := piko.New(
        piko.WithPostgresURL(cfg.DatabaseURL),
        piko.WithCSRFSecret([]byte(cfg.CSRFSecret)),
    )
    if err := ssr.Run(os.Args[1]); err != nil {
        log.Fatal(err)
    }
}
```

`config.Load` automatically registers the dependency-free defaults (`env:`, `file:`, `base64:`), so the loader resolves the `env:CSRF_SECRET` placeholder above without any extra wiring. Heavier resolvers (Vault, AWS, Azure, GCP, Kubernetes) require their own client, so you must construct and pass them in explicitly.

The Vault resolver builds its own client from `VAULT_ADDR`, `VAULT_TOKEN`, and `VAULT_NAMESPACE`. Its constructor returns `(*Resolver, error)` because client construction can fail. Always handle the error.

To register resolvers once at process start, use the global registry:

```go
if err := config_resolver_vault.Register(); err != nil {
    log.Fatalf("registering vault resolver: %v", err)
}

if _, err := config.Load(ctx, &cfg, config.LoaderOptions{
    UseGlobalResolvers: true,
}); err != nil {
    log.Fatalf("resolving config: %v", err)
}
```

`config_resolver_vault.Register()` is shorthand for calling `NewResolver()` followed by `config.RegisterResolver(...)`. The other built-in resolvers (`config_resolver_aws`, `config_resolver_azure`, `config_resolver_gcp`, `config_resolver_kubernetes`) expose the same `NewResolver` / `Register` pair. The GCP and Azure variants take a `context.Context` as their first argument because they authenticate eagerly.

## Write a custom resolver

The `Resolver` interface is intentionally small:

```go
type Resolver interface {
    GetPrefix() string
    Resolve(ctx context.Context, value string) (string, error)
}
```

Implement it for any source that hosts your secrets. The example below uses the prefix `myvault:` so it does not clash with the built-in `vault:` resolver. The registry rejects duplicate prefixes:

```go
package myvault

import (
    "context"
    "fmt"
    "strings"

    vault "github.com/hashicorp/vault/api"
)

type Resolver struct {
    client *vault.Client
}

func NewResolver(client *vault.Client) *Resolver {
    return &Resolver{client: client}
}

func (r *Resolver) GetPrefix() string {
    return "myvault:"
}

func (r *Resolver) Resolve(ctx context.Context, value string) (string, error) {
    path, field, ok := strings.Cut(value, "#")
    if !ok {
        return "", fmt.Errorf("myvault: expected path#field, got %q", value)
    }
    secret, err := r.client.KVv2("secret").Get(ctx, path)
    if err != nil {
        return "", fmt.Errorf("myvault: %w", err)
    }
    raw, ok := secret.Data[field]
    if !ok {
        return "", fmt.Errorf("myvault: field %q not found at %q", field, path)
    }
    return fmt.Sprint(raw), nil
}
```

For resolvers that fetch multiple values per call (such as AWS Secrets Manager batch APIs), implement the optional batch interface:

```go
func (r *Resolver) ResolveBatch(ctx context.Context, values []string) (map[string]string, error) {
    // single API call that pulls every requested secret
    // return a map from input value to resolved value
}
```

The loader uses the batch form when multiple placeholders share the same prefix.

## Pass resolved values to Piko options

After resolution, the values are plain strings. Pipe them into the relevant `With*` option:

```go
ssr := piko.New(
    piko.WithPostgresURL(cfg.DatabaseURL),
    piko.WithCSRFSecret([]byte(cfg.CSRFSecret)),
    piko.WithStoragePresign(piko.StoragePresignConfig{
        Secret: cfg.PresignSecret,
    }),
)
```

You can also use Piko's `Secret[T]` primitive when you want to delay materialising the value until just before use. See the [secrets API reference](../../reference/secrets-api.md) and the [secrets how-to](../secrets.md).

## Custom AWS Secrets Manager resolver example

Piko already ships an AWS Secrets Manager resolver under the `aws-secret:` prefix in `piko.sh/piko/wdk/config/config_resolver_aws`. The example below shows how you would write your own from scratch, say, to inject a pre-configured client or a different key-extraction policy. It uses the prefix `aws-sm:` to avoid colliding with the built-in resolver:

```go
package awssecretsmanager

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"

    "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type Resolver struct {
    client *secretsmanager.Client
}

func NewResolver(client *secretsmanager.Client) *Resolver {
    return &Resolver{client: client}
}

func (r *Resolver) GetPrefix() string { return "aws-sm:" }

func (r *Resolver) Resolve(ctx context.Context, value string) (string, error) {
    secretID, field, hasField := strings.Cut(value, "#")

    out, err := r.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
        SecretId: &secretID,
    })
    if err != nil {
        return "", fmt.Errorf("aws-sm: %w", err)
    }
    if out.SecretString == nil {
        return "", fmt.Errorf("aws-sm: %q has no string value", secretID)
    }
    if !hasField {
        return *out.SecretString, nil
    }
    var parsed map[string]string
    if err := json.Unmarshal([]byte(*out.SecretString), &parsed); err != nil {
        return "", fmt.Errorf("aws-sm: %q is not a JSON object", secretID)
    }
    raw, ok := parsed[field]
    if !ok {
        return "", fmt.Errorf("aws-sm: field %q not found in %q", field, secretID)
    }
    return raw, nil
}
```

Reference the custom resolver with `aws-sm:prod/app-db#password`. The built-in resolver covers most cases via `aws-secret:prod/app-db#password`.

## Caching

The loader caches resolved values for the duration of one resolution pass. Two placeholders with the same value share one API call. Resolvers themselves do not cache across passes. If you reload, every placeholder is re-fetched.

Custom resolvers that talk to slow back-ends should cache internally if reload frequency makes the memory trade-off worthwhile.

## Error handling

If a resolver returns an error, your `Resolve` call fails and `func main` should bail out before calling `piko.New`. A missing secret means the server would run misconfigured. Treat resolver errors as fatal.

For optional values (secrets that default to empty), let the resolver return the empty string and validate the field yourself.

## Test the resolver

Table-driven tests with a fake client:

```go
func TestVaultResolver_Resolve(t *testing.T) {
    client := &fakeVaultClient{
        secrets: map[string]map[string]any{
            "secret/data/app": {"db_password": "hunter2"},
        },
    }
    r := myvault.NewResolver(client)

    got, err := r.Resolve(context.Background(), "secret/data/app#db_password")
    if err != nil {
        t.Fatalf("resolve: %v", err)
    }
    if got != "hunter2" {
        t.Fatalf("got %q, want %q", got, "hunter2")
    }
}
```

## See also

- [Configuration philosophy](../../explanation/about-configuration.md) for why Piko stays out of the loader business.
- [How to manage secrets](../secrets.md) for using `Secret[T]` with resolvers at request time.
- [Bootstrap options reference](../../reference/bootstrap-options.md) for the full surface of `With*` options the resolved values can flow into.
- [Secrets API reference](../../reference/secrets-api.md) for Piko's typed `Secret[T]` primitives.
