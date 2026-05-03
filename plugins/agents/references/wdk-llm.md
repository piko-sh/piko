# WDK LLM Integration

Use this guide when adding AI/LLM features - completions, streaming, tool calling, structured output, embeddings, vector search, or RAG.

## Overview

The LLM package provides a provider-agnostic API for interacting with large language models.

### Supported providers

| Provider | Package | Default model | Embeddings |
|----------|---------|---------------|------------|
| OpenAI | `llm_provider_openai` | gpt-4.1 | Yes |
| Anthropic | `llm_provider_anthropic` | claude-sonnet-4-5-20250929 | No |
| Google Gemini | `llm_provider_gemini` | gemini-2.5-flash | Yes |
| Grok | `llm_provider_grok` | grok-3 | No |
| Mistral | `llm_provider_mistral` | mistral-large-latest | Yes |
| Ollama | `llm_provider_ollama` | llama3.2 | Yes |
| Voyage | `llm_provider_voyage` | voyage-3.5 | Yes |
| Zolt.ai | `llm_provider_zoltai` | zoltai-1 | Yes |

## Setup

```go
import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/llm"
    "piko.sh/piko/wdk/llm/llm_provider_openai"
)

provider, err := llm_provider_openai.NewOpenAIProvider(llm_provider_openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})

app := piko.New(
    piko.WithLLMProvider("openai", provider),
    piko.WithDefaultLLMProvider("openai"),
)
```

## Completions

```go
svc, _ := llm.GetDefaultService()

resp, err := svc.NewCompletion().
    Model("gpt-4o").
    System("You are a helpful assistant.").
    User("What is the capital of France?").
    Temperature(0.7).
    MaxTokens(500).
    Do(ctx)

fmt.Println(resp.Content())
```

## Streaming

```go
events, err := svc.NewCompletion().
    Model("gpt-4o").
    User("Write a story.").
    Stream(ctx)

for event := range events {
    switch event.Type {
    case llm.StreamEventChunk:
        if event.Chunk != nil && event.Chunk.Delta != nil && event.Chunk.Delta.Content != nil {
            fmt.Print(*event.Chunk.Delta.Content)
        }
    case llm.StreamEventDone:
        fmt.Println("\n[Done]")
    case llm.StreamEventError:
        return event.Error
    }
}
```

## Tool calling

### Manual tool handling

```go
resp, err := svc.NewCompletion().
    Model("gpt-4o").
    User("What's the weather in London?").
    Tool("get_weather", "Get weather for a city", &llm.JSONSchema{
        Type: "object",
        Properties: map[string]*llm.JSONSchema{
            "city": new(llm.StringSchema()),
        },
        Required: []string{"city"},
    }).
    Do(ctx)

if resp.HasToolCalls() {
    for _, tc := range resp.ToolCalls() {
        result := executeWeatherTool(tc.Function.Arguments)
        resp, err = svc.NewCompletion().
            Model("gpt-4o").
            Messages(resp.FirstChoice().Message).
            ToolResult(tc.ID, result).
            Do(ctx)
    }
}
```

### Auto-dispatch tool loop

Register handlers directly - the loop runs automatically until the model stops calling tools:

```go
resp, err := svc.NewCompletion().
    Model("gpt-4o").
    User("What's the weather in London and Paris?").
    ToolFunc("get_weather", "Get weather", schema,
        func(ctx context.Context, args string) (string, error) {
            var p struct{ City string }
            json.Unmarshal([]byte(args), &p)
            return fetchWeather(p.City), nil
        }).
    MaxToolRounds(5).
    Do(ctx)
```

## Structured output

```go
resp, err := svc.NewCompletion().
    Model("gpt-4o").
    User("Extract entities from: 'John works at Acme'").
    StructuredResponse("entities", llm.ObjectSchema(
        map[string]*llm.JSONSchema{
            "people":    new(llm.ArraySchema(llm.StringSchema())),
            "companies": new(llm.ArraySchema(llm.StringSchema())),
        },
        []string{"people", "companies"},
    )).
    Do(ctx)
```

## Vision / multi-modal

```go
resp, err := svc.NewCompletion().
    Model("gpt-4o").
    UserWithImage("What's in this image?", "https://example.com/image.png").
    Do(ctx)
```

## Retry and fallback

```go
resp, err := svc.NewCompletion().
    Model("gpt-4o").
    User("Hello").
    DefaultRetry().
    FallbackProviders("openai", "anthropic", "gemini").
    Do(ctx)
```

## Response caching

```go
resp, err := svc.NewCompletion().
    Model("gpt-4o").
    User("What is the capital of France?").
    Cache(time.Hour).
    Do(ctx)
```

## Budget and rate limiting

```go
import "piko.sh/piko/wdk/maths"

svc.SetBudget("user:123", &llm.BudgetConfig{
    MaxDailySpend: maths.NewMoneyFromFloat(10.0, "USD"),
})

resp, err := svc.NewCompletion().
    Model("gpt-4o").
    User("Hello").
    BudgetScope("user:123").
    Do(ctx)
```

## Embeddings

```go
resp, err := svc.NewEmbedding().
    Model("text-embedding-3-small").
    Input("Hello, world!").
    Embed(ctx)

vector := resp.FirstVector()
```

## RAG (retrieval-augmented generation)

### Auto-RAG (convenience)

```go
resp, err := svc.NewCompletion().
    Model("gpt-4o").
    System("Answer using only the provided context.").
    User("How does caching work?").
    RAG("knowledge-base", 5).
    Do(ctx)
```

### Document ingestion

```go
// Ingest from directory
err := svc.NewIngest("knowledge-base").
    FromDirectory("./docs", "**/*.md").
    Transform(llm.StripFrontmatter()).
    Splitter(llm.NewRecursiveCharacterSplitter(1000, 200)).
    Do(ctx)

// Add individual documents
err := svc.AddText(ctx, "knowledge-base", "faq-1", "How do I install Piko?")
```

## LLM mistake checklist

- Forgetting to set `MaxTokens` for Anthropic (required, unlike OpenAI)
- Not closing the service on shutdown (`svc.Close(ctx)`)
- Using `Do(ctx)` without a context timeout in production
- Forgetting to register an embedding provider for RAG (Anthropic and Grok have no embeddings; the others do)
- Not setting a budget scope (can lead to unexpected costs)
- Using `ToolFunc` when you want manual tool handling (auto-dispatch runs automatically)

## Related

- `references/wdk-data.md` - cache for response caching, vector store backends
- `references/server-actions.md` - calling LLM from server actions
