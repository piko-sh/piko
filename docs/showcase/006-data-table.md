---
title: "006: Sortable data table"
description: Query parameter binding and server-side sorting
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 70
---

# 006: Sortable data table

A sortable data table that reads URL query parameters on the server, sorts data accordingly, and renders the result. No client-side JavaScript required.

## What this demonstrates

Inside `Render`, `r.QueryParam("sort")` and `r.QueryParam("dir")` read the query string and keep sorting state in the URL, which makes the sorted view bookmarkable and shareable. A `p-for` loop iterates over `[]Employee` with `(i, emp)` destructuring. The `p-class:active="state.SortColumn == 'name'"` directive marks the sorted column and works alongside static `class` attributes.

Template expressions like `{{ emp.Name }}` and `{{ state.EmployeeCount }}` render data. Piko type-checks struct fields at compile time. Server-side sorting uses `sort.Slice` in Go, driven by the query parameters.

## Project structure

```text
src/
  pages/
    index.pk          The sortable data table page
```

## How it works

The browser navigates to `/index?sort=department&dir=desc`. The `Render` function reads the query parameters, sorts the employee slice with `sort.Slice`, and returns the sorted data. The template renders column headers as links that toggle sort direction, using `p-if`/`p-else-if`/`p-else` chains to show the correct direction indicator.

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/006_data_table/src/
go mod tidy
air
```

## See also

- [Server actions reference](../reference/server-actions.md).
- [How to dynamic routes](../how-to/routing/dynamic-routes.md).
