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

Piko ships a Go-level LLM service that wraps any provider implementing the `llm_domain.LLMProviderPort` interface (OpenAI, Anthropic, local Ollama, custom). Embeddings follow the same pattern. Retrieval-augmented generation (RAG) pairs embeddings with a vector store for semantic search. This guide covers the common patterns. See the [LLM API reference](../reference/llm-api.md) for the full API. The tests under [`tests/integration/llm/`](https://github.com/piko-sh/piko/tree/master/tests/integration/llm) exercise the behaviour.

## Register an LLM provider

Pass a provider instance at bootstrap. The first registered provider becomes the default unless overridden:

```go
import (
    "piko.sh/piko"
    "piko.sh/piko/adapters/llm/openai"
    "piko.sh/piko/adapters/llm/anthropic"
)

ssr := piko.New(
    piko.WithLLMProvider("openai", openai.NewProvider(openai.Config{APIKey: apiKey})),
    piko.WithLLMProvider("anthropic", anthropic.NewProvider(anthropic.Config{APIKey: anthropicKey})),
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
    "piko.sh/piko/internal/llm/llm_dto"
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
    svc := piko.GetLLMService()

    resp, err := svc.Complete(a.Ctx(), &llm_dto.CompletionRequest{
        Messages: []llm_dto.Message{
            {Role: "user", Content: input.Prompt},
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

`Complete(ctx, req)` targets the default provider. Use `CompleteWithProvider(ctx, "anthropic", req)` to pick a specific one, or use the fluent builder `svc.NewCompletion().WithMessages(...).WithTemperature(...).Do(ctx)`.

## Stream completions

For long responses, stream tokens as they arrive:

```go
events, err := svc.Stream(a.Ctx(), req)
if err != nil {
    return ChatResponse{}, err
}

for event := range events {
    if event.Delta != "" {
        // Handle each token chunk. Typically forward over SSE to the client
        // via stream.Send("token", event.Delta) in a StreamProgress method.
    }
}
```

Combine with the [streaming-action pattern](actions/streaming-with-sse.md) to pipe tokens straight to the browser as they arrive.

## Generate embeddings

```go
resp, err := piko.GetLLMService().Embed(a.Ctx(), &llm_dto.EmbeddingRequest{
    Input: []string{"the text to embed"},
})
if err != nil {
    return err
}

vector := resp.Embeddings[0]
// vector is []float32 ready to store or compare.
```

## Ingest documents for RAG

Piko's RAG path combines an embedding provider with a vector-store port. Ingest documents through the builder:

```go
ingest := piko.GetLLMService().NewIngest("docs")
if err := ingest.
    AddDocument("intro.md", introBody).
    AddDocument("guide.md", guideBody).
    WithChunker(llm_domain.MarkdownChunker{MaxTokens: 500}).
    Do(a.Ctx()); err != nil {
    return err
}
```

Piko chunks each document, embeds it with the default embedding provider, and stores it in the configured vector store under the namespace `"docs"`.

For one-off additions, `svc.AddText(ctx, namespace, id, content)` is a shorter path.

Register a vector-store implementation at bootstrap (`svc.SetVectorStore(store)` during initial setup, or via a dedicated bootstrap option when the provider exposes one).

## Retrieve and augment

Query the vector store and pass the retrieved context into the next completion request. The builder API composes both in one call:

```go
resp, err := svc.NewCompletion().
    WithRetrievedContext("docs", input.Prompt, 5).   // pulls top-5 matches
    WithMessage("user", input.Prompt).
    Do(a.Ctx())
```

The resulting completion sees the user prompt plus the most relevant chunks from the `docs` namespace.

## Budgets, rate limits, and caching

Piko tracks cost for every completion against the configured budget (`svc.SetBudget(scope, config)`) and rate-limits it (`svc.SetRateLimits(scope, rpm, tpm)`). A completion that would exceed the budget returns an error you can surface as a user-visible message:

```go
status, _ := svc.GetBudgetStatus(a.Ctx(), "project:alpha")
if status.Remaining <= 0 {
    return ChatResponse{}, piko.NewError("this feature is temporarily unavailable", errors.New("budget exhausted"))
}
```

The service caches identical completion requests when you attach a cache manager via `svc.SetCacheManager(...)`. Cached hits do not count against the budget.

## See also

- [Bootstrap options reference: LLM and embeddings](../reference/bootstrap-options.md#llm-and-embedding).
- [How to streaming with SSE](actions/streaming-with-sse.md) for forwarding token streams to the browser.
- Integration tests: [`tests/integration/llm/`](https://github.com/piko-sh/piko/tree/master/tests/integration/llm) exercises completions, streaming, embeddings, RAG, budgets, and fallback chains.
