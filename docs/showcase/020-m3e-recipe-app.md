---
title: "020: M3E recipe app"
description: Full-featured recipe app with M3E components, SSE streaming, emails, and LLM
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 180
---

# 020: M3E recipe app
<!-- vale Piko.TitleCase = NO -->
A full-featured recipe web application ("The Norman Kitchen") that brings together M3E components, server actions, SSE streaming, email sending, and LLM integration.
<!-- vale Piko.TitleCase = YES -->
This is the most complete example, showing how multiple Piko features compose into a real application.

## What this demonstrates

The M3E component library provides cards, carousels, buttons, chips, modals, text fields, and more, woven into a real layout. An SSE streaming action powers the live kitchen activity feed through `StreamProgress`, and heartbeats keep the stream alive through proxies. A contact form sends confirmation and internal notification emails through email actions. The email templates live as `.pk` files in `emails/` and use the same template syntax as pages.

The "Ask the Mouse" Q&A page uses LLM integration through the Zoltai fake provider, so local development requires no API keys. The project has more than 10 pages with partials, layouts, and nested routes. A shared layout, navigation bar, footer, and recipe card partial demonstrate partials and slots. This example combines nearly every major Piko feature in a single application.

## Project structure

```text
src/
  cmd/main/
    main.go                           Registers M3E, Zoltai LLM provider
  actions/
    email/
      contact.go                      Contact form action with email sending
      helpers.go                      Email helper utilities
    kitchen/
      feed.go                         SSE streaming action for live kitchen events
  emails/
    contact_confirmation.pk           Confirmation email template
    contact_internal.pk               Internal notification email template
  internal/
    dto/emails.go                     Email data transfer objects
    env/env.go                        Environment configuration
    fortunes/cheese.go                Cheese-themed fortunes for the LLM oracle
    kitchen/events.go                 Kitchen event sequences and standalone events
  pages/
    index.pk                          Home with hero carousel and recipe grid
    recipes.pk                        Recipe browsing and filtering
    recipe-index.pk                   Recipe index page
    recipe/                           Individual recipe pages (8 recipes)
    kitchen-feed.pk                   Live SSE kitchen activity stream
    ask-the-mouse.pk                  LLM-powered Q&A with Gouda the Gourmet
    meal-planner.pk                   Weekly meal planning
    cheese-guide.pk                   Cheese encyclopaedia
    about.pk                          About page with contact form
    favourites.pk                     Saved recipes
  partials/
    layout.pk                         Shared page layout
    nav.pk                            Navigation bar
    footer.pk                         Page footer
    recipe-card.pk                    Reusable recipe card component
```

## How it works

The entry point wires together M3E components and the Zoltai LLM provider:

```go
zoltaiProvider, err := llm_provider_zoltai.NewZoltaiProvider(llm_provider_zoltai.Config{
    Fortunes: fortunes.Cheese,
})

ssr := piko.New(
    piko.WithComponents(components.M2()...),
    piko.WithComponents(components.M3E()...),
    piko.WithLLMProvider("zoltai", zoltaiProvider),
    piko.WithDefaultLLMProvider("zoltai"),
)
```

The kitchen feed streams randomised events via SSE, drawing from sequence chains and standalone event pools:

```go
func (*FeedAction) StreamProgress(stream *piko.SSEStream) error {
    for {
        stream.Send("event", map[string]string{
            "text": evt.Text, "category": evt.Category,
        })
        // Wait 5-12 seconds before the next event
    }
}
```

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/020_m3e_recipe_app/src/
go mod tidy
air
```

## See also

- [M3E component library reference](../reference/m3e-components.md).
- [How to email templates](../how-to/email-templates.md).
- [How to LLMs and embeddings](../how-to/llm-and-embeddings.md).
