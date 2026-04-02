---
title: Your first page
description: Create and customise your first Piko page step-by-step
nav:
  sidebar:
    section: "get-started"
    subsection: "basics"
    order: 30
---

# Your first page

Let's build your first Piko page from scratch. In this tutorial, you'll create a simple "About" page with dynamic data, styling, and conditional rendering.

## Prerequisites

Make sure you've:
- Read [Core concepts](/docs/get-started/core-concepts) to understand Piko's mental model
- Installed Piko CLI ([Introduction](/docs/get-started/introduction))
- Created a new project with `piko new myapp`
- Started the dev server with `piko dev`

## Step 1: Create the page file

Create a new file at `pages/about.pk`:

```bash
touch pages/about.pk
```

Every `.pk` file has multiple sections:
1. `<template>` - the HTML structure
2. `<script type="application/x-go">` - the Go code
3. `<script lang="ts">` - the frontend Typescript (optional)
4. `<style>` - the CSS (optional)
5. `<i18n>` - the JSON i18n configuration (optional)

You can do lang="go" for the Go block, or the mime-type for ts/js in the frontend script block. The reason why we use the convention of them being inconsistent like this, is because some IDE's seem to have limited support for application/x-go without the need for plugins. Likewise for lang="ts".

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

**Visit**: `http://localhost:8080/about`

You should see your page!

> **Note**: Piko's dev server automatically reloads when you save changes.

## Step 3: Add dynamic data

Let's make the page dynamic by defining a `Response` struct. Update the `<script>` section:

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

```piko
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

**How it works**:
- The `Response` struct defines what data is available to the template
- Values are returned from the `Render()` function
- In the template, access them with `{{ state.FieldName }}`

## Step 4: Add scoped styles

Add a `<style>` section to style your page:

```piko
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

Note that in this snippet, we are using natively nested CSS (which has been widely supported since about mid-2023). You can use anything that is supported by CSS, but please note that we do not support preprocessors like Sass out of the box at the moment.

> **Tip**: Styles in `.pk` files are automatically scoped to the component. This means `.box` in one partial won't affect `.box` in another, because they are isolated by design. Piko achieves this by adding a unique attribute to elements and transforming CSS selectors to include that attribute.
>
> It is worth noting that styles do not have as strict scoping as VueJS or other frontend frameworks do, as we do not hash id's and classes. This is because we do not 'sit-between' conversations with the DOM, as primarily pk files are server-side rendered. This means, that styling scopes can sometimes bleed due to the HTML attribute referencing an elements 'owners', and we scope styles to point to elements with the same 'owners'. Root elements of a partial confuse this because they technically have multiple 'owners', their own scope, and the scope of the parent, which might want to style its children.
>
> You can use `:deep()` to for-go the strictness scoping on child elements, and style everything below. You can use a `<style global>` to completely ignore the scope system.

## Step 5: Add conditional rendering

Let's show different content based on data. Update your Response:

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

Change `IsHiring: true` to `false` and watch the banner change! The `p-else` directive must immediately follow a `p-if` element.

> **Note**: `<piko:a>` is a meta element, it actually renders as `<a>`. If you ever see an element with the name `<piko:xxx>` it will always render out as the `xxx`, but with special behaviour injected into it. So if you want to match it in CSS or JS you just use `xxx`, or in this case, `a`. In this scenario, `<a>` injects custom logic onto interaction to allow soft-navigation on the frontend, to give a SPA feeling.

## Step 6: Add a list

Let's show team members. Update your Response:

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

```html
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

The `p-for` directive uses the syntax `(index, item) in collection`. If you do not want the index, you can drop it and just loop over the item.

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

> **Note**: The order in the for-loop is different than you might expect if you are used to Javascript tooling. We use (index, item), as this is more aligned with what is expected in Go.

## Step 7: Use a layout partial

Most pages share a common layout (header, footer, etc.). Let's create a reusable partial.

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

## Step 8: Add navigation highlighting

Update your layout to highlight the current page. Modify `partials/layout.pk`:

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

```piko
<layout is="layout" :server.company_name="state.CompanyName" :server.current_page="'about'">
  <!-- content -->
</layout>
```

> **Note**: In the expression you will see `==`; we use `==` for strict comparison. `===` is unsupported in Piko. If you want truthy comparison, you can use `~=`.

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

## Understanding the Render function

Every `.pk` file has a `Render` function with this signature:

```go
func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error)
```

| Parameter | Description |
|-----------|-------------|
| `r *piko.RequestData` | Request context including URL, query params, form data, and locale |
| `props Props` | Properties passed from parent components via `:server.*` attributes |
| `Response` | Your struct containing data accessible in the template as `state.*` |
| `piko.Metadata` | Page metadata like title, description, and caching settings |
| `error` | Return an error to trigger error handling |

For the complete RequestData API and all Metadata fields, see [PK templates](/docs/guide/pk-templates) and [metadata](/docs/guide/metadata).

## Going further: server actions

Server actions handle form submissions and client-triggered operations. They embed `piko.ActionMetadata` and implement a typed `Call` method:

```go
type ContactSubmitAction struct {
    piko.ActionMetadata
}

func (a ContactSubmitAction) Call(name string, email string) (ContactResponse, error) {
    // Process the submission
    a.Response().AddHelper("showToast", "Message sent!", "success")
    return ContactResponse{OK: true}, nil
}
```

Actions are discovered automatically from the `actions/` directory. For the full API including response helpers, cookie management, and error handling, see [server actions](/docs/guide/server-actions).

## Next steps

Now that you've built your first page, dive deeper:

- **[Template syntax](/docs/guide/template-syntax)** → All template expressions and operators
- **[Directives](/docs/guide/directives)** → Complete directive reference (`p-if`, `p-for`, `p-show`, etc.)
- **[Partials](/docs/guide/partials)** → Build reusable components with props and slots
- **[Server actions](/docs/guide/server-actions)** → Type-safe, validated request handling with typed responses
