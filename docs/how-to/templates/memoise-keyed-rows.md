---
title: How to memoise keyed rows with p-memo
description: Skip the patch walk for unchanged rows in a large PKC list by declaring the values each row renders from.
nav:
  sidebar:
    section: "how-to"
    subsection: "templates"
    order: 25
---

# How to memoise keyed rows with `p-memo`

> `p-memo` applies to `.pkc` components only. It tells the client-side vdom reconciler what to skip. The server renders a `.pk` page and never runs that diff, so the binding does nothing there.

On every re-render the client renderer walks each element's props and diffs its children, even when nothing about that element changed. Across ten rows the cost is too small to notice. Across a long keyed list that re-renders on every animation frame, the walk takes most of the frame.

`p-memo` lets you declare what a row renders from. When none of those dependencies change, the renderer skips the whole props walk and child diff for that row.

```html
<template>
    <ul>
        <li p-for="row in state.Rows" p-key="row.ID" p-memo="row">
            <span p-text="row.Label"></span>
            <span p-text="row.Status"></span>
        </li>
    </ul>
</template>
```

If `row` is the same object as last render, the renderer leaves that `<li>` and everything inside it alone.

## Declare every value the row reads

`p-memo` is a promise you make to the renderer: *nothing inside this element changes unless one of these dependencies changes.* Pass an array when a row renders from more than one source.

```html
<template>
    <ul>
        <li
            p-for="row in state.Rows"
            p-key="row.ID"
            p-memo="[row, state.SelectedID, state.Compact]"
        >
            <span p-text="row.Label"></span>
            <span p-class="{ selected: row.ID == state.SelectedID }"></span>
        </li>
    </ul>
</template>
```

The row reads `row`, `state.SelectedID` and `state.Compact`, so all three are dependencies. If you leave `state.SelectedID` out, every row that did not otherwise change keeps its old selection styling.

The renderer compares dependencies by reference, one array item at a time. Reactive state hands back a new proxy wrapper on each property access. The renderer therefore unwraps each dependency to the object underneath before it compares, so reading `state.Rows[0]` twice still counts as the same dependency.

## Replace objects instead of changing them

The renderer compares by reference, so changing a row in place does not count as a change:

```js
// Does NOT invalidate the memo: same object, new contents.
state.Rows[0].Label = "Renamed";

// Invalidates the memo: a different object lands at index 0.
state.Rows[0] = { ...state.Rows[0], Label: "Renamed" };
```

If you prefer mutation, add an explicit version dependency the row can change: `p-memo="[row, row.Revision]"`.

## When not to use it

**Rows containing controlled inputs.** A skipped patch also skips setting `value` and `checked` again. An input the user has typed into then keeps the typed text instead of matching state. Leave `p-memo` off any row with an `<input>`, `<select>` or `<textarea>` bound through `p-model`.

**Rows whose dependencies you cannot list.** A missed dependency does more than delay an update. After a skip, the renderer's stored props hold values it never wrote to the DOM. A later patch compares against those stored props. A field left out of the dependencies can therefore stay wrong *forever*, instead of catching up on the next render.

**Short lists.** The comparison is cheap, but it is not free, and the risk above is real. Use `p-memo` when profiling shows that patching costs too much, not by default.

**Static values.** Writing `_memo="row"` as a plain attribute binds the string `"row"`. That string never changes, so the element stops updating after its first render. Use the `p-memo` directive instead. The compiler warns about the plain attribute with code `T032`.

## Confirm it is working

Stamp a render counter onto each row and watch which rows stop updating:

```html
<template>
    <ul>
        <li
            p-for="row in state.Rows"
            p-key="row.ID"
            p-memo="row"
            :data-render="state.Generation"
        >
            <span p-text="row.Label"></span>
        </li>
    </ul>
</template>
```

Raise `state.Generation` and re-render. Rows that skipped keep their old `data-render` value. Rows whose dependency changed show the new one. That is the memo doing its job, not a bug, because `state.Generation` sits outside the dependencies on purpose.

## See also

- [Directives reference](../../reference/directives.md) for `p-for` and `p-key`.
- [How to loop over data in a template](loops.md) for the list-rendering basics `p-memo` builds on.
