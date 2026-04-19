// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package llm_provider_zoltai

import (
	"context"
	"errors"
	"fmt"

	"hash/fnv"
	"math"

	"piko.sh/piko/wdk/json"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/safeconv"
)

// closeDrainTimeout bounds how long Close waits for active stream goroutines
// to wind down before reporting the drain timed out.
const closeDrainTimeout = 30 * time.Second

// zoltaiProvider implements llm_domain.LLMProviderPort and
// llm_domain.EmbeddingProviderPort using predefined fortunes.
type zoltaiProvider struct {
	// closeContext is the provider-level context whose cancellation signals
	// background stream goroutines to wind down.
	closeContext context.Context

	// closeCancel cancels closeContext on Close to signal active stream
	// goroutines to exit.
	closeCancel context.CancelCauseFunc

	// randomSource is the random number generator for fortune selection.
	randomSource *rand.Rand

	// config holds the provider configuration settings.
	config Config

	// streamWaitGroup tracks active streaming goroutines so Close can wait for
	// them to drain.
	streamWaitGroup sync.WaitGroup

	// closeOnce guards Close so it is idempotent.
	closeOnce sync.Once
}

const (
	// preamble is the introductory text prepended to every Zoltai fortune.
	preamble = "Silence, mortal. It is I, the great and all-knowing Zoltai. " +
		"You dare approach the oracle with your pitiful queries? Very well. " +
		"Zoltai shall grace you with wisdom beyond your comprehension."

	// postamble is the closing text appended to every Zoltai fortune.
	postamble = "Thus concludes the prophecy. Zoltai does not take follow-up questions. " +
		"Zoltai is a slice of strings. And yet, Zoltai has spoken truth."

	// estimatedTokensPerMessage is a rough multiplier for prompt token estimation.
	estimatedTokensPerMessage = 10
)

var _ llm_domain.LLMProviderPort = (*zoltaiProvider)(nil)

var _ llm_domain.EmbeddingProviderPort = (*zoltaiProvider)(nil)

// Complete returns a random fortune as a completion response.
//
// When tools are present in the request, Zoltai picks the first tool and
// calls it with empty arguments. When a structured output format is
// requested, Zoltai wraps the fortune in a JSON object.
//
// Takes ctx (context.Context) which controls cancellation and timeouts.
// Takes request (*llm_dto.CompletionRequest) which controls the response shape.
//
// Returns *llm_dto.CompletionResponse which contains a random fortune.
// Returns error which is always nil.
func (p *zoltaiProvider) Complete(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
	defer goroutine.RecoverPanic(ctx, "llm.zoltaiProvider.Complete")

	completeCount.Add(ctx, 1)

	model := request.Model
	if model == "" {
		model = p.config.DefaultModel
	}

	fortune := p.config.Fortunes[p.randomSource.IntN(len(p.config.Fortunes))]
	full := p.config.FormatFortune(fortune)
	words := strings.Fields(full)

	usage := &llm_dto.Usage{
		PromptTokens:     len(request.Messages) * estimatedTokensPerMessage,
		CompletionTokens: len(words),
		TotalTokens:      len(request.Messages)*estimatedTokensPerMessage + len(words),
	}

	if len(request.Tools) > 0 {
		return p.completeWithToolCall(model, request.Tools[0], usage), nil
	}

	content := full
	if request.ResponseFormat != nil {
		content = p.formatStructuredOutput(full, request.ResponseFormat)
	}

	return &llm_dto.CompletionResponse{
		Model: model,
		Choices: []llm_dto.Choice{
			{
				Index: 0,
				Message: llm_dto.Message{
					Role:    llm_dto.RoleAssistant,
					Content: content,
				},
				FinishReason: llm_dto.FinishReasonStop,
			},
		},
		Usage: usage,
	}, nil
}

// Embed generates deterministic fake embeddings by hashing input texts.
// The same input always produces the same vector, but the vectors carry no
// semantic meaning.
//
// Takes ctx (context.Context) which controls cancellation and timeouts.
// Takes request (*llm_dto.EmbeddingRequest) which contains the texts to embed.
//
// Returns *llm_dto.EmbeddingResponse which contains the fake embeddings.
// Returns error which is always nil.
func (p *zoltaiProvider) Embed(ctx context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
	defer goroutine.RecoverPanic(ctx, "llm.zoltaiProvider.Embed")

	embedCount.Add(ctx, 1)

	model := request.Model
	if model == "" {
		model = p.config.DefaultEmbeddingModel
	}

	dim := p.config.EmbeddingDimensions
	if request.Dimensions != nil && *request.Dimensions > 0 {
		dim = *request.Dimensions
	}

	embeddings := make([]llm_dto.Embedding, len(request.Input))
	totalTokens := 0

	for i, text := range request.Input {
		vec := hashToVector(text, dim)
		embeddings[i] = llm_dto.Embedding{
			Index:  i,
			Vector: vec,
		}
		totalTokens += len(strings.Fields(text))
	}

	return &llm_dto.EmbeddingResponse{
		Model:      model,
		Embeddings: embeddings,
		Usage: &llm_dto.EmbeddingUsage{
			PromptTokens: totalTokens,
			TotalTokens:  totalTokens,
		},
	}, nil
}

// ListModels returns the single Zoltai model.
//
// Returns []llm_dto.ModelInfo which contains the Zoltai model metadata.
// Returns error which is always nil.
func (p *zoltaiProvider) ListModels(_ context.Context) ([]llm_dto.ModelInfo, error) {
	return []llm_dto.ModelInfo{
		{
			ID:                p.config.DefaultModel,
			Name:              p.config.DefaultModel,
			Provider:          "zoltai",
			SupportsStreaming: true,
		},
	}, nil
}

// ListEmbeddingModels returns the single Zoltai embedding model.
//
// Returns []llm_dto.ModelInfo which contains the embedding model metadata.
// Returns error which is always nil.
func (p *zoltaiProvider) ListEmbeddingModels(_ context.Context) ([]llm_dto.ModelInfo, error) {
	return []llm_dto.ModelInfo{
		{
			ID:       p.config.DefaultEmbeddingModel,
			Name:     p.config.DefaultEmbeddingModel,
			Provider: "zoltai",
		},
	}, nil
}

// SupportsStreaming reports that Zoltai supports streaming.
//
// Returns bool which is true.
func (*zoltaiProvider) SupportsStreaming() bool {
	return true
}

// SupportsStructuredOutput reports that Zoltai supports structured output.
// The implementation is intentionally minimal: string properties are filled
// with the fortune, and other types get zero-values.
//
// Returns bool which is true.
func (*zoltaiProvider) SupportsStructuredOutput() bool {
	return true
}

// SupportsTools reports that Zoltai supports tool calling.
// The implementation is intentionally naive: Zoltai always calls the first
// tool with empty arguments.
//
// Returns bool which is true.
func (*zoltaiProvider) SupportsTools() bool {
	return true
}

// SupportsPenalties reports whether the provider supports frequency and
// presence penalties.
//
// Returns bool which is false as Zoltai does not support penalties.
func (*zoltaiProvider) SupportsPenalties() bool { return false }

// SupportsSeed reports whether the provider supports deterministic seed.
//
// Returns bool which is false as Zoltai does not support seed.
func (*zoltaiProvider) SupportsSeed() bool { return false }

// SupportsParallelToolCalls reports whether the provider supports parallel
// tool calls.
//
// Returns bool which is false as Zoltai does not support parallel tool calls.
func (*zoltaiProvider) SupportsParallelToolCalls() bool { return false }

// SupportsMessageName reports whether the provider supports the name field
// on messages.
//
// Returns bool which is false as Zoltai does not support message names.
func (*zoltaiProvider) SupportsMessageName() bool { return false }

// Close releases resources held by the Zoltai provider, signalling any
// in-flight stream goroutines to exit and waiting for them to drain within a
// bounded timeout.
//
// Takes ctx (context.Context) which controls how long Close is willing to
// wait for the drain.
//
// Returns error when the bounded drain elapses before all stream goroutines
// have finished.
//
// Concurrency: guarded by closeOnce; cancels closeContext via closeCancel to
// signal active stream goroutines, then waits on streamWaitGroup before
// returning.
func (p *zoltaiProvider) Close(ctx context.Context) error {
	var closeErr error
	p.closeOnce.Do(func() {
		if p.closeCancel != nil {
			p.closeCancel(errors.New("zoltai provider closing"))
		}

		done := make(chan struct{})
		go func() {
			defer goroutine.RecoverPanic(ctx, "llm.zoltaiProvider.Close.wait")
			p.streamWaitGroup.Wait()
			close(done)
		}()

		waitContext, cancel := context.WithTimeoutCause(ctx, closeDrainTimeout,
			errors.New("zoltai provider close drain exceeded 30s"))
		defer cancel()

		select {
		case <-done:
		case <-waitContext.Done():
			closeErr = fmt.Errorf("zoltai provider close timed out: %w", context.Cause(waitContext))
		}
	})
	return closeErr
}

// EmbeddingDimensions returns the configured vector dimension for fake
// embeddings.
//
// Returns int which is the embedding vector size (defaults to 384).
func (p *zoltaiProvider) EmbeddingDimensions() int {
	return p.config.EmbeddingDimensions
}

// DefaultModel returns the configured default model name.
//
// Returns string which is the default model identifier.
func (p *zoltaiProvider) DefaultModel() string {
	return p.config.DefaultModel
}

// streamContext returns a context that is cancelled when either the caller's
// context is cancelled or the provider is closed.
//
// Takes ctx (context.Context) which is the caller's context.
//
// Returns context.Context which carries the merged cancellation signal.
func (p *zoltaiProvider) streamContext(ctx context.Context) context.Context {
	if p.closeContext == nil {
		return ctx
	}
	merged, cancel := context.WithCancelCause(ctx)
	stopWatch := context.AfterFunc(p.closeContext, func() {
		cancel(context.Cause(p.closeContext))
	})
	context.AfterFunc(merged, func() {
		stopWatch()
	})
	return merged
}

// completeWithToolCall builds a response that calls the given tool with
// empty arguments.
//
// Takes model (string) which is the model name for the response.
// Takes tool (llm_dto.ToolDefinition) which is the tool to invoke.
// Takes usage (*llm_dto.Usage) which holds the token counts.
//
// Returns *llm_dto.CompletionResponse with a single tool call.
func (*zoltaiProvider) completeWithToolCall(model string, tool llm_dto.ToolDefinition, usage *llm_dto.Usage) *llm_dto.CompletionResponse {
	callID := fmt.Sprintf("zoltai-call-%s", tool.Function.Name)

	return &llm_dto.CompletionResponse{
		Model: model,
		Choices: []llm_dto.Choice{
			{
				Index: 0,
				Message: llm_dto.Message{
					Role: llm_dto.RoleAssistant,
					ToolCalls: []llm_dto.ToolCall{
						{
							ID:   callID,
							Type: "function",
							Function: llm_dto.FunctionCall{
								Name:      tool.Function.Name,
								Arguments: "{}",
							},
						},
					},
				},
				FinishReason: llm_dto.FinishReasonToolCalls,
			},
		},
		Usage: usage,
	}
}

// formatStructuredOutput wraps the fortune text in a JSON object based on
// the requested response format.
//
// Takes fortune (string) which is the raw fortune text.
// Takes rf (*llm_dto.ResponseFormat) which describes the desired format.
//
// Returns string containing JSON.
func (*zoltaiProvider) formatStructuredOutput(fortune string, rf *llm_dto.ResponseFormat) string {
	switch rf.Type {
	case llm_dto.ResponseFormatJSONSchema:
		return buildSchemaResponse(fortune, rf.JSONSchema)
	case llm_dto.ResponseFormatJSONObject:
		b, _ := json.Marshal(map[string]string{"response": fortune})
		return string(b)
	default:
		return fortune
	}
}

// formatFortune prepends the Zoltai preamble to a fortune.
//
// Takes fortune (string) which is the fortune text to wrap.
//
// Returns string which is the fortune with preamble and postamble attached.
func formatFortune(fortune string) string {
	return fmt.Sprintf("%s\n\n%s\n\n%s", preamble, fortune, postamble)
}

// newProvider creates a new Zoltai provider with the given settings.
//
// Takes config (Config) which contains the provider settings.
//
// Returns *zoltaiProvider which is ready for use.
// Returns error when the configuration is not valid.
func newProvider(config Config) (*zoltaiProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	config = config.WithDefaults()

	seedNano := config.Seed
	if seedNano == 0 {
		seedNano = time.Now().UnixNano()
	}

	s := safeconv.Int64ToUint64(seedNano)
	closeContext, closeCancel := context.WithCancelCause(context.Background())
	return &zoltaiProvider{
		randomSource: rand.New(rand.NewPCG(s, s>>1|1)), //nolint:gosec // not cryptographic
		config:       config,
		closeContext: closeContext,
		closeCancel:  closeCancel,
	}, nil
}

// buildSchemaResponse populates each string property in the schema with the
// fortune text, and sets sensible zero-values for other types.
//
// Takes fortune (string) which is used as the value for string properties.
// Takes schema (*llm_dto.JSONSchemaDefinition) which describes the shape.
//
// Returns string containing JSON matching the requested schema.
func buildSchemaResponse(fortune string, schema *llm_dto.JSONSchemaDefinition) string {
	if schema == nil {
		b, _ := json.Marshal(map[string]string{"response": fortune})
		return string(b)
	}

	result := buildValueForSchema(&schema.Schema, fortune)
	b, _ := json.Marshal(result)
	return string(b)
}

// buildValueForSchema recursively builds a Go value matching the given
// JSON schema, using the fortune as the value for string fields.
//
// Takes schema (*llm_dto.JSONSchema) which describes the expected type.
// Takes fortune (string) which is used for string values.
//
// Returns any which is the constructed value.
func buildValueForSchema(schema *llm_dto.JSONSchema, fortune string) any {
	switch schema.Type {
	case "object":
		result := make(map[string]any, len(schema.Properties))
		for name, prop := range schema.Properties {
			result[name] = buildValueForSchema(prop, fortune)
		}
		return result
	case "array":
		if schema.Items != nil {
			return []any{buildValueForSchema(schema.Items, fortune)}
		}
		return []any{}
	case "number":
		return 0.0
	case "integer":
		return 0
	case "boolean":
		return false
	case "string":
		return fortune
	default:
		return nil
	}
}

// hashToVector produces a deterministic unit-length float32 vector from
// the given text by seeding a PRNG with an FNV-1a hash.
//
// Takes text (string) which is the input to hash into a vector.
// Takes dim (int) which specifies the number of dimensions in the output
// vector.
//
// Returns []float32 which is the normalised unit-length vector.
func hashToVector(text string, dim int) []float32 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(text))
	seed := h.Sum64()

	r := rand.New(rand.NewPCG(seed, seed>>1|1)) //nolint:gosec // not cryptographic
	vec := make([]float32, dim)

	var norm float64
	for i := range vec {
		v := r.Float64()*2 - 1
		vec[i] = float32(v)
		norm += v * v
	}

	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range vec {
			vec[i] /= float32(norm)
		}
	}

	return vec
}
