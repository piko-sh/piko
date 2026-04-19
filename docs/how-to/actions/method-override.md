---
title: How to override an action's HTTP method
description: Make an action respond to GET, PUT, PATCH, DELETE, HEAD, or OPTIONS instead of the default POST.
nav:
  sidebar:
    section: "how-to"
    subsection: "actions"
    order: 720
---

# How to override an action's HTTP method

Actions respond to POST by default. Implement `MethodOverridable` to bind an action to a different HTTP method. For the action structure and response helpers see [server actions reference](../../reference/server-actions.md). For the rationale behind the action protocol see [about the action protocol](../../explanation/about-the-action-protocol.md).

## Define the HTTP method

Implement `MethodOverridable` by adding a `Method()` receiver that returns the chosen `piko.HTTPMethod`:

```go
type MethodOverridable interface {
    Method() piko.HTTPMethod
}
```

Available methods:

| Constant | HTTP method |
|---|---|
| `piko.MethodGet` | GET |
| `piko.MethodPost` | POST (default) |
| `piko.MethodPut` | PUT |
| `piko.MethodPatch` | `PATCH` |
| `piko.MethodDelete` | DELETE |
| `piko.MethodHead` | HEAD |
| `piko.MethodOptions` | OPTIONS |

## Bind a DELETE action

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
    if err := dal.DeleteProduct(a.Ctx(), id); err != nil {
        return DeleteResponse{}, fmt.Errorf("deleting product: %w", err)
    }

    a.Response().AddHelper("showToast", "Product deleted", "success")
    a.Response().AddHelper("reloadPartial", `[data-table="products"]`)

    return DeleteResponse{Deleted: true}, nil
}
```

The generated TypeScript client routes the call to DELETE automatically. Form submissions stay declarative because the framework reads the method from the action type.

## Compose with other interfaces

`MethodOverridable` composes with `Cacheable`, `RateLimitable`, and `ResourceLimitable`. Implement them on the same struct to combine behaviours. See [How to cache action responses](caching.md), [How to rate-limit an action](rate-limiting.md), and [How to set resource limits on an action](resource-limits.md).

## See also

- [Server actions reference](../../reference/server-actions.md) for the full action surface.
- [About the action protocol](../../explanation/about-the-action-protocol.md) for the design rationale.
- [How to test actions](../testing.md) for `NewActionTester` and method-aware assertions.
