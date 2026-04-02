---
title: Template syntax
description: Complete guide to PK template expressions and interpolation
nav:
  sidebar:
    section: "guide"
    subsection: "concepts"
    order: 35
---

# Template syntax

Piko's template syntax lets you insert dynamic data, evaluate expressions, and build flexible templates. All templates are processed server-side, generating static HTML sent to the client. For the overall `.pk` file structure, see [PK File Format](/docs/guide/pk-templates).

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

> **Key Point**: Access Response fields with `state.FieldName`. The `state` variable holds your returned Response data.

## Expressions

You can use expressions inside `{{ }}`:

### Arithmetic operators

All standard arithmetic operators are supported: `+`, `-`, `*`, `/`, `%` (modulo).

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

```piko
<template>
  <p>{{ state.FirstName + " " + state.LastName }}</p>
  <p>{{ "Welcome back, " + state.Username + "!" }}</p>
  <!-- Numbers are automatically converted when concatenating with strings -->
  <p>{{ "Age: " + state.Age }}</p>
</template>
```

### Comparison operators

Piko uses **strict equality by default** (`==`, `!=`) with Go-style semantics. For loose equality with type coercion, use `~=` and `!~=`:

```piko
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

```piko
<template>
  <p>Can checkout: {{ state.ItemCount > 0 && state.IsLoggedIn }}</p>
  <p>Show warning: {{ state.Stock < 10 || state.Discontinued }}</p>
  <p>Not verified: {{ !state.EmailVerified }}</p>
</template>
```

**All logical operators**: `&&` (and), `||` (or), `!` (not)

### Nullish coalescing operator

Use `??` to provide a fallback value when the left operand is `nil`:

```piko
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

> **Key Difference**: `||` returns the right value for any falsy left value (0, "", false, nil), whilst `??` only returns the right value when the left is `nil`.

## Function calls

You can call functions from your Response struct:

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

> **Tip**: Use functions for complex formatting, calculations, or any logic that's too complex for inline expressions.

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

```piko
<template>
  <!-- Provide a fallback when optional chaining returns nil -->
  <p>Street: {{ state.User?.Address?.Street ?? "No address on file" }}</p>
  <p>First tag: {{ state.User?.Tags?.[0] ?? "No tags" }}</p>
</template>
```

**Why use optional chaining?**

```piko
<!-- Without optional chaining - can panic! -->
<p>{{ state.User.Address.Street }}</p>  <!-- Panics if User or Address is nil -->

<!-- With optional chaining - safe -->
<p>{{ state.User?.Address?.Street }}</p>  <!-- Returns "" if any part is nil -->
```

## Attribute binding

Use the `:` prefix to bind dynamic values to HTML attributes:

```piko
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

For all binding patterns (boolean attributes, style binding, `p-bind`, `p-class`, `p-style`), see [directives](/docs/guide/directives).

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

Piko supports several built-in literal types for common data:

### Decimal literals

For precise decimal arithmetic (useful for money calculations):

```piko
<template>
  <p>Price: {{ 99.99d }}</p>
  <p>Total: {{ 10.5d + 2.5d }}</p>
  <p>Is expensive: {{ state.Price > 100d }}</p>
</template>
```

### Date and time literals

```piko
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

```piko
<template>
  <p>Large number: {{ 12345678901234567890n }}</p>
  <p>Sum: {{ 100n + 50n }}</p>
</template>
```

### Rune literals

For single Unicode characters:

```piko
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

**Output** (malicious script is escaped):
```html
<div>
  <p>&lt;script&gt;alert('xss')&lt;/script&gt;</p>
</div>
```

> **Security**: Piko automatically escapes HTML in `{{ }}` interpolations. This prevents XSS attacks by converting `<`, `>`, `&`, `"`, and `'` to HTML entities.

### Rendering raw HTML

If you trust the content and need raw HTML, use the `p-html` directive (see [Directives](/docs/guide/directives)):

```piko
<template>
  <div>
    <!-- Raw HTML - use with caution! -->
    <div p-html="state.TrustedHTML"></div>
  </div>
</template>
```

> **Warning**: Only use `p-html` with content you fully control. Never use it with user input or untrusted sources!

## Common patterns

### Pluralisation

```piko
<template>
  <p>
    You have {{ state.Count }} item{{ state.Count != 1 ? "s" : "" }}
  </p>

  <!-- Or with template literal -->
  <p>{{ `You have ${state.Count} item${state.Count != 1 ? 's' : ''}` }}</p>

  <!-- More complex -->
  <p>
    {{ state.Count == 0 ? "No items" : state.Count == 1 ? "1 item" : `${state.Count} items` }}
  </p>
</template>
```

### Formatting numbers

```piko
<template>
  <div>
    <p>Price: ${{ state.FormatPrice(state.Price) }}</p>
    <p>Discount: {{ state.DiscountPercent }}%</p>
  </div>
</template>

<script type="application/x-go">
package main

import (
    "fmt"

    "piko.sh/piko"
)

type Response struct {
    Price           float64
    DiscountPercent int
    FormatPrice     func(float64) string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{
        Price:           12.345,
        DiscountPercent: 15,
        FormatPrice: func(p float64) string {
            return fmt.Sprintf("%.2f", p)
        },
    }, piko.Metadata{}, nil
}
</script>
```

### Displaying dates

```piko
<template>
  <div>
    <p>Created: {{ state.FormatDate(state.CreatedAt) }}</p>
    <p>Days ago: {{ state.DaysAgo(state.CreatedAt) }}</p>
  </div>
</template>

<script type="application/x-go">
package main

import (
    "time"

    "piko.sh/piko"
)

type Response struct {
    CreatedAt  time.Time
    FormatDate func(time.Time) string
    DaysAgo    func(time.Time) int
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    createdAt := time.Now().AddDate(0, 0, -5)

    return Response{
        CreatedAt: createdAt,
        FormatDate: func(t time.Time) string {
            return t.Format("January 2, 2006")
        },
        DaysAgo: func(t time.Time) int {
            return int(time.Since(t).Hours() / 24)
        },
    }, piko.Metadata{}, nil
}
</script>
```

### Default values

```piko
<template>
  <div>
    <!-- Fallback with ternary -->
    <p>Name: {{ state.Name != "" ? state.Name : "Anonymous" }}</p>

    <!-- Fallback with nullish coalescing -->
    <p>City: {{ state.User?.Address?.City ?? "Unknown" }}</p>

    <!-- Fallback with function -->
    <p>Email: {{ state.GetEmailOrDefault() }}</p>
  </div>
</template>
```

### Status badges

```piko
<template>
  <div>
    <span :class="`badge badge-${state.Status}`">
      {{ state.Status == "active" ? "Active" :
         state.Status == "pending" ? "⏱ Pending" :
         "✗ Inactive" }}
    </span>
  </div>
</template>
```

## Limitations

### No multi-statement expressions

```piko
<!-- Wrong: Multiple statements not allowed -->
<p>{{ x := 5; x * 2 }}</p>

<!-- Correct: Use function -->
<p>{{ state.Calculate() }}</p>
```

### No variable declarations

```piko
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

## Next steps

Now that you understand template syntax, explore:

- [Directives](/docs/guide/directives) → `p-if`, `p-for`, `p-on`, and more
- [Partials](/docs/guide/partials) → Reusable template components

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
