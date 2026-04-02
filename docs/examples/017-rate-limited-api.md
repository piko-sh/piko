---
title: "017: Rate-limited API"
description: Per-action rate limiting with token bucket algorithm
nav:
  sidebar:
    section: "examples"
    subsection: "examples"
    order: 150
---

# 017: Rate-limited API

An echo API endpoint demonstrating per-action rate limiting. The action allows three requests per minute per IP address. When the limit is exceeded, Piko returns HTTP 429 with standard rate-limit headers.

## What this demonstrates

- **`RateLimit()` interface**: returning `*piko.RateLimit` to opt into per-action rate limiting
- **`piko.RateLimitByIP` key function**: groups rate-limit buckets by client IP
- **Token-bucket algorithm**: refills at the sustained rate, allows bursts up to `BurstSize`
- **HTTP 429 Too Many Requests**: standard status code when the limit is exceeded; rate limiter runs before `Call`, so rejected requests never reach the action
- **Rate-limit headers**: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`, `Retry-After`
- `serverConfig.security.rateLimit.enabled` must be `true`; the middleware is disabled by default
- Set `RequestsPerMinute` based on legitimate usage patterns (5/min for login, 60/min for search); set `BurstSize` equal to or slightly above the per-minute rate for burst tolerance

## Project structure

```text
src/
  actions/
    api/
      echo.go                         Echo action with rate limiting
  pages/
    index.pk                          Demo page that fires rapid requests
```

## How it works

The action returns a `*piko.RateLimit` struct:

```go
func (a *EchoAction) RateLimit() *piko.RateLimit {
    return &piko.RateLimit{
        KeyFunc:           piko.RateLimitByIP,
        RequestsPerMinute: 3,
        BurstSize:         3,
    }
}
```

The page fires five rapid requests. The first three succeed; the fourth gets HTTP 429.

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/017_rate_limited_api/src/
go mod tidy
air
```
