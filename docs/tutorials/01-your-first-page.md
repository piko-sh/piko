---
title: Your first page
description: Build an About page with dynamic data, styling, and conditional rendering.
nav:
  sidebar:
    section: "tutorials"
    subsection: "getting-started"
    order: 10
---

# Your first page

Let us build your first Piko page from scratch. In this tutorial, you will create a simple "About" page with dynamic data, styling, and conditional rendering.

<p align="center">
  <img src="../diagrams/tutorial-01-preview.svg"
       alt="Preview of the finished About page in a browser at /about showing a heading, a bio paragraph, a conditional greeting block, and a scoped styled section."
       width="500"/>
</p>

## Prerequisites

Make sure you have:
- Read [Concepts](../get-started/concepts.md) for the Piko vocabulary
- Installed the Piko CLI and scaffolded a project (see [Install and run](../get-started/install.md))
- Started the dev server with `air` or `go run ./cmd/main/main.go dev`

## Step 1: Create the page file

Create a new file at `pages/about.pk`:

```bash
touch pages/about.pk
```

Every `.pk` file has up to five sections (template, Go script, optional TypeScript, style, i18n). For the full grammar see [pk-file format reference](../reference/pk-file-format.md). For the rationale behind the single-file shape see [about PK files](../explanation/about-pk-files.md).

## Step 2: Add a simple template

Open `pages/about.pk` and add this basic template:

```piko
<template>
  <div class="about-page">
    <h1>About Us</h1>
    <p>Welcome to our website!</p>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
    return piko.NoResponse{}, piko.Metadata{
        Title: "About Us | MyApp",
    }, nil
}
</script>
```

> **Note:** Both `type="application/x-go"` and `lang="go"` work for the Go script tag. We recommend `application/x-go` because more IDEs colour it by default. The entry function must use the exact name `Render`. The compiler discovers pages by looking for `Render` with the right signature, so renaming it silently breaks the route. See [pk-file format reference](../reference/pk-file-format.md) for the full grammar.

**Visit**: `http://localhost:8080/about`

You should see your page.

> **Note**: Piko's dev server automatically reloads when you save changes.

## Step 3: Add dynamic data

Let us make the page dynamic by defining a `Response` struct. Update the `<script>` section:

```piko
<script type="application/x-go">
package main

import "piko.sh/piko"

type Response struct {
    CompanyName string
    Founded     int
    TeamSize    int
    Mission     string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{
        CompanyName: "Acme Corporation",
        Founded:     2020,
        TeamSize:    12,
        Mission:     "Building amazing web applications with Piko",
    }, piko.Metadata{
        Title: "About Acme | MyApp",
    }, nil
}
</script>
```

Now update the `<template>` to use this data:

```html
<template>
  <div class="about-page">
    <h1>About {{ state.CompanyName }}</h1>
    <p>{{ state.Mission }}</p>

    <div class="stats">
      <div class="stat">
        <strong>Founded: </strong>{{ state.Founded }}
      </div>
      <div class="stat">
        <strong>Team size: </strong>{{ state.TeamSize }} people
      </div>
    </div>
  </div>
</template>
```

Reload `/about`. The page renders the company name, mission, founding year, and team size. For the full `{{ ... }}` expression grammar see [template syntax reference](../reference/template-syntax.md). For the `Render` signature see [pk-file format reference](../reference/pk-file-format.md#render).

## Step 4: Add scoped styles

Add a `<style>` section to style your page:

```html
<style>
.about-page {
    max-width: 800px;
    margin: 0 auto;
    padding: 2rem;
    font-family: system-ui, -apple-system, sans-serif;
    h1 {
        color: #6F47EB;
        font-size: 2.5rem;
        margin-bottom: 1rem;
    }
    p {
        font-size: 1.125rem;
        line-height: 1.7;
        color: #374151;
        margin-bottom: 2rem;
    }
    .stats {
        display: flex;
        gap: 2rem;
        margin-top: 2rem;
        .stat {
            background: #f3f4f6;
            padding: 1.5rem;
            border-radius: 0.5rem;
            flex: 1;
            text-align: center;
            strong {
                display: block;
                color: #6b7280;
                font-size: 0.875rem;
                text-transform: uppercase;
                letter-spacing: 0.05em;
                margin-bottom: 0.5rem;
            }
            span {
                display: block;
                color: #1f2937;
                font-size: 1.5rem;
                font-weight: 600;
            }
        }
    }
}
</style>
```

The snippet uses natively nested CSS, which browsers have supported since mid-2023.

Piko scopes styles to the component automatically. A `.box` in one partial does not affect a `.box` in another. For the full scoping model, including `:deep()` and `<style global>`, see [how to scope and bridge component CSS](../how-to/templates/scoped-css.md).

## Step 5: Add conditional rendering

Let us show different content based on data. Update your Response:

```go
type Response struct {
    CompanyName string
    Founded     int
    TeamSize    int
    Mission     string
    IsHiring    bool
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{
        CompanyName: "Acme Corporation",
        Founded:     2020,
        TeamSize:    12,
        Mission:     "Building amazing web applications with Piko",
        IsHiring:    true,
    }, piko.Metadata{
        Title: "About Acme | MyApp",
    }, nil
}
```

Update the template to show a hiring message conditionally:

```piko
<template>
  <div class="about-page">
    <h1>About {{ state.CompanyName }}</h1>
    <p>{{ state.Mission }}</p>

    <!-- Conditional rendering with p-if -->
    <div p-if="state.IsHiring" class="hiring-banner">
      <strong>We're hiring!</strong>
      <p>Join our growing team of {{ state.TeamSize }} talented people.</p>
      <piko:a href="/careers" class="button">View Open Positions</piko:a>
    </div>
    <div p-else class="not-hiring">
      <p>We're not currently hiring, but check back soon!</p>
    </div>

    <div class="stats">
      <div class="stat">
        <strong>Founded</strong>
        <span>{{ state.Founded }}</span>
      </div>
      <div class="stat">
        <strong>Team Size</strong>
        <span>{{ state.TeamSize }} people</span>
      </div>
    </div>
  </div>
</template>
```

Add styles for the banner:

```css
.hiring-banner {
    background: linear-gradient(135deg, #6F47EB 0%, #8B5CF6 100%);
    color: white;
    padding: 1.5rem;
    border-radius: 0.5rem;
    margin-bottom: 2rem;
    text-align: center;
    strong {
        display: block;
        font-size: 1.25rem;
        margin-bottom: 0.5rem;
    }
    p {
        color: rgba(255, 255, 255, 0.9);
        margin-bottom: 1rem;
    }
    .button {
        display: inline-block;
        background: white;
        color: #6F47EB;
        padding: 0.75rem 1.5rem;
        border-radius: 0.375rem;
        text-decoration: none;
        font-weight: 600;
        transition: transform 0.2s;
        &:hover {
            transform: translateY(-2px);
        }
    }
}
.not-hiring {
    background: #f3f4f6;
    padding: 1.5rem;
    border-radius: 0.5rem;
    margin-bottom: 2rem;
    text-align: center;
}
```

Change `IsHiring: true` to `false` and watch the banner change. The `p-else` directive must immediately follow a `p-if` element.

> **Note**: `<piko:a>` is a meta element, it actually renders as `<a>`. If you ever see an element with the name `<piko:xxx>` it will always render out as the `xxx`, but with special behaviour injected into it. So if you want to match it in CSS or JS you just use `xxx`, or in this case, `a`. In this scenario, `<a>` injects custom logic onto interaction to allow soft-navigation on the frontend, to give a SPA feeling.

## Step 6: Add a list

Let us show team members. Update your Response:

```go
type TeamMember struct {
    Name string
    Role string
}

type Response struct {
    CompanyName string
    Founded     int
    TeamSize    int
    Mission     string
    IsHiring    bool
    Leaders     []TeamMember
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{
        CompanyName: "Acme Corporation",
        Founded:     2020,
        TeamSize:    12,
        Mission:     "Building amazing web applications with Piko",
        IsHiring:    true,
        Leaders: []TeamMember{
            {Name: "Alice Williams", Role: "CEO"},
            {Name: "Bob Smith", Role: "CTO"},
            {Name: "Carol Davis", Role: "Head of Design"},
        },
    }, piko.Metadata{
        Title: "About Acme | MyApp",
    }, nil
}
```

Add this section to your template (below the `.stats` section):

```piko
<div class="team-section">
  <h2>Leadership Team</h2>
  <div class="team-grid">
    <!-- Loop with p-for: (index, item) or item in collection -->
    <div p-for="member in state.Leaders" class="team-member">
      <h3>{{ member.Name }}</h3>
      <p>{{ member.Role }}</p>
    </div>
  </div>
</div>
```

The `p-for` directive uses the syntax `(index, item) in collection`.

Add styles:

```css
.team-section {
    margin-top: 3rem;
    h2 {
        font-size: 1.75rem;
        color: #1f2937;
        margin-bottom: 1.5rem;
    }
    .team-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 1.5rem;
        .team-member {
            background: white;
            border: 1px solid #e5e7eb;
            padding: 1.5rem;
            border-radius: 0.5rem;
            text-align: center;
            h3 {
                font-size: 1.125rem;
                color: #1f2937;
                margin: 0 0 0.5rem 0;
            }
            p {
                color: #6b7280;
                font-size: 0.875rem;
                margin: 0;
            }
        }
    }
}
```

> **Note**: The order in the for-loop differs from what you might expect if you come from JavaScript tooling. We use (index, item) to match Go conventions.

## Step 7: Use a layout partial

Most pages share a common layout (header, footer, etc.). Let us create a reusable partial.

First, create `partials/layout.pk`:

```piko
<template>
  <div class="app-layout">
    <header class="header">
      <nav>
        <piko:a href="/">Home</piko:a>
        <piko:a href="/about">About</piko:a>
        <piko:a href="/contact">Contact</piko:a>
      </nav>
    </header>

    <main class="content">
      <!-- This is where the page content goes -->
      <piko:slot />
    </main>

    <footer class="footer">
      <p>&copy; 2025 {{ state.CompanyName }}. All rights reserved.</p>
    </footer>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Props struct {
    CompanyName string `prop:"company_name"`
}

type Response struct {
    CompanyName string
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    return Response{
        CompanyName: props.CompanyName,
    }, piko.Metadata{}, nil
}
</script>

<style>
.app-layout {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    font-family: system-ui, -apple-system, sans-serif;
    .header {
        background: #1f2937;
        padding: 1rem 2rem;
        nav {
            display: flex;
            gap: 1.5rem;
            max-width: 1200px;
            margin: 0 auto;
            a {
                color: white;
                text-decoration: none;
                font-weight: 500;
                transition: color .2s;
                &:hover {
                   color: #A194CC;
                }
            }
        }
    }
    .content {
        flex: 1;
    }
    .footer {
        background: #f3f4f6;
        padding: 2rem;
        text-align: center;
        color: #6b7280;
    }
}
</style>
```

Now update your about page to use the layout:

```piko
<template>
  <!-- Wrap your content with the layout partial -->
  <piko:partial is="layout" :server.company_name="state.CompanyName">
    <div class="about-page">
      <h1>About {{ state.CompanyName }}</h1>
      <!-- rest of your content -->
    </div>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    layout "myapp/partials/layout.pk"
)

// Rest of your script...
</script>
```

> **Key points**:
> - Import the partial with `layout "myapp/partials/layout.pk"` (replace `myapp` with your module name from `go.mod`)
> - Use `<piko:partial is="layout">` to include it. The `is` attribute uses the import alias
> - Pass props with `:server.prop_name="expression"`. The prop name matches the `prop:` tag in the Props struct
> - Your content goes inside and replaces `<piko:slot />`
>
> By default, props are also passed into the frontend as attributes. However, if you add `server.` before, you are saying this prop is server-side only.

## Step 8: Mark the active navigation link

Update your layout to show the current page. Modify `partials/layout.pk`:

```piko
<template>
  <div class="app-layout">
    <header class="header">
      <nav>
        <piko:a href="/" :class="state.CurrentPage == 'home' ? 'active' : ''">Home</piko:a>
        <piko:a href="/about" :class="state.CurrentPage == 'about' ? 'active' : ''">About</piko:a>
        <piko:a href="/contact" :class="state.CurrentPage == 'contact' ? 'active' : ''">Contact</piko:a>
      </nav>
    </header>

    <main class="content">
      <piko:slot />
    </main>

    <footer class="footer">
      <p>&copy; 2025 {{ state.CompanyName }}. All rights reserved.</p>
    </footer>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Props struct {
    CompanyName string `prop:"company_name"`
    CurrentPage string `prop:"current_page"`
}

type Response struct {
    CompanyName string
    CurrentPage string
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    return Response{
        CompanyName: props.CompanyName,
        CurrentPage: props.CurrentPage,
    }, piko.Metadata{}, nil
}
</script>

<style>
/* ... previous styles ... */

.header a.active {
    color: #A194CC;
    border-bottom: 2px solid #6F47EB;
}
</style>
```

Update your about page to pass `CurrentPage`:

```html
<layout is="layout" :server.company_name="state.CompanyName" :server.current_page="'about'">
  <!-- content -->
</layout>
```

The expression uses `==` for strict comparison. Piko does not support `===`.

## Complete example

Here's the full `pages/about.pk`:

```piko
<template>
  <piko:partial is="layout" :server.company_name="state.CompanyName" :server.current_page="'about'">
    <div class="about-page">
      <h1>About {{ state.CompanyName }}</h1>
      <p>{{ state.Mission }}</p>

      <!-- Conditional rendering with p-if -->
      <div p-if="state.IsHiring" class="hiring-banner">
        <strong>We're hiring!</strong>
        <p>Join our growing team of {{ state.TeamSize }} talented people.</p>
        <piko:a href="/careers" class="button">View Open Positions</piko:a>
      </div>
      <div p-else class="not-hiring">
        <p>We're not currently hiring, but check back soon!</p>
      </div>

      <div class="stats">
        <div class="stat">
          <strong>Founded</strong>
          <span>{{ state.Founded }}</span>
        </div>
        <div class="stat">
          <strong>Team Size</strong>
          <span>{{ state.TeamSize }} people</span>
        </div>
      </div>

      <div class="team-section">
        <h2>Leadership Team</h2>
        <div class="team-grid">
          <!-- Loop with p-for: (index, item) or item in collection -->
          <div p-for="member in state.Leaders" class="team-member">
            <h3>{{ member.Name }}</h3>
            <p>{{ member.Role }}</p>
          </div>
        </div>
      </div>
    </div>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
	"piko.sh/piko"
	layout "docs-dummy/partials/layout.pk"
)

type TeamMember struct {
    Name string
    Role string
}

type Response struct {
    CompanyName string
    Founded     int
    TeamSize    int
    Mission     string
    IsHiring    bool
    Leaders     []TeamMember
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{
        CompanyName: "Acme Corporation",
        Founded:     2020,
        TeamSize:    12,
        Mission:     "Building amazing web applications with Piko",
        IsHiring:    true,
        Leaders: []TeamMember{
            {Name: "Alice Williams", Role: "CEO"},
            {Name: "Bob Smith", Role: "CTO"},
            {Name: "Carol Davis", Role: "Head of Design"},
        },
    }, piko.Metadata{
        Title: "About Acme | MyApp",
    }, nil
}
</script>

<style>
body {
    margin: 0;
}
.about-page {
    max-width: 800px;
    margin: 0 auto;
    padding: 2rem;
    font-family: system-ui, -apple-system, sans-serif;
    h1 {
        color: #6F47EB;
        font-size: 2.5rem;
        margin-bottom: 1rem;
    }
    p {
        font-size: 1.125rem;
        line-height: 1.7;
        color: #374151;
        margin-bottom: 2rem;
    }
    .stats {
        display: flex;
        gap: 2rem;
        margin-top: 2rem;
        .stat {
            background: #f3f4f6;
            padding: 1.5rem;
            border-radius: 0.5rem;
            flex: 1;
            text-align: center;
            strong {
                display: block;
                color: #6b7280;
                font-size: 0.875rem;
                text-transform: uppercase;
                letter-spacing: 0.05em;
                margin-bottom: 0.5rem;
            }
            span {
                display: block;
                color: #1f2937;
                font-size: 1.5rem;
                font-weight: 600;
            }
        }
    }
}

.hiring-banner {
    background: linear-gradient(135deg, #6F47EB 0%, #8B5CF6 100%);
    color: white;
    padding: 1.5rem;
    border-radius: 0.5rem;
    margin-bottom: 2rem;
    text-align: center;
    strong {
        display: block;
        font-size: 1.25rem;
        margin-bottom: 0.5rem;
    }
    p {
        color: rgba(255, 255, 255, 0.9);
        margin-bottom: 1rem;
    }
    .button {
        display: inline-block;
        background: white;
        color: #6F47EB;
        padding: 0.75rem 1.5rem;
        border-radius: 0.375rem;
        text-decoration: none;
        font-weight: 600;
        transition: transform 0.2s;
        &:hover {
             transform: translateY(-2px);
         }
    }
}
.not-hiring {
    background: #f3f4f6;
    padding: 1.5rem;
    border-radius: 0.5rem;
    margin-bottom: 2rem;
    text-align: center;
}

.team-section {
    margin-top: 3rem;
    h2 {
        font-size: 1.75rem;
        color: #1f2937;
        margin-bottom: 1.5rem;
    }
    .team-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 1.5rem;
        .team-member {
            background: white;
            border: 1px solid #e5e7eb;
            padding: 1.5rem;
            border-radius: 0.5rem;
            text-align: center;
            h3 {
                font-size: 1.125rem;
                color: #1f2937;
                margin: 0 0 0.5rem 0;
            }
            p {
                color: #6b7280;
                font-size: 0.875rem;
                margin: 0;
            }
        }
    }
}
</style>
```

## What we built

We have a working About page that pulls data from a `Response` struct, styles itself with scoped CSS, and swaps between two layouts depending on a boolean. Every change in the template reflected a change in one of three places, the `Response` fields, the `Render` function, or the `<template>` markup.

## Next steps

- How-to: [passing props to partials](../how-to/partials/passing-props.md) to reuse this page's shape across other pages.
- Reference: [PK file format](../reference/pk-file-format.md) for every field on `RequestData` and `Metadata`.
- Next tutorial: [Adding interactivity](02-adding-interactivity.md) to add a client-side counter to the page.
