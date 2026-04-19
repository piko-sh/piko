---
title: How to resolve secrets from a custom source
description: Wire a Vault, AWS Secrets Manager, or custom resolver into Piko's config precedence chain.
nav:
  sidebar:
    section: "how-to"
    subsection: "deployment"
    order: 25
---

# How to resolve secrets from a custom source

Piko config values can reference secrets with a prefix syntax: `env:DB_PASSWORD`, `file:/run/secrets/db_password`, `base64:...`. A custom resolver extends this list with a new prefix, such as `vault:secret/data/app#db_password` or `aws-sm:/prod/app/db`. The resolver runs during the config-load pass, after the loader applies environment and flags but before programmatic overrides. For the full precedence order see [about config](../../explanation/about-config.md).

## Understand where resolvers fit

`ServerConfig` values may contain a placeholder string. When the loader reaches the resolver pass, it walks every field, strips the prefix, and calls the matching resolver. Built-in resolvers include `env:`, `file:`, and `base64:`. Custom resolvers plug in alongside.

Precedence order:

1. Struct-tag defaults.
2. Config files (`config.json`, `config.yaml`).
3. `.env` file.
4. Environment variables (`PIKO_*`).
5. CLI flags.
6. **Resolvers** (the placeholder substitution pass that your custom resolver joins).
7. Programmatic overrides (`WithServerConfigDefaults`).
8. Validation.

A resolver never overrides a concrete value set earlier. It only fires when a field's value is a placeholder string with a recognised prefix.

## Implement the resolver interface

```go
package vault

import (
    "context"
    "fmt"

    vault "github.com/hashicorp/vault/api"
)

type Resolver struct {
    client *vault.Client
}

func NewResolver(client *vault.Client) *Resolver {
    return &Resolver{client: client}
}

// GetPrefix returns the placeholder prefix this resolver responds to.
func (r *Resolver) GetPrefix() string {
    return "vault:"
}

// Resolve converts a Vault path-and-field pair into the stored value.
//
// Input format: "secret/data/app#db_password".
func (r *Resolver) Resolve(ctx context.Context, value string) (string, error) {
    path, field, ok := strings.Cut(value, "#")
    if !ok {
        return "", fmt.Errorf("vault: expected path#field, got %q", value)
    }

    secret, err := r.client.KVv2("secret").Get(ctx, path)
    if err != nil {
        return "", fmt.Errorf("vault: %w", err)
    }

    raw, ok := secret.Data[field]
    if !ok {
        return "", fmt.Errorf("vault: field %q not found at %q", field, path)
    }

    return fmt.Sprint(raw), nil
}
```

For resolvers that fetch multiple values in one call (such as AWS Secrets Manager's batch API), implement `BatchResolver` additionally:

```go
func (r *Resolver) ResolveBatch(ctx context.Context, values []string) (map[string]string, error) {
    // Single API call that pulls every requested secret.
    // Return a map from input value to resolved value.
}
```

Piko uses the batch form when multiple placeholders share the same prefix.

## Wire the resolver at bootstrap

```go
import (
    "piko.sh/piko"
    "myapp/resolvers/vault"
)

vaultClient, err := newVaultClient()
if err != nil {
    log.Fatal(err)
}

resolver := vault.NewResolver(vaultClient)

ssr := piko.New(
    piko.WithConfigResolver(resolver),
)
```

Register multiple resolvers in one bootstrap call. Each responds to its own prefix.

## Reference the resolver in config

With the resolver wired, use its prefix in any config source:

```json
{
  "database": {
    "password": "vault:secret/data/app#db_password"
  },
  "captcha": {
    "turnstile_secret": "vault:secret/data/captcha#turnstile"
  }
}
```

Or through environment variables:

```bash
PIKO_DATABASE_PASSWORD="vault:secret/data/app#db_password"
```

The loader resolves the placeholder during the resolver pass.

## Write an AWS Secrets Manager resolver

```go
package awssecretsmanager

import (
    "context"
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

func (r *Resolver) GetPrefix() string {
    return "aws-sm:"
}

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

Wire it the same way:

```go
ssr := piko.New(
    piko.WithConfigResolver(awssecretsmanager.NewResolver(smClient)),
)
```

Reference it with `aws-sm:prod/app-db#password`.

## Cache resolved values

The loader caches resolved values for the duration of the config-load pass. Within a single load, two placeholders with the same value resolve to the same string through one API call. Resolvers do not cache across loads. A future config reload re-fetches every placeholder.

Custom resolvers that talk to a slow backend should cache internally if reload frequency justifies the memory trade.

## Error handling

If a resolver returns an error, the config load fails and the server does not start. This is deliberate. A missing secret means the server would run misconfigured. Treat resolver errors as fatal.

For optional values (secrets that default to an empty string on miss), let the resolver return the empty string instead of an error. The validation pass catches invalid combinations.

## Test the resolver

Write table-driven tests with a fake client:

```go
func TestVaultResolver_Resolve(t *testing.T) {
    client := &fakeVaultClient{
        secrets: map[string]map[string]any{
            "secret/data/app": {"db_password": "hunter2"},
        },
    }
    r := vault.NewResolver(client)

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

- [About config](../../explanation/about-config.md) for the multi-pass precedence model and the resolver pass.
- [How to environment config](environment-config.md) for the base environment-variable workflow.
- [Config JSON schema reference](../../reference/config-json-schema.md) for every `ServerConfig` field.
- [Secrets API reference](../../reference/secrets-api.md) for the framework's typed `Secret[T]` primitives.
