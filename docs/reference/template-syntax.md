---
title: Template syntax
description: The Go-like expression DSL available inside .pk and .pkc templates.
nav:
  sidebar:
    section: "reference"
    subsection: "template-syntax"
    order: 5
---

# Template syntax

Piko templates share one expression language. The same DSL appears inside `{{ ... }}` and directive attributes in both PK files (compiled to Go, rendered server-side) and PKC files (compiled to JavaScript, executed in the browser). The DSL is intentionally Go-like so that lowering to Go is direct, but it adds ergonomic extensions that plain Go does not support.

> **Note:** The expression language is one DSL with two compile targets. PK templates lower expressions to Go at build time; PKC templates lower the same syntax to JavaScript. See [about PK files](../explanation/about-pk-files.md) for the rationale.

This page enumerates the interpolation syntax, the expression operators, the DSL extensions, and the template helpers available inside expressions. For the surrounding file structure see the [PK file format reference](pk-file-format.md). For directives (`p-if`, `p-for`, `p-on`, and others) see the [directives reference](directives.md). For the types and functions available inside expressions see the [runtime symbols reference](runtime-symbols.md).

## DSL extensions beyond Go

<p align="center">
  <img src="../diagrams/template-dsl-lowering.svg"
       alt="One shared DSL expression at the top branches into two lowered forms. The PK path lowers ternary to a wrapper closure, nullish coalescing to a nil check, template literals to a Sprintf call, and loose equality to a safeconv helper. The PKC path keeps ternary, nullish coalescing, and template literals native, and emits double-equals for loose equality."
       width="600"/>
</p>

The expression language adds conveniences that are not valid Go:

| Syntax | Meaning | Lowered form in Go | Lowered form in JavaScript |
|---|---|---|---|
| `cond ? a : b` | Ternary. | Immediately invoked function returning `a` or `b`. | Native `cond ? a : b`. |
| `a ~= b` | Non-strict equality: compares values allowing type coercion. | Helper call (`safeconv.Equals`). | `==` (as opposed to `===`). |
| `a !~= b` | Non-strict inequality. | Helper call. | `!=` (as opposed to `!==`). |
| `~expr` | Truthy coercion: true when the value is non-zero, non-empty, and not nil. | Helper call (`safeconv.Truthy`). | JavaScript truthiness check. |
| `a ?? b` | Nullish coalescing: returns `a` if non-nil, otherwise `b`. | Nil check. | Native `??`. |
| `` `text ${expr} text` `` | Template literal with embedded expressions. | Lowered to `fmt.Sprintf` with the expressions as arguments. | Native template literal. |

Plain Go operators (`==`, `!=`, `<`, `>`, `<=`, `>=`, `&&`, `||`, `!`, `+`, `-`, `*`, `/`, `%`) work as expected. Field access, method calls on exposed methods, and slice/map indexing all follow Go's rules.

PKC templates use `==` for strict equality, the same way Go does. The DSL does not recognise JavaScript's `===`. Use `==` for strict comparison and `~=` for type coercion.

> **Note:** In the DSL, `==` is strict (Go semantics) and `~=` is the loose form. JavaScript's `===` does not parse, which inverts the JS muscle memory of "use triple-equals to be safe."

The DSL does not support struct literals, function declarations, or block constructs. Build those in the script section and reference the result from the template.

## Basic interpolation

Use double curly braces `{{ }}` to insert values from your Response struct:

```piko
<template>
  <div>
    <h1>Hello, {{ state.Name }}!</h1>
    <p>You have {{ state.MessageCount }} messages.</p>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Response struct {
    Name         string
    MessageCount int
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{
        Name:         "Alice",
        MessageCount: 5,
    }, piko.Metadata{}, nil
}
</script>
```

**Output**:
```html
<div>
  <h1>Hello, Alice!</h1>
  <p>You have 5 messages.</p>
</div>
```

Inside `{{ }}`, access Response fields through `state.FieldName`. See [pk-file format reference](pk-file-format.md) for where `state` comes from.

## Expressions

Expressions inside `{{ }}`:

### Arithmetic operators

The DSL accepts the standard arithmetic operators: `+`, `-`, `*`, `/`, `%` (modulo).

```piko
<template>
  <div>
    <p>Price: ${{ state.Price }}</p>
    <p>Quantity: {{ state.Quantity }}</p>
    <p>Total: ${{ state.Price * state.Quantity }}</p>
    <p>Tax (10%): ${{ state.Price * state.Quantity * 0.10 }}</p>
    <p>Grand Total: ${{ state.Price * state.Quantity * 1.10 }}</p>
    <p>Remainder: {{ 10 % 3 }}</p>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Response struct {
    Price    float64
    Quantity int
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{
        Price:    9.99,
        Quantity: 3,
    }, piko.Metadata{}, nil
}
</script>
```

**Output**:
```html
<div>
  <p>Price: $9.99</p>
  <p>Quantity: 3</p>
  <p>Total: $29.97</p>
  <p>Tax (10%): $2.997</p>
  <p>Grand Total: $32.967</p>
  <p>Remainder: 1</p>
</div>
```

### String concatenation

Use `+` to concatenate strings:

```html
<template>
  <p>{{ state.FirstName + " " + state.LastName }}</p>
  <p>{{ "Welcome back, " + state.Username + "!" }}</p>
  <!-- Numbers are automatically converted when concatenating with strings -->
  <p>{{ "Age: " + state.Age }}</p>
</template>
```

### Comparison operators

Piko uses **strict equality by default** (`==`, `!=`) with Go-style semantics. For loose equality with type coercion, use `~=` and `!~=`:

```html
<template>
  <!-- Strict equality (default, no type coercion) -->
  <p>5 == "5": {{ 5 == "5" }}</p>       <!-- false (different types) -->
  <p>5 == 5: {{ 5 == 5 }}</p>           <!-- true -->

  <!-- Loose equality (with type coercion) -->
  <p>5 ~= "5": {{ 5 ~= "5" }}</p>       <!-- true (coerced) -->
  <p>0 ~= "0": {{ 0 ~= "0" }}</p>       <!-- true (coerced) -->

  <!-- Truthiness check with ~ operator -->
  <p>Is truthy: {{ ~state.Value }}</p>  <!-- true if value is truthy -->

  <!-- Other comparisons -->
  <p>Is adult: {{ state.Age >= 18 }}</p>
  <p>Has discount: {{ state.Total > 100 }}</p>
  <p>Is premium: {{ state.Tier == "premium" }}</p>
</template>
```

**All comparison operators**: `==`, `!=`, `~=`, `!~=`, `<`, `>`, `<=`, `>=`

**Truthiness operator**: `~` (unary) - converts any value to its boolean truthiness

### Logical operators

```html
<template>
  <p>Can checkout: {{ state.ItemCount > 0 && state.IsLoggedIn }}</p>
  <p>Show warning: {{ state.Stock < 10 || state.Discontinued }}</p>
  <p>Not verified: {{ !state.EmailVerified }}</p>
</template>
```

**All logical operators**: `&&` (and), `||` (or), `!` (not)

### Nullish coalescing operator

Use `??` to provide a fallback value when the left operand is `nil`:

```html
<template>
  <!-- Returns "default" only if nilValue is nil -->
  <p>{{ state.MaybeNil ?? "default" }}</p>

  <!-- Unlike ||, ?? only checks for nil (not falsy values) -->
  <p>{{ state.EmptyString ?? "fallback" }}</p>  <!-- Shows empty string, not fallback -->
  <p>{{ state.ZeroValue ?? 100 }}</p>           <!-- Shows 0, not 100 -->
  <p>{{ state.FalseValue ?? true }}</p>         <!-- Shows false, not true -->

  <!-- Chaining -->
  <p>{{ state.First ?? state.Second ?? "default" }}</p>
</template>
```

`||` returns the right operand for any falsy left operand (`0`, `""`, `false`, `nil`). `??` returns the right operand only when the left operand is `nil`.

## Function calls

Templates can call functions attached to the Response struct:

```piko
<template>
  <div>
    <p>Full Name: {{ state.GetFullName() }}</p>
    <p>Formatted Date: {{ state.FormatDate(state.CreatedAt) }}</p>
  </div>
</template>

<script type="application/x-go">
package main

import (
    "fmt"
    "time"

    "piko.sh/piko"
)

type Response struct {
    FirstName   string
    LastName    string
    CreatedAt   time.Time
    GetFullName func() string
    FormatDate  func(t time.Time) string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    firstName := "John"
    lastName := "Doe"
    createdAt := time.Now()

    return Response{
        FirstName: firstName,
        LastName:  lastName,
        CreatedAt: createdAt,
        GetFullName: func() string {
            return fmt.Sprintf("%s %s", firstName, lastName)
        },
        FormatDate: func(t time.Time) string {
            return t.Format("January 2, 2006")
        },
    }, piko.Metadata{}, nil
}
</script>
```

Formatting, calculations, and logic too complex for inline expressions belong in Response-struct functions. The `Render` function is the natural home for that work because it sees the full Go type system and standard library.

## Ternary operator

Use ternary expressions for conditional values: `condition ? valueIfTrue : valueIfFalse`

```piko
<template>
  <div>
    <!-- Simple ternary -->
    <p>Status: {{ state.IsActive ? "Active" : "Inactive" }}</p>

    <!-- In attributes -->
    <div :class="state.IsActive ? 'status-active' : 'status-inactive'">
      {{ state.IsActive ? "Online" : "✗ Offline" }}
    </div>

    <!-- Nested ternaries -->
    <p>Priority: {{ state.Score > 80 ? "High" : state.Score > 50 ? "Medium" : "Low" }}</p>

    <!-- With numbers -->
    <p>Discount: {{ state.IsPremium ? 20 : 10 }}%</p>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Response struct {
    IsActive  bool
    Score     int
    IsPremium bool
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{
        IsActive:  true,
        Score:     75,
        IsPremium: true,
    }, piko.Metadata{}, nil
}
</script>
```

**Output**:
```html
<div>
  <p>Status: Active</p>
  <div class="status-active">Online</div>
  <p>Priority: Medium</p>
  <p>Discount: 20%</p>
</div>
```

## Optional chaining

Use `?.` to safely access nested properties that might be nil:

### Basic optional chaining

```piko
<template>
  <div>
    <!-- Safe access - won't panic if User or Address is nil -->
    <p>Street: {{ state.User?.Address?.Street }}</p>

    <!-- Returns empty string if any part is nil -->
    <p>City: {{ state.User?.Address?.City }}</p>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Address struct {
    Street string
    City   string
}

type User struct {
    Name    string
    Address *Address
}

type Response struct {
    User *User
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{
        User: &User{
            Name: "Alice",
            Address: &Address{
                Street: "123 Main St",
                City:   "Springfield",
            },
        },
    }, piko.Metadata{}, nil
}
</script>
```

### Optional chaining with arrays

Use `?.[index]` for safe array access:

```piko
<template>
  <div>
    <!-- Safe array access -->
    <p>First tag: {{ state.User?.Tags?.[0] }}</p>

    <!-- Out of bounds returns empty string -->
    <p>100th tag: {{ state.User?.Tags?.[99] }}</p>

    <!-- Chaining with array and properties -->
    <p>First order: {{ state.User?.Orders?.[0]?.ItemName }}</p>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Order struct {
    ItemName string
}

type User struct {
    Tags   []string
    Orders []*Order
}

type Response struct {
    User *User
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{
        User: &User{
            Tags: []string{"admin", "active"},
            Orders: []*Order{
                {ItemName: "Laptop"},
                {ItemName: "Mouse"},
            },
        },
    }, piko.Metadata{}, nil
}
</script>
```

### Combining with nullish coalescing

```html
<template>
  <!-- Provide a fallback when optional chaining returns nil -->
  <p>Street: {{ state.User?.Address?.Street ?? "No address on file" }}</p>
  <p>First tag: {{ state.User?.Tags?.[0] ?? "No tags" }}</p>
</template>
```

**Why use optional chaining?**

```html
<!-- Without optional chaining - can panic! -->
<p>{{ state.User.Address.Street }}</p>  <!-- Panics if User or Address is nil -->

<!-- With optional chaining - safe -->
<p>{{ state.User?.Address?.Street }}</p>  <!-- Returns "" if any part is nil -->
```

## Attribute binding

Use the `:` prefix to bind dynamic values to HTML attributes:

```html
<template>
  <!-- Dynamic href -->
  <a :href="state.Link">{{ state.LinkText }}</a>

  <!-- Conditional class -->
  <div :class="state.IsActive ? 'active' : 'inactive'">Status</div>

  <!-- Data attributes -->
  <div :data-user-id="state.UserID" :aria-label="`Profile for ${state.Username}`">
    User card
  </div>
</template>
```

For all binding patterns (boolean attributes, style binding, `p-bind`, `p-class`, `p-style`), see [directives](directives.md).

## Template literals

Use backticks for template literals (string interpolation with `${expression}`):

```piko
<template>
  <div>
    <!-- Template literal in attribute -->
    <div :class="`card theme-${state.Theme} status--${state.Status}`">
      User card
    </div>

    <!-- Multiple interpolations -->
    <p :aria-label="`User profile for ${state.Username} (ID: ${state.UserID})`">
      {{ state.Username }}
    </p>

    <!-- In content -->
    <h1>{{ `Welcome, ${state.FirstName} ${state.LastName}!` }}</h1>

    <!-- With expressions -->
    <p>{{ `You have ${state.Count} item${state.Count != 1 ? 's' : ''}` }}</p>

    <!-- Escaping backticks and dollar signs -->
    <p>{{ `Use \`backticks\` and \${expressions}` }}</p>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Response struct {
    Theme     string
    Status    string
    Username  string
    UserID    int
    FirstName string
    LastName  string
    Count     int
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{
        Theme:     "dark",
        Status:    "active",
        Username:  "jdoe",
        UserID:    123,
        FirstName: "John",
        LastName:  "Doe",
        Count:     5,
    }, piko.Metadata{}, nil
}
</script>
```

**Output**:
```html
<div>
  <div class="card theme-dark status--active">User card</div>
  <p aria-label="User profile for jdoe (ID: 123)">jdoe</p>
  <h1>Welcome, John Doe!</h1>
  <p>You have 5 items</p>
  <p>Use `backticks` and ${expressions}</p>
</div>
```

## Built-in literal types

Piko supports three built-in literal types for common data:

### Decimal literals

For precise decimal arithmetic (useful for money calculations):

```html
<template>
  <p>Price: {{ 99.99d }}</p>
  <p>Total: {{ 10.5d + 2.5d }}</p>
  <p>Is expensive: {{ state.Price > 100d }}</p>
</template>
```

### Date and time literals

```html
<template>
  <!-- Date literal (YYYY-MM-DD format) -->
  <p>Launch date: {{ d'2025-01-15' }}</p>

  <!-- Time literal (HH:mm:ss format) -->
  <p>Start time: {{ t'09:30:00' }}</p>

  <!-- DateTime literal (RFC3339 format) -->
  <p>Event: {{ dt'2025-01-15T10:00:00Z' }}</p>

  <!-- Duration literal -->
  <p>Duration: {{ du'1h30m' }}</p>

  <!-- Date arithmetic -->
  <p>Tomorrow: {{ d'2025-01-15' + du'24h' }}</p>
  <p>Time difference: {{ dt'2025-01-02T00:00:00Z' - dt'2025-01-01T00:00:00Z' }}</p>
</template>
```

### BigInt literals

For arbitrary-precision integers:

```html
<template>
  <p>Large number: {{ 12345678901234567890n }}</p>
  <p>Sum: {{ 100n + 50n }}</p>
</template>
```

### Rune literals

For single Unicode characters:

```html
<template>
  <p>Letter: {{ r'a' }}</p>
  <p>Emoji: {{ r'🚀' }}</p>
  <p>Escaped: {{ r'\n' }}</p>
</template>
```

## Array and object literals

You can create arrays and objects directly in templates:

```piko
<template>
  <!-- Array literal -->
  <div p-for="item in [1, 2, 3]">{{ item }}</div>

  <!-- Object literal -->
  <div :data-config="{ theme: 'dark', fontSize: 16 }">Config</div>

  <!-- Mixed types -->
  <div p-for="item in ['text', 123, true, nil]">{{ item }}</div>

  <!-- Trailing commas are allowed -->
  <div :data-list="[1, 2, 3,]">List</div>
</template>
```

## HTML escaping

By default, all interpolated content is HTML-escaped to prevent XSS attacks:

```piko
<template>
  <div>
    <!-- HTML is escaped automatically -->
    <p>{{ state.UserInput }}</p>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Response struct {
    UserInput string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{
        UserInput: "<script>alert('xss')</script>",
    }, piko.Metadata{}, nil
}
</script>
```

**Output** (Piko escapes the malicious script):
```html
<div>
  <p>&lt;script&gt;alert('xss')&lt;/script&gt;</p>
</div>
```

`{{ }}` interpolations auto-escape `<`, `>`, `&`, `"`, and `'` to HTML entities. Attribute bindings escape the same characters for attribute context.

### Rendering raw HTML

The `p-html` directive renders a string as raw HTML, bypassing escaping. Use it only for content you fully control. See [directives reference](directives.md#p-html).

```html
<template>
  <div p-html="state.TrustedHTML"></div>
</template>
```

For worked formatting patterns see [how to pluralise translations](../how-to/i18n/pluralisation.md), [how to bind typed variables to translations](../how-to/i18n/variable-binding.md), and [how to format dates and times for a locale](../how-to/i18n/date-time-formatting.md).

## Limitations

### No multi-statement expressions

```html
<!-- Wrong: Multiple statements not allowed -->
<p>{{ x := 5; x * 2 }}</p>

<!-- Correct: Use function -->
<p>{{ state.Calculate() }}</p>
```

### No variable declarations

```html
<!-- Wrong: Can't declare variables in templates -->
<p>{{ var total = state.Price * state.Qty; total }}</p>

<!-- Correct: Return from Render() -->
<p>{{ state.Total }}</p>
```

### No loops in expressions

```piko
<!-- Wrong: Use p-for directive instead -->
<p>{{ for _, item := range state.Items { } }}</p>

<!-- Correct: Use p-for -->
<div p-for="(_, item) in state.Items">
  {{ item.Name }}
</div>
```

## Cheat sheet

| Feature | Syntax | Example |
|---------|--------|---------|
| Interpolation | `{{ expression }}` | `{{ state.Name }}` |
| Ternary | `condition ? yes : no` | `{{ state.Active ? "On" : "Off" }}` |
| Optional Chain | `?.` | `{{ state.User?.Name }}` |
| Optional Array | `?.[index]` | `{{ state.Items?.[0] }}` |
| Nullish Coalescing | `??` | `{{ state.Value ?? "default" }}` |
| Template Literal | `` `text ${expr}` `` | `` `Hello ${state.Name}` `` |
| Attribute Bind | `:attr="expr"` | `:class="state.Theme"` |
| Arithmetic | `+`, `-`, `*`, `/`, `%` | `{{ state.Price * 1.10 }}` |
| Comparison | `==`, `!=`, `<`, `>`, `<=`, `>=` | `{{ state.Count > 0 }}` |
| Loose Equality | `~=`, `!~=` | `{{ state.Value ~= "5" }}` |
| Truthiness | `~` | `{{ ~state.Value }}` |
| Logical | `&&`, `\|\|`, `!` | `{{ state.A && state.B }}` |
| Function Call | `func()` | `{{ state.Format() }}` |
| Decimal | `99.99d` | `{{ state.Price > 100d }}` |
| Date | `d'YYYY-MM-DD'` | `{{ d'2025-01-15' }}` |
| Time | `t'HH:mm:ss'` | `{{ t'15:04:05' }}` |
| DateTime | `dt'RFC3339'` | `{{ dt'2025-01-15T10:00:00Z' }}` |
| Duration | `du'...'` | `{{ du'1h30m' }}` |
| BigInt | `123n` | `{{ 12345678901234567890n }}` |
| Rune | `r'x'` | `{{ r'🚀' }}` |


## See also

- [Directives reference](directives.md) for the `p-*` directives that use these expressions.
- [Runtime symbols reference](runtime-symbols.md) for every helper callable inside expressions.
- [PK file format reference](pk-file-format.md) for the surrounding file structure.
- [How to conditionals](../how-to/templates/conditionals.md) and [how to loops](../how-to/templates/loops.md) for task recipes.
