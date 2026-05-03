---
title: Secrets API
description: Types and functions for lazy-loaded, scoped access to secret configuration values.
nav:
  sidebar:
    section: "reference"
    subsection: "bootstrap"
    order: 30
---

# Secrets API

`Secret[T]` provides lazy-loaded, scoped access to secret configuration values (API keys, tokens, signing secrets). Secrets resolve on first `Acquire()` instead of at startup, track their own reference count, and release their memory when the last holder closes its handle. This page documents the surface. For task recipes see the [secrets how-to](../how-to/secrets.md). Source of truth: [`secrets.go`](https://github.com/piko-sh/piko/blob/master/secrets.go).

## Types

### `Secret[T any]`

Type alias for `config_domain.Secret[T]`. Holds a resolver reference and resolves the value on demand.

Supported type parameters:

| `T` | Storage characteristics |
|---|---|
| `string` | Memory zeroing is best-effort (Go's garbage collector may retain copies). |
| `[]byte` | Stored in `SecureBytes` backed by `mmap` + `mlock`; no GC copies. |

Security properties:

| Property | Behaviour |
|---|---|
| Lazy loading | The value does not enter memory until the caller invokes `Acquire()`. |
| Explicit lifecycle | Every `Acquire()` call pairs with a `Release()` call, or the holder closes the handle. |
| Reference counting | Concurrent `Acquire()` calls are safe, and the value stays in memory while any handle is active. |
| Automatic registration | Every `Secret[T]` registers with the global `SecretManager` for coordinated shutdown. |
| Finaliser safety net | If a holder leaks a handle, the garbage collector's finaliser releases it. |

### `SecretHandle[T any]`

Type alias for `config_domain.SecretHandle[T]`. A scoped accessor to a resolved secret value.

- Implements `io.Closer`, which makes it suitable for `defer handle.Close()`.
- `Value()` returns the resolved value while the handle is active.
- `Release()` and `Close()` are equivalent.

### `SecretManager`

Type alias for `config_domain.SecretManager`. Singleton that coordinates the lifecycle of every `Secret[T]` instance.

Access via `GetSecretManager()`:

```go
stats := piko.GetSecretManager().Stats()
log.Info("active secrets", "count", stats.TotalSecrets)
```

### `SecretStats`

Type alias for `config_domain.SecretStats`. Returned by `SecretManager.Stats()`.

## Functions

### `GetSecretManager() *SecretManager`

Returns the singleton secret manager. Use it for statistics access or manual shutdown coordination.

## Errors

| Error | Returned when |
|---|---|
| `ErrSecretNotSet` | A caller invokes `Acquire()` on a secret that never received a value. |
| `ErrSecretClosed` | A caller invokes `Acquire()` on a secret that has already closed. |
| `ErrSecretResolutionFailed` | A resolver returns an error. |
| `ErrSecretHandleClosed` | A caller accesses an already-released handle. |
| `ErrNoResolver` | Piko finds no resolver for the secret's prefix. |

## Resolver registration

Applications register resolvers via the [`WithConfigResolvers`](bootstrap-options.md#security) bootstrap option. A resolver matches a URI prefix (for example `env://`, `vault://`, `awssm://`) and returns the raw value on demand.

## See also

- [Secrets how-to](../how-to/secrets.md) for usage recipes.
- [Bootstrap options reference](bootstrap-options.md) for `WithConfigResolvers`.
- Source: [`secrets.go`](https://github.com/piko-sh/piko/blob/master/secrets.go).
