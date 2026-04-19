---
title: How to use LLMs, embeddings, and RAG
description: Register LLM and embedding providers, run completions, stream responses, and ingest documents into a vector store for retrieval.
nav:
  sidebar:
    section: "how-to"
    subsection: "operations"
    order: 60
---

# How to use LLMs, embeddings, and RAG

Piko ships a Go-level LLM service that wraps any provider implementing the `llm.ProviderPort` interface (OpenAI, Anthropic, local Ollama, custom). Embeddings follow the same pattern. Retrieval-augmented generation (RAG) pairs embeddings with a vector store for semantic search. This guide covers the common patterns. See the [LLM API reference](../reference/llm-api.md) for the full API. The tests under [`tests/integration/llm/`](https://github.com/piko-sh/piko/tree/master/tests/integration/llm) exercise the behaviour.

## Register an LLM provider

Pass a provider instance at bootstrap. The first registered provider becomes the default unless overridden:

```go
import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/llm/llm_provider_openai"
    "piko.sh/piko/wdk/llm/llm_provider_anthropic"
)

openaiProvider, err := llm_provider_openai.NewOpenAIProvider(llm_provider_openai.Config{APIKey: apiKey})
if err != nil {
    panic(err)
}
anthropicProvider, err := llm_provider_anthropic.NewAnthropicProvider(llm_provider_anthropic.Config{APIKey: anthropicKey})
if err != nil {
    panic(err)
}

ssr := piko.New(
    piko.WithLLMProvider("openai", openaiProvider),
    piko.WithLLMProvider("anthropic", anthropicProvider),
    piko.WithDefaultLLMProvider("anthropic"),
)
```

Use `piko.WithEmbeddingProvider(name, provider)` and `piko.WithDefaultEmbeddingProvider(name)` for the embedding side. Embedding providers can be the same adapter as the completion provider (for providers that handle both) or a separate adapter (for example, Voyage embeddings alongside an OpenAI completion provider).

## Run a completion

Access the service through the bootstrap-registered handle. A typical action looks like:

```go
package chat

import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/llm"
)

type ChatInput struct {
    Prompt string `json:"prompt" validate:"required"`
}

type ChatResponse struct {
    Reply string `json:"reply"`
}

type ChatAction struct {
    piko.ActionMetadata
}

func (a *ChatAction) Call(input ChatInput) (ChatResponse, error) {
    svc, err := llm.GetDefaultService()
    if err != nil {
        return ChatResponse{}, piko.Errorf(
            "the assistant service is unavailable",
            "get llm service: %w", err,
        )
    }

    resp, err := svc.Complete(a.Ctx(), &llm.CompletionRequest{
        Messages: []llm.Message{
            {Role: llm.RoleUser, Content: input.Prompt},
        },
    })
    if err != nil {
        return ChatResponse{}, piko.Errorf(
            "sorry, the assistant is unavailable",
            "llm completion failed: %w", err,
        )
    }

    return ChatResponse{Reply: resp.Choices[0].Message.Content}, nil
}
```

`Complete(ctx, req)` targets the default provider. Use `CompleteWithProvider(ctx, "anthropic", req)` to pick a specific one, or use the fluent builder `svc.NewCompletion().System(...).User(...).Temperature(...).Do(ctx)`.

## Stream completions

For long responses, stream tokens as they arrive:

```go
events, err := svc.Stream(a.Ctx(), req)
if err != nil {
    return ChatResponse{}, err
}

for event := range events {
    if event.Type != llm.StreamEventChunk || event.Chunk == nil || event.Chunk.Delta == nil {
        continue
    }
    if event.Chunk.Delta.Content == nil {
        continue
    }
    token := *event.Chunk.Delta.Content
    _ = token
}
```

Combine with the [streaming-action pattern](actions/streaming-with-sse.md) to pipe tokens straight to the browser as they arrive.

## Generate embeddings

```go
svc, err := llm.GetDefaultService()
if err != nil {
    return err
}

resp, err := svc.Embed(a.Ctx(), &llm.EmbeddingRequest{
    Input: []string{"the text to embed"},
})
if err != nil {
    return err
}

vector := resp.FirstEmbedding().Vector
```

## Ingest documents for RAG

Piko's RAG path combines an embedding provider with a vector-store port. Ingest documents through the builder, which loads, optionally transforms, splits, and embeds them in one pass:

```go
svc, _ := llm.GetDefaultService()

splitter, err := llm.NewMarkdownSplitter(500, 50)
if err != nil {
    return err
}

if err := svc.NewIngest("docs").
    FromDirectory("./content", "**/*.md").
    Splitter(splitter).
    Do(a.Ctx()); err != nil {
    return err
}
```

`FromDirectory` reads files from disk. `FromFS` accepts any `fs.FS`, which is handy for `embed.FS`. Chain `Transform(...)` for per-document pre-processing such as frontmatter stripping or markup cleanup, and `PostSplitTransform(...)` to enrich or filter individual chunks. Piko embeds every chunk with the default embedding provider and stores it in the configured vector store under the namespace `"docs"`.

For one-off additions, `svc.AddText(ctx, namespace, id, content)` is a shorter path.

Register a vector-store implementation at bootstrap (`svc.SetVectorStore(store)` during initial setup, or via a dedicated bootstrap option when the provider exposes one).

## Retrieve and augment

Embed the user's question, search the vector store, and feed the matches into a completion. The builder accepts both message helpers (`User`, `System`, `Assistant`) and retrieved context:

```go
embedResp, err := svc.Embed(a.Ctx(), &llm.EmbeddingRequest{
    Input: []string{input.Prompt},
})
if err != nil {
    return ChatResponse{}, err
}

store := svc.GetVectorStore()
search, err := store.Search(a.Ctx(), &llm.VectorSearchRequest{
    Namespace: "docs",
    Vector:    embedResp.FirstEmbedding().Vector,
    TopK:      5,
})
if err != nil {
    return ChatResponse{}, err
}

resp, err := svc.NewCompletion().
    WithVectorContext(search.Results).
    User(input.Prompt).
    Do(a.Ctx())
```

`WithVectorContext` injects the retrieved chunks as system context, then `User` adds the prompt. The completion sees both the user message and the most relevant passages from the `docs` namespace.

## Budgets, rate limits, and caching

Piko tracks cost for every completion against the configured budget (`svc.SetBudget(scope, config)`) and rate-limits it (`svc.SetRateLimits(scope, rpm, tpm)`). `SetBudget` takes a `*llm.BudgetConfig` pointer, so reuse the same value across goroutines without copying:

```go
budget := &llm.BudgetConfig{
    MaxDailySpend: maths.NewMoneyFromString("10.00", "USD"),
    MaxTotalSpend: maths.NewMoneyFromString("100.00", "USD"),
}
svc.SetBudget("project:alpha", budget)
```

A completion that would exceed the budget returns an error you can surface as a user-visible message:

```go
status, _ := svc.GetBudgetStatus(a.Ctx(), "project:alpha")
if status.RemainingBudget.CheckIsZero() {
    return ChatResponse{}, piko.NewError("this feature is temporarily unavailable", errors.New("budget exhausted"))
}
```

The service caches identical completion requests when you attach a cache manager via `svc.SetCacheManager(...)`. Cached hits do not count against the budget.

## See also

- [LLM API reference](../reference/llm-api.md) for the typed provider, model, and embedding surface.
- [Bootstrap options reference: LLM and embeddings](../reference/bootstrap-options.md#llm-and-embedding).
- [How to streaming with SSE](actions/streaming-with-sse.md) for forwarding token streams to the browser.
- Integration tests: [`tests/integration/llm/`](https://github.com/piko-sh/piko/tree/master/tests/integration/llm) exercises completions, streaming, embeddings, RAG, budgets, and fallback chains.
