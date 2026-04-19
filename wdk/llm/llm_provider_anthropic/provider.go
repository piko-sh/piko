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

package llm_provider_anthropic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/safeerror"
	"piko.sh/piko/wdk/logger"
)

const (
	// providerNameAnthropic is the provider name used in model information.
	providerNameAnthropic = "anthropic"

	// toolTypeFunction is the type identifier for function tools.
	toolTypeFunction = "function"

	// ContextWindowClaude4 is the context window size for Claude 4 family models.
	ContextWindowClaude4 = 200_000

	// ContextWindowClaude4Opus is the context window size for Claude 4 Opus
	// models.
	ContextWindowClaude4Opus = 200_000

	// MaxOutputTokensClaude4 is the maximum output tokens for Claude 4
	// Sonnet/Haiku models.
	MaxOutputTokensClaude4 = 16384

	// MaxOutputTokensClaude4Opus is the maximum output tokens for Claude 4 Opus
	// models.
	MaxOutputTokensClaude4Opus = 32000

	// httpClientTimeout is a top-level HTTP client timeout that bounds requests
	// even when the caller does not supply a per-request deadline. It is
	// generous enough to allow long-running completions but prevents stuck
	// connections from leaking goroutines indefinitely.
	httpClientTimeout = 30 * time.Minute
)

// anthropicProvider implements llm_domain.LLMProviderPort for Anthropic Claude.
type anthropicProvider struct {
	// closeContext is the provider-level context whose cancellation signals
	// background stream goroutines to exit.
	closeContext context.Context

	// closeCancel cancels closeContext on Close to signal in-flight stream
	// goroutines to wind down.
	closeCancel context.CancelCauseFunc

	// httpClient is the underlying *http.Client injected into the Anthropic
	// SDK; retained so tests can verify the configured top-level timeout and
	// idle connections can be released on Close.
	httpClient *http.Client

	// defaultModel is the model identifier to use when none is specified.
	defaultModel string

	// client is the Anthropic API client for making requests.
	client anthropic.Client

	// config holds the provider configuration settings.
	config Config

	// streamWaitGroup tracks active streaming goroutines so Close can wait for
	// them to drain.
	streamWaitGroup sync.WaitGroup

	// defaultMaxToken is the maximum number of tokens for API requests.
	defaultMaxToken int

	// closeOnce guards Close so it is idempotent.
	closeOnce sync.Once
}

var _ llm_domain.LLMProviderPort = (*anthropicProvider)(nil)

// Complete sends a completion request to Anthropic.
//
// Takes request (*llm_dto.CompletionRequest) which specifies the completion
// parameters including model, messages, and optional response format.
//
// Returns *llm_dto.CompletionResponse which contains the model's response.
// Returns error when the Anthropic API call fails.
func (p *anthropicProvider) Complete(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
	defer goroutine.RecoverPanic(ctx, "llm.anthropicProvider.Complete")

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

	if request.ResponseFormat != nil && request.ResponseFormat.Type == llm_dto.ResponseFormatJSONSchema {
		return p.completeWithStructuredOutput(ctx, request, model)
	}

	params := p.buildMessageParams(request, model)

	l.Debug("Sending Anthropic completion request",
		logger.String("model", model),
		logger.Int("message_count", len(request.Messages)),
	)

	message, err := p.client.Messages.New(ctx, params)
	if err != nil {
		completeErrorCount.Add(ctx, 1)
		wrapped := fmt.Errorf("anthropic completion failed: %w", wrapError(err))
		return nil, sanitiseProviderError(wrapped, "anthropic request rejected")
	}

	return p.convertResponse(message, model), nil
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
func (*anthropicProvider) SupportsStreaming() bool {
	return true
}

// SupportsStructuredOutput reports whether the provider supports structured
// output. Anthropic supports this via tool-use translation.
//
// Returns bool which is true when structured output is supported.
func (*anthropicProvider) SupportsStructuredOutput() bool {
	return true
}

// SupportsTools reports whether the provider supports tool calling.
//
// Returns bool which is true if the provider supports tool calling.
func (*anthropicProvider) SupportsTools() bool {
	return true
}

// SupportsPenalties reports whether the provider supports frequency and
// presence penalties.
//
// Returns bool which is false as Anthropic does not support penalties.
func (*anthropicProvider) SupportsPenalties() bool { return false }

// SupportsSeed reports whether the provider supports deterministic seed.
//
// Returns bool which is false as Anthropic does not support seed.
func (*anthropicProvider) SupportsSeed() bool { return false }

// SupportsParallelToolCalls reports whether the provider supports parallel
// tool calls.
//
// Returns bool which is false as Anthropic does not support parallel tool
// calls.
func (*anthropicProvider) SupportsParallelToolCalls() bool { return false }

// SupportsMessageName reports whether the provider supports the name field
// on messages.
//
// Returns bool which is false as Anthropic does not support message names.
func (*anthropicProvider) SupportsMessageName() bool { return false }

// ListModels returns available models.
//
// Returns []llm_dto.ModelInfo which contains the known Anthropic models.
// Returns error which is always nil as this uses a static list.
func (*anthropicProvider) ListModels(_ context.Context) ([]llm_dto.ModelInfo, error) {
	return []llm_dto.ModelInfo{
		{
			ID:                       "claude-opus-4-6",
			Name:                     "Claude Opus 4.6",
			Provider:                 providerNameAnthropic,
			ContextWindow:            ContextWindowClaude4Opus,
			MaxOutputTokens:          MaxOutputTokensClaude4Opus,
			SupportsStreaming:        true,
			SupportsTools:            true,
			SupportsStructuredOutput: true,
			SupportsVision:           true,
		},
		{
			ID:                       "claude-sonnet-4-5-20250929",
			Name:                     "Claude Sonnet 4.5",
			Provider:                 providerNameAnthropic,
			ContextWindow:            ContextWindowClaude4,
			MaxOutputTokens:          MaxOutputTokensClaude4,
			SupportsStreaming:        true,
			SupportsTools:            true,
			SupportsStructuredOutput: true,
			SupportsVision:           true,
		},
		{
			ID:                       "claude-haiku-4-5-20251001",
			Name:                     "Claude Haiku 4.5",
			Provider:                 providerNameAnthropic,
			ContextWindow:            ContextWindowClaude4,
			MaxOutputTokens:          MaxOutputTokensClaude4,
			SupportsStreaming:        true,
			SupportsTools:            true,
			SupportsStructuredOutput: true,
			SupportsVision:           true,
		},
		{
			ID:                       "claude-sonnet-4-20250514",
			Name:                     "Claude Sonnet 4",
			Provider:                 providerNameAnthropic,
			ContextWindow:            ContextWindowClaude4,
			MaxOutputTokens:          MaxOutputTokensClaude4,
			SupportsStreaming:        true,
			SupportsTools:            true,
			SupportsStructuredOutput: true,
			SupportsVision:           true,
		},
		{
			ID:                       "claude-opus-4-20250514",
			Name:                     "Claude Opus 4",
			Provider:                 providerNameAnthropic,
			ContextWindow:            ContextWindowClaude4Opus,
			MaxOutputTokens:          MaxOutputTokensClaude4Opus,
			SupportsStreaming:        true,
			SupportsTools:            true,
			SupportsStructuredOutput: true,
			SupportsVision:           true,
		},
	}, nil
}

// Close releases resources held by the provider, cancelling any in-flight
// stream goroutines and waiting for them to drain within a bounded timeout.
//
// Returns error when the provider close drain exceeds its bounded wait.
//
// Concurrency: guarded by closeOnce; cancels closeContext via closeCancel to
// signal active stream goroutines, then waits on streamWaitGroup before
// returning.
func (p *anthropicProvider) Close(ctx context.Context) error {
	var closeErr error
	p.closeOnce.Do(func() {
		if p.closeCancel != nil {
			p.closeCancel(errors.New("anthropic provider closing"))
		}

		done := make(chan struct{})
		go func() {
			defer goroutine.RecoverPanic(ctx, "llm.anthropicProvider.Close.wait")
			p.streamWaitGroup.Wait()
			close(done)
		}()

		waitContext, cancel := context.WithTimeoutCause(ctx, 30*time.Second,
			errors.New("anthropic provider close drain exceeded 30s"))
		defer cancel()

		select {
		case <-done:
		case <-waitContext.Done():
			closeErr = fmt.Errorf("anthropic provider close timed out: %w", context.Cause(waitContext))
		}

		if p.httpClient != nil {
			p.httpClient.CloseIdleConnections()
		}
	})
	return closeErr
}

// DefaultModel implements LLMProviderPort.DefaultModel.
//
// Returns string which is the default model identifier for this provider.
func (p *anthropicProvider) DefaultModel() string {
	return p.defaultModel
}

// buildMessageParams converts a CompletionRequest to Anthropic
// MessageNewParams.
//
// Takes request (*llm_dto.CompletionRequest) which contains the completion request
// to convert.
// Takes model (string) which specifies the Anthropic model to use.
//
// Returns anthropic.MessageNewParams which is the converted parameters ready
// for the Anthropic API.
func (p *anthropicProvider) buildMessageParams(request *llm_dto.CompletionRequest, model string) anthropic.MessageNewParams {
	maxTokens := p.defaultMaxToken
	if request.MaxTokens != nil {
		maxTokens = *request.MaxTokens
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: int64(maxTokens),
		Messages:  p.convertMessages(request.Messages),
	}

	system := p.extractSystemMessage(request.Messages)
	if system != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: system},
		}
	}

	if request.Temperature != nil {
		params.Temperature = anthropic.Float(*request.Temperature)
	}
	if request.TopP != nil {
		params.TopP = anthropic.Float(*request.TopP)
	}
	if len(request.Stop) > 0 {
		params.StopSequences = request.Stop
	}

	if userID, ok := request.Metadata["user_id"]; ok {
		params.Metadata = anthropic.MetadataParam{
			UserID: anthropic.String(userID),
		}
	}

	if len(request.Tools) > 0 {
		params.Tools = p.convertTools(request.Tools)
	}
	if request.ToolChoice != nil {
		params.ToolChoice = p.convertToolChoice(request.ToolChoice)
	}

	return params
}

// extractSystemMessage extracts the system message content from messages.
//
// Takes messages ([]llm_dto.Message) which contains the messages to search.
//
// Returns string which is the system message content, or empty if not found.
func (*anthropicProvider) extractSystemMessage(messages []llm_dto.Message) string {
	for _, message := range messages {
		if message.Role == llm_dto.RoleSystem {
			return message.Content
		}
	}
	return ""
}

// convertMessages converts an llm_dto.Message slice to Anthropic format.
// Filters out system messages as they are handled separately.
//
// Takes messages ([]llm_dto.Message) which contains the messages to convert.
//
// Returns []anthropic.MessageParam which contains the converted messages.
func (p *anthropicProvider) convertMessages(messages []llm_dto.Message) []anthropic.MessageParam {
	result := make([]anthropic.MessageParam, 0, len(messages))
	for _, message := range messages {
		if message.Role == llm_dto.RoleSystem {
			continue
		}
		result = append(result, p.convertMessage(message))
	}
	return result
}

// convertMessage converts a single llm_dto.Message to Anthropic format.
//
// Takes message (llm_dto.Message) which is the message to convert.
//
// Returns anthropic.MessageParam which is the converted message in Anthropic
// API format.
func (p *anthropicProvider) convertMessage(message llm_dto.Message) anthropic.MessageParam {
	switch message.Role {
	case llm_dto.RoleAssistant:
		return p.convertAssistantMessage(message)
	case llm_dto.RoleTool:
		toolCallID := ""
		if message.ToolCallID != nil {
			toolCallID = *message.ToolCallID
		}
		return anthropic.NewUserMessage(anthropic.NewToolResultBlock(toolCallID, message.Content, false))
	default:
		if len(message.ContentParts) > 0 {
			return anthropic.NewUserMessage(p.convertContentParts(message.ContentParts)...)
		}
		return anthropic.NewUserMessage(anthropic.NewTextBlock(message.Content))
	}
}

// convertAssistantMessage converts an assistant-role message to Anthropic
// format, handling both plain text and tool-call content.
//
// Takes message (llm_dto.Message) which is the assistant message to convert.
//
// Returns anthropic.MessageParam which is the converted assistant message.
func (*anthropicProvider) convertAssistantMessage(message llm_dto.Message) anthropic.MessageParam {
	if len(message.ToolCalls) == 0 {
		return anthropic.NewAssistantMessage(anthropic.NewTextBlock(message.Content))
	}

	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(message.ToolCalls)+1)
	if message.Content != "" {
		blocks = append(blocks, anthropic.NewTextBlock(message.Content))
	}
	for _, toolCall := range message.ToolCalls {
		var inputMap map[string]any
		if unmarshalError := json.Unmarshal([]byte(toolCall.Function.Arguments), &inputMap); unmarshalError != nil {
			_, warningLogger := logger.From(context.Background(), nil)
			warningLogger.Warn("failed to unmarshal tool call arguments",
				logger.String("function", toolCall.Function.Name),
				logger.Error(unmarshalError))
		}
		blocks = append(blocks, anthropic.NewToolUseBlock(toolCall.ID, inputMap, toolCall.Function.Name))
	}
	return anthropic.NewAssistantMessage(blocks...)
}

// convertContentParts converts multimodal content parts to Anthropic content
// blocks.
//
// Takes parts ([]llm_dto.ContentPart) which contains the content parts to
// convert.
//
// Returns []anthropic.ContentBlockParamUnion which contains the converted
// content blocks for the Anthropic API.
func (*anthropicProvider) convertContentParts(parts []llm_dto.ContentPart) []anthropic.ContentBlockParamUnion {
	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(parts))
	for _, part := range parts {
		switch part.Type {
		case llm_dto.ContentPartTypeText:
			if part.Text != nil {
				blocks = append(blocks, anthropic.NewTextBlock(*part.Text))
			}
		case llm_dto.ContentPartTypeImageURL:
			if part.ImageURL != nil {
				blocks = append(blocks, anthropic.NewImageBlock(anthropic.URLImageSourceParam{
					URL: part.ImageURL.URL,
				}))
			}
		case llm_dto.ContentPartTypeImageData:
			if part.ImageData != nil {
				blocks = append(blocks, anthropic.NewImageBlockBase64(
					part.ImageData.MIMEType,
					part.ImageData.Data,
				))
			}
		}
	}
	return blocks
}

// convertTools converts llm_dto.ToolDefinition slice to Anthropic format.
//
// Takes tools ([]llm_dto.ToolDefinition) which contains the tool definitions
// to convert.
//
// Returns []anthropic.ToolUnionParam which contains the converted tools ready
// for use with the Anthropic API.
func (p *anthropicProvider) convertTools(tools []llm_dto.ToolDefinition) []anthropic.ToolUnionParam {
	result := make([]anthropic.ToolUnionParam, len(tools))
	for i, tool := range tools {
		inputSchema := anthropic.ToolInputSchemaParam{
			Properties: p.schemaToProperties(tool.Function.Parameters),
		}
		if tool.Function.Parameters != nil && len(tool.Function.Parameters.Required) > 0 {
			inputSchema.ExtraFields = map[string]any{
				"required": tool.Function.Parameters.Required,
			}
		}

		toolParam := anthropic.ToolParam{
			Name:        tool.Function.Name,
			InputSchema: inputSchema,
		}
		if tool.Function.Description != nil {
			toolParam.Description = anthropic.String(*tool.Function.Description)
		}

		result[i] = anthropic.ToolUnionParam{
			OfTool: &toolParam,
		}
	}
	return result
}

// schemaToProperties converts a JSONSchema's properties to Anthropic's format.
//
// Takes schema (*llm_dto.JSONSchema) which contains the JSON schema to convert.
//
// Returns any which is a map of property names to their definitions, or an
// empty map if schema is nil or conversion fails.
func (*anthropicProvider) schemaToProperties(schema *llm_dto.JSONSchema) any {
	if schema == nil || schema.Properties == nil {
		return map[string]any{}
	}
	data, err := json.Marshal(schema.Properties)
	if err != nil {
		return map[string]any{}
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return map[string]any{}
	}
	return result
}

// convertToolChoice converts llm_dto.ToolChoice to Anthropic format.
//
// Takes choice (*llm_dto.ToolChoice) which specifies the tool selection mode.
//
// Returns anthropic.ToolChoiceUnionParam which is the Anthropic-compatible
// tool choice. Defaults to auto if the choice type is unknown or if function
// type is specified without a function name.
func (*anthropicProvider) convertToolChoice(choice *llm_dto.ToolChoice) anthropic.ToolChoiceUnionParam {
	switch choice.Type {
	case llm_dto.ToolChoiceTypeAuto:
		return anthropic.ToolChoiceUnionParam{
			OfAuto: &anthropic.ToolChoiceAutoParam{},
		}
	case llm_dto.ToolChoiceTypeNone:
		return anthropic.ToolChoiceUnionParam{
			OfNone: &anthropic.ToolChoiceNoneParam{},
		}
	case llm_dto.ToolChoiceTypeRequired:
		return anthropic.ToolChoiceUnionParam{
			OfAny: &anthropic.ToolChoiceAnyParam{},
		}
	case llm_dto.ToolChoiceTypeFunction:
		if choice.Function != nil {
			return anthropic.ToolChoiceUnionParam{
				OfTool: &anthropic.ToolChoiceToolParam{
					Name: choice.Function.Name,
				},
			}
		}
	}
	return anthropic.ToolChoiceUnionParam{
		OfAuto: &anthropic.ToolChoiceAutoParam{},
	}
}

// convertResponse converts Anthropic's response to
// llm_dto.CompletionResponse.
//
// Takes anthropicMessage (*anthropic.Message) which is the raw
// Anthropic API response.
// Takes model (string) which specifies the model name for the
// response.
//
// Returns *llm_dto.CompletionResponse which contains the
// normalised completion data including message content, tool
// calls, and usage statistics.
func (p *anthropicProvider) convertResponse(anthropicMessage *anthropic.Message, model string) *llm_dto.CompletionResponse {
	message := llm_dto.Message{
		Role: llm_dto.RoleAssistant,
	}

	for i := range anthropicMessage.Content {
		block := &anthropicMessage.Content[i]
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			message.Content += b.Text
		case anthropic.ToolUseBlock:
			inputJSON, _ := json.Marshal(b.Input)
			message.ToolCalls = append(message.ToolCalls, llm_dto.ToolCall{
				ID:   b.ID,
				Type: toolTypeFunction,
				Function: llm_dto.FunctionCall{
					Name:      b.Name,
					Arguments: string(inputJSON),
				},
			})
		}
	}

	finishReason := p.convertStopReason(anthropic.StopReason(anthropicMessage.StopReason))

	response := &llm_dto.CompletionResponse{
		ID:      anthropicMessage.ID,
		Model:   model,
		Created: time.Now().Unix(),
		Choices: []llm_dto.Choice{
			{
				Index:        0,
				Message:      message,
				FinishReason: finishReason,
			},
		},
	}

	if anthropicMessage.Usage.InputTokens > 0 || anthropicMessage.Usage.OutputTokens > 0 {
		response.Usage = &llm_dto.Usage{
			PromptTokens:     int(anthropicMessage.Usage.InputTokens),
			CompletionTokens: int(anthropicMessage.Usage.OutputTokens),
			TotalTokens:      int(anthropicMessage.Usage.InputTokens + anthropicMessage.Usage.OutputTokens),
			CachedTokens:     int(anthropicMessage.Usage.CacheReadInputTokens),
		}
	}

	return response
}

// convertStopReason converts Anthropic's stop reason to llm_dto.FinishReason.
//
// Takes reason (anthropic.StopReason) which is the Anthropic stop reason
// to convert.
//
// Returns llm_dto.FinishReason which is the corresponding finish reason.
func (*anthropicProvider) convertStopReason(reason anthropic.StopReason) llm_dto.FinishReason {
	switch reason {
	case anthropic.StopReasonMaxTokens:
		return llm_dto.FinishReasonLength
	case anthropic.StopReasonToolUse:
		return llm_dto.FinishReasonToolCalls
	default:
		return llm_dto.FinishReasonStop
	}
}

// New creates a new Anthropic provider.
//
// Takes config (Config) which contains the provider settings.
//
// Returns llm_domain.LLMProviderPort which is the configured provider.
// Returns error when the configuration is invalid.
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

	client := anthropic.NewClient(opts...)

	closeContext, closeCancel := context.WithCancelCause(context.Background())

	return &anthropicProvider{
		client:          client,
		httpClient:      httpClient,
		closeContext:    closeContext,
		closeCancel:     closeCancel,
		config:          config,
		defaultModel:    config.DefaultModel,
		defaultMaxToken: config.DefaultMaxTokens,
	}, nil
}
