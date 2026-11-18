---
title: How to rate-limit an action
description: Set per-IP, per-user, or per-session rate limits on a server action.
nav:
  sidebar:
    section: "how-to"
    subsection: "actions"
    order: 740
---

# How to rate-limit an action

Implement `RateLimitable` on an action to tighten the default rate limiter on sensitive endpoints. Common cases include login, password reset, and sign-up, or adding a per-user dimension on top of the always-on per-IP bucket. For the action surface see [server-actions](../../reference/server-actions.md).

The full bucket key is always `"ratelimit:" + keySuffix + ":" + clientIP`. `KeyFunc` controls the suffix. The client IP is always appended, so swapping `KeyFunc` adds a dimension instead of replacing IP.

## Throttle a login endpoint per IP

To allow five attempts per minute per IP, return a `*piko.RateLimit` from `RateLimit()`:

```go
func (a LoginAction) RateLimit() *piko.RateLimit {
    return &piko.RateLimit{
        RequestsPerMinute: 5,
        KeyFunc:           piko.RateLimitByIP,
    }
}
```

Five attempts per minute fits a typical login form without locking out legitimate users who fat-finger a password.

## Add a per-user dimension after authentication

To rate-limit by both user ID and IP, swap to `piko.RateLimitByUser`:

```go
func (a ExportAction) RateLimit() *piko.RateLimit {
    return &piko.RateLimit{
        RequestsPerMinute: 30,
        KeyFunc:           piko.RateLimitByUser,
    }
}
```

Piko composes the bucket key from the registered action name (`<package>.<StructName minus "Action">`) plus the value `KeyFunc` returns plus the client IP. For an `ExportAction` in package `accounts`, that gives `ratelimit:accounts.Export:<userID>:<clientIP>`. The same user from two IPs gets two buckets, and two users behind one NAT'd IP get separate buckets. Anonymous callers degrade gracefully because the key function returns the remote address when no session is present.

`piko.RateLimitBySession` works the same way against the session ID. To define your own suffix, supply a `RateLimitKeyFunc` that returns the suffix string for a given request.

## Share limits across instances

Each process keeps its own rate limiter state by default. To enforce one limit across instances, register a shared remote cache provider and name it for the rate limiter:

```go
ssr := piko.New(
    piko.WithCacheProvider("redis", redisProvider),
    piko.WithRateLimiterCacheProvider("redis"),
)
```

A remote counter can remain active for up to two seconds past its nominal
window end to tolerate clock skew between instances. Rate limit headers
still report the nominal reset time.

The server refuses to start if the named provider cannot be used for rate limiting.

## See also

- [Server actions reference](../../reference/server-actions.md) for the action surface.
- [About the action protocol](../../explanation/about-the-action-protocol.md) for where rate limits sit in the dispatch pipeline.
- [How to security](../security.md) for CSRF, captcha, and the wider abuse-prevention picture.
- [How to protect an action with a captcha](../captcha.md) for layering challenge-response on top of rate limits.
