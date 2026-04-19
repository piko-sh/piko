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

Map iteration is **deterministic** in Piko. The generator emits a sorted-by-key range so the same map renders in the same order every request. This is intentional (predictable HTML, stable diffs in cache layers). For a different ordering, by value or by insertion for example, sort the entries inside `Render` and pass a slice of key-value structs to the template.

> **Note:** This is a Piko-specific behaviour. Plain Go ranges over maps in randomised order; the template generator wraps the iteration with a key-sort so output is deterministic. See [`for_emitter.go:446`](https://github.com/piko-sh/piko/blob/master/internal/generator/generator_adapters/driven_code_emitter_go_literal/for_emitter.go) (`emitDeterministicMapLoopWithContext`).

## Iterate over a numeric range

There is no `range(start, end)` builtin in PK expressions. Build the slice in `Render` and iterate that:

```go
type Response struct {
    Steps []int
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    steps := make([]int, 0, 5)
    for i := 1; i < 6; i++ {
        steps = append(steps, i)
    }
    return Response{Steps: steps}, piko.Metadata{}, nil
}
```

```piko
<div p-for="i in state.Steps" p-key="i">
  Step {{ i }}
</div>
```

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
- [How to conditionally render elements](conditionals.md).
- [Template syntax reference](../../reference/template-syntax.md).
