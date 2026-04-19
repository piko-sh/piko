---
title: How to loop over data in a template
description: Iterate over slices, maps, and ranges with `p-for`, and keep identity stable with `p-key`.
nav:
  sidebar:
    section: "how-to"
    subsection: "templates"
    order: 20
---

# How to loop over data in a template

`p-for` iterates over slices, arrays, maps, and numeric ranges. Pair it with `p-key` for stable identity when the list can change. See the [directives reference](../../reference/directives.md) for the full syntax.

## Iterate over a slice

Use the value-only form when the index is not needed:

```piko
<template>
  <ul>
    <li p-for="item in state.Items" p-key="item.ID">
      {{ item.Name }}
    </li>
  </ul>
</template>
```

Access the index with the tuple form:

```piko
<li p-for="(idx, item) in state.Items" p-key="item.ID">
  {{ idx + 1 }}. {{ item.Name }}
</li>
```

If the index is irrelevant, use `_`:

```piko
<li p-for="(_, item) in state.Items" p-key="item.ID">
  {{ item.Name }}
</li>
```

## Iterate over a map

```piko
<dl>
  <div p-for="(key, value) in state.Config" p-key="key">
    <dt>{{ key }}</dt>
    <dd>{{ value }}</dd>
  </div>
</dl>
```

Map iteration order is unstable (Go ranges maps in random order). For sorted output, sort in `Render` and pass a slice of key-value structs.

> **Note:** `p-for` over a map inherits Go's randomised iteration order, so the same data renders in a different sequence on each request. For deterministic output, sort the entries inside `Render` and pass a slice of key-value structs to the template.

## Iterate over a numeric range

```piko
<div p-for="i in range(1, 6)">
  Step {{ i }}
</div>
```

`range(start, end)` yields `start`, `start+1`, through `end-1`.

## Use `p-key` for mutable lists

`p-key` gives Piko a stable identity for each rendered element. Without it, updates that reorder or remove items may reuse the wrong DOM nodes, which breaks focus, animation, and form state.

Good keys include database IDs, slugs, and UUIDs. Avoid array indices, because they change when items move.

```piko
<li p-for="item in state.Items" p-key="item.ID">
```

For static lists that never change after render, you can omit `p-key`.

## Nested loops

Nest `p-for` freely, and pair each inner loop with its own `p-key`:

```piko
<section p-for="group in state.Groups" p-key="group.ID">
  <h2>{{ group.Name }}</h2>
  <ul>
    <li p-for="item in group.Items" p-key="item.ID">
      {{ item.Name }}
    </li>
  </ul>
</section>
```

## Empty-list fallback

Use `p-if` alongside the loop to show a placeholder when the list is empty:

```piko
<ul p-if="len(state.Items) > 0">
  <li p-for="item in state.Items" p-key="item.ID">{{ item.Name }}</li>
</ul>
<p p-else>No items to display.</p>
```

## Filter and sort before the template

Templates are for presentation, not transformation. Do filtering and sorting in `Render`:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    visible := make([]Item, 0, len(all))
    for _, item := range all {
        if !item.Hidden {
            visible = append(visible, item)
        }
    }

    sort.Slice(visible, func(i, j int) bool {
        return visible[i].Created.After(visible[j].Created)
    })

    return Response{Items: visible}, piko.Metadata{}, nil
}
```

## See also

- [Directives reference](../../reference/directives.md) for `p-for`, `p-key`, and `p-if`.
- [How to conditionals](conditionals.md).
- [Template syntax reference](../../reference/template-syntax.md).
