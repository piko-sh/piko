# Examples

Use this guide for annotated code examples of common Piko patterns.

## Minimal page (hello world)

```piko
<!-- pages/index.pk -->
<template>
  <h1>{{ state.Greeting }}</h1>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Response struct {
    Greeting string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{Greeting: "Hello, World!"}, piko.Metadata{
        Title: "Home",
    }, nil
}
</script>
```

**Key points**: `package main`, Response struct, `piko.NoProps` for pages, `state.` prefix in template.

## Page with data fetching

```piko
<!-- pages/blog/{slug}.pk -->
<template>
  <piko:partial is="layout" :server.page_title="state.Post.Title">
    <article>
      <h1>{{ state.Post.Title }}</h1>
      <p>{{ state.Post.Body }}</p>
    </article>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    "myapp/pkg/domain"
    layout "myapp/partials/layout.pk"
)

type Response struct {
    Post domain.Post
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    slug := r.PathParam("slug")

    post, err := domain.GetPostBySlug(slug)
    if err != nil {
        return Response{}, piko.Metadata{}, piko.NotFound("post", slug)
    }

    return Response{Post: post}, piko.Metadata{Title: post.Title}, nil
}
</script>
```

**Key points**: Dynamic route `{slug}`, `r.PathParam("slug")`, typed `piko.NotFound` error to trigger `!404.pk`, partial import via `<piko:partial is="layout">`.

## Partial with props and slots (layout)

```piko
<!-- partials/layout.pk -->
<template>
  <html>
  <head>
    <title>{{ state.PageTitle }}</title>
  </head>
  <body>
    <piko:partial is="nav" :server.current_page="state.CurrentPage"></piko:partial>
    <main>
      <piko:slot />
    </main>
    <footer>
      <piko:slot name="footer">
        <p>Default footer</p>
      </piko:slot>
    </footer>
  </body>
  </html>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    nav "myapp/partials/nav.pk"
)

type Props struct {
    PageTitle   string `prop:"page_title"`
    CurrentPage string `prop:"current_page" default:"home"`
}

type Response struct {
    PageTitle   string
    CurrentPage string
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    return Response{
        PageTitle:   props.PageTitle,
        CurrentPage: props.CurrentPage,
    }, piko.Metadata{}, nil
}
</script>
```

**Key points**: Props struct with `prop:` tags, default values, `<piko:slot />` for default slot, `<piko:slot name="footer">` with fallback content, nested partial import.

## Usage of the layout partial

```piko
<!-- pages/about.pk -->
<template>
  <piko:partial is="layout" :server.page_title="'About Us'" :server.current_page="'about'">
    <h1>About Us</h1>
    <p>We build amazing things.</p>

    <piko:slot name="footer">
      <p>Custom about page footer</p>
    </piko:slot>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    layout "myapp/partials/layout.pk"
)

func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
    return piko.NoResponse{}, piko.Metadata{Title: "About Us"}, nil
}
</script>
```

**Key points**: `:server.` prefix for passing props, `piko.NoResponse` when page has no data, named slot override.

## PKC counter component

```piko
<!-- components/pp-counter.pkc -->
<template name="pp-counter">
  <div class="counter">
    <button p-on:click="decrement">-</button>
    <span p-class="{ negative: state.count < 0 }">{{ state.count }}</span>
    <button p-on:click="increment">+</button>
  </div>
</template>

<script lang="ts">
const state = {
    count: 0 as number,
    step: 1 as number,
};

function increment() {
    state.count += state.step;
}

function decrement() {
    state.count -= state.step;
}
</script>

<style>
.counter {
    display: flex;
    align-items: center;
    gap: 0.5rem;
}
button {
    padding: 0.25rem 0.75rem;
    cursor: pointer;
}
.negative { color: red; }

:host([step]) .counter {
    border: 1px solid #ccc;
    padding: 0.5rem;
}
</style>
```

**Key points**: `name="pp-counter"` on `<template>` (script `name` is ignored), `as number` annotation, snake_case state, `:host([step])` attribute sync, Shadow DOM styles.

Usage in a PK page:

```piko
<pp-counter></pp-counter>
<pp-counter step="5"></pp-counter>
```

## Form with server action

```piko
<!-- pages/contact.pk -->
<template>
  <piko:partial is="layout" :server.page_title="'Contact'">
    <h1>Contact Us</h1>

    <form p-on:submit.prevent="action.contact.Submit($form).call()">
      <div>
        <label for="name">Name</label>
        <input id="name" name="name" type="text" required />
      </div>

      <div>
        <label for="email">Email</label>
        <input id="email" name="email" type="email" required />
      </div>

      <div>
        <label for="message">Message</label>
        <textarea id="message" name="message" rows="5" required></textarea>
      </div>

      <button type="submit">Send Message</button>
    </form>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    layout "myapp/partials/layout.pk"
)

func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
    return piko.NoResponse{}, piko.Metadata{Title: "Contact"}, nil
}
</script>
```

The action (`actions/contact/submit.go`):

```go
package contact

import (
    "fmt"
    "piko.sh/piko"
)

type ContactResponse struct {
    OK bool `json:"ok"`
}

type SubmitAction struct {
    piko.ActionMetadata
}

func (a SubmitAction) Call(name string, email string, message string) (ContactResponse, error) {
    err := sendEmail(email, name, message)
    if err != nil {
        a.Response().AddHelper("showToast", "Could not send message.", "error")
        return ContactResponse{}, fmt.Errorf("sending email: %w", err)
    }

    a.Response().AddHelper("showToast", "Message sent!", "success")
    return ContactResponse{OK: true}, nil
}
```

Run `go run ./cmd/generator/main.go all` after creating new action files.

**Key points**: `action.<package>.<StructNameMinusActionSuffix>` (here `action.contact.Submit`), `$form` binds input `name` to `Call` parameters, `piko.ActionMetadata` embed, `showToast` helper, generator auto-registers.

## Product list with loops and conditionals

```piko
<!-- pages/products.pk -->
<template>
  <piko:partial is="layout" :server.page_title="'Products'">
    <h1>Products</h1>

    <div p-if="len(state.Products) == 0">
      <p>No products found.</p>
    </div>

    <div p-else class="product-grid">
      <div p-for="product in state.Products" p-key="product.ID" class="product">
        <h2 p-text="product.Name"></h2>
        <p>Price: {{ product.Price }}</p>
        <span p-if="product.InStock" class="badge-success">In Stock</span>
        <span p-else class="badge-danger">Out of Stock</span>
        <button
          p-on:click="action.cart.Add({id: product.ID}).call()"
          :disabled="!product.InStock"
        >
          Add to Cart
        </button>
      </div>
    </div>
  </piko:partial>
</template>
```

**Key points**: `p-for` with `p-key`, `p-if`/`p-else` chain, `p-text` for safe text, `:disabled` boolean binding, action with dynamic argument.

## Collection page (blog index)

```piko
<!-- pages/blog/index.pk -->
<template>
  <piko:partial is="layout" :server.page_title="'Blog'">
    <h1>Blog</h1>
    <article p-for="post in state.Posts" p-key="post.Slug">
      <h2><a :href="`/blog/${post.Slug}`">{{ post.Title }}</a></h2>
      <time>{{ post.Date }}</time>
    </article>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    layout "myapp/partials/layout.pk"
)

type Post struct {
    Title string
    Slug  string
    Date  string
}

type Response struct {
    Posts []Post
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    items, err := piko.GetAllCollectionItems("blog")
    if err != nil {
        return Response{}, piko.Metadata{}, err
    }

    posts := make([]Post, 0, len(items))
    for _, item := range items {
        posts = append(posts, Post{
            Title: stringField(item, "Title"),
            Slug:  stringField(item, "Slug"),
            Date:  stringField(item, "Date"),
        })
    }

    return Response{Posts: posts}, piko.Metadata{Title: "Blog"}, nil
}

func stringField(item map[string]any, key string) string {
    if value, ok := item[key]; ok {
        if str, ok := value.(string); ok {
            return str
        }
    }
    return ""
}
</script>
```

**Key points**: `piko.GetAllCollectionItems` returns `[]map[string]any` - convert to typed struct via a helper; template literals in `:href`; `p-for` with `p-key`.

## Related

- `references/pk-file-format.md` - full .pk file structure
- `references/template-syntax.md` - directives and expressions
- `references/server-actions.md` - action struct and response helpers
- `references/partials-and-slots.md` - partial props and slots
- `references/collections.md` - collection setup and markdown content
