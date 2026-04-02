---
title: "006: Sortable data table"
description: Query parameter binding and server-side sorting
nav:
  sidebar:
    section: "examples"
    subsection: "examples"
    order: 70
---

# 006: Sortable data table

A sortable data table that reads URL query parameters on the server, sorts data accordingly, and renders the result. No client-side JavaScript required.

## What this demonstrates

- **Query parameter binding**: `r.QueryParam("sort")` and `r.QueryParam("dir")` inside `Render`; keeps sorting state in the URL, making it bookmarkable and shareable
- **`p-for` with struct slices**: iterating over `[]Employee` with `(i, emp)` destructuring
- **`p-class` conditional classes**: `p-class:active="state.SortColumn == 'name'"` highlights the sorted column; can be combined with static `class` attributes
- **Template expressions**: `{{ emp.Name }}`, `{{ state.EmployeeCount }}` for data display; struct fields are type-checked at compile time
- **Server-side sorting**: `sort.Slice` in Go, driven by query parameters

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
