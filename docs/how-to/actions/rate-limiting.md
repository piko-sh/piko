---
title: How to rate-limit an action
description: Implement RateLimitable to set per-IP, per-user, or custom-key rate limits on a server action.
nav:
  sidebar:
    section: "how-to"
    subsection: "actions"
    order: 740
---

# How to rate-limit an action

Actions implement `RateLimitable` to override the default rate limiter. Use it to tighten limits on sensitive endpoints (login, password reset, sign-up) or to switch the bucket from IP to authenticated user. For the action structure see [server actions reference](../../reference/server-actions.md).

## Add a RateLimit receiver

Implement `RateLimitable` by adding a `RateLimit()` receiver that returns a `*piko.RateLimit`:

```go
type RateLimitable interface {
    RateLimit() *piko.RateLimit
}

type RateLimit struct {
    // RequestsPerMinute caps the steady-state rate.
    RequestsPerMinute int

    // BurstSize allows short spikes above the steady rate.
    BurstSize int

    // KeyFunc determines the bucket. Defaults to client IP.
    KeyFunc func(*http.Request) string
}
```

Built-in key functions:

| Key function | Bucket |
|---|---|
| `piko.RateLimitByIP` | Client IP address (default) |
| `piko.RateLimitByUser` | Authenticated user ID |
| `piko.RateLimitBySession` | Session ID |

## Throttle a login endpoint

```go
package auth

import (
    "net/http"

    "piko.sh/piko"
)

type LoginResponse struct {
    Message string `json:"message"`
}

type LoginAction struct {
    piko.ActionMetadata
}

func (a LoginAction) RateLimit() *piko.RateLimit {
    return &piko.RateLimit{
        RequestsPerMinute: 5,
        BurstSize:         3,
        KeyFunc:           piko.RateLimitByIP,
    }
}

func (a LoginAction) Call(email string, password string) (LoginResponse, error) {
    session, err := authenticate(a.Ctx(), email, password)
    if err != nil {
        return LoginResponse{}, piko.Unauthorised("Invalid credentials")
    }

    a.Response().SetCookie(piko.SessionCookie("session_id", session.ID, session.ExpiresAt))
    a.Response().AddHelper("redirect", "/dashboard")

    return LoginResponse{Message: "Login successful"}, nil
}
```

Five attempts per minute per IP, with a burst of three, fits a typical login form without locking out legitimate users who fat-finger a password.

## Switch the bucket to authenticated user

For endpoints that run after authentication, key the limiter on user ID so a single tenant cannot starve another:

```go
func (a ExportAction) RateLimit() *piko.RateLimit {
    return &piko.RateLimit{
        RequestsPerMinute: 30,
        BurstSize:         10,
        KeyFunc:           piko.RateLimitByUser,
    }
}
```

Anonymous callers fall back to IP-keyed limits because the user ID resolves to an empty string.

## Compose with other interfaces

`RateLimitable` composes with `MethodOverridable`, `Cacheable`, and `ResourceLimitable`. See [How to override an action's HTTP method](method-override.md), [How to cache action responses](caching.md), and [How to set resource limits on an action](resource-limits.md).

## See also

- [Server actions reference](../../reference/server-actions.md) for the action surface.
- [How to security](../security.md) for CSRF, captcha, and the wider abuse-prevention picture.
- [How to protect an action with a captcha](../captcha.md) for layering challenge-response on top of rate limits.
