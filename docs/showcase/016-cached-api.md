---
title: "016: Cached API"
description: Action response caching with TTL
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 140
---

# 016: Cached API

A weather forecast action demonstrating Piko's action response caching. The action implements `CacheConfig()` to opt into in-memory caching with a configurable TTL. Once the action opts in, Piko stores the serialised response after the first call and returns it directly for later calls, bypassing `Call()` until the TTL expires.

## What this demonstrates

An action opts into response caching by implementing `CacheConfig()` and returning `*piko.CacheConfig`. Piko does not cache actions by default, and only those implementing `CacheConfig()` participate. Repeated calls within the TTL window return cached responses. After expiry, the next call runs `Call()` and populates a fresh entry.

Piko keys the cache by action name plus serialised input, so different inputs produce separate entries. Piko sets an `X-Action-Cache` header on cached responses so the client can see what happened. Caching suits idempotent, read-heavy endpoints such as weather, exchange rates, and leaderboards. Do not cache actions with side effects.

## Project structure

```text
src/
  actions/
    weather/
      forecast.go                     Cached weather forecast action
  pages/
    index.pk                          Page with cache verification UI
```

## How it works

The action returns `&piko.CacheConfig{TTL: 10 * time.Second}` from `CacheConfig()`. Piko keys the cache by action name and serialised input:

```go
func (a *ForecastAction) CacheConfig() *piko.CacheConfig {
    return &piko.CacheConfig{TTL: 10 * time.Second}
}
```

The page fires two requests 100 ms apart. If the timestamps match, Piko served the second request from cache.

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/016_cached_api/src/
go mod tidy
air
```

## See also

- [Cache API reference](../reference/cache-api.md).
- [How to cache](../how-to/cache.md).
