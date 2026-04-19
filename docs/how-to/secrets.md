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

A `Secret[T]` stores a reference to a secret value and resolves the value on demand instead of at startup. Piko loads the value into memory only when an action needs it, and the holder releases it when finished. This guide shows the common patterns. See the [secrets API reference](../reference/secrets-api.md) for the full surface.

## Declare a secret in your application config

Use `piko.Secret[T]` fields where you would otherwise hold a raw string. Populate each via `UnmarshalText` with a resolver-prefixed placeholder:

```go
type AppConfig struct {
    OpenAIKey     piko.Secret[string]
    WebhookSecret piko.Secret[string]
    SigningKey    piko.Secret[[]byte]
}

var cfg AppConfig
_ = cfg.OpenAIKey.UnmarshalText([]byte("env:OPENAI_API_KEY"))
_ = cfg.WebhookSecret.UnmarshalText([]byte("vault:kv/data/stripe#webhook_secret"))
_ = cfg.SigningKey.UnmarshalText([]byte("awssm:myapp/signing-key"))
```

`Secret[T]` implements `encoding.TextUnmarshaler`. In production, your config loader (koanf, viper, envconfig) calls `UnmarshalText` on each field automatically when it reads the placeholder string from environment, file, or struct tags.

Each prefix maps to a resolver registered at bootstrap. The built-in `env:` resolver reads from environment variables. Custom resolvers handle the rest.

Use `Secret[string]` for text and `Secret[[]byte]` for binary secrets. The binary form stores the value in `SecureBytes` (backed by `mmap` + `mlock`), preventing the GC from copying it. See [secrets API reference](../reference/secrets-api.md) for the complete type surface.

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

A resolver implements `piko.ConfigResolver` (an alias for `config_domain.Resolver`):

```go
package resolvers

import (
    "context"
    "fmt"

    "github.com/hashicorp/vault/api"
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

func (r *VaultResolver) GetPrefix() string { return "vault:" }

func (r *VaultResolver) Resolve(ctx context.Context, value string) (string, error) {
    // value is the lookup key with the prefix already stripped, e.g.
    // "kv/data/stripe#webhook_secret".
    path, field := splitVaultPath(value)
    secret, err := r.client.Logical().ReadWithContext(ctx, path)
    if err != nil {
        return "", fmt.Errorf("reading %s: %w", path, err)
    }
    resolved, ok := secret.Data[field].(string)
    if !ok {
        return "", fmt.Errorf("field %s not found at %s", field, path)
    }
    return resolved, nil
}
```

The `Resolve` method receives the placeholder value with the prefix stripped, and returns the resolved string. Piko converts the result to `[]byte` automatically when the field type is `Secret[[]byte]`.

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
log.Info("secrets", "total", stats.TotalSecrets, "active", stats.ActiveSecrets)
```

## See also

- [Secrets API reference](../reference/secrets-api.md) for the full type surface.
- [About configuration](../explanation/about-configuration.md) for the rationale behind resolvers and deploy-time secrets.
- [Bootstrap options reference](../reference/bootstrap-options.md) for `WithConfigResolvers`.
