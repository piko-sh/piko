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

package llm_provider_mistral

import (
	"bytes"
	"context"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/safeerror"
	"piko.sh/piko/wdk/logger"
)

const (
	// toolTypeFunction is the type value for function tools in the Mistral API.
	toolTypeFunction = "function"

	// httpClientTimeout is the time limit for HTTP requests to the Mistral API.
	// 30 minutes provides a generous upper bound for long-running completions
	// while preventing stuck connections from leaking goroutines indefinitely.
	httpClientTimeout = 30 * time.Minute

	// maxLLMResponseBytes bounds the size of a third-party HTTP response body to
	// prevent unbounded memory consumption from a hostile or malfunctioning peer.
	maxLLMResponseBytes = 16 * 1024 * 1024
)

// errResponseTruncated indicates a provider response exceeded the configured
// size cap and was truncated.
var errResponseTruncated = errors.New("mistral response exceeded maximum size")

// readBoundedBody reads up to maxLLMResponseBytes+1 bytes from body and reports
// truncation when the cap is exceeded.
//
// Takes body (io.Reader) which is the response body to read.
//
// Returns []byte which contains the read bytes (capped at maxLLMResponseBytes).
// Returns error which wraps a read failure or signals truncation.
func readBoundedBody(body io.Reader) ([]byte, error) {
	limited := io.LimitReader(body, maxLLMResponseBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return data, err
	}
	if int64(len(data)) > maxLLMResponseBytes {
		return data[:maxLLMResponseBytes], errResponseTruncated
	}
	return data, nil
}

// decodeBoundedJSON decodes JSON from body with a size cap to prevent
// unbounded memory consumption.
//
// Takes body (io.Reader) which is the response body to decode.
// Takes target (any) which receives the decoded value.
//
// Returns error when the read fails, the body is truncated, or decoding fails.
func decodeBoundedJSON(body io.Reader, target any) error {
	data, err := readBoundedBody(body)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// drainAndClose drains any remaining bytes from response.Body before closing
// so that the underlying TCP connection can be reused by the HTTP client.
//
// Takes response (*http.Response) which is the response whose body should be
// drained and closed.
func drainAndClose(response *http.Response) {
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
}

// mistralProvider implements llm_domain.LLMProviderPort and
// llm_domain.EmbeddingProviderPort for Mistral AI.
type mistralProvider struct {
	// closeContext is the provider-level context whose cancellation signals
	// background stream goroutines to exit.
	closeContext context.Context

	// client is the HTTP client used for API requests.
	client *http.Client

	// closeCancel cancels the provider-level context to signal shutdown to any
	// in-flight stream goroutines.
	closeCancel context.CancelCauseFunc

	// defaultModel is the model identifier used when none is specified.
	defaultModel string

	// defaultEmbeddingModel is the embedding model name to use when not
	// specified in a request.
	defaultEmbeddingModel string

	// config holds the provider configuration settings.
	config Config

	// streamWaitGroup tracks active streaming goroutines so Close can wait for
	// them to drain.
	streamWaitGroup sync.WaitGroup

	// embeddingDimensions is the default vector dimension for the configured
	// embedding model.
	embeddingDimensions int

	// closeOnce ensures Close is idempotent.
	closeOnce sync.Once
}

var _ llm_domain.LLMProviderPort = (*mistralProvider)(nil)

var _ llm_domain.EmbeddingProviderPort = (*mistralProvider)(nil)

// Complete sends a completion request to Mistral.
//
// Takes request (*llm_dto.CompletionRequest) which specifies the
// messages and model settings for the completion.
//
// Returns *llm_dto.CompletionResponse which contains the generated response.
// Returns error when the request to Mistral fails.
func (p *mistralProvider) Complete(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
	defer goroutine.RecoverPanic(ctx, "llm.mistralProvider.Complete")

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

	apiReq := p.buildRequest(request, model, false)

	l.Debug("Sending Mistral completion request",
		logger.String("model", model),
		logger.Int("message_count", len(request.Messages)),
	)

	response, err := p.doRequest(ctx, apiReq)
	if err != nil {
		completeErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("mistral completion failed: %w", err)
	}

	return p.convertResponse(response), nil
}

// mistralRequest holds the data sent to the Mistral API.
type mistralRequest struct {
	// Temperature controls randomness in the response; range 0.0 to 1.0.
	Temperature *float64 `json:"temperature,omitempty"`

	// MaxTokens is the maximum number of tokens to generate; nil uses the
	// model default.
	MaxTokens *int `json:"max_tokens,omitempty"`

	// TopP controls nucleus sampling probability mass; nil uses the model
	// default.
	TopP *float64 `json:"top_p,omitempty"`

	// ResponseFormat specifies the desired output format; nil uses the default.
	ResponseFormat *mistralResponseFormat `json:"response_format,omitempty"`

	// FrequencyPenalty penalises tokens based on how often they have appeared
	// so far; nil uses the model default.
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`

	// PresencePenalty penalises tokens that have appeared at all so far; nil
	// uses the model default.
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`

	// RandomSeed is an optional seed for random number generation.
	RandomSeed *int64 `json:"random_seed,omitempty"`

	// ToolChoice specifies how the model should use tools; nil uses the default.
	ToolChoice any `json:"tool_choice,omitempty"`

	// Model specifies the Mistral model identifier to use for the request.
	Model string `json:"model"`

	// Messages contains the conversation history to send to the model.
	Messages []mistralMessage `json:"messages"`

	// Stop lists sequences that cause generation to halt when produced.
	Stop []string `json:"stop,omitempty"`

	// Tools contains the tools available for the model to call.
	Tools []mistralTool `json:"tools,omitempty"`

	// Stream enables streaming responses when true.
	Stream bool `json:"stream,omitempty"`
}

// mistralMessage represents a message in the Mistral API request format.
type mistralMessage struct {
	// Role specifies the message sender, such as "user", "assistant", or "system".
	Role string `json:"role"`

	// Content holds the message body as raw JSON. For text-only messages this
	// is a JSON string; for multimodal messages it is a JSON array of content
	// part objects.
	Content stdjson.RawMessage `json:"content"`

	// Name is an optional name for the message author, used to distinguish
	// between multiple participants with the same role.
	Name string `json:"name,omitempty"`

	// ToolCallID is the identifier of the tool call this message responds to.
	ToolCallID string `json:"tool_call_id,omitempty"`

	// ToolCalls contains the tool invocations requested by the model.
	ToolCalls []mistralToolCall `json:"tool_calls,omitempty"`
}

// mistralContentPart represents a single content part in a multimodal Mistral
// message.
type mistralContentPart struct {
	// ImageURL holds the image URL reference for image content parts.
	ImageURL *mistralImageURL `json:"image_url,omitempty"`

	// Text holds the text content for text content parts.
	Text string `json:"text,omitempty"`

	// Type identifies the content part kind ("text" or "image_url").
	Type string `json:"type"`
}

// mistralImageURL holds an image URL for the Mistral API.
type mistralImageURL struct {
	// URL is the image location. Supports HTTP URLs and data URIs.
	URL string `json:"url"`
}

// mistralToolCall represents a tool invocation request from the Mistral API.
type mistralToolCall struct {
	// ID string `json:"id"` // ID is the unique identifier for this tool call.
	ID string `json:"id"`

	// Type specifies the tool type; typically "function".
	Type string `json:"type"`

	// Function contains the function name and arguments for this tool call.
	Function mistralFunctionCall `json:"function"`
}

// mistralFunctionCall holds a function call request from the Mistral API.
type mistralFunctionCall struct {
	// Name is the name of the function to call.
	Name string `json:"name"`

	// Arguments contains the JSON-encoded function arguments.
	Arguments string `json:"arguments"`
}

// mistralTool represents a tool definition in the Mistral API format.
type mistralTool struct {
	// Function specifies the function definition for this tool.
	Function mistralFunction `json:"function"`

	// Type specifies the tool type; always "function" for Mistral.
	Type string `json:"type"`
}

// mistralFunction represents a function definition for the Mistral AI API.
type mistralFunction struct {
	// Parameters holds the JSON Schema object describing function arguments.
	Parameters map[string]any `json:"parameters,omitempty"`

	// Name is the function name exposed to the model.
	Name string `json:"name"`

	// Description is the human-readable explanation of what the function does.
	Description string `json:"description,omitempty"`
}

// mistralToolChoiceFunction represents a specific tool choice targeting a named
// function.
type mistralToolChoiceFunction struct {
	// Type specifies the tool type; always "function".
	Type string `json:"type"`

	// Function identifies the function to call by name.
	Function mistralToolChoiceFunctionName `json:"function"`
}

// mistralToolChoiceFunctionName identifies a function by name for tool choice
// selection.
type mistralToolChoiceFunctionName struct {
	// Name is the name of the function to select.
	Name string `json:"name"`
}

// mistralResponseFormat specifies the output format for Mistral API responses.
type mistralResponseFormat struct {
	// Type specifies the response format type.
	Type string `json:"type"`
}

// mistralResponse holds the data returned from Mistral's API.
type mistralResponse struct {
	// ID is the unique identifier for this response.
	ID string `json:"id"`

	// Object is the type of object returned by the API.
	Object string `json:"object"`

	// Model is the identifier of the Mistral model used to generate the response.
	Model string `json:"model"`

	// Usage contains token usage statistics for this response.
	Usage *mistralUsage `json:"usage,omitempty"`

	// Choices contains the list of generated completions from the model.
	Choices []mistralChoice `json:"choices"`

	// Created is the Unix timestamp when the response was generated.
	Created int64 `json:"created"`
}

// mistralChoice represents a single response option from the Mistral API.
type mistralChoice struct {
	// FinishReason indicates why the model stopped generating tokens.
	FinishReason string `json:"finish_reason"`

	// Message contains the assistant's response content.
	Message mistralChoiceMessage `json:"message"`

	// Index is the position of this choice in the response list.
	Index int `json:"index"`
}

// mistralChoiceMessage holds the message content from a Mistral API response.
type mistralChoiceMessage struct {
	// Role is the role of the message author (e.g. "assistant", "user").
	Role string `json:"role"`

	// Content is the text content of the assistant's response.
	Content string `json:"content"`

	// ToolCalls contains the tool invocations requested by the model.
	ToolCalls []mistralToolCall `json:"tool_calls,omitempty"`
}

// mistralUsage holds token usage statistics from a Mistral API response.
type mistralUsage struct {
	// PromptTokens is the number of tokens in the input prompt.
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the number of tokens generated in the response.
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the sum of prompt and completion tokens used.
	TotalTokens int `json:"total_tokens"`
}

// SupportsStreaming reports whether the provider supports streaming.
//
// Returns bool which is true if streaming responses are supported.
func (*mistralProvider) SupportsStreaming() bool {
	return true
}

// SupportsStructuredOutput reports whether the provider supports structured
// output.
//
// Returns bool which is true if structured output is supported.
func (*mistralProvider) SupportsStructuredOutput() bool {
	return true
}

// SupportsTools reports whether the provider supports tool calling.
//
// Returns bool which is true if tool calling is supported.
func (*mistralProvider) SupportsTools() bool {
	return true
}

// SupportsPenalties reports whether the provider supports frequency and
// presence penalties.
//
// Returns bool which is true if penalties are supported.
func (*mistralProvider) SupportsPenalties() bool { return true }

// SupportsSeed reports whether the provider supports deterministic seed.
//
// Returns bool which is true if seed is supported.
func (*mistralProvider) SupportsSeed() bool { return true }

// SupportsParallelToolCalls reports whether the provider supports parallel
// tool calls.
//
// Returns bool which is false as Mistral does not support parallel tool calls.
func (*mistralProvider) SupportsParallelToolCalls() bool { return false }

// SupportsMessageName reports whether the provider supports the name field
// on messages.
//
// Returns bool which is true if message names are supported.
func (*mistralProvider) SupportsMessageName() bool { return true }

// ListModels returns available models from the Mistral API.
//
// Returns []llm_dto.ModelInfo which contains the available model details.
// Returns error when the request fails or the API returns an error.
func (p *mistralProvider) ListModels(ctx context.Context) ([]llm_dto.ModelInfo, error) {
	defer goroutine.RecoverPanic(ctx, "llm.mistralProvider.ListModels")

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.config.BaseURL+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	response, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list mistral models: %w", err)
	}
	defer drainAndClose(response)

	if response.StatusCode != http.StatusOK {
		return nil, classifyMistralAPIError(response, "mistral API error", "mistral request rejected")
	}

	var listResp struct {
		Data []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
		} `json:"data"`
	}

	if err := decodeBoundedJSON(response.Body, &listResp); err != nil {
		if errors.Is(err, errResponseTruncated) {
			return nil, fmt.Errorf("mistral models response exceeded %d bytes: %w", maxLLMResponseBytes, err)
		}
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := make([]llm_dto.ModelInfo, len(listResp.Data))
	for i, model := range listResp.Data {
		result[i] = llm_dto.ModelInfo{
			ID:                       model.ID,
			Name:                     model.ID,
			Provider:                 "mistral",
			Created:                  model.Created,
			SupportsStreaming:        true,
			SupportsTools:            true,
			SupportsStructuredOutput: true,
		}
	}
	return result, nil
}

// Close releases resources held by the Mistral provider, signalling any
// in-flight stream goroutines to exit and waiting for them to drain.
//
// Returns error when resources cannot be released within the bounded wait.
//
// Concurrency: guarded by closeOnce; cancels closeContext via closeCancel to
// signal active stream goroutines, waits on streamWaitGroup, then closes idle
// HTTP connections.
func (p *mistralProvider) Close(ctx context.Context) error {
	var closeErr error
	p.closeOnce.Do(func() {
		if p.closeCancel != nil {
			p.closeCancel(errors.New("mistral provider closing"))
		}

		done := make(chan struct{})
		go func() {
			defer goroutine.RecoverPanic(ctx, "llm.mistralProvider.Close.wait")
			p.streamWaitGroup.Wait()
			close(done)
		}()

		waitContext, cancel := context.WithTimeoutCause(ctx, 30*time.Second,
			errors.New("mistral provider close drain exceeded 30s"))
		defer cancel()

		select {
		case <-done:
		case <-waitContext.Done():
			closeErr = fmt.Errorf("mistral provider close timed out: %w", context.Cause(waitContext))
		}

		p.client.CloseIdleConnections()
	})
	return closeErr
}

// DefaultModel implements LLMProviderPort.DefaultModel.
//
// Returns string which is the default model name for this provider.
func (p *mistralProvider) DefaultModel() string {
	return p.defaultModel
}

// mistralEmbedRequest holds the data sent to the Mistral embeddings API.
type mistralEmbedRequest struct {
	// EncodingFormat specifies the output encoding; typically "float".
	EncodingFormat string `json:"encoding_format,omitempty"`

	// Model specifies the Mistral embedding model.
	Model string `json:"model"`

	// Input is the list of texts to embed.
	Input []string `json:"input"`
}

// mistralEmbedResponse holds the data returned from Mistral's embeddings API.
type mistralEmbedResponse struct {
	// Usage contains token usage statistics.
	Usage *mistralUsage `json:"usage,omitempty"`

	// ID is the unique identifier for this response.
	ID string `json:"id"`

	// Model is the model that generated the embeddings.
	Model string `json:"model"`

	// Data contains the generated embeddings.
	Data []mistralEmbedData `json:"data"`
}

// mistralEmbedData holds a single embedding from the Mistral embeddings API.
type mistralEmbedData struct {
	// Embedding is the vector of float64 values.
	Embedding []float64 `json:"embedding"`

	// Index is the position in the input list.
	Index int `json:"index"`
}

// Embed generates embeddings for the given input texts via the Mistral
// embeddings API.
//
// Takes ctx (context.Context) which controls cancellation and timeouts.
// Takes request (*llm_dto.EmbeddingRequest) which contains the embedding
// parameters.
//
// Returns *llm_dto.EmbeddingResponse which contains the generated
// embeddings.
// Returns error when the request fails.
func (p *mistralProvider) Embed(ctx context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
	defer goroutine.RecoverPanic(ctx, "llm.mistralProvider.Embed")

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

	l.Debug("Sending Mistral embedding request",
		logger.String("model", model),
		logger.Int("input_count", len(request.Input)),
	)

	apiResp, err := p.doEmbedRequest(ctx, model, request.Input)
	if err != nil {
		embedErrorCount.Add(ctx, 1)
		return nil, err
	}

	return convertEmbedResponse(apiResp), nil
}

// ListEmbeddingModels returns available models from the Mistral API.
// Mistral does not distinguish between completion and embedding models in
// its listing API, so this returns all available models.
//
// Returns []llm_dto.ModelInfo which contains model metadata.
// Returns error when the API request fails.
func (p *mistralProvider) ListEmbeddingModels(ctx context.Context) ([]llm_dto.ModelInfo, error) {
	return p.ListModels(ctx)
}

// EmbeddingDimensions returns the default vector dimension for the configured
// embedding model.
//
// Returns int which is the vector dimension.
func (p *mistralProvider) EmbeddingDimensions() int {
	return p.embeddingDimensions
}

// doEmbedRequest sends the embedding HTTP request and decodes
// the response.
//
// Takes model (string) which specifies the Mistral embedding
// model to use.
// Takes input ([]string) which contains the texts to embed.
//
// Returns *mistralEmbedResponse which holds the decoded API
// response.
// Returns error when the request fails or the API returns a
// non-OK status.
func (p *mistralProvider) doEmbedRequest(
	ctx context.Context, model string, input []string,
) (*mistralEmbedResponse, error) {
	apiReq := &mistralEmbedRequest{
		Model:          model,
		Input:          input,
		EncodingFormat: "float",
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.BaseURL+"/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	response, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("mistral embedding request failed: %w", err)
	}
	defer drainAndClose(response)

	if response.StatusCode != http.StatusOK {
		return nil, classifyMistralAPIError(response, "mistral embedding API error", "mistral embedding request rejected")
	}

	var apiResp mistralEmbedResponse
	if err := decodeBoundedJSON(response.Body, &apiResp); err != nil {
		if errors.Is(err, errResponseTruncated) {
			return nil, fmt.Errorf("mistral embedding response exceeded %d bytes: %w", maxLLMResponseBytes, err)
		}
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	return &apiResp, nil
}

// buildRequest converts a CompletionRequest to Mistral's request format.
//
// Takes request (*llm_dto.CompletionRequest) which contains the completion
// settings.
// Takes model (string) which specifies the Mistral model to use.
// Takes stream (bool) which indicates whether to stream the response.
//
// Returns *mistralRequest which is the formatted request for the Mistral API.
func (p *mistralProvider) buildRequest(request *llm_dto.CompletionRequest, model string, stream bool) *mistralRequest {
	apiReq := &mistralRequest{
		Model:    model,
		Messages: p.convertMessages(request.Messages),
		Stream:   stream,
	}

	if request.Temperature != nil {
		apiReq.Temperature = request.Temperature
	}
	if request.MaxTokens != nil {
		apiReq.MaxTokens = request.MaxTokens
	}
	if request.TopP != nil {
		apiReq.TopP = request.TopP
	}
	if len(request.Stop) > 0 {
		apiReq.Stop = request.Stop
	}
	if request.Seed != nil {
		apiReq.RandomSeed = request.Seed
	}
	if request.FrequencyPenalty != nil {
		apiReq.FrequencyPenalty = request.FrequencyPenalty
	}
	if request.PresencePenalty != nil {
		apiReq.PresencePenalty = request.PresencePenalty
	}

	if len(request.Tools) > 0 {
		apiReq.Tools = p.convertTools(request.Tools)
	}
	if request.ToolChoice != nil {
		apiReq.ToolChoice = p.convertToolChoice(request.ToolChoice)
	}

	if request.ResponseFormat != nil {
		apiReq.ResponseFormat = p.convertResponseFormat(request.ResponseFormat)
	}

	return apiReq
}

// convertMessages converts an llm_dto.Message slice to Mistral format.
//
// Takes messages ([]llm_dto.Message) which contains the messages to convert.
//
// Returns []mistralMessage which contains the converted messages.
func (p *mistralProvider) convertMessages(messages []llm_dto.Message) []mistralMessage {
	result := make([]mistralMessage, len(messages))
	for i, message := range messages {
		result[i] = p.convertMessage(message)
	}
	return result
}

// convertMessage converts a single llm_dto.Message to Mistral format.
//
// Takes message (llm_dto.Message) which is the message to convert.
//
// Returns mistralMessage which is the converted message in Mistral format.
func (p *mistralProvider) convertMessage(message llm_dto.Message) mistralMessage {
	mm := mistralMessage{
		Role: string(message.Role),
	}

	if len(message.ContentParts) > 0 && (message.Role == llm_dto.RoleUser || message.Role == llm_dto.RoleSystem) {
		mm.Content = p.convertContentParts(message.ContentParts)
	} else {
		mm.Content = marshalStringContent(message.Content)
	}

	if len(message.ToolCalls) > 0 {
		mm.ToolCalls = make([]mistralToolCall, len(message.ToolCalls))
		for i, tc := range message.ToolCalls {
			mm.ToolCalls[i] = mistralToolCall{
				ID:   tc.ID,
				Type: toolTypeFunction,
				Function: mistralFunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}

	if message.ToolCallID != nil {
		mm.ToolCallID = *message.ToolCallID
	}

	if message.Name != nil {
		mm.Name = *message.Name
	}

	return mm
}

// convertContentParts converts multimodal content parts to Mistral's
// OpenAI-compatible content array format.
//
// Takes parts ([]llm_dto.ContentPart) which contains the content parts to
// convert.
//
// Returns stdjson.RawMessage which is the JSON-encoded content array.
func (*mistralProvider) convertContentParts(parts []llm_dto.ContentPart) stdjson.RawMessage {
	mParts := make([]mistralContentPart, 0, len(parts))
	for _, part := range parts {
		switch part.Type {
		case llm_dto.ContentPartTypeText:
			if part.Text != nil {
				mParts = append(mParts, mistralContentPart{
					Type: "text",
					Text: *part.Text,
				})
			}
		case llm_dto.ContentPartTypeImageURL:
			if part.ImageURL != nil {
				mParts = append(mParts, mistralContentPart{
					Type:     "image_url",
					ImageURL: &mistralImageURL{URL: part.ImageURL.URL},
				})
			}
		case llm_dto.ContentPartTypeImageData:
			if part.ImageData != nil {
				dataURI := "data:" + part.ImageData.MIMEType + ";base64," + part.ImageData.Data
				mParts = append(mParts, mistralContentPart{
					Type:     "image_url",
					ImageURL: &mistralImageURL{URL: dataURI},
				})
			}
		}
	}
	data, _ := json.Marshal(mParts)
	return data
}

// convertTools converts llm_dto.ToolDefinition slice to Mistral format.
//
// Takes tools ([]llm_dto.ToolDefinition) which contains the tool definitions
// to convert.
//
// Returns []mistralTool which contains the converted tools in Mistral format.
func (p *mistralProvider) convertTools(tools []llm_dto.ToolDefinition) []mistralTool {
	result := make([]mistralTool, len(tools))
	for i, tool := range tools {
		mt := mistralTool{
			Type: toolTypeFunction,
			Function: mistralFunction{
				Name: tool.Function.Name,
			},
		}
		if tool.Function.Description != nil {
			mt.Function.Description = *tool.Function.Description
		}
		if tool.Function.Parameters != nil {
			mt.Function.Parameters = p.schemaToMap(tool.Function.Parameters)
		}
		result[i] = mt
	}
	return result
}

// convertToolChoice converts llm_dto.ToolChoice to Mistral format.
//
// Takes choice (*llm_dto.ToolChoice) which specifies the tool selection mode.
//
// Returns any which is the Mistral-compatible tool choice value.
func (*mistralProvider) convertToolChoice(choice *llm_dto.ToolChoice) any {
	switch choice.Type {
	case llm_dto.ToolChoiceTypeAuto:
		return "auto"
	case llm_dto.ToolChoiceTypeNone:
		return "none"
	case llm_dto.ToolChoiceTypeRequired:
		return "any"
	case llm_dto.ToolChoiceTypeFunction:
		if choice.Function != nil {
			return mistralToolChoiceFunction{
				Type:     toolTypeFunction,
				Function: mistralToolChoiceFunctionName{Name: choice.Function.Name},
			}
		}
	}
	return "auto"
}

// convertResponseFormat converts llm_dto.ResponseFormat to Mistral format.
//
// Takes format (*llm_dto.ResponseFormat) which specifies the desired response
// format type.
//
// Returns *mistralResponseFormat which contains the Mistral-specific format
// configuration.
func (*mistralProvider) convertResponseFormat(format *llm_dto.ResponseFormat) *mistralResponseFormat {
	switch format.Type {
	case llm_dto.ResponseFormatJSONObject:
		return &mistralResponseFormat{Type: "json_object"}
	default:
		return &mistralResponseFormat{Type: "text"}
	}
}

// schemaToMap converts a JSONSchema to a map for Mistral's parameter format.
//
// Takes schema (*llm_dto.JSONSchema) which is the schema to convert.
//
// Returns map[string]any which is the converted schema, or nil if the schema
// is nil or conversion fails.
func (*mistralProvider) schemaToMap(schema *llm_dto.JSONSchema) map[string]any {
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

// doRequest sends a request to the Mistral API.
//
// Takes apiReq (*mistralRequest) which contains the chat completion request.
//
// Returns *mistralResponse which contains the API response.
// Returns error when the request fails or the API returns a non-OK status.
func (p *mistralProvider) doRequest(ctx context.Context, apiReq *mistralRequest) (*mistralResponse, error) {
	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	response, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer drainAndClose(response)

	if response.StatusCode != http.StatusOK {
		return nil, classifyMistralAPIError(response, "mistral API error", "mistral request rejected")
	}

	var apiResp mistralResponse
	if err := decodeBoundedJSON(response.Body, &apiResp); err != nil {
		if errors.Is(err, errResponseTruncated) {
			return nil, fmt.Errorf("mistral response exceeded %d bytes: %w", maxLLMResponseBytes, err)
		}
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &apiResp, nil
}

// convertResponse converts Mistral's response to llm_dto.CompletionResponse.
//
// Takes response (*mistralResponse) which is the raw response
// from the Mistral API.
//
// Returns *llm_dto.CompletionResponse which is the normalised completion
// response.
func (p *mistralProvider) convertResponse(response *mistralResponse) *llm_dto.CompletionResponse {
	choices := make([]llm_dto.Choice, len(response.Choices))
	for i, choice := range response.Choices {
		message := llm_dto.Message{
			Role:    llm_dto.RoleAssistant,
			Content: choice.Message.Content,
		}

		if len(choice.Message.ToolCalls) > 0 {
			message.ToolCalls = make([]llm_dto.ToolCall, len(choice.Message.ToolCalls))
			for j, tc := range choice.Message.ToolCalls {
				message.ToolCalls[j] = llm_dto.ToolCall{
					ID:   tc.ID,
					Type: toolTypeFunction,
					Function: llm_dto.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		choices[i] = llm_dto.Choice{
			Index:        choice.Index,
			Message:      message,
			FinishReason: p.convertFinishReason(choice.FinishReason),
		}
	}

	result := &llm_dto.CompletionResponse{
		ID:      response.ID,
		Model:   response.Model,
		Created: response.Created,
		Choices: choices,
	}

	if response.Usage != nil {
		result.Usage = &llm_dto.Usage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		}
	}

	return result
}

// convertFinishReason converts Mistral's finish reason to llm_dto.FinishReason.
//
// Takes reason (string) which is the Mistral API finish reason value.
//
// Returns llm_dto.FinishReason which is the normalised finish reason.
func (*mistralProvider) convertFinishReason(reason string) llm_dto.FinishReason {
	switch reason {
	case "length", "model_length":
		return llm_dto.FinishReasonLength
	case "tool_calls":
		return llm_dto.FinishReasonToolCalls
	default:
		return llm_dto.FinishReasonStop
	}
}

// New creates a new Mistral provider with the given configuration.
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

	closeContext, closeCancel := context.WithCancelCause(context.Background())

	return &mistralProvider{
		client: &http.Client{
			Timeout: httpClientTimeout,
		},
		closeContext:          closeContext,
		closeCancel:           closeCancel,
		config:                config,
		defaultModel:          config.DefaultModel,
		defaultEmbeddingModel: config.DefaultEmbeddingModel,
		embeddingDimensions:   config.EmbeddingDimensions,
	}, nil
}

// convertEmbedResponse converts the raw Mistral embedding
// response to the domain DTO.
//
// Takes apiResp (*mistralEmbedResponse) which is the raw API
// response to convert.
//
// Returns *llm_dto.EmbeddingResponse which contains the
// normalised embeddings.
func convertEmbedResponse(apiResp *mistralEmbedResponse) *llm_dto.EmbeddingResponse {
	embeddings := make([]llm_dto.Embedding, len(apiResp.Data))
	for i := range apiResp.Data {
		d := &apiResp.Data[i]
		f32 := make([]float32, len(d.Embedding))
		for j, v := range d.Embedding {
			f32[j] = float32(v)
		}
		embeddings[i] = llm_dto.Embedding{
			Index:  d.Index,
			Vector: f32,
		}
	}

	result := &llm_dto.EmbeddingResponse{
		Model:      apiResp.Model,
		Embeddings: embeddings,
	}

	if apiResp.Usage != nil {
		result.Usage = &llm_dto.EmbeddingUsage{
			PromptTokens: apiResp.Usage.PromptTokens,
			TotalTokens:  apiResp.Usage.TotalTokens,
		}
	}

	return result
}

// marshalStringContent encodes a plain string as a JSON string for use in
// mistralMessage.Content.
//
// Takes s (string) which is the string to encode as JSON.
//
// Returns stdjson.RawMessage which contains the JSON-encoded string.
func marshalStringContent(s string) stdjson.RawMessage {
	data, _ := json.Marshal(s)
	return data
}

// classifyMistralAPIError converts a non-OK Mistral response into either a
// safeerror-wrapped 4xx (user-rejection) or a transient 5xx-style error.
// The caller is responsible for draining and closing the response body
// (typically via a deferred drainAndClose).
//
// Takes response (*http.Response) which is the non-OK upstream response.
// Takes errorContext (string) which prefixes the error message
// (e.g. "mistral API error", "mistral embedding API error").
// Takes rejectMessage (string) which is the user-safe message used when the
// status code is a 4xx client error.
//
// Returns error which carries the classified upstream failure with a
// *llm_domain.ProviderError underneath so retry classification and
// Retry-After hints flow through.
func classifyMistralAPIError(response *http.Response, errorContext, rejectMessage string) error {
	respBody, readErr := readBoundedBody(response.Body)
	detail := http.StatusText(response.StatusCode)
	if len(respBody) > 0 {
		detail = string(respBody)
	}
	if readErr != nil && !errors.Is(readErr, errResponseTruncated) {
		detail = fmt.Sprintf("%s (read error: %v)", detail, readErr)
	}
	baseErr := fmt.Errorf("%s (status %d): %s", errorContext, response.StatusCode, detail)
	providerErr := newProviderError(response, fmt.Sprintf("%s: %s", errorContext, detail), baseErr)
	if response.StatusCode >= http.StatusBadRequest && response.StatusCode < http.StatusInternalServerError {
		return safeerror.NewError(rejectMessage, providerErr)
	}
	return providerErr
}
