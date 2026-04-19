---
title: LLM API
description: Completion, embedding, memory, RAG, cost, budget, and rate-limit APIs for language-model providers.
nav:
  sidebar:
    section: "reference"
    subsection: "services"
    order: 140
---

# LLM API

Piko's LLM service abstracts over Anthropic, OpenAI, Gemini, Ollama, and test providers. It supports plain completions, streaming, tool calling, structured JSON, embeddings, retrieval-augmented generation, cost tracking, budgets, rate limits, and response caching. For task recipes see [how to LLMs and embeddings](../how-to/llm-and-embeddings.md). Source of truth: [`wdk/llm/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/llm/facade.go).

## Service

| Function | Returns |
|---|---|
| `llm.NewService(defaultProviderName string, opts ...ServiceOption) Service` | Constructs a new service. |
| `llm.GetDefaultService() (Service, error)` | Returns the bootstrap-configured service. |

## Builders

```go
func NewCompletionBuilder(service Service) *CompletionBuilder
func NewCompletionBuilderFromDefault() (*CompletionBuilder, error)
func NewEmbeddingBuilder(service Service) *EmbeddingBuilder
func NewEmbeddingBuilderFromDefault() (*EmbeddingBuilder, error)
```

Fluent methods on `CompletionBuilder`: `.Model(...)`, `.Messages(...)`, `.Tools(...)`, `.ResponseFormat(...)`, `.Stream(...)`, `.Temperature(...)`, `.MaxTokens(...)`, `.Do(ctx)`.

## Message helpers

```go
NewSystemMessage(content string) Message
NewUserMessage(content string) Message
NewAssistantMessage(content string) Message
NewToolResultMessage(toolCallID, content string) Message
NewUserMessageWithImages(content string, images ...ContentPart) Message
NewUserMessageWithImageURL(content string, url, detail string) Message
NewUserMessageWithImageData(content string, data []byte, mimeType, detail string) Message
```

## Tools and structured output

```go
NewFunctionTool(name, description string, params JSONSchema) ToolDefinition
NewStrictFunctionTool(name, description string, params JSONSchema) ToolDefinition
ToolChoiceAuto() ToolChoice
ToolChoiceNone() ToolChoice
ToolChoiceRequired() ToolChoice
ToolChoiceSpecific(name string) ToolChoice
```

`JSONSchemaDefinition` and `JSONSchema` describe response formats. Set `ResponseFormat.Type = ResponseFormatJSONSchema` with `ResponseFormat.JSONSchema` to force a structured output.

## Cost and budget

| Function | Purpose |
|---|---|
| `NewCostCalculator()` | Cost calculator with the default pricing table. |
| `NewCostCalculatorWithPricing(table *PricingTable)` | Custom pricing. |
| `NewBudgetManager(store, calculator)` | Enforces spending limits. |
| `DefaultPricingTable` | The shipped pricing table. |

`BudgetStorePort` persists spending counters. Provide a `cache.Cache` or custom backing store.

## Rate limiting

| Function | Purpose |
|---|---|
| `NewRateLimiter(store, opts...)` | Per-model, per-user, or global rate limit. |
| `RateLimiterStorePort` | Driven port for counter state. |
| `WithRateLimiterClock(clk)` | Injects a test clock. |

## Response cache

`NewCacheManager(store, defaultTTL)` returns a manager that short-circuits completions whose inputs match a previously cached request.

## Memory

| Function | Purpose |
|---|---|
| `NewBufferMemory(store, opts...)` | Fixed-size conversation buffer. |
| `NewWindowMemory(store, opts...)` | Token-window buffer. |
| `NewSummaryMemory(store, service, config)` | LLM-summarised long history. |

## Retrieval-augmented generation

| Function | Purpose |
|---|---|
| `NewVectorStore(factory)` | Cache-backed vector store. |
| `NewIngestBuilder` (via service) | Loads, splits, transforms, vectorises documents. |
| `NewRecursiveFSLoader(fsys, patterns...)` | Filesystem document loader. |
| `NewRecursiveCharacterSplitter(chunkSize, overlap)` | Token-agnostic chunker. |
| `NewMarkdownSplitter(chunkSize, overlap, opts...)` | Markdown-aware chunker. |
| `StripFrontmatter()`, `ExtractFrontmatter(opts...)` | Frontmatter transforms. |
| `PrependChunkContext()` | Attaches chunk position metadata. |
| `LLMQueryRewriter(opts...)` | Multi-query expansion. |

Attach RAG to a completion with `CompletionBuilder.WithRAG(...)` options: `WithRAGQuery`, `WithRAGMinScore`, `WithRAGFilter`, `WithRAGEmbeddingProvider`, `WithRAGEmbeddingModel`, `WithRAGQueryRewriter`, `WithRAGHybridSearch`.

## Errors

`ErrProviderNotFound`, `ErrNoDefaultProvider`, `ErrProviderAlreadyExists`, `ErrStreamingNotSupported`, `ErrToolsNotSupported`, `ErrStructuredOutputNotSupported`, `ErrEmptyMessages`, `ErrEmptyModel`, `ErrInvalidTemperature`, `ErrInvalidTopP`, `ErrInvalidMaxTokens`, `ErrBudgetExceeded`, `ErrRateLimited`, `ErrMaxCostExceeded`, `ErrUnknownModelPrice`, `ErrProviderOverloaded`, `ErrProviderTimeout`, `ErrConversationNotFound`.

## Providers

| Sub-package | Backend |
|---|---|
| `llm_provider_anthropic` | Claude models. |
| `llm_provider_openai` | OpenAI GPT and o-series. |
| `llm_provider_gemini` | Google Gemini. |
| `llm_provider_ollama` | Local Ollama. |
| `llm_provider_mock` | In-memory test double. |

## Bootstrap options

| Option | Purpose |
|---|---|
| `piko.WithLLMProvider(name, provider)` | Registers a completion provider. |
| `piko.WithDefaultLLMProvider(name)` | Marks the default completion provider. |
| `piko.WithEmbeddingProvider(name, provider)` | Registers an embedding provider. |
| `piko.WithDefaultEmbeddingProvider(name)` | Marks the default embedding provider. |
| `piko.WithLLMService(service)` | Registers a fully configured service. |

## See also

- [How to LLMs and embeddings](../how-to/llm-and-embeddings.md) for RAG and streaming recipes.
- [Scenario 020: M3E recipe app](../showcase/020-m3e-recipe-app.md) uses completions.
