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

package vector_cache

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

// DefaultVectorSearchTopK is the default number of results returned by
// similarity searches when no explicit TopK is specified.
const DefaultVectorSearchTopK = 10

var (
	// errDocumentNil is returned when a nil document is passed to the vector
	// store.
	errDocumentNil = errors.New("document cannot be nil")

	// errDocumentIDEmpty is returned when a document with an empty ID is
	// passed to the vector store.
	errDocumentIDEmpty = errors.New("document ID cannot be empty")
)

// CacheFactory creates a cache instance for a given namespace configuration.
// The factory is responsible for configuring the search schema with vector
// fields and selecting the appropriate cache provider.
type CacheFactory func(namespace string, config *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error)

// Store implements llm_domain.VectorStorePort by delegating to cache
// instances. Each namespace has its own cache with vector search support.
type Store struct {
	// caches maps namespace names to their vector document caches.
	caches map[string]cache_domain.Cache[string, llm_dto.VectorDocument]

	// configs maps namespace names to their configuration.
	configs map[string]*llm_domain.VectorNamespaceConfig

	// factory creates cache instances for namespaces.
	factory CacheFactory

	// retired holds caches removed by DeleteNamespace so they can be closed
	// during Store.Close without prematurely closing shared resources (e.g.
	// a Redis client shared across namespaces).
	retired []cache_domain.Cache[string, llm_dto.VectorDocument]

	// mu guards concurrent access to store fields.
	mu sync.RWMutex

	// closed indicates the store has been shut down.
	closed bool
}

var _ llm_domain.VectorStorePort = (*Store)(nil)

// Store adds or updates a single document in the vector store.
//
// Takes namespace (string) which specifies the namespace for the document.
// Takes document (*llm_dto.VectorDocument) which is the document to store.
//
// Returns error when document is nil, document.ID is empty, or the namespace
// does not exist.
func (s *Store) Store(ctx context.Context, namespace string, document *llm_dto.VectorDocument) error {
	if document == nil {
		return errDocumentNil
	}
	if document.ID == "" {
		return errDocumentIDEmpty
	}

	c, err := s.getCache(namespace)
	if err != nil {
		return fmt.Errorf("storing document in namespace %q: %w", namespace, err)
	}

	return c.Set(ctx, document.ID, *document)
}

// BulkStore adds or updates multiple documents in a single operation.
//
// Takes namespace (string) which identifies the collection/index.
// Takes docs ([]*llm_dto.VectorDocument) which are the documents to store.
//
// Returns error when the namespace does not exist or a document is invalid.
func (s *Store) BulkStore(ctx context.Context, namespace string, docs []*llm_dto.VectorDocument) error {
	c, err := s.getCache(namespace)
	if err != nil {
		return fmt.Errorf("bulk storing in namespace %q: %w", namespace, err)
	}

	items := make(map[string]llm_dto.VectorDocument, len(docs))
	for _, document := range docs {
		if document == nil {
			continue
		}
		if document.ID == "" {
			return errDocumentIDEmpty
		}
		items[document.ID] = *document
	}

	return c.BulkSet(ctx, items)
}

// Search performs a similarity search using the provided query vector.
//
// Takes request (*llm_dto.VectorSearchRequest) which contains search parameters.
//
// Returns *llm_dto.VectorSearchResponse which contains matching documents.
// Returns error when the namespace does not exist or search fails.
func (s *Store) Search(ctx context.Context, request *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error) {
	if request == nil {
		return &llm_dto.VectorSearchResponse{}, nil
	}

	c := s.lookupCache(request.Namespace)
	if c == nil {
		return &llm_dto.VectorSearchResponse{}, nil
	}

	topK := request.TopK
	if topK <= 0 {
		topK = DefaultVectorSearchTopK
	}

	searchResult, err := c.Search(ctx, request.TextQuery, &cache_dto.SearchOptions{
		Vector:   request.Vector,
		TopK:     topK,
		MinScore: request.MinScore,
		Limit:    topK,
	})
	if err != nil {
		return nil, fmt.Errorf("cache search failed: %w", err)
	}

	response := &llm_dto.VectorSearchResponse{
		Results:    make([]llm_dto.VectorSearchResult, 0, len(searchResult.Items)),
		TotalCount: int(searchResult.Total),
	}

	for _, hit := range searchResult.Items {
		if !matchesFilter(hit.Value, request.Filter) {
			continue
		}
		result := llm_dto.VectorSearchResult{
			ID:      hit.Value.ID,
			Content: hit.Value.Content,
			Score:   float32(hit.Score),
		}
		if request.IncludeVectors {
			result.Vector = hit.Value.Vector
		}
		if request.IncludeMetadata {
			result.Metadata = hit.Value.Metadata
		}
		response.Results = append(response.Results, result)
	}

	return response, nil
}

// Get retrieves a single document by its ID.
//
// Takes namespace (string) which identifies the collection/index.
// Takes id (string) which is the document ID.
//
// Returns *llm_dto.VectorDocument which is the document, or nil if not found.
// Returns error when the namespace does not exist.
func (s *Store) Get(ctx context.Context, namespace, id string) (*llm_dto.VectorDocument, error) {
	c, err := s.getCache(namespace)
	if err != nil {
		return nil, fmt.Errorf("getting document from namespace %q: %w", namespace, err)
	}

	document, found, err := c.GetIfPresent(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting document %q from namespace %q: %w", id, namespace, err)
	}
	if !found {
		return nil, nil
	}

	return &document, nil
}

// Delete removes a document by its ID.
//
// Takes namespace (string) which identifies the collection/index.
// Takes id (string) which is the document ID.
//
// Returns error when the namespace does not exist.
func (s *Store) Delete(ctx context.Context, namespace, id string) error {
	c, err := s.getCache(namespace)
	if err != nil {
		return fmt.Errorf("deleting from namespace %q: %w", namespace, err)
	}

	return c.Invalidate(ctx, id)
}

// DeleteByFilter removes all documents matching the filter criteria.
//
// Takes namespace (string) which identifies the collection/index.
// Takes filter (map[string]any) which specifies the filter criteria.
//
// Returns int which is the number of documents deleted.
// Returns error when the namespace does not exist.
func (s *Store) DeleteByFilter(ctx context.Context, namespace string, filter map[string]any) (int, error) {
	c, err := s.getCache(namespace)
	if err != nil {
		return 0, fmt.Errorf("deleting by filter from namespace %q: %w", namespace, err)
	}

	var count int
	for id, document := range c.All() {
		if matchesFilter(document, filter) {
			_ = c.Invalidate(ctx, id)
			count++
		}
	}

	return count, nil
}

// CreateNamespace creates a new namespace backed by a cache instance.
//
// Takes namespace (string) which is the name of the namespace to create.
// Takes config (*VectorNamespaceConfig) which configures the namespace.
//
// Returns error when the factory fails to create the cache.
//
// Safe for concurrent use. Uses a mutex to protect the namespace map.
func (s *Store) CreateNamespace(_ context.Context, namespace string, config *llm_domain.VectorNamespaceConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.caches[namespace]; exists {
		return nil
	}

	c, err := s.factory(namespace, config)
	if err != nil {
		return fmt.Errorf("creating cache for namespace %q: %w", namespace, err)
	}

	s.caches[namespace] = c
	s.configs[namespace] = config
	return nil
}

// DeleteNamespace removes a namespace, invalidates all its cached
// entries, and retires the underlying cache instance.
//
// All data is cleared from the backing store (Redis, Valkey, etc.) via
// InvalidateAll before the namespace is forgotten. The cache is retired
// rather than closed immediately because some providers share a single
// client across namespaces; closing one would break others and any
// subsequent CreateNamespace calls. Retired caches are closed during
// [Store.Close].
//
// Takes namespace (string) which is the name of the namespace to delete.
//
// Returns error when closing the cache fails.
//
// Safe for concurrent use. Acquires the store's mutex for the full operation.
func (s *Store) DeleteNamespace(ctx context.Context, namespace string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, exists := s.caches[namespace]
	if !exists {
		return nil
	}

	_ = c.InvalidateAll(ctx)

	s.retired = append(s.retired, c)
	delete(s.caches, namespace)
	delete(s.configs, namespace)
	return nil
}

// Close releases all resources held by the vector store.
//
// Returns error (always nil for this implementation).
//
// Safe for concurrent use.
func (s *Store) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for namespace, c := range s.caches {
		_ = c.Close(ctx)
		delete(s.caches, namespace)
		delete(s.configs, namespace)
	}

	for _, c := range s.retired {
		_ = c.Close(ctx)
	}
	s.retired = nil

	s.closed = true
	return nil
}

// lookupCache returns the cache for the given namespace without
// auto-creating. Returns nil if the namespace does not exist.
//
// Takes namespace (string) which identifies the namespace to look up.
//
// Returns cache_domain.Cache[string, llm_dto.VectorDocument] which is
// the cache for the namespace, or nil if not found.
//
// Safe for concurrent use. Access is serialised by an internal
// read lock.
func (s *Store) lookupCache(namespace string) cache_domain.Cache[string, llm_dto.VectorDocument] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.caches[namespace]
}

// getCache returns the cache for the given namespace, creating it lazily
// with a default configuration if it does not exist.
//
// Takes namespace (string) which identifies the namespace.
//
// Returns cache_domain.Cache[string, llm_dto.VectorDocument] which is
// the cache for the namespace.
// Returns error when the store is closed or cache creation fails.
//
// Safe for concurrent use. Uses a read lock for the fast path and
// promotes to a write lock for lazy creation.
func (s *Store) getCache(namespace string) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
	s.mu.RLock()
	c, exists := s.caches[namespace]
	s.mu.RUnlock()

	if exists {
		return c, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, errors.New("store is closed")
	}

	if c, exists = s.caches[namespace]; exists {
		return c, nil
	}

	c, err := s.factory(namespace, nil)
	if err != nil {
		return nil, fmt.Errorf("auto-creating cache for namespace %q: %w", namespace, err)
	}

	s.caches[namespace] = c
	return c, nil
}

// New creates a new cache-backed vector store.
//
// Takes factory (CacheFactory) which creates cache instances per namespace.
//
// Returns *Store ready for use.
func New(factory CacheFactory) *Store {
	return &Store{
		caches:  make(map[string]cache_domain.Cache[string, llm_dto.VectorDocument]),
		configs: make(map[string]*llm_domain.VectorNamespaceConfig),
		factory: factory,
	}
}

// matchesFilter checks whether a document's metadata satisfies all filter
// criteria. Each filter key must exist in metadata and its value must match.
//
// Takes document (llm_dto.VectorDocument) which is the document to check.
// Takes filter (map[string]any) which specifies the key-value pairs to match.
//
// Returns bool which is true if all filter criteria are satisfied.
func matchesFilter(document llm_dto.VectorDocument, filter map[string]any) bool {
	if len(filter) == 0 {
		return true
	}
	if document.Metadata == nil {
		return false
	}
	for k, v := range filter {
		metaVal, exists := document.Metadata[k]
		if !exists {
			return false
		}
		if fmt.Sprintf("%v", metaVal) != fmt.Sprintf("%v", v) {
			return false
		}
	}
	return true
}
