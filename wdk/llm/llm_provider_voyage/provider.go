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

package llm_provider_voyage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/logger"
)

const (
	// httpClientTimeout is the time limit for HTTP requests to the Voyage API.
	httpClientTimeout = 5 * time.Minute

	// providerName is the provider identifier used in model listings.
	providerName = "voyage"
)

// voyageProvider implements llm_domain.EmbeddingProviderPort for the Voyage AI
// embedding service.
type voyageProvider struct {
	// client is the HTTP client used for API requests.
	client *http.Client

	// defaultModel is the embedding model name to use when not specified in a
	// request.
	defaultModel string

	// config holds the provider configuration settings.
	config Config

	// embeddingDimensions is the default vector dimension for the configured
	// embedding model.
	embeddingDimensions int
}

var _ llm_domain.EmbeddingProviderPort = (*voyageProvider)(nil)

// voyageEmbedRequest holds the data sent to the Voyage embeddings API.
type voyageEmbedRequest struct {
	// OutputDimension optionally overrides the output vector dimension.
	OutputDimension *int `json:"output_dimension,omitempty"`

	// Truncation controls whether long inputs are truncated. Defaults to true
	// server-side.
	Truncation *bool `json:"truncation,omitempty"`

	// Model specifies the Voyage embedding model.
	Model string `json:"model"`

	// InputType optionally specifies the type of input for retrieval
	// optimisation ("query" or "document").
	InputType string `json:"input_type,omitempty"`

	// Input is the list of texts to embed.
	Input []string `json:"input"`
}

// voyageEmbedResponse holds the data returned from Voyage's embeddings API.
type voyageEmbedResponse struct {
	// Usage contains token usage statistics.
	Usage *voyageUsage `json:"usage,omitempty"`

	// Model is the model that generated the embeddings.
	Model string `json:"model"`

	// Data contains the generated embeddings.
	Data []voyageEmbedData `json:"data"`
}

// voyageEmbedData holds a single embedding from the Voyage embeddings API.
type voyageEmbedData struct {
	// Embedding is the vector of float64 values.
	Embedding []float64 `json:"embedding"`

	// Index is the position in the input list.
	Index int `json:"index"`
}

// voyageUsage holds token usage statistics from a Voyage API response.
type voyageUsage struct {
	// TotalTokens is the total number of tokens used.
	TotalTokens int `json:"total_tokens"`
}

// Embed generates embeddings for the given input texts via the Voyage AI
// embeddings API.
//
// Takes ctx (context.Context) which controls cancellation and timeouts.
// Takes request (*llm_dto.EmbeddingRequest) which contains the embedding
// parameters.
//
// Returns *llm_dto.EmbeddingResponse which contains the generated
// embeddings.
// Returns error when the request fails.
func (p *voyageProvider) Embed(ctx context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
	ctx, l := logger.From(ctx, log)
	embedCount.Add(ctx, 1)
	start := time.Now()

	defer func() {
		embedDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
	}()

	model := request.Model
	if model == "" {
		model = p.defaultModel
	}

	l.Debug("Sending Voyage embedding request",
		logger.String("model", model),
		logger.Int("input_count", len(request.Input)),
	)

	apiReq := buildVoyageEmbedRequest(model, request)

	apiResp, err := p.executeEmbedRequest(ctx, apiReq)
	if err != nil {
		embedErrorCount.Add(ctx, 1)
		return nil, err
	}

	return convertVoyageResponse(apiResp), nil
}

// ListEmbeddingModels returns the known Voyage AI embedding models.
// Voyage does not provide a model listing API, so this returns a static list of
// known models.
//
// Returns []llm_dto.ModelInfo which contains model metadata.
// Returns error (always nil for Voyage).
func (*voyageProvider) ListEmbeddingModels(_ context.Context) ([]llm_dto.ModelInfo, error) {
	models := []llm_dto.ModelInfo{
		{ID: "voyage-3.5", Name: "voyage-3.5", Provider: providerName},
		{ID: "voyage-3.5-lite", Name: "voyage-3.5-lite", Provider: providerName},
		{ID: "voyage-4", Name: "voyage-4", Provider: providerName},
		{ID: "voyage-4-lite", Name: "voyage-4-lite", Provider: providerName},
		{ID: "voyage-4-large", Name: "voyage-4-large", Provider: providerName},
		{ID: "voyage-3-large", Name: "voyage-3-large", Provider: providerName},
		{ID: "voyage-code-3", Name: "voyage-code-3", Provider: providerName},
		{ID: "voyage-finance-2", Name: "voyage-finance-2", Provider: providerName},
		{ID: "voyage-law-2", Name: "voyage-law-2", Provider: providerName},
	}
	return models, nil
}

// EmbeddingDimensions returns the default vector dimension for the configured
// embedding model.
//
// Returns int which is the vector dimension.
func (p *voyageProvider) EmbeddingDimensions() int {
	return p.embeddingDimensions
}

// Close releases resources held by the Voyage provider.
//
// Returns error when resources cannot be released.
func (*voyageProvider) Close(_ context.Context) error {
	return nil
}

// executeEmbedRequest serialises the request, sends it to the Voyage API, and
// decodes the response.
//
// Takes ctx (context.Context) which controls cancellation and timeouts.
// Takes apiReq (*voyageEmbedRequest) which is the request to send.
//
// Returns *voyageEmbedResponse which contains the raw API response.
// Returns error when marshalling, sending, or decoding fails.
func (p *voyageProvider) executeEmbedRequest(ctx context.Context, apiReq *voyageEmbedRequest) (*voyageEmbedResponse, error) {
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
		return nil, fmt.Errorf("voyage embedding request failed: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("voyage embedding API error (status %d): %s", response.StatusCode, string(respBody))
	}

	var apiResp voyageEmbedResponse
	if err := json.NewDecoder(response.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	return &apiResp, nil
}

// buildVoyageEmbedRequest constructs the Voyage API request from the embedding
// parameters.
//
// Takes model (string) which is the resolved model name.
// Takes request (*llm_dto.EmbeddingRequest) which contains the
// embedding inputs and options.
//
// Returns *voyageEmbedRequest which is ready to be serialised and sent.
func buildVoyageEmbedRequest(model string, request *llm_dto.EmbeddingRequest) *voyageEmbedRequest {
	apiReq := &voyageEmbedRequest{
		Model: model,
		Input: request.Input,
	}

	if request.Dimensions != nil {
		apiReq.OutputDimension = request.Dimensions
	}

	if request.ProviderOptions != nil {
		if inputType, ok := request.ProviderOptions["input_type"].(string); ok {
			apiReq.InputType = inputType
		}
	}

	return apiReq
}

// convertVoyageResponse transforms the Voyage API response into the standard
// embedding response format.
//
// Takes apiResp (*voyageEmbedResponse) which contains the raw API embeddings.
//
// Returns *llm_dto.EmbeddingResponse which contains the converted embeddings
// with float32 vectors.
func convertVoyageResponse(apiResp *voyageEmbedResponse) *llm_dto.EmbeddingResponse {
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
			TotalTokens: apiResp.Usage.TotalTokens,
		}
	}

	return result
}
