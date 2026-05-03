---
title: How to override an action's HTTP method
description: Current behaviour of the Method() override on a server action.
nav:
  sidebar:
    section: "how-to"
    subsection: "actions"
    order: 720
---

# How to override an action's HTTP method

Actions respond to POST by default. The action protocol reserves a `Method()` receiver that returns a `piko.HTTPMethod`, intended to let you switch an action to GET, PUT, PATCH, DELETE, HEAD, or OPTIONS. For the action surface see [server actions reference](../../reference/server-actions.md). For the rationale behind the action protocol see [about the action protocol](../../explanation/about-the-action-protocol.md).

> [!IMPORTANT]
> The annotator currently registers every action with `Method()` as POST regardless of return value. If you need a non-POST endpoint today, use a normal HTTP handler instead. See [about the action protocol](../../explanation/about-the-action-protocol.md#http-method-override) for why.

## Available constants

| Constant | HTTP method |
|---|---|
| `piko.MethodGet` | GET |
| `piko.MethodPost` | POST (default) |
| `piko.MethodPut` | PUT |
| `piko.MethodPatch` | PATCH (partial update) |
| `piko.MethodDelete` | DELETE |
| `piko.MethodHead` | HEAD |
| `piko.MethodOptions` | OPTIONS |

Piko exports the constants and they are safe to reference today, even though the annotator does not yet honour the return value.

## Declare an intended method

```go
func (a DeleteAction) Method() piko.HTTPMethod {
    return piko.MethodDelete
}
```

After regeneration the action mounts at `/_piko/actions/<package>.<StructName minus "Action">` and accepts only POST. Declaring `Method()` keeps the source aligned with the planned interface for when codegen begins reading the return value.

## Compose with other interfaces

`Cacheable`, `RateLimitable`, and `ResourceLimitable` all compose freely on the same struct. Implement them on the same type to combine behaviours. See [How to cache action responses](caching.md), [How to rate-limit an action](rate-limiting.md), and [How to set resource limits on an action](resource-limits.md).

## See also

- [Server actions reference](../../reference/server-actions.md) for the full action surface.
- [About the action protocol](../../explanation/about-the-action-protocol.md) for the design rationale.
- [How to test actions](../testing.md) for `NewActionTester` and method-aware assertions.
