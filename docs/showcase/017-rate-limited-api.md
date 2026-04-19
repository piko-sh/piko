---
title: "017: Rate-limited API"
description: Per-action rate limiting with token bucket algorithm
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 150
---

# 017: Rate-limited API

An echo API endpoint demonstrating per-action rate limiting. The action allows three requests per minute per IP address. Once a client passes the limit, Piko returns HTTP 429 with standard rate-limit headers.

## What this demonstrates

An action opts into per-action rate limiting by implementing `RateLimit()` and returning `*piko.RateLimit`. The `piko.RateLimitByIP` key function groups rate-limit buckets by client IP. The token-bucket algorithm refills at the sustained rate and allows bursts up to `BurstSize`. When the bucket empties, Piko returns HTTP 429. The rate limiter runs before `Call`, so rejected requests never reach the action.

Piko sets four rate-limit response headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`, and `Retry-After`. The middleware does not run when `serverConfig.security.rateLimit.enabled` is absent, since the default turns it off. Set `RequestsPerMinute` to match legitimate usage (5 per minute for login, 60 per minute for search). Set `BurstSize` equal to or slightly above the per-minute rate to tolerate bursts.

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

The page fires five rapid requests. The first three succeed, and the fourth gets HTTP 429.

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/017_rate_limited_api/src/
go mod tidy
air
```

## See also

- [Server actions reference](../reference/server-actions.md).
- [How to security](../how-to/security.md).
