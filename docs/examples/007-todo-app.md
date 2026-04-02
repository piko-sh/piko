---
title: "007: Todo app"
description: Interactive todo list with array reactivity and two-way binding
nav:
  sidebar:
    section: "examples"
    subsection: "examples"
    order: 80
---

# 007: Todo app

An interactive todo list built as a PKC component, demonstrating add, toggle, and delete operations with Piko's reactivity system handling DOM updates automatically.

## What this demonstrates

- **`p-for` with `p-key`**: iterating over objects and tracking items by unique ID for efficient DOM updates; always provide `p-key` when rendering lists. Without it, Piko uses index-based diffing which causes incorrect behaviour on deletions
- **`p-model`**: two-way binding between a text input and reactive state; works with text inputs, checkboxes, selects, and textareas
- **Event handling with arguments**: `p-on:click="deleteItem(item.id)"` inside a `p-for` loop
- **Array reactivity**: mutation methods (`push`, `splice`, `pop`, `shift`) and reassignment (`state.items = state.items.filter(...)`) both trigger automatic re-renders
- **`p-if` conditionals**: removes elements from the DOM entirely; state inside conditionally rendered subtrees is lost when the condition becomes false
- **Template functions**: calling `countCompleted()` in expressions to derive values

## Project structure

```text
src/
  components/
    pp-todo-list.pkc          The todo list component
  pages/
    index.pk                  Host page mounting <pp-todo-list>
```

## How it works

The `state` object holds an `items` array, `newItem` string (bound via `p-model`), and a `nextId` counter. The template uses `p-for="(index, item) in state.items" p-key="item.id"` to render each item.

Key patterns:

```ts
// Add - push triggers reactivity
state.items.push({ id: state.nextId++, text: state.newItem, done: false });
state.newItem = "";  // clears the input via p-model

// Toggle - map returns a new array
state.items = state.items.map(i => i.id === id ? {...i, done: !i.done} : i);

// Delete - filter returns a new array
state.items = state.items.filter(i => i.id !== id);
```

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/007_todo_app/src/
go mod tidy
air
```
