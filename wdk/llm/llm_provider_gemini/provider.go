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

package llm_provider_gemini

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"path"
	"slices"
	"time"

	"google.golang.org/genai"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/safeconv"
)

// geminiProvider implements llm_domain.LLMProviderPort and
// llm_domain.EmbeddingProviderPort for Google Gemini.
type geminiProvider struct {
	// client is the Gemini API client for making requests.
	client *genai.Client

	// defaultModel is the model name to use when none is specified.
	defaultModel string

	// defaultEmbeddingModel is the embedding model name to use when not
	// specified in a request.
	defaultEmbeddingModel string

	// config holds the Gemini provider settings.
	config Config

	// embeddingDimensions is the default vector dimension for the configured
	// embedding model.
	embeddingDimensions int
}

var _ llm_domain.LLMProviderPort = (*geminiProvider)(nil)

var _ llm_domain.EmbeddingProviderPort = (*geminiProvider)(nil)

// Complete sends a completion request to Gemini.
//
// Takes request (*llm_dto.CompletionRequest) which specifies the completion
// parameters including model, messages, and generation settings.
//
// Returns *llm_dto.CompletionResponse which contains the generated response.
// Returns error when the Gemini API call fails.
func (p *geminiProvider) Complete(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
	ctx, l := logger.From(ctx, log)
	completeCount.Add(ctx, 1)
	start := time.Now()

	defer func() {
		completeDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
	}()

	modelName := request.Model
	if modelName == "" {
		modelName = p.defaultModel
	}

	config := p.buildGenerateContentConfig(request)
	contents := p.buildContents(request.Messages)

	l.Debug("Sending Gemini completion request",
		logger.String("model", modelName),
		logger.Int("message_count", len(request.Messages)),
	)

	response, err := p.client.Models.GenerateContent(ctx, modelName, contents, config)
	if err != nil {
		completeErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("gemini completion failed: %w", err)
	}

	return p.convertResponse(response, modelName), nil
}

// SupportsStreaming reports whether the provider supports streaming.
//
// Returns bool which is true if streaming is supported.
func (*geminiProvider) SupportsStreaming() bool {
	return true
}

// SupportsStructuredOutput reports whether the provider supports structured
// output.
//
// Returns bool which is true if structured output is supported.
func (*geminiProvider) SupportsStructuredOutput() bool {
	return true
}

// SupportsTools reports whether the provider supports tool calling.
//
// Returns bool which is true if tool calling is supported.
func (*geminiProvider) SupportsTools() bool {
	return true
}

// SupportsPenalties reports whether the provider supports frequency and
// presence penalties.
//
// Returns bool which is true if penalties are supported.
func (*geminiProvider) SupportsPenalties() bool { return true }

// SupportsSeed reports whether the provider supports deterministic seed.
//
// Returns bool which is true if seed is supported.
func (*geminiProvider) SupportsSeed() bool { return true }

// SupportsParallelToolCalls reports whether the provider supports parallel
// tool calls.
//
// Returns bool which is false as Gemini does not support parallel tool calls.
func (*geminiProvider) SupportsParallelToolCalls() bool { return false }

// SupportsMessageName reports whether the provider supports the name field
// on messages.
//
// Returns bool which is false as Gemini does not support message names.
func (*geminiProvider) SupportsMessageName() bool { return false }

// ListModels returns available models from the Gemini provider.
//
// Returns []llm_dto.ModelInfo which contains the available models.
// Returns error when the model listing fails.
func (p *geminiProvider) ListModels(ctx context.Context) ([]llm_dto.ModelInfo, error) {
	var models []llm_dto.ModelInfo

	for m, err := range p.client.Models.All(ctx) {
		if err != nil {
			return models, fmt.Errorf("gemini list models failed: %w", err)
		}
		models = append(models, llm_dto.ModelInfo{
			ID:                       m.Name,
			Name:                     m.DisplayName,
			Provider:                 "gemini",
			ContextWindow:            int(m.InputTokenLimit),
			MaxOutputTokens:          int(m.OutputTokenLimit),
			SupportsStreaming:        true,
			SupportsTools:            true,
			SupportsStructuredOutput: true,
		})
	}

	return models, nil
}

// Close releases resources.
//
// Returns error (always nil as the new SDK client does not require closing).
func (*geminiProvider) Close(_ context.Context) error {
	return nil
}

// DefaultModel returns the default model identifier.
//
// Implements LLMProviderPort.DefaultModel.
//
// Returns string which is the configured default model name.
func (p *geminiProvider) DefaultModel() string {
	return p.defaultModel
}

// Embed generates embeddings for the given input texts via the Gemini
// embedding API.
//
// Takes ctx (context.Context) which controls cancellation and timeouts.
// Takes request (*llm_dto.EmbeddingRequest) which contains the embedding
// parameters.
//
// Returns *llm_dto.EmbeddingResponse which contains the generated
// embeddings.
// Returns error when the request fails.
func (p *geminiProvider) Embed(ctx context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
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

	contents := make([]*genai.Content, len(request.Input))
	for i, text := range request.Input {
		contents[i] = genai.NewContentFromText(text, genai.RoleUser)
	}

	var embedConfig *genai.EmbedContentConfig
	if request.Dimensions != nil && *request.Dimensions > 0 {
		embedConfig = &genai.EmbedContentConfig{
			OutputDimensionality: new(safeconv.IntToInt32(*request.Dimensions)),
		}
	}

	l.Debug("Sending Gemini embedding request",
		logger.String("model", model),
		logger.Int("input_count", len(request.Input)),
	)

	response, err := p.client.Models.EmbedContent(ctx, model, contents, embedConfig)
	if err != nil {
		embedErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("gemini embedding failed: %w", err)
	}

	embeddings := make([]llm_dto.Embedding, len(response.Embeddings))
	for i, ce := range response.Embeddings {
		embeddings[i] = llm_dto.Embedding{
			Index:  i,
			Vector: ce.Values,
		}
	}

	return &llm_dto.EmbeddingResponse{
		Model:      model,
		Embeddings: embeddings,
	}, nil
}

// ListEmbeddingModels returns available embedding models from Gemini.
//
// Returns []llm_dto.ModelInfo which contains the embedding-capable models.
// Returns error when the model listing fails.
func (p *geminiProvider) ListEmbeddingModels(ctx context.Context) ([]llm_dto.ModelInfo, error) {
	var models []llm_dto.ModelInfo

	for m, err := range p.client.Models.All(ctx) {
		if err != nil {
			return models, fmt.Errorf("gemini list models failed: %w", err)
		}
		if !isGeminiEmbeddingModel(m) {
			continue
		}
		models = append(models, llm_dto.ModelInfo{
			ID:            m.Name,
			Name:          m.DisplayName,
			Provider:      "gemini",
			ContextWindow: int(m.InputTokenLimit),
		})
	}

	return models, nil
}

// EmbeddingDimensions returns the default vector dimension for the configured
// embedding model.
//
// Returns int which is the vector dimension.
func (p *geminiProvider) EmbeddingDimensions() int {
	return p.embeddingDimensions
}

// buildGenerateContentConfig creates a GenerateContentConfig from a
// completion request.
//
// Takes request (*llm_dto.CompletionRequest) which provides the configuration
// values including system instruction, temperature, max tokens, and tools.
//
// Returns *genai.GenerateContentConfig which is the configured generation
// settings for the Gemini API.
func (p *geminiProvider) buildGenerateContentConfig(request *llm_dto.CompletionRequest) *genai.GenerateContentConfig {
	config := &genai.GenerateContentConfig{}

	for _, message := range request.Messages {
		if message.Role == llm_dto.RoleSystem {
			config.SystemInstruction = genai.NewContentFromText(message.Content, genai.RoleUser)
			break
		}
	}

	if request.Temperature != nil {
		config.Temperature = new(float32(*request.Temperature))
	}
	if request.MaxTokens != nil {
		config.MaxOutputTokens = safeconv.IntToInt32(*request.MaxTokens)
	}
	if request.TopP != nil {
		config.TopP = new(float32(*request.TopP))
	}
	if len(request.Stop) > 0 {
		config.StopSequences = request.Stop
	}

	if request.FrequencyPenalty != nil {
		config.FrequencyPenalty = new(float32(*request.FrequencyPenalty))
	}
	if request.PresencePenalty != nil {
		config.PresencePenalty = new(float32(*request.PresencePenalty))
	}
	if request.Seed != nil {
		config.Seed = new(safeconv.Int64ToInt32(*request.Seed))
	}

	if len(request.Tools) > 0 {
		config.Tools = []*genai.Tool{
			{FunctionDeclarations: p.convertTools(request.Tools)},
		}
	}

	p.applyResponseFormat(config, request.ResponseFormat)

	return config
}

// applyResponseFormat configures the response MIME type and optional
// JSON schema on the generation config.
//
// Takes config (*genai.GenerateContentConfig) which is the
// generation config to modify in place.
// Takes rf (*llm_dto.ResponseFormat) which specifies the desired
// response format; nil is a no-op.
func (p *geminiProvider) applyResponseFormat(config *genai.GenerateContentConfig, rf *llm_dto.ResponseFormat) {
	if rf == nil {
		return
	}

	switch rf.Type {
	case llm_dto.ResponseFormatJSONObject:
		config.ResponseMIMEType = "application/json"
	case llm_dto.ResponseFormatJSONSchema:
		config.ResponseMIMEType = "application/json"
		if rf.JSONSchema != nil {
			config.ResponseSchema = p.convertSchema(&rf.JSONSchema.Schema)
		}
	}
}

// buildContents converts the message history to Gemini content format.
//
// Takes messages ([]llm_dto.Message) which contains the conversation history
// to convert.
//
// Returns []*genai.Content which holds the converted messages, excluding
// system messages (handled via SystemInstruction in config).
func (p *geminiProvider) buildContents(messages []llm_dto.Message) []*genai.Content {
	result := make([]*genai.Content, 0, len(messages))
	for _, message := range messages {
		if message.Role == llm_dto.RoleSystem {
			continue
		}
		result = append(result, p.convertMessage(message))
	}
	return result
}

// convertHistory converts llm_dto.Message history to Gemini format,
// excluding system messages and the last user message.
//
// Takes messages ([]llm_dto.Message) which contains the conversation history
// to convert.
//
// Returns []*genai.Content which holds the converted messages, excluding
// system messages (handled via SystemInstruction) and the last user message
// (sent separately).
func (p *geminiProvider) convertHistory(messages []llm_dto.Message) []*genai.Content {
	result := make([]*genai.Content, 0, len(messages))
	lastIndex := len(messages) - 1
	for i, message := range messages {
		if message.Role == llm_dto.RoleSystem {
			continue
		}
		if message.Role == llm_dto.RoleUser && i == lastIndex {
			continue
		}
		result = append(result, p.convertMessage(message))
	}
	return result
}

// convertMessage converts a single message to Gemini format.
//
// Takes message (llm_dto.Message) which is the message to convert.
//
// Returns *genai.Content which contains the role and parts in Gemini format.
func (p *geminiProvider) convertMessage(message llm_dto.Message) *genai.Content {
	role := "user"
	if message.Role == llm_dto.RoleAssistant {
		role = "model"
	}

	parts := make([]*genai.Part, 0)

	if len(message.ContentParts) > 0 {
		parts = append(parts, p.convertContentParts(message.ContentParts)...)
	} else if message.Content != "" {
		parts = append(parts, genai.NewPartFromText(message.Content))
	}

	for _, tc := range message.ToolCalls {
		var arguments map[string]any
		if unmarshalError := json.Unmarshal([]byte(tc.Function.Arguments), &arguments); unmarshalError != nil {
			_, warningLogger := logger.From(context.Background(), nil)
			warningLogger.Warn("failed to unmarshal tool call arguments",
				logger.String("function", tc.Function.Name),
				logger.Error(unmarshalError))
		}
		parts = append(parts, genai.NewPartFromFunctionCall(tc.Function.Name, arguments))
	}

	if message.Role == llm_dto.RoleTool && message.ToolCallID != nil {
		var result map[string]any
		if err := json.Unmarshal([]byte(message.Content), &result); err != nil {
			result = map[string]any{"result": message.Content}
		}
		parts = []*genai.Part{genai.NewPartFromFunctionResponse(*message.ToolCallID, result)}
		role = "user"
	}

	return genai.NewContentFromParts(parts, genai.Role(role))
}

// convertContentParts converts multimodal content parts to Gemini part format.
//
// Takes parts ([]llm_dto.ContentPart) which contains the content parts to
// convert.
//
// Returns []*genai.Part which contains the converted parts for the Gemini API.
func (*geminiProvider) convertContentParts(parts []llm_dto.ContentPart) []*genai.Part {
	result := make([]*genai.Part, 0, len(parts))
	for _, part := range parts {
		if p := convertSingleContentPart(part); p != nil {
			result = append(result, p)
		}
	}
	return result
}

// convertTools converts tool definitions to Gemini format.
//
// Takes tools ([]llm_dto.ToolDefinition) which contains the tool definitions
// to convert.
//
// Returns []*genai.FunctionDeclaration which contains the converted function
// declarations ready for use with the Gemini API.
func (p *geminiProvider) convertTools(tools []llm_dto.ToolDefinition) []*genai.FunctionDeclaration {
	result := make([]*genai.FunctionDeclaration, len(tools))
	for i, tool := range tools {
		fd := &genai.FunctionDeclaration{
			Name: tool.Function.Name,
		}
		if tool.Function.Description != nil {
			fd.Description = *tool.Function.Description
		}
		if tool.Function.Parameters != nil {
			fd.Parameters = p.convertSchema(tool.Function.Parameters)
		}
		result[i] = fd
	}
	return result
}

// convertSchema converts llm_dto.JSONSchema to Gemini schema.
//
// Takes schema (*llm_dto.JSONSchema) which is the schema to convert.
//
// Returns *genai.Schema which is the converted Gemini schema, or nil if the
// input is nil.
func (p *geminiProvider) convertSchema(schema *llm_dto.JSONSchema) *genai.Schema {
	if schema == nil {
		return nil
	}

	s := &genai.Schema{
		Type: p.convertSchemaType(schema.Type),
	}

	if schema.Description != nil {
		s.Description = *schema.Description
	}

	if len(schema.Properties) > 0 {
		s.Properties = make(map[string]*genai.Schema)
		for name, prop := range schema.Properties {
			s.Properties[name] = p.convertSchema(prop)
		}
	}

	if len(schema.Required) > 0 {
		s.Required = schema.Required
	}

	if schema.Items != nil {
		s.Items = p.convertSchema(schema.Items)
	}

	if len(schema.Enum) > 0 {
		for _, v := range schema.Enum {
			if str, ok := v.(string); ok {
				s.Enum = append(s.Enum, str)
			}
		}
	}

	return s
}

// convertSchemaType converts a JSON Schema type to Gemini type.
//
// Takes t (string) which is the JSON Schema type name to convert.
//
// Returns genai.Type which is the corresponding Gemini type, or
// genai.TypeUnspecified if the type is not recognised.
func (*geminiProvider) convertSchemaType(t string) genai.Type {
	switch t {
	case "string":
		return genai.TypeString
	case "number":
		return genai.TypeNumber
	case "integer":
		return genai.TypeInteger
	case "boolean":
		return genai.TypeBoolean
	case "array":
		return genai.TypeArray
	case "object":
		return genai.TypeObject
	default:
		return genai.TypeUnspecified
	}
}

// convertResponse converts Gemini's response to llm_dto.CompletionResponse.
//
// Takes response (*genai.GenerateContentResponse) which is the raw Gemini API
// response to convert.
// Takes model (string) which identifies the model name for the response.
//
// Returns *llm_dto.CompletionResponse which contains the converted response
// with choices, usage metadata, and a generated unique ID.
func (p *geminiProvider) convertResponse(response *genai.GenerateContentResponse, model string) *llm_dto.CompletionResponse {
	choices := make([]llm_dto.Choice, 0, len(response.Candidates))

	for i, candidate := range response.Candidates {
		message := llm_dto.Message{
			Role: llm_dto.RoleAssistant,
		}

		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					message.Content += part.Text
				}
				if part.FunctionCall != nil {
					argsJSON, _ := json.Marshal(part.FunctionCall.Args)
					message.ToolCalls = append(message.ToolCalls, llm_dto.ToolCall{
						ID:   fmt.Sprintf("call_%d", len(message.ToolCalls)),
						Type: "function",
						Function: llm_dto.FunctionCall{
							Name:      part.FunctionCall.Name,
							Arguments: string(argsJSON),
						},
					})
				}
			}
		}

		finishReason := p.convertFinishReason(candidate.FinishReason)
		choices = append(choices, llm_dto.Choice{
			Index:        i,
			Message:      message,
			FinishReason: finishReason,
		})
	}

	result := &llm_dto.CompletionResponse{
		ID:      fmt.Sprintf("gemini-%d", time.Now().UnixNano()),
		Model:   model,
		Created: time.Now().Unix(),
		Choices: choices,
	}

	if response.UsageMetadata != nil {
		result.Usage = &llm_dto.Usage{
			PromptTokens:     int(response.UsageMetadata.PromptTokenCount),
			CompletionTokens: int(response.UsageMetadata.CandidatesTokenCount),
			TotalTokens:      int(response.UsageMetadata.TotalTokenCount),
		}
	}

	return result
}

// convertFinishReason converts Gemini's finish reason to llm_dto.FinishReason.
//
// Takes reason (genai.FinishReason) which is the Gemini API finish reason.
//
// Returns llm_dto.FinishReason which is the normalised finish reason.
func (*geminiProvider) convertFinishReason(reason genai.FinishReason) llm_dto.FinishReason {
	switch reason {
	case genai.FinishReasonMaxTokens:
		return llm_dto.FinishReasonLength
	case genai.FinishReasonSafety, genai.FinishReasonRecitation:
		return llm_dto.FinishReasonContentFilter
	default:
		return llm_dto.FinishReasonStop
	}
}

// New creates a new Gemini provider.
//
// Takes config (Config) which holds the provider settings.
//
// Returns llm_domain.LLMProviderPort which is the configured provider.
// Returns error when the configuration is not valid.
func New(config Config) (llm_domain.LLMProviderPort, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	config = config.WithDefaults()

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  config.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	return &geminiProvider{
		client:                client,
		config:                config,
		defaultModel:          config.DefaultModel,
		defaultEmbeddingModel: config.DefaultEmbeddingModel,
		embeddingDimensions:   config.EmbeddingDimensions,
	}, nil
}

// isGeminiEmbeddingModel reports whether a Gemini model supports embedding.
//
// Takes m (*genai.Model) which is the Gemini model to check.
//
// Returns bool which is true if the model supports the embedContent action.
func isGeminiEmbeddingModel(m *genai.Model) bool {
	return slices.Contains(m.SupportedActions, "embedContent")
}

// convertSingleContentPart converts a single content part to a
// Gemini Part.
//
// Takes part (llm_dto.ContentPart) which is the content part to
// convert.
//
// Returns *genai.Part which is the converted part, or nil when the
// part cannot be converted.
func convertSingleContentPart(part llm_dto.ContentPart) *genai.Part {
	switch part.Type {
	case llm_dto.ContentPartTypeText:
		if part.Text != nil {
			return genai.NewPartFromText(*part.Text)
		}
	case llm_dto.ContentPartTypeImageURL:
		if part.ImageURL != nil {
			return genai.NewPartFromURI(
				part.ImageURL.URL,
				inferImageMIMEType(part.ImageURL.URL),
			)
		}
	case llm_dto.ContentPartTypeImageData:
		if part.ImageData != nil {
			decoded, err := base64.StdEncoding.DecodeString(part.ImageData.Data)
			if err == nil {
				return genai.NewPartFromBytes(decoded, part.ImageData.MIMEType)
			}
		}
	}
	return nil
}

// inferImageMIMEType infers the MIME type from an image URL based on its file
// extension. Returns "image/jpeg" as a fallback when the extension is not
// recognised.
//
// Takes imageURL (string) which is the URL whose file extension determines
// the MIME type.
//
// Returns string which is the inferred MIME type, defaulting to "image/jpeg".
func inferImageMIMEType(imageURL string) string {
	if mimeType := mime.TypeByExtension(path.Ext(imageURL)); mimeType != "" {
		return mimeType
	}
	return "image/jpeg"
}
