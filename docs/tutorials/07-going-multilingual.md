---
title: Going multilingual
description: Add English and French to the blog from tutorial 04. Locale files, prefix routing, translated metadata, and locale-scoped content.
nav:
  sidebar:
    section: "tutorials"
    subsection: "getting-started"
    order: 70
---

# Going multilingual

In this tutorial we will add French to the blog from [Shipping a real site](04-shipping-a-real-site.md). Visitors will reach `/about` in English and `/fr/about` in French. The navigation, the metadata, and the post bodies all localise. By the end the site is bilingual, and adding a third language is a five-minute job.

<p align="center">
  <img src="../diagrams/tutorial-07-preview.svg"
       alt="Preview of the finished bilingual site: two browser windows side by side. The left shows example.com/about in English with an EN badge, an English heading, and English body copy. The right shows example.com/fr/about in French with an FR badge, a French heading, and translated body. Each window has a language switcher pointing to the other locale."
       width="500"/>
</p>

You should have completed [Shipping a real site](04-shipping-a-real-site.md) first. The querier and testing tutorials are not prerequisites.

## Step 1: Create global locale files

Global translations live under `i18n/` at the project root. Create `i18n/en.json`:

```json
{
  "nav": {
    "home": "Home",
    "about": "About",
    "blog": "Blog"
  },
  "footer": {
    "built_with": "Built with Piko."
  },
  "site": {
    "title": "MyBlog",
    "tagline": "Notes on web, code, and coffee."
  },
  "signup": {
    "label": "Get new posts by email",
    "submit": "Subscribe",
    "success": "Subscribed. Thanks for signing up.",
    "invalid_email": "Enter a valid email address."
  }
}
```

And `i18n/fr.json`:

```json
{
  "nav": {
    "home": "Accueil",
    "about": "À propos",
    "blog": "Blog"
  },
  "footer": {
    "built_with": "Propulsé par Piko."
  },
  "site": {
    "title": "MonBlog",
    "tagline": "Notes sur le web, le code et le café."
  },
  "signup": {
    "label": "Recevoir les nouveaux articles par email",
    "submit": "S'abonner",
    "success": "Inscription réussie. Merci !",
    "invalid_email": "Entrez une adresse email valide."
  }
}
```

Nested JSON keys flatten to dot notation at runtime. `nav.home` becomes the lookup key.

## Step 2: Declare the locales in config.json

Add an `i18n` section to the project's `config.json`:

```json
{
  "name": "MyBlog",
  "i18n": {
    "defaultLocale": "en",
    "strategy": "prefix_except_default",
    "locales": ["en", "fr"]
  }
}
```

The three fields matter as follows:

- `defaultLocale` is the locale served when the URL does not select one explicitly.
- `strategy: prefix_except_default` maps `/about` to English and `/fr/about` to French. Other strategies exist for subdomain routing or always-prefixed URLs; see [how to i18n routing strategy](../how-to/i18n/routing-strategy.md).
- `locales` lists every supported locale. Each code needs a matching `i18n/<code>.json` file.

## Step 3: Translate the layout

Update `partials/layout.pk`. Replace hardcoded strings with `T()` calls and swap the nav hrefs for `piko:a` tags so they pick up the current locale:

```piko
<template>
  <!DOCTYPE html>
  <html :lang="r.Locale()">
    <head>
      <meta charset="UTF-8" />
      <meta name="viewport" content="width=device-width, initial-scale=1" />
      <title>{{ state.PageTitle }}</title>
      <meta name="description" :content="state.PageDescription" />
    </head>
    <body>
      <header class="site-header">
        <piko:a href="/" class="brand">{{ T("site.title") }}</piko:a>
        <nav>
          <piko:a href="/">{{ T("nav.home") }}</piko:a>
          <piko:a href="/about">{{ T("nav.about") }}</piko:a>
          <piko:a href="/blog">{{ T("nav.blog") }}</piko:a>
        </nav>
      </header>

      <main>
        <piko:slot />
      </main>

      <footer>
        <p>{{ T("footer.built_with") }}</p>
        <piko:slot name="footer" />
      </footer>
    </body>
  </html>
</template>
```

`T("key")` is the global translation helper. It returns a `*Translation` builder that the template renders by calling `String()` automatically (it implements `fmt.Stringer`). The `piko:a` tag rewrites the href for the active locale: a click on the "Accueil" link on `/fr/about` goes to `/fr/`, not `/`.

Reload `/about` and `/fr/about`. The nav labels and footer text change. The browser's `<html lang="...">` attribute also updates, which helps screen readers announce the page in the right language.

## Step 4: Translate a page from inline i18n

Some strings belong to one page only. Put them inline instead of polluting the global namespace.

Update `pages/about.pk`:

```piko
<template>
  <piko:partial is="layout" :server.page_title="LT('title').String()">
    <article>
      <h1>{{ LT("heading") }}</h1>
      <p>{{ LT("intro") }}</p>
      <p>{{ LT("body") }}</p>
    </article>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    layout "myapp/partials/layout.pk"
)

type Response struct{}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{}, piko.Metadata{
        Title:       r.LT("title").String(),
        Description: r.LT("description").String(),
    }, nil
}
</script>

<i18n lang="json">
{
  "en": {
    "title": "About MyBlog",
    "description": "A short history of the blog.",
    "heading": "About MyBlog",
    "intro": "MyBlog is a small, quiet site about web, code, and coffee.",
    "body": "It ships as a single Go binary. The whole system fits in your head."
  },
  "fr": {
    "title": "À propos de MonBlog",
    "description": "Une courte histoire du blog.",
    "heading": "À propos de MonBlog",
    "intro": "MonBlog est un petit site discret sur le web, le code et le café.",
    "body": "Il se déploie comme un binaire Go unique. Tout le système tient dans votre tête."
  }
}
</i18n>
```

`LT("key")` is the local translation helper. It searches the `<i18n>` block embedded in this PK file. `T("key")` searches the global files first and falls back to the local block; pick `LT` when you want to be explicit that the string is page-scoped.

`r.T(...)` and `r.LT(...)` are the Go-side equivalents. Both return a `*Translation`. Inside templates, `fmt.Stringer` resolves the value to a string automatically. Inside `Render` the `Title` and `Description` fields on `piko.Metadata` are plain strings, so call `.String()` explicitly to terminate the builder.

For the full reference on `T`, `LT`, and the `*Translation` API, see [reference: i18n API](../reference/i18n-api.md).

## Step 5: Translate strings with variables

Some strings have variables. Consider a welcome line on a user dashboard.

In `i18n/en.json`:

```json
{
  "welcome": "Welcome back, ${name}. You have ${count} new messages."
}
```

In `i18n/fr.json`:

```json
{
  "welcome": "Bon retour, ${name}. Vous avez ${count} nouveaux messages."
}
```

Bind variables fluently on the `*Translation` builder:

```piko
<p>
  {{ T("welcome").
       StringVar("name", state.Username).
       IntVar("count", state.MessageCount) }}
</p>
```

Each `*Var` setter takes a typed Go value. The compiler enforces argument types: `IntVar` rejects a string, `MoneyVar` rejects a `float64`. See [how to bind typed variables to translations](../how-to/i18n/variable-binding.md) for the full setter list (`StringVar`, `IntVar`, `FloatVar`, `DecimalVar`, `MoneyVar`, `BigIntVar`, `TimeVar`, `DateTimeVar`).

For pluralisation, declare pipe-separated forms and call `.Count(n)`:

```json
{
  "messages": "no new messages|one new message|${count} new messages"
}
```

```piko
<p>{{ T("messages").Count(state.MessageCount) }}</p>
```

`Count` selects the right plural variant for the active locale using CLDR rules. English uses two forms (singular versus plural). French treats 0 as singular. Russian needs three forms. The framework handles all of this; you write the variants. See [how to pluralise translations](../how-to/i18n/pluralisation.md) for per-language ordering tables.

## Step 6: Localise the blog post content

Blog posts have their own content per language. The markdown collection from tutorial 04 lives under `content/blog/`. Restructure it per locale:

```
content/
  blog/
    en/
      hello-world.md
      deployment.md
    fr/
      hello-world.md
      deployment.md
```

Each file keeps the same frontmatter shape. The filename (`hello-world.md`) becomes the slug in both languages.

The `pages/blog/{slug}.pk` template stays unchanged from tutorial 04:

```piko
<template p-collection="blog">
  <piko:partial
    is="layout"
    :server.page_title="state.Title"
    :server.page_description="state.Description"
  >
    <!-- existing post body -->
  </piko:partial>
</template>
```

> **Note:** The markdown collection driver detects the locale from the path. A directory named after a configured locale (`en/`, `fr/`) tags every file inside it with that locale. At request time, the collection service filters items to match `r.Locale()`, so a request to `/blog/hello-world` reads the English file and `/fr/blog/hello-world` reads the French one. You do not declare the locale in the directive.

The driver also accepts a filename suffix (`hello-world.fr.md`) instead of a locale directory if a flatter layout suits your project. See [routing rules reference](../reference/routing-rules.md) and [about collections](../explanation/about-collections.md) for the collection grammar.

## Step 7: Offer a language switcher

Add a small language switcher to the layout. It links the current page in the other locale.

Extend the footer in `partials/layout.pk`:

```piko
<nav class="lang-switcher">
  <piko:a href="" data-locale="en">English</piko:a>
  <piko:a href="" data-locale="fr">Français</piko:a>
</nav>
```

The `piko:a` tag with an empty `href` and a `data-locale` attribute asks Piko to rewrite the URL to the current path under the named locale. A user on `/fr/about` clicks "English" and lands on `/about`. No JavaScript needed.

## Step 8: Test the localisation

Run the dev server and walk through the checklist:

- Visit `/`. The header reads "MyBlog", the nav reads "Home / About / Blog".
- Visit `/fr/`. The header reads "MonBlog", the nav reads "Accueil / À propos / Blog".
- Visit `/about`. The page title in the browser tab reads "About MyBlog".
- Visit `/fr/about`. The tab reads "À propos de MonBlog".
- View the page source. The `<html lang>` attribute matches.
- View the `<meta name="description">` tag. The text is locale-appropriate.
- Click "Français" on `/about`. You land on `/fr/about`.
- Visit `/blog/hello-world` and `/fr/blog/hello-world`. Each serves the locale-specific post body.

Pikotest covers the rendering. Add a test case per locale:

```go
for _, locale := range []string{"en", "fr"} {
    t.Run(locale, func(t *testing.T) {
        req := piko.NewTestRequest("GET", "/about").
            WithLocale(locale).
            Build()
        view := tester.Render(req, piko.NoProps{})

        switch locale {
        case "en":
            view.QueryAST("h1").HasText("About MyBlog")
        case "fr":
            view.QueryAST("h1").HasText("À propos de MonBlog")
        }
    })
}
```

## Step 9: Add a third language

Repeat the pattern:

1. Add the locale to `config.json`: `"locales": ["en", "fr", "de"]`.
2. Create `i18n/de.json` with translations for every key.
3. Add `content/blog/de/` with the German posts.
4. Redeploy.

Every page using `T` and `LT` picks up German without further code changes. Pages whose markdown lives under `content/blog/de/` gain German posts automatically through the markdown driver's locale detection.

## Common pitfalls

Three gotchas to watch for:

- **Missing keys fall back to the literal key.** A key present in `en.json` but absent in `fr.json` returns the key itself (`"nav.home"`) on a French request, not an empty string. Add a CI lint over `i18n/*.json` to catch divergence, or write a test that walks the JSON trees and asserts keys match.
- **Word order is locale-specific.** The `${name}` placeholder works identically across locales, but word order in the surrounding sentence often differs. Translators rearrange the sentence; do not hard-code the variable position.
- **Metadata fields are strings, not Translations.** The `Title`, `Description`, and `CanonicalURL` fields on `piko.Metadata` come from `Render` and are plain strings. Use `r.T(...).String()` or `r.LT(...).String()` to terminate the builder before assigning.

## Next steps

- [i18n API reference](../reference/i18n-api.md) for the full `T`, `LT`, and `*Translation` surface.
- [How to i18n routing strategy](../how-to/i18n/routing-strategy.md) for subdomain routing.
- [How to pluralise translations](../how-to/i18n/pluralisation.md), [how to bind typed variables](../how-to/i18n/variable-binding.md), and [how to format dates and times for a locale](../how-to/i18n/date-time-formatting.md).
- [About i18n](../explanation/about-i18n.md) for the rationale behind the runtime store and fluent builder.
