---
title: How to manage secrets
description: Declare a Secret[T], acquire scoped access, release the handle, and write a custom resolver.
nav:
  sidebar:
    section: "how-to"
    subsection: "operations"
    order: 20
---

# How to manage secrets

A `Secret[T]` stores a reference to a secret value and resolves the value on demand instead of at startup. The framework loads the value into memory only when an action needs it, and the holder releases it when finished. This guide shows the common patterns. See the [secrets API reference](../reference/secrets-api.md) for the full surface.

## Declare a secret in configuration

Reference the secret by prefix in `piko.yaml`:

```yaml
openai:
  apiKey: "env://OPENAI_API_KEY"

stripe:
  webhookSecret: "vault://kv/data/stripe#webhook_secret"

signing:
  key: "awssm://myapp/signing-key"
```

Each prefix maps to a resolver registered at bootstrap. The built-in `env://` resolver reads from environment variables. Custom resolvers handle the others.

Declare the Go type:

```go
type Config struct {
    OpenAI struct {
        APIKey piko.Secret[string] `yaml:"apiKey"`
    } `yaml:"openai"`

    Stripe struct {
        WebhookSecret piko.Secret[string] `yaml:"webhookSecret"`
    } `yaml:"stripe"`

    Signing struct {
        Key piko.Secret[[]byte] `yaml:"key"`
    } `yaml:"signing"`
}
```

Use `Secret[string]` for text secrets and `Secret[[]byte]` for binary ones. The binary form stores the value in `SecureBytes` backed by `mmap` and `mlock`, which stops Go's garbage collector from copying the value.

## Acquire and release

Acquire the value for the minimum time necessary:

```go
func (a CallOpenAIAction) Call(prompt string) (Response, error) {
    handle, err := config.OpenAI.APIKey.Acquire(a.Ctx())
    if err != nil {
        return Response{}, fmt.Errorf("acquiring openai key: %w", err)
    }
    defer handle.Close()

    client := openai.NewClient(handle.Value())
    result, err := client.Complete(a.Ctx(), prompt)
    if err != nil {
        return Response{}, err
    }

    return Response{Text: result.Text}, nil
}
```

`Close()` releases the handle. Always defer it. The reference-counting `SecretManager` keeps the value alive while any handle holds it and releases it when the last one closes.

## Errors

`Acquire` can return:

| Error | Reason |
|---|---|
| `piko.ErrSecretNotSet` | The secret never populated, usually because the config key is missing. |
| `piko.ErrSecretClosed` | The shutdown sequence has closed the secret. |
| `piko.ErrSecretResolutionFailed` | The resolver returned an error (for example, Vault unreachable). |
| `piko.ErrNoResolver` | No resolver handles the secret's URI prefix. |

Use `errors.Is` to distinguish them.

## Write a custom resolver

A resolver implements `config.Resolver`:

```go
package resolvers

import (
    "context"
    "fmt"

    "github.com/hashicorp/vault/api"
    "piko.sh/piko"
)

type VaultResolver struct {
    client *api.Client
}

func NewVaultResolver(addr, token string) (*VaultResolver, error) {
    cfg := api.DefaultConfig()
    cfg.Address = addr
    client, err := api.NewClient(cfg)
    if err != nil {
        return nil, err
    }
    client.SetToken(token)
    return &VaultResolver{client: client}, nil
}

func (r *VaultResolver) Prefix() string { return "vault://" }

func (r *VaultResolver) Resolve(ctx context.Context, uri string) ([]byte, error) {
    // uri is e.g. "vault://kv/data/stripe#webhook_secret"
    path, field := splitVaultURI(uri)
    secret, err := r.client.Logical().ReadWithContext(ctx, path)
    if err != nil {
        return nil, fmt.Errorf("reading %s: %w", path, err)
    }
    value, ok := secret.Data[field].(string)
    if !ok {
        return nil, fmt.Errorf("field %s not found at %s", field, path)
    }
    return []byte(value), nil
}
```

Register it at bootstrap:

```go
vaultResolver, err := resolvers.NewVaultResolver(os.Getenv("VAULT_ADDR"), os.Getenv("VAULT_TOKEN"))
if err != nil {
    log.Fatal(err)
}

ssr := piko.New(
    piko.WithConfigResolvers(vaultResolver),
)
```

## Monitor secret usage

The singleton manager exposes statistics:

```go
stats := piko.GetSecretManager().Stats()
log.Info("active secrets", "count", stats.TotalSecrets, "active handles", stats.ActiveHandles)
```

## See also

- [Secrets API reference](../reference/secrets-api.md).
- [Bootstrap options reference](../reference/bootstrap-options.md) for `WithConfigResolvers`.
