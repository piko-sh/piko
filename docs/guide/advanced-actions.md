---
title: Advanced actions
description: Advanced action patterns including caching, rate limiting, resource limits, and HTTP method overrides
nav:
  sidebar:
    section: "guide"
    subsection: "advanced"
    order: 740
---

# Advanced actions

This guide covers advanced action patterns not found in the individual feature guides. You should already be familiar with [server actions](/docs/guide/server-actions).

For foundational topics covered in dedicated guides, see:

- **[Server actions](/docs/guide/server-actions)** → Action structure, typed parameters/responses, response helpers, cookie management, error handling
- **[Testing](/docs/guide/testing)** → `NewActionTester`, assertion methods, table-driven action tests

## HTTP method override

By default, actions respond to POST requests. Implement `MethodOverridable` to use a different HTTP method.

### Interface definition

```go
type MethodOverridable interface {
    Method() piko.HTTPMethod
}
```

### Available methods

```go
piko.MethodGet     // GET
piko.MethodPost    // POST (default)
piko.MethodPut     // PUT
piko.MethodPatch   // PATCH
piko.MethodDelete  // DELETE
piko.MethodHead    // HEAD
piko.MethodOptions // OPTIONS
```

### Example: DELETE action

```go
package product

import (
    "fmt"

    "piko.sh/piko"
    "myapp/pkg/dal"
)

type DeleteResponse struct {
    Deleted bool `json:"deleted"`
}

type DeleteAction struct {
    piko.ActionMetadata
}

func (a DeleteAction) Method() piko.HTTPMethod {
    return piko.MethodDelete
}

func (a DeleteAction) Call(id int64) (DeleteResponse, error) {
    err := dal.DeleteProduct(a.Ctx(), id)
    if err != nil {
        return DeleteResponse{}, fmt.Errorf("deleting product: %w", err)
    }

    a.Response().AddHelper("showToast", "Product deleted", "success")
    a.Response().AddHelper("reloadPartial", `[data-table="products"]`)

    return DeleteResponse{Deleted: true}, nil
}
```

## Caching

Actions can implement `Cacheable` to configure response caching behaviour.

### Interface definition

```go
type Cacheable interface {
    CacheConfig() *piko.CacheConfig
}
```

### CacheConfig structure

```go
type CacheConfig struct {
    // TTL is the cache time-to-live duration.
    TTL time.Duration

    // VaryHeaders lists headers that affect the cache key.
    // Different header values produce different cache entries.
    VaryHeaders []string

    // KeyFunc is an optional function to generate custom cache keys.
    // If nil, the default key is based on the action name and arguments.
    KeyFunc func(*http.Request) string
}
```

### Example: cacheable action

```go
package product

import (
    "net/http"
    "time"

    "piko.sh/piko"
    "myapp/pkg/dal"
)

type ListResponse struct {
    Products []dal.Product `json:"products"`
}

type ListAction struct {
    piko.ActionMetadata
}

func (a ListAction) Method() piko.HTTPMethod {
    return piko.MethodGet
}

func (a ListAction) CacheConfig() *piko.CacheConfig {
    return &piko.CacheConfig{
        TTL:         5 * time.Minute,
        VaryHeaders: []string{"Accept-Language"},
        KeyFunc: func(r *http.Request) string {
            return r.URL.Path + "?" + r.URL.RawQuery
        },
    }
}

func (a ListAction) Call(category string) (ListResponse, error) {
    products, err := dal.ListProducts(a.Ctx(), category)
    if err != nil {
        return ListResponse{}, fmt.Errorf("listing products: %w", err)
    }

    return ListResponse{Products: products}, nil
}
```

## Rate limiting

Actions can implement `RateLimitable` to override default rate limiting. This is useful for sensitive operations like authentication where stricter limits are required.

### Interface definition

```go
type RateLimitable interface {
    RateLimit() *piko.RateLimit
}
```

### RateLimit structure

```go
type RateLimit struct {
    // RequestsPerMinute is the maximum requests allowed per minute.
    RequestsPerMinute int

    // BurstSize is the maximum burst size for the rate limiter.
    BurstSize int

    // KeyFunc determines the rate limit key (e.g., by IP, user, or custom).
    // If nil, defaults to rate limiting by IP address.
    KeyFunc func(*http.Request) string
}
```

Piko provides built-in key functions:

| Key function | Description |
|-------------|-------------|
| `piko.RateLimitByIP` | Limit by client IP address (default) |
| `piko.RateLimitByUser` | Limit by authenticated user ID |
| `piko.RateLimitBySession` | Limit by session ID |

### Example: strict rate limiting for login

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
        KeyFunc: func(r *http.Request) string {
            return r.RemoteAddr
        },
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

## Resource limits

Actions can implement `ResourceLimitable` to set constraints on resource usage, protecting against resource exhaustion or denial-of-service.

### Interface definition

```go
type ResourceLimitable interface {
    ResourceLimits() *piko.ResourceLimits
}
```

### ResourceLimits structure

```go
type ResourceLimits struct {
    MaxRequestBodySize int64         // Maximum request body size in bytes (0 = default)
    MaxResponseSize    int64         // Maximum response size in bytes (0 = no limit)
    Timeout            time.Duration // Maximum execution time (0 = default)
    SlowThreshold      time.Duration // Duration after which a request is logged as slow (0 = default)
    MaxConcurrent      int           // Maximum concurrent executions (0 = no limit)
    MaxMemoryUsage     int64         // Maximum memory usage in bytes, advisory (0 = no limit)
    MaxSSEDuration     time.Duration // Maximum SSE connection duration
    SSEHeartbeatInterval time.Duration // Interval between SSE heartbeat messages
}
```

### Example: upload with resource limits

```go
package upload

import (
    "fmt"
    "time"

    "piko.sh/piko"
)

type FileResponse struct {
    Filename string `json:"filename"`
    Size     int64  `json:"size"`
}

type FileAction struct {
    piko.ActionMetadata
}

func (a FileAction) ResourceLimits() *piko.ResourceLimits {
    return &piko.ResourceLimits{
        MaxRequestBodySize: 50 * 1024 * 1024, // 50 MB
        Timeout:            2 * time.Minute,
        MaxConcurrent:      5,
    }
}

func (a FileAction) Call(file piko.FileUpload) (FileResponse, error) {
    data, err := file.ReadAll()
    if err != nil {
        return FileResponse{}, fmt.Errorf("reading upload: %w", err)
    }

    header := file.Header()

    if err := storage.Save(a.Ctx(), header.Filename, data); err != nil {
        return FileResponse{}, fmt.Errorf("saving file: %w", err)
    }

    a.Response().AddHelper("showToast", "File uploaded!", "success")

    return FileResponse{
        Filename: header.Filename,
        Size:     header.Size,
    }, nil
}
```

## Combining interfaces

Actions can implement multiple optional interfaces to combine behaviours:

```go
package admin

import (
    "fmt"
    "net/http"
    "time"

    "piko.sh/piko"
    "myapp/pkg/dal"
)

type DeleteUserResponse struct {
    Deleted bool `json:"deleted"`
}

type DeleteUserAction struct {
    piko.ActionMetadata
}

// MethodOverridable: Use DELETE method.
func (a DeleteUserAction) Method() piko.HTTPMethod {
    return piko.MethodDelete
}

// Cacheable: Prevent caching of DELETE operations.
func (a DeleteUserAction) CacheConfig() *piko.CacheConfig {
    return &piko.CacheConfig{
        TTL: 0, // No caching
    }
}

// RateLimitable: Stricter rate limits for destructive operations.
func (a DeleteUserAction) RateLimit() *piko.RateLimit {
    return &piko.RateLimit{
        RequestsPerMinute: 10,
        BurstSize:         3,
        KeyFunc: func(r *http.Request) string {
            return r.RemoteAddr
        },
    }
}

// ResourceLimitable: Set a short timeout for delete operations.
func (a DeleteUserAction) ResourceLimits() *piko.ResourceLimits {
    return &piko.ResourceLimits{
        Timeout: 10 * time.Second,
    }
}

func (a DeleteUserAction) Call(user_id int64) (DeleteUserResponse, error) {
    if err := dal.DeleteUser(a.Ctx(), user_id); err != nil {
        return DeleteUserResponse{}, fmt.Errorf("deleting user: %w", err)
    }

    a.Response().AddHelper("showToast", "User deleted", "success")
    a.Response().AddHelper("reloadPartial", `[data-table="users"]`)

    return DeleteUserResponse{Deleted: true}, nil
}
```

## Next steps

- [Server actions](/docs/guide/server-actions) → Action structure, response helpers, cookie management
- [Testing](/docs/guide/testing) → Action testing with NewActionTester
