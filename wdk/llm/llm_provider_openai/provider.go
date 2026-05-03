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

package llm_provider_openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/safeerror"
	"piko.sh/piko/wdk/logger"
)

const (
	// httpClientTimeout is a top-level HTTP client timeout that bounds requests
	// even when the caller does not supply a per-request deadline. It is
	// generous enough to allow long-running completions but prevents stuck
	// connections from leaking goroutines indefinitely.
	httpClientTimeout = 30 * time.Minute
)

// openaiProvider implements llm_domain.LLMProviderPort and
// llm_domain.EmbeddingProviderPort for OpenAI.
type openaiProvider struct {
	// config holds the provider configuration settings.
	config Config

	// closeContext is the provider-level context whose cancellation signals
	// background stream goroutines to exit.
	closeContext context.Context

	// closeCancel cancels closeContext on Close to signal in-flight stream
	// goroutines to wind down.
	closeCancel context.CancelCauseFunc

	// httpClient is the underlying *http.Client injected into the OpenAI SDK;
	// retained so tests can verify the configured top-level timeout and idle
	// connections can be released on Close.
	httpClient *http.Client

	// defaultModel is the model name to use when not specified in a request.
	defaultModel string

	// defaultEmbeddingModel is the embedding model name to use when not
	// specified in a request.
	defaultEmbeddingModel string

	// client is the OpenAI API client for making requests.
	client openai.Client

	// streamWaitGroup tracks active streaming goroutines so Close can wait for
	// them to drain.
	streamWaitGroup sync.WaitGroup

	// closeOnce guards Close so it is idempotent.
	closeOnce sync.Once

	// embeddingDimensions is the default vector dimension for the configured
	// embedding model.
	embeddingDimensions int
}

var _ llm_domain.LLMProviderPort = (*openaiProvider)(nil)

var _ llm_domain.EmbeddingProviderPort = (*openaiProvider)(nil)

// Complete sends a completion request to OpenAI.
//
// Takes request (*llm_dto.CompletionRequest) which specifies the prompt and model
// settings.
//
// Returns *llm_dto.CompletionResponse which contains the generated completion.
// Returns error when the API request fails.
func (p *openaiProvider) Complete(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
	defer goroutine.RecoverPanic(ctx, "llm.openaiProvider.Complete")

	ctx, l := logger.From(ctx, log)
	completeCount.Add(ctx, 1)
	start := time.Now()

	defer func() {
		completeDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
	}()

	model := request.Model
	if model == "" {
		model = p.defaultModel
	}

	params := p.buildChatParams(request, model)

	l.Debug("Sending OpenAI completion request",
		logger.String("model", model),
		logger.Int("message_count", len(request.Messages)),
	)

	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		completeErrorCount.Add(ctx, 1)
		wrapped := fmt.Errorf("openai completion failed: %w", wrapError(err))
		return nil, sanitiseProviderError(wrapped, "openai request rejected")
	}

	return p.convertResponse(completion), nil
}

// sanitiseProviderError wraps a 4xx provider error in a safeerror so that HTTP
// edges can sanitise it before returning to the user. Non-4xx errors are
// returned unchanged so retry classification continues to work.
//
// Takes err (error) which is the error to inspect.
// Takes safeMessage (string) which is shown to end users for 4xx errors.
//
// Returns error which is wrapped when err carries a 4xx status code.
func sanitiseProviderError(err error, safeMessage string) error {
	if providerErr, ok := errors.AsType[*llm_domain.ProviderError](err); ok && providerErr.StatusCode >= http.StatusBadRequest && providerErr.StatusCode < http.StatusInternalServerError {
		return safeerror.NewError(safeMessage, err)
	}
	return err
}

// SupportsStreaming reports whether the provider supports streaming.
//
// Returns bool which is true if streaming is supported.
func (*openaiProvider) SupportsStreaming() bool {
	return true
}

// SupportsStructuredOutput reports whether the provider supports structured
// output.
//
// Returns bool which is true if structured output is supported.
func (*openaiProvider) SupportsStructuredOutput() bool {
	return true
}

// SupportsTools reports whether the provider supports tool calling.
//
// Returns bool which is true if tool calling is supported.
func (*openaiProvider) SupportsTools() bool {
	return true
}

// SupportsPenalties reports whether the provider supports frequency and
// presence penalties.
//
// Returns bool which is true if penalties are supported.
func (*openaiProvider) SupportsPenalties() bool { return true }

// SupportsSeed reports whether the provider supports deterministic seed.
//
// Returns bool which is true if seed is supported.
func (*openaiProvider) SupportsSeed() bool { return true }

// SupportsParallelToolCalls reports whether the provider supports parallel
// tool calls.
//
// Returns bool which is true if parallel tool calls are supported.
func (*openaiProvider) SupportsParallelToolCalls() bool { return true }

// SupportsMessageName reports whether the provider supports the name field
// on messages.
//
// Returns bool which is true if message names are supported.
func (*openaiProvider) SupportsMessageName() bool { return true }

// ListModels returns available models.
//
// Returns []llm_dto.ModelInfo which contains the filtered list of chat models.
// Returns error when the API request to list models fails.
func (p *openaiProvider) ListModels(ctx context.Context) ([]llm_dto.ModelInfo, error) {
	page, err := p.client.Models.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list openai models: %w", wrapError(err))
	}

	result := make([]llm_dto.ModelInfo, 0)
	for i := range page.Data {
		model := &page.Data[i]
		if isOAChatModel(model.ID) {
			result = append(result, llm_dto.ModelInfo{
				ID:                       model.ID,
				Name:                     model.ID,
				Provider:                 "openai",
				Created:                  model.Created,
				SupportsStreaming:        true,
				SupportsTools:            true,
				SupportsStructuredOutput: true,
			})
		}
	}
	return result, nil
}

// Close releases resources held by the provider, cancelling any in-flight
// stream goroutines and waiting for them to drain within a bounded timeout.
//
// Returns error when the close drain exceeds its bounded wait.
//
// Concurrency: guarded by closeOnce; cancels closeContext via closeCancel to
// signal active stream goroutines, then waits on streamWaitGroup before
// returning.
func (p *openaiProvider) Close(ctx context.Context) error {
	var closeErr error
	p.closeOnce.Do(func() {
		if p.closeCancel != nil {
			p.closeCancel(errors.New("openai provider closing"))
		}

		done := make(chan struct{})
		go func() {
			defer goroutine.RecoverPanic(ctx, "llm.openaiProvider.Close.wait")
			p.streamWaitGroup.Wait()
			close(done)
		}()

		waitContext, cancel := context.WithTimeoutCause(ctx, 30*time.Second,
			errors.New("openai provider close drain exceeded 30s"))
		defer cancel()

		select {
		case <-done:
		case <-waitContext.Done():
			closeErr = fmt.Errorf("openai provider close timed out: %w", context.Cause(waitContext))
		}

		if p.httpClient != nil {
			p.httpClient.CloseIdleConnections()
		}
	})
	return closeErr
}

// DefaultModel implements LLMProviderPort.DefaultModel.
//
// Returns string which is the name of the default model for this provider.
func (p *openaiProvider) DefaultModel() string {
	return p.defaultModel
}

// Embed generates embeddings for the given input texts via the OpenAI
// embeddings API.
//
// Takes ctx (context.Context) which controls cancellation and timeouts.
// Takes request (*llm_dto.EmbeddingRequest) which contains the embedding
// parameters.
//
// Returns *llm_dto.EmbeddingResponse which contains the generated
// embeddings.
// Returns error when the request fails.
func (p *openaiProvider) Embed(ctx context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
	defer goroutine.RecoverPanic(ctx, "llm.openaiProvider.Embed")

	ctx, l := logger.From(ctx, log)
	embedCount.Add(ctx, 1)
	start := time.Now()

	defer func() {
		embedDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
	}()

	model := request.Model
	if model == "" {
		model = p.defaultEmbeddingModel
	}

	params := openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(model),
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: request.Input,
		},
	}
	if request.Dimensions != nil && *request.Dimensions > 0 {
		params.Dimensions = openai.Int(int64(*request.Dimensions))
	}

	l.Debug("Sending OpenAI embedding request",
		logger.String("model", model),
		logger.Int("input_count", len(request.Input)),
	)

	response, err := p.client.Embeddings.New(ctx, params)
	if err != nil {
		embedErrorCount.Add(ctx, 1)
		wrapped := fmt.Errorf("openai embedding failed: %w", wrapError(err))
		return nil, sanitiseProviderError(wrapped, "openai embedding rejected")
	}

	embeddings := make([]llm_dto.Embedding, len(response.Data))
	for i := range response.Data {
		d := &response.Data[i]
		f32 := make([]float32, len(d.Embedding))
		for j, v := range d.Embedding {
			f32[j] = float32(v)
		}
		embeddings[i] = llm_dto.Embedding{
			Index:  int(d.Index),
			Vector: f32,
		}
	}

	return &llm_dto.EmbeddingResponse{
		Model:      response.Model,
		Embeddings: embeddings,
		Usage: &llm_dto.EmbeddingUsage{
			PromptTokens: int(response.Usage.PromptTokens),
			TotalTokens:  int(response.Usage.TotalTokens),
		},
	}, nil
}

// ListEmbeddingModels returns available embedding models from OpenAI.
//
// Returns []llm_dto.ModelInfo which contains the filtered list of embedding
// models.
// Returns error when the API request to list models fails.
func (p *openaiProvider) ListEmbeddingModels(ctx context.Context) ([]llm_dto.ModelInfo, error) {
	page, err := p.client.Models.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list openai models: %w", err)
	}

	result := make([]llm_dto.ModelInfo, 0)
	for i := range page.Data {
		model := &page.Data[i]
		if isOAEmbeddingModel(model.ID) {
			result = append(result, llm_dto.ModelInfo{
				ID:       model.ID,
				Name:     model.ID,
				Provider: "openai",
				Created:  model.Created,
			})
		}
	}
	return result, nil
}

// EmbeddingDimensions returns the default vector dimension for the configured
// embedding model.
//
// Returns int which is the vector dimension.
func (p *openaiProvider) EmbeddingDimensions() int {
	return p.embeddingDimensions
}

// buildChatParams converts a CompletionRequest to OpenAI ChatCompletionNewParams.
//
// Takes request (*llm_dto.CompletionRequest) which contains the
// completion settings to convert.
// Takes model (string) which specifies the OpenAI model identifier.
//
// Returns openai.ChatCompletionNewParams which contains the converted parameters
// ready for the OpenAI API.
func (p *openaiProvider) buildChatParams(request *llm_dto.CompletionRequest, model string) openai.ChatCompletionNewParams {
	params := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: p.convertMessages(request.Messages),
	}

	if request.Temperature != nil {
		params.Temperature = openai.Float(*request.Temperature)
	}
	if request.MaxTokens != nil {
		params.MaxCompletionTokens = openai.Int(int64(*request.MaxTokens))
	}
	if request.TopP != nil {
		params.TopP = openai.Float(*request.TopP)
	}
	if request.FrequencyPenalty != nil {
		params.FrequencyPenalty = openai.Float(*request.FrequencyPenalty)
	}
	if request.PresencePenalty != nil {
		params.PresencePenalty = openai.Float(*request.PresencePenalty)
	}
	if len(request.Stop) > 0 {
		params.Stop = openai.ChatCompletionNewParamsStopUnion{
			OfStringArray: request.Stop,
		}
	}
	if request.Seed != nil {
		params.Seed = openai.Int(*request.Seed)
	}

	if len(request.Tools) > 0 {
		params.Tools = p.convertTools(request.Tools)
	}
	if request.ToolChoice != nil {
		params.ToolChoice = p.convertToolChoice(request.ToolChoice)
	}
	if request.ParallelToolCalls != nil {
		params.ParallelToolCalls = openai.Bool(*request.ParallelToolCalls)
	}

	if request.ResponseFormat != nil {
		params.ResponseFormat = p.convertResponseFormat(request.ResponseFormat)
	}

	return params
}

// convertMessages converts an llm_dto.Message slice to OpenAI format.
//
// Takes messages ([]llm_dto.Message) which contains the messages to convert.
//
// Returns []openai.ChatCompletionMessageParamUnion which contains the
// converted messages ready for the OpenAI API.
func (p *openaiProvider) convertMessages(messages []llm_dto.Message) []openai.ChatCompletionMessageParamUnion {
	result := make([]openai.ChatCompletionMessageParamUnion, len(messages))
	for i, message := range messages {
		result[i] = p.convertMessage(message)
	}
	return result
}

// convertMessage converts a single llm_dto.Message to OpenAI format.
//
// Takes message (llm_dto.Message) which is the message to convert.
//
// Returns openai.ChatCompletionMessageParamUnion which is the converted
// message in OpenAI's expected format.
func (p *openaiProvider) convertMessage(message llm_dto.Message) openai.ChatCompletionMessageParamUnion {
	switch message.Role {
	case llm_dto.RoleSystem:
		sm := openai.SystemMessage(message.Content)
		if message.Name != nil {
			sm.OfSystem.Name = openai.String(*message.Name)
		}
		return sm
	case llm_dto.RoleAssistant:
		return p.convertAssistantMessage(message)
	case llm_dto.RoleTool:
		toolCallID := ""
		if message.ToolCallID != nil {
			toolCallID = *message.ToolCallID
		}
		return openai.ToolMessage(message.Content, toolCallID)
	default:
		return p.convertUserMessage(message)
	}
}

// convertAssistantMessage converts an assistant message,
// handling tool calls when present.
//
// Takes message (llm_dto.Message) which is the assistant message to
// convert.
//
// Returns openai.ChatCompletionMessageParamUnion which is the
// converted message in OpenAI format.
func (*openaiProvider) convertAssistantMessage(message llm_dto.Message) openai.ChatCompletionMessageParamUnion {
	if len(message.ToolCalls) > 0 {
		toolCalls := make([]openai.ChatCompletionMessageToolCallUnionParam, len(message.ToolCalls))
		for i, tc := range message.ToolCalls {
			toolCalls[i] = openai.ChatCompletionMessageToolCallUnionParam{
				OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
					ID: tc.ID,
					Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				},
			}
		}
		am := openai.ChatCompletionMessageParamUnion{
			OfAssistant: &openai.ChatCompletionAssistantMessageParam{
				Content: openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(message.Content),
				},
				ToolCalls: toolCalls,
			},
		}
		if message.Name != nil {
			am.OfAssistant.Name = openai.String(*message.Name)
		}
		return am
	}

	am := openai.AssistantMessage(message.Content)
	if message.Name != nil {
		am.OfAssistant.Name = openai.String(*message.Name)
	}
	return am
}

// convertUserMessage converts a user message, handling
// multimodal content parts when present.
//
// Takes message (llm_dto.Message) which is the user message to
// convert.
//
// Returns openai.ChatCompletionMessageParamUnion which is the
// converted message in OpenAI format.
func (p *openaiProvider) convertUserMessage(message llm_dto.Message) openai.ChatCompletionMessageParamUnion {
	var um openai.ChatCompletionMessageParamUnion
	if len(message.ContentParts) > 0 {
		um = openai.UserMessage(p.convertContentParts(message.ContentParts))
	} else {
		um = openai.UserMessage(message.Content)
	}
	if message.Name != nil {
		um.OfUser.Name = openai.String(*message.Name)
	}
	return um
}

// convertContentParts converts multimodal content parts to OpenAI content part
// format.
//
// Takes parts ([]llm_dto.ContentPart) which contains the content parts to
// convert.
//
// Returns []openai.ChatCompletionContentPartUnionParam which contains the
// converted content parts for the OpenAI API.
func (*openaiProvider) convertContentParts(parts []llm_dto.ContentPart) []openai.ChatCompletionContentPartUnionParam {
	result := make([]openai.ChatCompletionContentPartUnionParam, 0, len(parts))
	for _, part := range parts {
		if converted, ok := convertSingleContentPart(part); ok {
			result = append(result, converted)
		}
	}
	return result
}

// convertTools converts llm_dto.ToolDefinition slice to OpenAI format.
//
// Takes tools ([]llm_dto.ToolDefinition) which contains the tool definitions
// to convert.
//
// Returns []openai.ChatCompletionToolUnionParam which contains the converted
// tool parameters ready for OpenAI API calls.
func (p *openaiProvider) convertTools(tools []llm_dto.ToolDefinition) []openai.ChatCompletionToolUnionParam {
	result := make([]openai.ChatCompletionToolUnionParam, len(tools))
	for i, tool := range tools {
		funcDef := shared.FunctionDefinitionParam{
			Name: tool.Function.Name,
		}
		if tool.Function.Description != nil {
			funcDef.Description = openai.String(*tool.Function.Description)
		}
		if tool.Function.Parameters != nil {
			funcDef.Parameters = shared.FunctionParameters(p.schemaToMap(tool.Function.Parameters))
		}
		if tool.Function.Strict != nil && *tool.Function.Strict {
			funcDef.Strict = openai.Bool(true)
		}

		result[i] = openai.ChatCompletionToolUnionParam{
			OfFunction: &openai.ChatCompletionFunctionToolParam{
				Function: funcDef,
			},
		}
	}
	return result
}

// convertToolChoice converts llm_dto.ToolChoice to OpenAI format.
//
// Takes choice (*llm_dto.ToolChoice) which specifies the tool choice setting.
//
// Returns openai.ChatCompletionToolChoiceOptionUnionParam which is the OpenAI
// equivalent of the tool choice. Defaults to auto if the choice type is not
// recognised or if a function choice lacks function details.
func (*openaiProvider) convertToolChoice(choice *llm_dto.ToolChoice) openai.ChatCompletionToolChoiceOptionUnionParam {
	switch choice.Type {
	case llm_dto.ToolChoiceTypeAuto:
		return openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String("auto"),
		}
	case llm_dto.ToolChoiceTypeNone:
		return openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String("none"),
		}
	case llm_dto.ToolChoiceTypeRequired:
		return openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String("required"),
		}
	case llm_dto.ToolChoiceTypeFunction:
		if choice.Function != nil {
			return openai.ToolChoiceOptionFunctionToolChoice(
				openai.ChatCompletionNamedToolChoiceFunctionParam{
					Name: choice.Function.Name,
				},
			)
		}
	}
	return openai.ChatCompletionToolChoiceOptionUnionParam{
		OfAuto: openai.String("auto"),
	}
}

// convertResponseFormat converts llm_dto.ResponseFormat to OpenAI format.
//
// Takes format (*llm_dto.ResponseFormat) which specifies the desired response
// format type and optional JSON schema configuration.
//
// Returns openai.ChatCompletionNewParamsResponseFormatUnion which is the
// OpenAI-compatible format specification, defaulting to text format when the
// type is unrecognised or when JSON schema format lacks schema details.
func (p *openaiProvider) convertResponseFormat(format *llm_dto.ResponseFormat) openai.ChatCompletionNewParamsResponseFormatUnion {
	switch format.Type {
	case llm_dto.ResponseFormatText:
		return openai.ChatCompletionNewParamsResponseFormatUnion{
			OfText: &shared.ResponseFormatTextParam{},
		}
	case llm_dto.ResponseFormatJSONObject:
		return openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
		}
	case llm_dto.ResponseFormatJSONSchema:
		if format.JSONSchema != nil {
			schemaParam := shared.ResponseFormatJSONSchemaJSONSchemaParam{
				Name:   format.JSONSchema.Name,
				Schema: p.schemaToMap(&format.JSONSchema.Schema),
			}
			if format.JSONSchema.Description != nil {
				schemaParam.Description = openai.String(*format.JSONSchema.Description)
			}
			if format.JSONSchema.Strict != nil && *format.JSONSchema.Strict {
				schemaParam.Strict = openai.Bool(true)
			}
			return openai.ChatCompletionNewParamsResponseFormatUnion{
				OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
					JSONSchema: schemaParam,
				},
			}
		}
	}
	return openai.ChatCompletionNewParamsResponseFormatUnion{
		OfText: &shared.ResponseFormatTextParam{},
	}
}

// schemaToMap converts a JSONSchema to a map for OpenAI's parameter format.
//
// Takes schema (*llm_dto.JSONSchema) which is the schema to convert.
//
// Returns map[string]any which is the converted schema, or nil if the schema
// is nil or conversion fails.
func (*openaiProvider) schemaToMap(schema *llm_dto.JSONSchema) map[string]any {
	if schema == nil {
		return nil
	}
	data, err := json.Marshal(schema)
	if err != nil {
		return nil
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}

// convertResponse converts OpenAI's response to llm_dto.CompletionResponse.
//
// Takes completion (*openai.ChatCompletion) which is the raw OpenAI response.
//
// Returns *llm_dto.CompletionResponse which contains the converted response
// with choices, messages, tool calls, and usage statistics.
func (p *openaiProvider) convertResponse(completion *openai.ChatCompletion) *llm_dto.CompletionResponse {
	choices := make([]llm_dto.Choice, len(completion.Choices))
	for i := range completion.Choices {
		choice := &completion.Choices[i]
		message := llm_dto.Message{
			Role:    llm_dto.RoleAssistant,
			Content: choice.Message.Content,
		}

		if len(choice.Message.ToolCalls) > 0 {
			message.ToolCalls = make([]llm_dto.ToolCall, len(choice.Message.ToolCalls))
			for j := range choice.Message.ToolCalls {
				tc := &choice.Message.ToolCalls[j]
				message.ToolCalls[j] = llm_dto.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: llm_dto.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		choices[i] = llm_dto.Choice{
			Index:        int(choice.Index),
			Message:      message,
			FinishReason: p.convertFinishReason(choice.FinishReason),
		}
	}

	response := &llm_dto.CompletionResponse{
		ID:      completion.ID,
		Model:   completion.Model,
		Created: completion.Created,
		Choices: choices,
	}

	if completion.Usage.TotalTokens > 0 {
		response.Usage = &llm_dto.Usage{
			PromptTokens:     int(completion.Usage.PromptTokens),
			CompletionTokens: int(completion.Usage.CompletionTokens),
			TotalTokens:      int(completion.Usage.TotalTokens),
			CachedTokens:     int(completion.Usage.PromptTokensDetails.CachedTokens),
		}
	}

	return response
}

// convertFinishReason converts OpenAI's finish reason to llm_dto.FinishReason.
//
// Takes reason (string) which is the OpenAI finish reason string.
//
// Returns llm_dto.FinishReason which is the mapped internal finish reason.
func (*openaiProvider) convertFinishReason(reason string) llm_dto.FinishReason {
	switch reason {
	case "length":
		return llm_dto.FinishReasonLength
	case "tool_calls":
		return llm_dto.FinishReasonToolCalls
	case "content_filter":
		return llm_dto.FinishReasonContentFilter
	default:
		return llm_dto.FinishReasonStop
	}
}

// New creates a new OpenAI provider with the given settings.
//
// Takes config (Config) which contains the provider settings.
//
// Returns llm_domain.LLMProviderPort which is the configured provider.
// Returns error when the configuration is not valid.
func New(config Config) (llm_domain.LLMProviderPort, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	config = config.WithDefaults()

	httpClient := &http.Client{Timeout: httpClientTimeout}

	opts := []option.RequestOption{
		option.WithAPIKey(config.APIKey),
		option.WithHTTPClient(httpClient),
	}
	if config.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.BaseURL))
	}
	if config.Organisation != "" {
		opts = append(opts, option.WithOrganization(config.Organisation))
	}

	client := openai.NewClient(opts...)

	closeContext, closeCancel := context.WithCancelCause(context.Background())

	return &openaiProvider{
		client:                client,
		httpClient:            httpClient,
		closeContext:          closeContext,
		closeCancel:           closeCancel,
		config:                config,
		defaultModel:          config.DefaultModel,
		defaultEmbeddingModel: config.DefaultEmbeddingModel,
		embeddingDimensions:   config.EmbeddingDimensions,
	}, nil
}

// isOAChatModel checks if a model ID is a chat model.
//
// Takes id (string) which is the model identifier to check.
//
// Returns bool which is true if the model ID matches a known chat model prefix.
func isOAChatModel(id string) bool {
	chatPrefixes := []string{"gpt-5", "gpt-4o", "gpt-4", "gpt-3.5", "o1", "o3", "o4", "chatgpt"}
	for _, prefix := range chatPrefixes {
		if !strings.HasPrefix(id, prefix) {
			continue
		}
		if len(id) == len(prefix) {
			return true
		}
		next := id[len(prefix)]
		if next == '-' || next == '.' {
			return true
		}
	}
	return false
}

// isOAEmbeddingModel checks if a model ID is an embedding model.
//
// Takes id (string) which is the model identifier to check.
//
// Returns bool which is true if the model ID matches a known embedding prefix.
func isOAEmbeddingModel(id string) bool {
	return strings.HasPrefix(id, "text-embedding")
}

// convertSingleContentPart converts a single content part to
// OpenAI format.
//
// Takes part (llm_dto.ContentPart) which is the content part to
// convert.
//
// Returns openai.ChatCompletionContentPartUnionParam which is
// the converted part.
// Returns bool which is true if the conversion succeeded, or
// false when the part cannot be converted.
func convertSingleContentPart(
	part llm_dto.ContentPart,
) (openai.ChatCompletionContentPartUnionParam, bool) {
	switch part.Type {
	case llm_dto.ContentPartTypeText:
		if part.Text != nil {
			return openai.TextContentPart(*part.Text), true
		}
	case llm_dto.ContentPartTypeImageURL:
		if part.ImageURL != nil {
			imgParam := openai.ChatCompletionContentPartImageImageURLParam{
				URL: part.ImageURL.URL,
			}
			if part.ImageURL.Detail != nil {
				imgParam.Detail = *part.ImageURL.Detail
			}
			return openai.ImageContentPart(imgParam), true
		}
	case llm_dto.ContentPartTypeImageData:
		if part.ImageData != nil {
			dataURI := "data:" + part.ImageData.MIMEType + ";base64," + part.ImageData.Data
			return openai.ImageContentPart(
				openai.ChatCompletionContentPartImageImageURLParam{
					URL: dataURI,
				},
			), true
		}
	}
	return openai.ChatCompletionContentPartUnionParam{}, false
}
