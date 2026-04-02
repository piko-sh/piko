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

package llm_dto

// EmbeddingRequest holds the settings for creating text embeddings.
type EmbeddingRequest struct {
	// EncodingFormat specifies the output format ("float" or "base64").
	// Nil defaults to "float".
	EncodingFormat *string

	// Dimensions specifies the output dimension (for models that support it).
	// If nil, uses the model's default dimensions.
	Dimensions *int

	// User is an optional identifier for the end user making the request.
	User *string

	// ProviderOptions holds settings specific to the chosen provider.
	ProviderOptions map[string]any

	// Metadata holds key-value pairs for tracking and logging.
	Metadata map[string]string

	// Model specifies which embedding model to use (e.g. "text-embedding-3-small").
	Model string

	// Input is the list of text strings to embed.
	Input []string
}

// EmbeddingResponse contains the response from an embedding request.
type EmbeddingResponse struct {
	// Usage holds token usage statistics; nil when usage data is not available.
	Usage *EmbeddingUsage

	// ID is the unique identifier for this embedding response.
	ID string

	// Model is the name of the model that generated the embeddings.
	Model string

	// Embeddings holds the list of generated embeddings from the response.
	Embeddings []Embedding
}

// Embedding represents a single embedding vector.
type Embedding struct {
	// Base64 is the base64-encoded embedding vector. This is set when
	// EncodingFormat is "base64".
	Base64 *string

	// Vector contains the embedding values as float32 numbers.
	// This is filled when EncodingFormat is "float" or nil.
	Vector []float32

	// Index is the position of this embedding in the response, starting from 0.
	// It matches the position of the input text in the request.
	Index int
}

// EmbeddingUsage holds token usage figures for an embedding request.
type EmbeddingUsage struct {
	// EstimatedCost holds the cost estimate for this request.
	EstimatedCost *CostEstimate

	// PromptTokens is the number of tokens in the input text.
	PromptTokens int

	// TotalTokens is the total number of tokens used in the request.
	TotalTokens int
}

// FirstEmbedding returns the first embedding from the response, or an empty
// Embedding if none exist. This is a convenience method for single-input requests.
//
// Returns Embedding which is the first embedding, or empty if none exist.
func (r *EmbeddingResponse) FirstEmbedding() Embedding {
	if len(r.Embeddings) == 0 {
		return Embedding{}
	}
	return r.Embeddings[0]
}

// FirstVector returns the first embedding vector from the response, or nil
// if none exist. This is a convenience method for single-input requests.
//
// Returns []float32 which is the first embedding vector, or nil if none exist.
func (r *EmbeddingResponse) FirstVector() []float32 {
	if len(r.Embeddings) == 0 {
		return nil
	}
	return r.Embeddings[0].Vector
}

// EncodingFormatFloat returns a pointer to "float" for use in EmbeddingRequest.
//
// Returns *string which contains the value "float".
func EncodingFormatFloat() *string {
	return new("float")
}

// EncodingFormatBase64 returns a pointer to "base64" for use in EmbeddingRequest.
//
// Returns *string which is a pointer to the literal string "base64".
func EncodingFormatBase64() *string {
	return new("base64")
}
