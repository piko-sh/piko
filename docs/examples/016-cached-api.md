---
title: "016: Cached API"
description: Action response caching with TTL
nav:
  sidebar:
    section: "examples"
    subsection: "examples"
    order: 140
---

# 016: Cached API

A weather forecast action demonstrating Piko's action response caching. The action implements `CacheConfig()` to opt into in-memory caching with a configurable TTL. When caching is enabled, Piko stores the serialised response after the first call and returns it directly for subsequent calls, bypassing `Call()` until the TTL expires.

## What this demonstrates

- **`CacheConfig()` interface**: returning `*piko.CacheConfig` to opt into response caching; actions are not cached by default; only those implementing `CacheConfig()` participate
- **TTL-based caching**: repeated calls within the TTL window return cached responses; after expiry, the next call executes `Call()` and populates a fresh entry
- **Cache key**: action name + serialised input; different inputs produce separate entries
- **`X-Action-Cache` header**: Piko sets this header on cached responses for client visibility
- Best for idempotent, read-heavy endpoints (weather, exchange rates, leaderboards); avoid caching actions with side effects

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

The page fires two requests 100ms apart. If the timestamps match, the second was served from cache.

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/016_cached_api/src/
go mod tidy
air
```
