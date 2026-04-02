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

package llm_provider_ollama

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ollama/ollama/api"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/safeconv"
)

// ollamaProvider implements llm_domain.LLMProviderPort and
// llm_domain.EmbeddingProviderPort for Ollama.
type ollamaProvider struct {
	// client is the Ollama API client.
	client *api.Client

	// transport is the HTTP transport used by the client, kept for cleanup.
	transport *http.Transport

	// process is non-nil if we spawned the Ollama server.
	process *managedProcess

	// imageFetcher is an HTTP client used to download URL-referenced images.
	// Only set when Config.ImageFetch is non-nil.
	imageFetcher *http.Client

	// defaultModel is the model reference to use for completions.
	defaultModel ModelRef

	// defaultEmbeddingModel is the model reference to use for embeddings.
	defaultEmbeddingModel ModelRef

	// config holds the provider configuration settings.
	config Config

	// embeddingDim caches the vector dimension reported by the embedding
	// model. Populated eagerly from Show during construction, or lazily
	// from the first Embed response.
	embeddingDim atomic.Int32
}

var _ llm_domain.LLMProviderPort = (*ollamaProvider)(nil)

var _ llm_domain.EmbeddingProviderPort = (*ollamaProvider)(nil)

// Complete sends a completion request to Ollama.
//
// Takes request (*llm_dto.CompletionRequest) which specifies the prompt and model
// settings.
//
// Returns *llm_dto.CompletionResponse which contains the generated completion.
// Returns error when the API request fails.
func (p *ollamaProvider) Complete(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
	ctx, l := logger.From(ctx, log)
	completeCount.Add(ctx, 1)
	start := time.Now()

	defer func() {
		completeDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
	}()

	model, ref := p.resolveModel(request.Model, p.defaultModel)

	if err := p.ensureModel(ctx, model, ref); err != nil {
		completeErrorCount.Add(ctx, 1)
		return nil, err
	}

	chatRequest := p.buildChatRequest(ctx, request, model)
	chatRequest.Stream = new(bool)

	l.Debug("Sending Ollama completion request",
		logger.String("model", model),
		logger.Int("message_count", len(request.Messages)),
	)

	var chatResp api.ChatResponse

	err := p.client.Chat(ctx, chatRequest, func(response api.ChatResponse) error {
		chatResp = response
		return nil
	})
	if err != nil {
		completeErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("ollama completion failed: %w", wrapError(err))
	}

	return p.convertChatResponse(&chatResp, model), nil
}

// Embed generates embeddings for the given input texts.
//
// Takes request (*llm_dto.EmbeddingRequest) which contains the
// embedding parameters.
//
// Returns *llm_dto.EmbeddingResponse which contains the generated embeddings.
// Returns error when the request fails.
func (p *ollamaProvider) Embed(ctx context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
	ctx, l := logger.From(ctx, log)
	embedCount.Add(ctx, 1)
	start := time.Now()

	defer func() {
		embedDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
	}()

	model, ref := p.resolveModel(request.Model, p.defaultEmbeddingModel)

	if err := p.ensureModel(ctx, model, ref); err != nil {
		embedErrorCount.Add(ctx, 1)
		return nil, err
	}

	l.Debug("Sending Ollama embedding request",
		logger.String("model", model),
		logger.Int("input_count", len(request.Input)),
	)

	embedResp, err := p.client.Embed(ctx, &api.EmbedRequest{
		Model: model,
		Input: request.Input,
	})
	if err != nil {
		embedErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("ollama embedding failed: %w", wrapError(err))
	}

	embeddings := make([]llm_dto.Embedding, len(embedResp.Embeddings))
	for i, vec := range embedResp.Embeddings {
		f32 := make([]float32, len(vec))
		for j, v := range vec {
			f32[j] = float32(v)
		}
		embeddings[i] = llm_dto.Embedding{
			Index:  i,
			Vector: f32,
		}
	}

	if p.embeddingDim.Load() == 0 && len(embeddings) > 0 {
		p.embeddingDim.Store(safeconv.IntToInt32(len(embeddings[0].Vector)))
	}

	return &llm_dto.EmbeddingResponse{
		Model:      model,
		Embeddings: embeddings,
		Usage: &llm_dto.EmbeddingUsage{
			PromptTokens: embedResp.PromptEvalCount,
			TotalTokens:  embedResp.PromptEvalCount,
		},
	}, nil
}

// ListModels returns available models from the Ollama server.
//
// Returns []llm_dto.ModelInfo which contains model metadata.
// Returns error when the API request fails.
func (p *ollamaProvider) ListModels(ctx context.Context) ([]llm_dto.ModelInfo, error) {
	response, err := p.client.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing ollama models: %w", wrapError(err))
	}

	result := make([]llm_dto.ModelInfo, len(response.Models))
	for i := range response.Models {
		m := &response.Models[i]
		result[i] = llm_dto.ModelInfo{
			ID:                m.Name,
			Name:              m.Name,
			Provider:          "ollama",
			Created:           m.ModifiedAt.Unix(),
			SupportsStreaming: true,
		}
	}
	return result, nil
}

// ListEmbeddingModels returns available models from the Ollama server.
// Ollama does not distinguish between completion and embedding models in
// its listing API, so this returns all available models.
//
// Returns []llm_dto.ModelInfo which contains model metadata.
// Returns error when the API request fails.
func (p *ollamaProvider) ListEmbeddingModels(ctx context.Context) ([]llm_dto.ModelInfo, error) {
	return p.ListModels(ctx)
}

// EmbeddingDimensions returns the cached vector dimension for the default
// embedding model. The value is populated eagerly during construction (if the
// model is already pulled) or lazily after the first successful Embed call.
//
// Returns int which is the vector dimension, or 0 if not yet known.
func (p *ollamaProvider) EmbeddingDimensions() int {
	return int(p.embeddingDim.Load())
}

// SupportsStreaming reports whether the provider supports streaming.
//
// Returns bool which is true.
func (*ollamaProvider) SupportsStreaming() bool {
	return true
}

// SupportsStructuredOutput reports whether the provider supports structured
// output.
//
// Returns bool which is false.
func (*ollamaProvider) SupportsStructuredOutput() bool {
	return false
}

// SupportsTools reports whether the provider supports tool calling.
//
// Returns bool which is true.
func (*ollamaProvider) SupportsTools() bool {
	return true
}

// SupportsPenalties reports whether the provider supports frequency and
// presence penalties.
//
// Returns bool which is true.
func (*ollamaProvider) SupportsPenalties() bool { return true }

// SupportsSeed reports whether the provider supports deterministic seed.
//
// Returns bool which is true.
func (*ollamaProvider) SupportsSeed() bool { return true }

// SupportsParallelToolCalls reports whether the provider supports parallel
// tool calls.
//
// Returns bool which is false.
func (*ollamaProvider) SupportsParallelToolCalls() bool { return false }

// SupportsMessageName reports whether the provider supports the name field
// on messages.
//
// Returns bool which is false.
func (*ollamaProvider) SupportsMessageName() bool { return false }

// Close releases resources. If the provider started a managed Ollama
// subprocess, it is terminated.
//
// Returns error when resource cleanup fails.
func (p *ollamaProvider) Close(_ context.Context) error {
	p.transport.CloseIdleConnections()

	if p.process != nil {
		return p.process.Stop()
	}
	return nil
}

// DefaultModel returns the name of the default model.
//
// Implements LLMProviderPort.DefaultModel.
//
// Returns string which is the configured default model name.
func (p *ollamaProvider) DefaultModel() string {
	return p.defaultModel.Name
}

// tryPopulateEmbeddingDim queries the Ollama Show API for the given model and
// caches the embedding dimension if found. This is best-effort: if the model
// is not yet pulled or the server does not report the dimension, the cached
// value remains 0.
//
// Takes ctx (context.Context) which bounds the Show API call.
// Takes model (string) which is the model name to query for its embedding
// dimension.
func (p *ollamaProvider) tryPopulateEmbeddingDim(ctx context.Context, model string) {
	if p.embeddingDim.Load() != 0 {
		return
	}

	ctx, cancel := context.WithTimeoutCause(ctx, 5*time.Second, errors.New("ollama health check exceeded 5s timeout"))
	defer cancel()

	response, err := p.client.Show(ctx, &api.ShowRequest{Model: model})
	if err != nil {
		return
	}

	if dim := embeddingDimFromShow(response); dim > 0 {
		p.embeddingDim.Store(safeconv.IntToInt32(dim))
	}
}

// buildChatRequest converts a CompletionRequest to an Ollama ChatRequest.
//
// Takes ctx (context.Context) which controls cancellation for image fetches.
// Takes request (*llm_dto.CompletionRequest) which contains the
// completion settings.
// Takes model (string) which specifies the model to use.
//
// Returns *api.ChatRequest ready for the Ollama API.
func (p *ollamaProvider) buildChatRequest(ctx context.Context, request *llm_dto.CompletionRequest, model string) *api.ChatRequest {
	messages := make([]api.Message, len(request.Messages))
	for i, message := range request.Messages {
		messages[i] = p.convertMessage(ctx, message)
	}

	chatRequest := &api.ChatRequest{
		Model:    model,
		Messages: messages,
	}

	if len(request.Tools) > 0 {
		chatRequest.Tools = convertTools(request.Tools)
		chatRequest.Think = &api.ThinkValue{Value: false}
	}

	if options := buildChatOptions(request); len(options) > 0 {
		chatRequest.Options = options
	}

	if request.ResponseFormat != nil {
		switch request.ResponseFormat.Type {
		case llm_dto.ResponseFormatJSONObject, llm_dto.ResponseFormatJSONSchema:
			chatRequest.Format = json.RawMessage(`"json"`)
		}
	}

	return chatRequest
}

// convertMessage converts a single llm_dto.Message to an Ollama
// api.Message, handling multimodal content parts, tool calls,
// and tool results.
//
// Takes message (llm_dto.Message) which is the message to convert.
//
// Returns api.Message which is the converted Ollama message.
func (p *ollamaProvider) convertMessage(ctx context.Context, message llm_dto.Message) api.Message {
	m := api.Message{
		Role:    string(message.Role),
		Content: message.Content,
	}

	if len(message.ContentParts) > 0 {
		text, images := p.convertContentParts(ctx, message.ContentParts)
		m.Content = text
		m.Images = images
	}

	if message.Role == llm_dto.RoleAssistant && len(message.ToolCalls) > 0 {
		m.ToolCalls = convertDTOToolCalls(message.ToolCalls)
	}

	if message.Role == llm_dto.RoleTool && message.ToolCallID != nil {
		m.ToolCallID = *message.ToolCallID
	}

	return m
}

// convertContentParts extracts text and images from multimodal
// content parts.
//
// Takes parts ([]llm_dto.ContentPart) which contains the
// content parts to process.
//
// Returns string which is the concatenated text content.
// Returns []api.ImageData which holds decoded image bytes.
func (p *ollamaProvider) convertContentParts(
	ctx context.Context, parts []llm_dto.ContentPart,
) (string, []api.ImageData) {
	var textParts []string
	var images []api.ImageData

	for _, part := range parts {
		switch part.Type {
		case llm_dto.ContentPartTypeText:
			if part.Text != nil {
				textParts = append(textParts, *part.Text)
			}
		case llm_dto.ContentPartTypeImageData:
			if img := decodeImageData(part); img != nil {
				images = append(images, img)
			}
		case llm_dto.ContentPartTypeImageURL:
			if img := p.fetchImagePart(ctx, part); img != nil {
				images = append(images, img)
			}
		}
	}

	return strings.Join(textParts, ""), images
}

// fetchImagePart downloads an image from a URL content part.
//
// Takes part (llm_dto.ContentPart) which contains the image URL
// to fetch.
//
// Returns api.ImageData which is the downloaded bytes, or nil
// when the part has no URL or fetching fails.
func (p *ollamaProvider) fetchImagePart(ctx context.Context, part llm_dto.ContentPart) api.ImageData {
	if part.ImageURL == nil || p.imageFetcher == nil {
		return nil
	}
	data, err := p.fetchImage(ctx, part.ImageURL.URL)
	if err != nil {
		return nil
	}
	return data
}

// fetchImage downloads an image from the given URL, respecting the configured
// size limit.
//
// Takes imageURL (string) which is the URL of the image to download.
//
// Returns the raw image bytes or an error when fetching fails, the response
// status is not OK, or the image exceeds the configured size limit.
func (p *ollamaProvider) fetchImage(ctx context.Context, imageURL string) ([]byte, error) {
	if p.imageFetcher == nil {
		return nil, errors.New("image fetching is not enabled")
	}

	maxBytes := p.config.ImageFetch.MaxBytes
	timeout := p.config.ImageFetch.Timeout

	ctx, cancel := context.WithTimeoutCause(ctx, timeout,
		fmt.Errorf("image fetch timed out after %s for %s", timeout, imageURL),
	)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating image request: %w", err)
	}

	response, err := p.imageFetcher.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetching image: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching image: HTTP %d", response.StatusCode)
	}

	limited := io.LimitReader(response.Body, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("reading image body: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("image exceeds maximum size of %d bytes", maxBytes)
	}

	return data, nil
}

// convertChatResponse converts an Ollama ChatResponse to a CompletionResponse.
//
// Takes response (*api.ChatResponse) which is the Ollama response.
// Takes model (string) which is the model that was used.
//
// Returns *llm_dto.CompletionResponse with the converted data.
func (*ollamaProvider) convertChatResponse(response *api.ChatResponse, model string) *llm_dto.CompletionResponse {
	message := llm_dto.Message{
		Role:    llm_dto.RoleAssistant,
		Content: response.Message.Content,
	}

	finishReason := llm_dto.FinishReasonStop

	if len(response.Message.ToolCalls) > 0 {
		message.ToolCalls = convertOllamaToolCalls(response.Message.ToolCalls)
		finishReason = llm_dto.FinishReasonToolCalls
	}

	return &llm_dto.CompletionResponse{
		Model: model,
		Choices: []llm_dto.Choice{
			{
				Index:        0,
				Message:      message,
				FinishReason: finishReason,
			},
		},
		Usage: &llm_dto.Usage{
			PromptTokens:     response.PromptEvalCount,
			CompletionTokens: response.EvalCount,
			TotalTokens:      response.PromptEvalCount + response.EvalCount,
		},
	}
}

// newProvider creates a new Ollama provider with the given settings.
//
// Takes config (Config) which contains the provider settings.
//
// Returns *ollamaProvider which is the configured provider that also
// implements llm_domain.EmbeddingProviderPort.
// Returns error when the configuration is not valid or the server cannot
// be reached or started.
func newProvider(config Config) (*ollamaProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	config = config.WithDefaults()

	p := &ollamaProvider{
		config:                config,
		defaultModel:          config.DefaultModel,
		defaultEmbeddingModel: config.DefaultEmbeddingModel,
	}

	if !isServerReachable(config.Host) {
		if !*config.AutoStart {
			return nil, fmt.Errorf(
				"ollama server not reachable at %s and AutoStart is disabled",
				config.Host,
			)
		}

		_, l := logger.From(context.Background(), log)
		l.Info("Ollama server not reachable, starting managed instance",
			logger.String("host", config.Host),
		)

		proc, err := startOllama(config.BinaryPath, config.Host)
		if err != nil {
			return nil, fmt.Errorf("auto-starting ollama: %w", err)
		}
		p.process = proc
	}

	ollamaURL, err := url.Parse(config.Host)
	if err != nil {
		return nil, fmt.Errorf("parsing ollama host URL: %w", err)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	p.transport = transport
	p.client = api.NewClient(ollamaURL, &http.Client{
		Transport: transport,
		Timeout:   config.HTTPTimeout,
	})

	if config.ImageFetch != nil {
		p.imageFetcher = &http.Client{
			Timeout: config.ImageFetch.Timeout,
		}
	}

	p.tryPopulateEmbeddingDim(context.Background(), config.DefaultEmbeddingModel.Name)

	return p, nil
}

// embeddingDimFromShow extracts the embedding vector dimension from an Ollama
// ShowResponse. The dimension is stored in ModelInfo under the key
// "<family>.embedding_length" where family comes from Details.Family.
//
// Takes response (*api.ShowResponse) which contains the model metadata including
// family and embedding length.
//
// Returns int which is the embedding dimension, or 0 when the dimension
// cannot be determined.
func embeddingDimFromShow(response *api.ShowResponse) int {
	family := response.Details.Family
	if family == "" {
		return 0
	}

	key := family + ".embedding_length"

	v, ok := response.ModelInfo[key]
	if !ok {
		return 0
	}

	f, ok := v.(float64)
	if !ok || f <= 0 {
		return 0
	}

	return int(f)
}

// decodeImageData decodes base64 image data from a content part.
//
// Takes part (llm_dto.ContentPart) which contains the base64
// encoded image data.
//
// Returns api.ImageData which is the decoded bytes, or nil when
// the part has no image data or decoding fails.
func decodeImageData(part llm_dto.ContentPart) api.ImageData {
	if part.ImageData == nil {
		return nil
	}
	decoded, err := base64.StdEncoding.DecodeString(part.ImageData.Data)
	if err != nil {
		return nil
	}
	return decoded
}

// buildChatOptions builds the Ollama options map from the
// completion request.
//
// Takes request (*llm_dto.CompletionRequest) which provides the
// generation parameters to translate into Ollama options.
//
// Returns map[string]any which holds the Ollama option keys and
// values.
func buildChatOptions(request *llm_dto.CompletionRequest) map[string]any {
	options := map[string]any{}
	if request.Temperature != nil {
		options["temperature"] = *request.Temperature
	}
	if request.TopP != nil {
		options["top_p"] = *request.TopP
	}
	if request.MaxTokens != nil {
		options["num_predict"] = *request.MaxTokens
	}
	if len(request.Stop) > 0 {
		options["stop"] = request.Stop
	}
	if request.Seed != nil {
		options["seed"] = *request.Seed
	}
	if request.FrequencyPenalty != nil {
		options["frequency_penalty"] = *request.FrequencyPenalty
	}
	if request.PresencePenalty != nil {
		options["presence_penalty"] = *request.PresencePenalty
	}
	maps.Copy(options, request.ProviderOptions)
	return options
}

// convertTools maps DTO tool definitions to Ollama's api.Tools
// type.
//
// Takes tools ([]llm_dto.ToolDefinition) which contains the
// tool definitions to convert.
//
// Returns api.Tools which holds the converted Ollama tools.
func convertTools(tools []llm_dto.ToolDefinition) api.Tools {
	out := make(api.Tools, len(tools))
	for i, td := range tools {
		params := api.ToolFunctionParameters{
			Type: "object",
		}

		if td.Function.Parameters != nil {
			params.Required = td.Function.Parameters.Required
			params.Properties = convertSchemaProperties(td.Function.Parameters.Properties)
		}

		description := ""
		if td.Function.Description != nil {
			description = *td.Function.Description
		}

		out[i] = api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        td.Function.Name,
				Description: description,
				Parameters:  params,
			},
		}
	}
	return out
}

// convertSchemaProperties maps JSONSchema properties to
// Ollama's ordered ToolPropertiesMap, sorting keys
// alphabetically for deterministic output.
//
// Takes props (map[string]*llm_dto.JSONSchema) which contains
// the schema properties to convert.
//
// Returns *api.ToolPropertiesMap which holds the converted
// properties, or nil when props is empty.
func convertSchemaProperties(props map[string]*llm_dto.JSONSchema) *api.ToolPropertiesMap {
	if len(props) == 0 {
		return nil
	}

	keys := slices.Sorted(maps.Keys(props))

	pm := api.NewToolPropertiesMap()
	for _, k := range keys {
		pm.Set(k, convertSchemaToProperty(props[k]))
	}
	return pm
}

// convertSchemaToProperty recursively converts a single
// JSONSchema to an Ollama ToolProperty.
//
// Takes schema (*llm_dto.JSONSchema) which is the schema to
// convert.
//
// Returns api.ToolProperty which holds the converted property.
func convertSchemaToProperty(schema *llm_dto.JSONSchema) api.ToolProperty {
	if schema == nil {
		return api.ToolProperty{}
	}

	prop := api.ToolProperty{
		Type: api.PropertyType{schema.Type},
		Enum: schema.Enum,
	}

	if schema.Description != nil {
		prop.Description = *schema.Description
	}

	if len(schema.Properties) > 0 {
		prop.Properties = convertSchemaProperties(schema.Properties)
	}

	if len(schema.AnyOf) > 0 {
		prop.AnyOf = make([]api.ToolProperty, len(schema.AnyOf))
		for i, s := range schema.AnyOf {
			prop.AnyOf[i] = convertSchemaToProperty(s)
		}
	}

	return prop
}

// convertOllamaToolCalls maps Ollama tool calls to the DTO
// representation.
//
// Takes calls ([]api.ToolCall) which contains the Ollama tool
// calls to convert.
//
// Returns []llm_dto.ToolCall which holds the converted calls.
func convertOllamaToolCalls(calls []api.ToolCall) []llm_dto.ToolCall {
	out := make([]llm_dto.ToolCall, len(calls))
	for i, tc := range calls {
		out[i] = llm_dto.ToolCall{
			ID:   tc.ID,
			Type: "function",
			Function: llm_dto.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments.String(),
			},
		}
	}
	return out
}

// convertDTOToolCalls maps DTO tool calls back to Ollama's
// api.ToolCall for assistant messages that contain prior tool
// calls in the conversation history.
//
// Takes calls ([]llm_dto.ToolCall) which contains the DTO tool
// calls to convert.
//
// Returns []api.ToolCall which holds the converted Ollama calls.
func convertDTOToolCalls(calls []llm_dto.ToolCall) []api.ToolCall {
	out := make([]api.ToolCall, len(calls))
	for i, tc := range calls {
		var arguments api.ToolCallFunctionArguments
		if unmarshalError := json.Unmarshal([]byte(tc.Function.Arguments), &arguments); unmarshalError != nil {
			_, warningLogger := logger.From(context.Background(), nil)
			warningLogger.Warn("failed to unmarshal tool call arguments",
				logger.String("function", tc.Function.Name),
				logger.Error(unmarshalError))
		}

		out[i] = api.ToolCall{
			ID: tc.ID,
			Function: api.ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: arguments,
			},
		}
	}
	return out
}
