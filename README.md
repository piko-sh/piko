<div align="center">

# Piko

**A website development kit for Go**

Build fast, type-safe, server-rendered websites with smooth client-server interactivity.

<img src="docs/mascot.png" alt="Piko" width="480"/>

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Alpha-orange.svg)]()

[![Go Coverage](https://img.shields.io/badge/Go_Coverage-74%25-yellowgreen?logo=go)](hack/test/test.sh)
[![Frontend Coverage](https://img.shields.io/badge/Frontend_Coverage-92%25-brightgreen?logo=typescript)](frontend/core/)
[![VSCode Plugin Coverage](https://img.shields.io/badge/VSCode_Plugin-44%25-red?logo=visualstudiocode)](plugins/vscode/)
[![IntelliJ Plugin Coverage](https://img.shields.io/badge/IntelliJ_Plugin-32%25-red?logo=intellijidea)](plugins/idea/)

[Getting Started](#getting-started) |
[Documentation](docs/) |
[Examples](#usage) |
[Contributing](CONTRIBUTING.md)

</div>

---

> **Alpha:** Piko is under active development. We are currently focused on stabilising the API design, consolidating the public surface area, and reducing the required client-side JavaScript. Expect breaking changes between releases.  
> Certain interfaces are still experimental or at left-overs from previous iterations of the project which are due to be fully removed by version 0.5.0, which will signify our intended stable API for version 1.0.0 release.

---

## About

Piko is a website development kit for Go. It combines server-side rendering with component-based architecture, covering templating, routing, caching, email, and observability. 

A single import gets you started; pluggable adapters let you swap storage, email, cache, LLM, and other providers without changing domain code.

```go
import "piko.sh/piko"
```

A Piko application is just a Go program you control:

```go
func main() {
    ssr := piko.New(
        persistence.WithDriver(sqlite.New(sqlite.Config{})),
        piko.WithImageTransformer("imaging", imaging.NewProvider(imaging.Config{})),
    )

    // Add your own chi middleware and routes alongside file-based routing
    ssr.AppRouter.Use(myAuthMiddleware)
    ssr.AppRouter.Get("/api/health", healthHandler)

    if err := ssr.Run(piko.RunModeDev); err != nil {
        log.Fatal(err)
    }
}
```

**Why Piko?**

- **Pure Go.** Zero CGO. No Node.js. No external runtime dependencies. Just `go build`.
- **Single import.** One package gives you the full toolkit. No dependency maze.
- **Type-safe end-to-end.** Go structs in your backend flow through to TypeScript types on the frontend, validated at compile time.
- **Test without a browser.** Built-in `ComponentTester` and `ActionTester` let you verify pages and actions at the AST level, without spinning up a server or browser.

Piko grows with your project. Every provider sits behind a clean adapter interface, so you can start with zero-dependency defaults and swap in scalable infrastructure when the time comes, without needing to rewire domain code.

### Features

- **Server-first rendering.** Complete HTML rendered on the server with progressive enhancement.
- **File-based routing.** Routing based on file structure, with full access to the chi router for custom routes.
- **Single file components.** `.pk` files combine Go logic, HTML templates, and scoped CSS in one file.
- **Type-safe actions.** Server actions with natural Go function signatures and generated TypeScript clients.
- **Content collections.** Content model supporting Markdown, CMS APIs, and databases.
- **Image optimisation.** Automatic responsive images with format conversion and CDN support.
- **Email templating.** PML components for responsive email templates.
- **SSE support.** Real-time server-sent event streaming built in.
- **Full-text search.** Search indexing over collections.
- **Multi-level caching.** In-memory and Redis caching with encryption and compression.
- **Observability.** OpenTelemetry integration with structured logging.
- **Pluggable providers.** Swap out storage, email, cache, events, and LLM providers without changing domain code:

```go
ssr := piko.New(
    // Storage: disk, S3, R2, GCS
    piko.WithStorageProvider("s3", s3Provider),

    // Email: SMTP, SES, SendGrid, Postmark
    piko.WithEmailProvider("ses", sesProvider),

    // Cache: in-memory, Redis, Redis Cluster
    piko.WithCacheProvider("redis", redisProvider),

    // LLMs: Anthropic, OpenAI, Gemini, Mistral
    piko.WithLLMProvider("anthropic", anthropicProvider),
)
```

---

## Getting Started

### Prerequisites

- Go 1.26 or later

### Installation

```bash
# Install the Piko CLI
go install piko.sh/piko/cmd/piko@latest

# Run the interactive project wizard
piko

# After the wizard creates your project:
cd my-app
go run ./cmd/generator/main.go all   # Compile .pk templates into typed Go code
air                                   # Start development server with hot reload
```

The generation step compiles `.pk` templates into Go source code. After this, everything is statically typed and there is no runtime template parsing. [air](https://github.com/air-verse/air) is optional; you can also run `go run ./cmd/main/main.go dev` directly.

### Project Structure

```text
my-app/
├── actions/            # Server actions
├── cmd/
│   ├── generator/      # Asset compilation entry point
│   └── main/           # Application entry point
├── components/         # Client-side components
├── pages/              # Page templates (.pk files)
│   └── index.pk        # → /
├── partials/           # Reusable layout fragments (.pk files)
├── pkg/                # Your domain logic
├── dist/               # Generated code (do not edit)
├── piko.yaml           # Piko configuration
└── config.json         # Theme configuration (fonts, colours)
```

---

## Usage

### Pages

Pages are `.pk` single file components that combine a Go script block, an HTML template, and scoped CSS. The generator statically analyses the Go types in your project and compiles each template into typed Go code. 

A `p-if` becomes a Go `if`, a `p-for` becomes a Go `for`, and a comparison like `state.Count == 1` compiles to a direct integer comparison because the generator already knows the type. There is no reflection, no `interface{}` boxing, and no runtime type assertion for template expressions. If you rename a field or change a type, you get a compile error, not a runtime panic.

Our priority is to do as much validating and processing as possible at compile time, not runtime.

```html
<!-- pages/customers.pk -->
<template>
  <piko:partial is="layout" :server.page_title="state.Title">
    <h1>{{ state.Title }}</h1>

    <p p-if="state.Total == 0">No customers found.</p>

    <ul p-else>
      <li p-for="(i, customer) in state.Customers"
          p-key="customer.ID"
          :class="{ 'highlight': customer.IsVIP }">
        {{ customer.Name }} - {{ customer.Email }}
        <button p-on:click="action.customer.delete(customer.ID)">Remove</button>
      </li>
    </ul>

    <p>Showing {{ state.Total }} customers</p>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    layout "my-app/partials/layout.pk"
)

type Customer struct {
    ID    int64
    Name  string
    Email string
    IsVIP bool
}

type Response struct {
    Title     string
    Customers []Customer
    Total     int
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    customers := fetchCustomers(r.Context())
    return Response{
        Title:     "Customers",
        Customers: customers,
        Total:     len(customers),
    }, piko.Metadata{}, nil
}
</script>

<style>
.highlight { background-color: #fef3c7; }
</style>
```

The `Render` function runs on the server. Its return value is available in the template as `state`, and the `Metadata` struct controls SEO tags, caching, and other page-level behaviour.

The `layout "my-app/partials/layout.pk"` import looks unusual, but the generator rewrites it into a standard Go import of the layout's generated package. This means you can import public types and functions from other `.pk` files the same way you would between normal Go packages.

### Directives

Piko templates use directives for dynamic rendering. Here are the most commonly used:

| Directive | Shorthand | Purpose | Example |
|-----------|-----------|---------|---------|
| `p-if` | | Conditional rendering | `<div p-if="state.IsActive">` |
| `p-for` | | Iteration | `<li p-for="(i, item) in state.Items">` |
| `p-bind:attr` | `:attr` | Dynamic attribute | `<a :href="state.URL">` |
| `p-on:event` | `@event` | Event handler | `<button @click="handleClick()">` |
| `p-model` | | Two-way binding | `<input p-model="form.name">` |
| `p-text` | | Text content | `<span p-text="state.Name">` |

Piko also supports `p-else-if`, `p-else`, `p-show`, `p-html`, `p-class`, `p-style`, `p-key`, `p-ref`, `p-slot`, formatting functions (`F()`, `LF()`), and event modifiers (`.prevent`, `.stop`, `.once`, `.capture`). See the [full directive reference](docs/) for details.

### Actions

Actions are type-safe server functions callable from the frontend. Define a struct, embed `ActionMetadata`, and write a `Call` method with any signature. Piko generates the dispatch code and TypeScript client types automatically:

```go
type CreateCustomerAction struct {
    piko.ActionMetadata
}

type CreateCustomerResponse struct {
    ID   int64
    Name string
}

func (a CreateCustomerAction) Call(name string, email string) (CreateCustomerResponse, error) {
    ctx := a.Ctx()

    // Access session, headers, etc.
    userID := a.Request().Session.UserID

    // Your business logic here...

    // Trigger a client-side helper (e.g. show a toast notification)
    a.Response().AddHelper("showToast", "Customer created", "success")

    return CreateCustomerResponse{ID: 42, Name: name}, nil
}
```

Actions support multiple transports (HTTP, SSE), structured error types that map to HTTP status codes (`ValidationError` -> 422, `NotFoundError` -> 404, `ForbiddenError` -> 403), file uploads, and per-action middleware.

### Collections

Define content collections for blogs, documentation, or any structured content:

```html
<template p-collection="posts">
  <article>
    <h1>{{ state.Title }}</h1>
    <time>{{ state.Date.Format "2 January 2006" }}</time>
    <div p-html="state.Content"></div>
  </article>
</template>
```

### Email templates

Create responsive emails using PML (Piko Markup Language) elements. Email templates are `.pk` files with the same structure as pages (`<template>`, `<style>`, and `<script>` blocks) but using PML elements (`pml-row`, `pml-col`, `pml-p`, `pml-button`, `pml-img`, etc.) for Outlook-compliant output:

```html
<!-- emails/welcome.pk -->
<template>
  <pml-row class="header">
    <pml-col>
      <pml-img src="@/lib/email/logo.png" width="180px" alt="My App"></pml-img>
    </pml-col>
  </pml-row>

  <pml-row>
    <pml-col>
      <pml-p class="greeting">Welcome, {{ props.Name }}!</pml-p>
      <pml-p>Click the button below to activate your account.</pml-p>
      <pml-button :href="props.ActivationURL" background-color="#007bff">
        Activate Account
      </pml-button>
    </pml-col>
  </pml-row>
</template>

<style>
.header { background-color: #f5f5f5; }
.greeting { font-size: 20px; font-weight: bold; }
</style>

<script type="application/x-go">
package main

import "piko.sh/piko"

type WelcomeProps struct {
    Name          string
    ActivationURL string
}

func Render(r *piko.RequestData, props WelcomeProps) (piko.NoResponse, piko.Metadata, error) {
    return piko.NoResponse{}, piko.Metadata{Title: "Welcome!"}, nil
}
</script>
```

PML elements provide a lenient email layout system. Loose content is automatically wrapped in the required `pml-row`/`pml-col` structure, so you can write flat markup without boilerplate. PML elements produce Outlook-compliant HTML with VML fallbacks.

---

## Configuration

### piko.yaml

```yaml
network:
  port: "8080"
  publicDomain: "localhost:8080"
  forceHttps: false

build:
  defaultServeMode: "render"

paths:
  baseDir: "."
  pagesSourceDir: "pages"

assets:
  image:
    profiles:
      web-hero:
        - capability: image-transform
          params:
            width: "1200"
            format: "webp"
            quality: "85"
```

### Environment overrides

- `piko-dev.yaml` - Development configuration
- `piko-prod.yaml` - Production configuration

---

## IDE Support

Piko includes IDE plugins and a language server that bring full Go type awareness into `.pk` templates. Because the generator statically analyses your Go types, the IDE can offer the same level of support you expect in pure Go files, hover documentation, completions, and real-time diagnostics, directly inside your HTML templates.

- **VS Code** - Syntax highlighting, completions, and diagnostics ([plugin](plugins/vscode/))
- **JetBrains** - Full support for JetBrains IDEs ([plugin](plugins/idea/))
- **Language Server** - LSP implementation for any compatible editor

The LSP currently requires around 512MB to 1024MB of RAM.  
If you are using JetBrains please make sure [LSP4IJ](https://plugins.jetbrains.com/plugin/23257-lsp4ij) is installed for full integration, ensure latest version.  
For ideal support for JetBrains IDE's you need either IntelliJ IDEA or ideally Goland. This is because we embed their language system to power the Go support.

### Hover documentation

Hover over `props` or `state` references in a template to see the full Go struct definition, including field names, types, and struct tags.

<img src="docs/ide-1.png" alt="IDE hover popup showing the Go struct definition for a pagination component's Props type, with fields CurrentPage, TotalPages, TotalItems, ItemsPerPage, BasePath, and QueryParams" width="830"/>

### Autocomplete

Get context-aware completions for all available properties, with their types displayed inline.

<img src="docs/ide-2.png" alt="IDE autocomplete dropdown listing available props fields - BasePath (string), CurrentPage (int), ItemsPerPage (int), QueryParams (string), TotalItems (int), TotalPages (int)" width="830"/>

### Property validation

Reference a property that does not exist and the IDE flags it.

<img src="docs/ide-3.png" alt="IDE error tooltip showing 'Property DoesNotExist does not exist on type partials_pagination Props' when accessing an invalid property in a template" width="830"/>

### Type checking

Type mismatches in template expressions are displayed proactively are caught at compile-time.  
Here, comparing a `string` field to an `int` field produces an inline diagnostic.

<img src="docs/ide-4.png" alt="IDE error tooltip showing 'Invalid operation: cannot strictly compare type string to int' when comparing props.BasePath to props.TotalPages in a template expression" width="830"/>

---

## Contributing

Contributions are welcome. Please read the [Contributing Guide](CONTRIBUTING.md) for details on our development process, how to submit pull requests, and coding standards.

---

## License

Distributed under the Apache 2.0 License. See [LICENSE](LICENSE) for more information.
# piko
