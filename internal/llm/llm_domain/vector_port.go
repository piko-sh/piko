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

package llm_domain

import (
	"context"

	"piko.sh/piko/internal/llm/llm_dto"
)

// VectorNamespaceConfig configures a vector store namespace.
type VectorNamespaceConfig struct {
	// Metric is the similarity metric to use for search.
	Metric llm_dto.SimilarityMetric

	// IndexType is the index type, which is specific to the implementation
	// (e.g. "hnsw", "ivfflat").
	IndexType string

	// Dimension is the number of elements in each embedding vector.
	Dimension int
}

// VectorStorePort is the driven port for vector storage and similarity search.
// Implementations provide storage for embedding vectors and efficient
// similarity search capabilities.
type VectorStorePort interface {
	// Store adds or updates a single document in the vector store.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes namespace (string) which identifies the collection/index.
	// Takes document (*llm_dto.VectorDocument) which is the document to store.
	//
	// Returns error if the operation fails.
	Store(ctx context.Context, namespace string, document *llm_dto.VectorDocument) error

	// BulkStore adds or updates multiple documents in a single operation.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes namespace (string) which identifies the collection/index.
	// Takes docs ([]*llm_dto.VectorDocument) which are the documents to store.
	//
	// Returns error if the operation fails.
	BulkStore(ctx context.Context, namespace string, docs []*llm_dto.VectorDocument) error

	// Search performs a similarity search using the provided query vector.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes request (*llm_dto.VectorSearchRequest) which contains search parameters.
	//
	// Returns *llm_dto.VectorSearchResponse which contains matching documents.
	// Returns error if the operation fails.
	Search(ctx context.Context, request *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error)

	// Get retrieves a single document by its ID.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes namespace (string) which identifies the collection/index.
	// Takes id (string) which is the document ID.
	//
	// Returns *llm_dto.VectorDocument which is the document, or nil if not found.
	// Returns error if the operation fails.
	Get(ctx context.Context, namespace, id string) (*llm_dto.VectorDocument, error)

	// Delete removes a document by its ID.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes namespace (string) which identifies the collection/index.
	// Takes id (string) which is the document ID.
	//
	// Returns error if the operation fails.
	Delete(ctx context.Context, namespace, id string) error

	// DeleteByFilter removes all documents matching the filter criteria.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes namespace (string) which identifies the collection/index.
	// Takes filter (map[string]any) which specifies the filter criteria.
	//
	// Returns int which is the number of documents deleted.
	// Returns error if the operation fails.
	DeleteByFilter(ctx context.Context, namespace string, filter map[string]any) (int, error)

	// CreateNamespace creates a new namespace or collection with the specified
	// configuration.
	//
	// Takes namespace (string) which is the name of the namespace to create.
	// Takes config (*VectorNamespaceConfig) which configures the namespace.
	//
	// Returns error when the operation fails.
	CreateNamespace(ctx context.Context, namespace string, config *VectorNamespaceConfig) error

	// DeleteNamespace removes a namespace and all its documents.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes namespace (string) which is the name of the namespace to delete.
	//
	// Returns error if the operation fails.
	DeleteNamespace(ctx context.Context, namespace string) error

	// Close releases any resources held by the vector store.
	//
	// Takes ctx (context.Context) which controls cancellation.
	//
	// Returns error if the operation fails.
	Close(ctx context.Context) error
}
