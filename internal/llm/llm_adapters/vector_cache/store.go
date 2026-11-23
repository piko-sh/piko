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
	"math"
	"sync"

	"golang.org/x/sync/singleflight"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/safeconv"
	"piko.sh/vectormaths"
)

const (
	// DefaultVectorSearchTopK is the default number of results returned by similarity
	// searches when no explicit TopK is specified.
	DefaultVectorSearchTopK = 10

	// DefaultMaxFilteredCandidates is the default ceiling on the ranked candidates a
	// filtered or thresholded search inspects before returning fewer than TopK results.
	DefaultMaxFilteredCandidates = 10_000
)

var (
	// log is the package-level logger for the vector store.
	log = logger_domain.GetLogger("piko/internal/llm/llm_adapters/vector_cache")

	// errDocumentNil is returned when a nil document is passed to the vector store.
	errDocumentNil = errors.New("document cannot be nil")

	// errDocumentIDEmpty is returned when a document with an empty ID is passed to the
	// vector store.
	errDocumentIDEmpty = errors.New("document ID cannot be empty")

	// errStoreClosed is returned when a namespace is used after the store is closed.
	errStoreClosed = errors.New("vector store is closed")

	// errNamespaceDeleting is returned when a namespace began deleting before its cache
	// could be created, so the caller waits for the deletion and tries again.
	errNamespaceDeleting = errors.New("vector namespace is being deleted")

	// errUnexpectedCacheType is returned when a shared cache creation yields a value that is
	// not a cache.
	errUnexpectedCacheType = errors.New("cache creation returned an unexpected type")
)

// CacheFactory creates a cache instance for a given namespace configuration.
//
// The context bounds the creation, which can be a network call.
type CacheFactory func(
	ctx context.Context,
	namespace string,
	config *llm_domain.VectorNamespaceConfig,
) (cache_domain.Cache[string, llm_dto.VectorDocument], error)

// Store implements llm_domain.VectorStorePort by delegating to cache instances.
type Store struct {
	// creating runs the factory once per namespace when callers race to create it.
	creating singleflight.Group

	// caches maps namespace names to their vector document caches.
	caches map[string]cache_domain.Cache[string, llm_dto.VectorDocument]

	// configs maps namespace names to their configuration.
	configs map[string]*llm_domain.VectorNamespaceConfig

	// factory creates cache instances for namespaces.
	factory CacheFactory

	// deleting maps each namespace whose entries are being invalidated to a channel that is
	// closed once the invalidation returns.
	deleting map[string]chan struct{}

	// retired keeps removed caches open until Close to preserve shared resources.
	retired []cache_domain.Cache[string, llm_dto.VectorDocument]

	// maxCandidates caps the ranked candidates a filtered search inspects.
	maxCandidates int

	// mu guards concurrent access to store fields.
	mu sync.RWMutex

	// closed indicates the store has been shut down.
	closed bool
}

// Option configures a Store.
type Option func(*Store)

var (
	_ llm_domain.VectorStorePort = (*Store)(nil)
)

// Store adds or updates a single document in the vector store.
//
// Takes namespace (string) which specifies the namespace for the document.
// Takes document (*llm_dto.VectorDocument) which is the document to store.
//
// Returns error when document is nil, its ID is empty, or cache creation, vector
// validation, or storage fails. Invalid vectors wrap llm_domain.ErrInvalidVector.
func (s *Store) Store(ctx context.Context, namespace string, document *llm_dto.VectorDocument) error {
	if document == nil {
		return errDocumentNil
	}
	if document.ID == "" {
		return errDocumentIDEmpty
	}

	if err := validateVector(document.Vector, s.lookupConfig(namespace)); err != nil {
		return fmt.Errorf("storing document %q in namespace %q: %w", document.ID, namespace, err)
	}

	c, err := s.getCache(ctx, namespace)
	if err != nil {
		return fmt.Errorf("storing document in namespace %q: %w", namespace, err)
	}

	return c.Set(ctx, document.ID, *document)
}

// BulkStore adds or updates multiple documents in a single operation.
//
// Nil documents are skipped.
//
// Takes namespace (string) which identifies the collection/index.
// Takes docs ([]*llm_dto.VectorDocument) which are the documents to store.
//
// Returns error when the namespace cache cannot be created or a document is invalid.
// Invalid vectors wrap llm_domain.ErrInvalidVector. Nothing is stored when a document is
// invalid.
func (s *Store) BulkStore(ctx context.Context, namespace string, docs []*llm_dto.VectorDocument) error {
	c, err := s.getCache(ctx, namespace)
	if err != nil {
		return fmt.Errorf("bulk storing in namespace %q: %w", namespace, err)
	}

	config := s.lookupConfig(namespace)
	items := make(map[string]llm_dto.VectorDocument, len(docs))
	for _, document := range docs {
		if document == nil {
			continue
		}
		if document.ID == "" {
			return errDocumentIDEmpty
		}
		if err := validateVector(document.Vector, config); err != nil {
			return fmt.Errorf("bulk storing document %q in namespace %q: %w", document.ID, namespace, err)
		}
		items[document.ID] = *document
	}

	return c.BulkSet(ctx, items)
}

// Search performs a similarity search using the provided query vector.
//
// Takes request (*llm_dto.VectorSearchRequest) which contains search parameters.
//
// Returns *llm_dto.VectorSearchResponse which contains matching documents, or none when
// the namespace does not exist.
// Returns error when the search fails, or wraps both cache_domain.ErrInvalidQueryVector
// and llm_domain.ErrInvalidVector when a query vector is given that the namespace cannot
// compare.
func (s *Store) Search(ctx context.Context, request *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error) {
	if request == nil {
		return &llm_dto.VectorSearchResponse{}, nil
	}

	c := s.lookupCache(request.Namespace)
	if c == nil {
		return &llm_dto.VectorSearchResponse{}, nil
	}
	if len(request.Vector) > 0 {
		if err := validateVector(request.Vector, s.lookupConfig(request.Namespace)); err != nil {
			return nil, fmt.Errorf("searching namespace %q: %w: %w", request.Namespace, cache_domain.ErrInvalidQueryVector, err)
		}
	}

	ceiling := s.candidateCeiling()
	topK := request.TopK
	if topK <= 0 {
		topK = DefaultVectorSearchTopK
	}
	if topK > ceiling {
		_, l := logger_domain.From(ctx, log)
		l.Debug("Clamping vector search TopK to the candidate ceiling",
			logger_domain.String("namespace", request.Namespace),
			logger_domain.Int("requested_top_k", topK),
			logger_domain.Int("ceiling", ceiling))
		topK = ceiling
	}
	if len(request.Filter) == 0 && request.MinScore == nil {
		response, _, err := rankedSearch(ctx, c, request, topK, topK, false)
		return response, err
	}

	window := topK
	for {
		response, candidates, err := rankedSearch(ctx, c, request, topK, window, true)
		if err != nil {
			return nil, err
		}
		if len(response.Results) >= topK || candidates < window || window >= ceiling {
			return response, nil
		}
		window = nextWindow(window, ceiling)
	}
}

// Get retrieves a single document by its ID.
//
// Takes namespace (string) which identifies the collection/index.
// Takes id (string) which is the document ID.
//
// Returns *llm_dto.VectorDocument which is the document, or nil if not found.
// Returns error when the namespace does not exist.
func (s *Store) Get(ctx context.Context, namespace, id string) (*llm_dto.VectorDocument, error) {
	c, err := s.getCache(ctx, namespace)
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
	c, err := s.getCache(ctx, namespace)
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
	c, err := s.getCache(ctx, namespace)
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

// CreateNamespace creates a namespace with the given configuration. Creating a namespace
// that already exists leaves it unchanged.
//
// Calls for the same namespace can overlap. The factory runs outside the store lock.
//
// Takes namespace (string) which is the name of the namespace to create.
// Takes config (*llm_domain.VectorNamespaceConfig) which configures the namespace.
//
// Returns error when the store is closed or the cache cannot be created.
func (s *Store) CreateNamespace(ctx context.Context, namespace string, config *llm_domain.VectorNamespaceConfig) error {
	if _, err := s.ensureCache(ctx, namespace, config); err != nil {
		return fmt.Errorf("creating namespace %q: %w", namespace, err)
	}
	return nil
}

// DeleteNamespace removes a namespace, invalidates all its cached entries, and retires
// the underlying cache instance.
//
// Takes namespace (string) which is the name of the namespace to delete.
//
// Returns error when invalidating the cached entries fails, in which case the namespace
// is kept so the deletion can be retried.
//
// Concurrency: Safe for concurrent use. The entries are invalidated without the store's
// lock, as invalidation can be a network call. Until it returns, callers that would
// create the namespace again wait, so a new cache cannot write data that the invalidation
// then removes.
func (s *Store) DeleteNamespace(ctx context.Context, namespace string) error {
	s.mu.Lock()
	c, exists := s.caches[namespace]
	if !exists {
		s.mu.Unlock()
		return nil
	}
	config := s.configs[namespace]
	delete(s.caches, namespace)
	delete(s.configs, namespace)
	done := make(chan struct{})
	s.deleting[namespace] = done
	s.mu.Unlock()

	invalidateErr := c.InvalidateAll(ctx)

	s.mu.Lock()
	delete(s.deleting, namespace)
	closed := s.closed
	switch {
	case closed:
	case invalidateErr != nil:
		s.caches[namespace] = c
		s.configs[namespace] = config
	default:
		s.retired = append(s.retired, c)
	}
	close(done)
	s.mu.Unlock()

	var closeErr error
	if closed {
		closeErr = c.Close(ctx)
	}
	if err := errors.Join(invalidateErr, closeErr); err != nil {
		return fmt.Errorf("invalidating namespace %q: %w", namespace, err)
	}
	return nil
}

// Close releases all resources held by the vector store.
//
// Returns error which joins every cache's close error.
//
// Concurrency: Safe for concurrent use. Caches are closed after the store's lock is
// released.
func (s *Store) Close(ctx context.Context) error {
	s.mu.Lock()
	caches := make([]cache_domain.Cache[string, llm_dto.VectorDocument], 0, len(s.caches)+len(s.retired))
	for namespace, c := range s.caches {
		caches = append(caches, c)
		delete(s.caches, namespace)
		delete(s.configs, namespace)
	}
	caches = append(caches, s.retired...)
	s.retired = nil
	s.closed = true
	s.mu.Unlock()

	var closeErrors []error
	for _, c := range caches {
		if err := c.Close(ctx); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	if err := errors.Join(closeErrors...); err != nil {
		return fmt.Errorf("closing vector store caches: %w", err)
	}
	return nil
}

// lookupCache returns the cache for the given namespace without auto-creating.
//
// Takes namespace (string) which identifies the namespace to look up.
//
// Returns cache_domain.Cache[string, llm_dto.VectorDocument] which is the cache for the
// namespace, or nil if not found.
//
// Concurrency: Safe for concurrent use. Access is serialised by an internal read lock.
func (s *Store) lookupCache(namespace string) cache_domain.Cache[string, llm_dto.VectorDocument] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.caches[namespace]
}

// getCache returns the namespace's cache, creating one without configuration when the
// namespace does not exist yet.
//
// Calls for the same namespace share the cache creation.
//
// Takes namespace (string) which identifies the namespace.
//
// Returns cache_domain.Cache[string, llm_dto.VectorDocument] which is the cache.
// Returns error when the store is closed or the cache cannot be created.
func (s *Store) getCache(ctx context.Context, namespace string) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
	c, err := s.ensureCache(ctx, namespace, nil)
	if err != nil {
		return nil, fmt.Errorf("auto-creating cache for namespace %q: %w", namespace, err)
	}
	return c, nil
}

// ensureCache returns the namespace's cache, creating it when missing.
//
// The factory runs without the store's lock, as creating a cache can be a network call,
// and runs once per namespace however many callers race to create it. While the namespace
// is being deleted, the caller waits for the deletion to finish first.
//
// Takes namespace (string) which identifies the namespace.
// Takes config (*llm_domain.VectorNamespaceConfig) which configures a new cache, or nil
// for none.
//
// Returns cache_domain.Cache[string, llm_dto.VectorDocument] which is the cache.
// Returns error when the store is closed, ctx ends while waiting for a deletion, or the
// factory fails.
//
// Concurrency: Safe for concurrent use.
func (s *Store) ensureCache(ctx context.Context, namespace string, config *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
	for {
		s.mu.RLock()
		existing, exists := s.caches[namespace]
		closed := s.closed
		deleting := s.deleting[namespace]
		s.mu.RUnlock()
		switch {
		case exists:
			return existing, nil
		case closed:
			return nil, errStoreClosed
		case deleting != nil:
			select {
			case <-deleting:
				continue
			case <-ctx.Done():
				return nil, fmt.Errorf("waiting for namespace %q to finish deleting: %w", namespace, context.Cause(ctx))
			}
		}

		created, err := s.createCacheOnce(ctx, namespace, config)
		if errors.Is(err, errNamespaceDeleting) {
			continue
		}
		return created, err
	}
}

// createCacheOnce creates the namespace's cache, sharing one factory call between callers
// that race to create the same namespace.
//
// Calls share the factory result. Cancellation by the caller that starts the factory can
// make the shared creation fail.
//
// Takes namespace (string) which identifies the namespace.
// Takes config (*llm_domain.VectorNamespaceConfig) which configures the new cache.
//
// Returns cache_domain.Cache[string, llm_dto.VectorDocument] which is the cache.
// Returns error when the store is closed, the namespace began deleting, or the factory
// fails.
func (s *Store) createCacheOnce(ctx context.Context, namespace string, config *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
	value, err, _ := s.creating.Do(namespace, func() (any, error) {
		return s.createCache(ctx, namespace, config)
	})
	if err != nil {
		return nil, err
	}
	created, ok := value.(cache_domain.Cache[string, llm_dto.VectorDocument])
	if !ok {
		return nil, fmt.Errorf("%w: %T", errUnexpectedCacheType, value)
	}
	return created, nil
}

// createCache runs the factory for a namespace that has no cache and stores the result.
//
// Takes namespace (string) which identifies the namespace.
// Takes config (*llm_domain.VectorNamespaceConfig) which configures the new cache.
//
// Returns cache_domain.Cache[string, llm_dto.VectorDocument] which is the stored cache,
// or the existing one when an earlier creation already stored it.
// Returns error when the store is closed, the namespace began deleting, ctx is done, or
// the factory fails.
//
// Concurrency: Must run inside the namespace's single-flight call, so no other creation
// of the namespace runs at the same time.
func (s *Store) createCache(ctx context.Context, namespace string, config *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
	s.mu.RLock()
	existing, exists := s.caches[namespace]
	closed := s.closed
	_, deleting := s.deleting[namespace]
	s.mu.RUnlock()
	switch {
	case exists:
		return existing, nil
	case closed:
		return nil, errStoreClosed
	case deleting:
		return nil, errNamespaceDeleting
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("creating cache: %w", context.Cause(ctx))
	}

	created, err := s.factory(ctx, namespace, config)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	closed = s.closed
	if !closed {
		s.caches[namespace] = created
		s.configs[namespace] = config
	}
	s.mu.Unlock()

	if closed {
		if closeErr := created.Close(ctx); closeErr != nil {
			return nil, errors.Join(errStoreClosed, closeErr)
		}
		return nil, errStoreClosed
	}
	return created, nil
}

// candidateCeiling returns the configured candidate ceiling, falling back to the default
// when the store was built without New.
//
// Returns int which is the positive candidate ceiling.
func (s *Store) candidateCeiling() int {
	if s.maxCandidates <= 0 {
		return DefaultMaxFilteredCandidates
	}
	return s.maxCandidates
}

// lookupConfig returns the configuration a namespace was created with.
//
// Takes namespace (string) which is the namespace to look up.
//
// Returns *llm_domain.VectorNamespaceConfig which is the configuration, or nil when the
// namespace was created without one.
//
// Concurrency: Safe for concurrent use.
func (s *Store) lookupConfig(namespace string) *llm_domain.VectorNamespaceConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.configs[namespace]
}

// WithMaxFilteredCandidates sets the ceiling on the ranked candidates a filtered or
// thresholded search inspects. Values below one are ignored.
//
// Takes limit (int) which is the candidate ceiling.
//
// Returns Option which applies the ceiling.
func WithMaxFilteredCandidates(limit int) Option {
	return func(s *Store) {
		if limit > 0 {
			s.maxCandidates = limit
		}
	}
}

// New creates a new cache-backed vector store.
//
// Takes factory (CacheFactory) which creates cache instances per namespace.
// Takes opts (...Option) which adjust search limits.
//
// Returns *Store ready for use.
func New(factory CacheFactory, opts ...Option) *Store {
	s := &Store{
		caches:        make(map[string]cache_domain.Cache[string, llm_dto.VectorDocument]),
		configs:       make(map[string]*llm_domain.VectorNamespaceConfig),
		factory:       factory,
		deleting:      make(map[string]chan struct{}),
		retired:       nil,
		creating:      singleflight.Group{},
		mu:            sync.RWMutex{},
		maxCandidates: DefaultMaxFilteredCandidates,
		closed:        false,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// validateVector checks that vector can be stored in a namespace with config, so a
// document is refused at the edge rather than cached but left out of vector search.
//
// Takes vector ([]float32) which is the document's vector.
// Takes config (*llm_domain.VectorNamespaceConfig) which is the namespace's
// configuration. When nil, only checks that need no configuration run.
//
// Returns error which wraps llm_domain.ErrInvalidVector when the vector is empty, holds a
// NaN or infinite value, has a magnitude too large to represent, its dimension differs
// from the namespace's, or it has zero magnitude in a namespace that uses cosine
// similarity, which is also the default.
func validateVector(vector []float32, config *llm_domain.VectorNamespaceConfig) error {
	if len(vector) == 0 {
		return fmt.Errorf("%w: vector is empty", llm_domain.ErrInvalidVector)
	}
	for index, value := range vector {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return fmt.Errorf("%w: component %d is %v", llm_domain.ErrInvalidVector, index, value)
		}
	}
	norm := vectormaths.NormF32(vector)
	if math.IsNaN(float64(norm)) || math.IsInf(float64(norm), 0) {
		return fmt.Errorf("%w: vector magnitude overflows", llm_domain.ErrInvalidVector)
	}
	if config == nil {
		return nil
	}
	if config.Dimension > 0 && len(vector) != config.Dimension {
		return fmt.Errorf("%w: vector has dimension %d, want %d", llm_domain.ErrInvalidVector, len(vector), config.Dimension)
	}
	usesCosine := config.Metric == "" || config.Metric == llm_dto.SimilarityCosine
	if usesCosine && norm == 0 {
		return fmt.Errorf("%w: vector has zero magnitude, which cosine similarity cannot compare", llm_domain.ErrInvalidVector)
	}
	return nil
}

// nextWindow doubles the candidate window without passing the ceiling or overflowing.
//
// Takes window (int) which is the current candidate window.
// Takes ceiling (int) which is the largest window allowed.
//
// Returns int which is twice window, or ceiling when doubling would pass it.
func nextWindow(window, ceiling int) int {
	if window > ceiling/2 {
		return ceiling
	}
	return window * 2
}

// matchesFilter checks whether a document's metadata satisfies all filter criteria.
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

// rankedSearch runs one ranked provider query over a candidate window and converts the
// hits that pass the request's metadata filter.
//
// Takes c (cache_domain.Cache[string, llm_dto.VectorDocument]) which stores the
// namespace.
// Takes request (*llm_dto.VectorSearchRequest) which holds the query and filter.
// Takes topK (int) which caps the returned results.
// Takes window (int) which is the provider candidate count.
// Takes countMatches (bool) which reports the number of filtered matches in the window as
// the total, rather than the provider's total.
//
// Returns *llm_dto.VectorSearchResponse which holds the converted matches.
// Returns int which is the number of ranked candidates the provider reported, counted
// before entries that expired or failed to decode were dropped.
// Returns error when the provider search fails or ctx is cancelled.
func rankedSearch(
	ctx context.Context,
	c cache_domain.Cache[string, llm_dto.VectorDocument],
	request *llm_dto.VectorSearchRequest,
	topK, window int,
	countMatches bool,
) (*llm_dto.VectorSearchResponse, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, fmt.Errorf("vector search cancelled: %w", context.Cause(ctx))
	}
	searchResult, err := c.Search(ctx, request.TextQuery, &cache_dto.SearchOptions{
		Vector:      request.Vector,
		TopK:        window,
		MinScore:    request.MinScore,
		Limit:       window,
		SortBy:      "",
		VectorField: "",
		Filters:     nil,
		Fields:      nil,
		Offset:      0,
		SortOrder:   0,
		Highlight:   false,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("cache search failed: %w", err)
	}

	response := &llm_dto.VectorSearchResponse{
		Results:    make([]llm_dto.VectorSearchResult, 0, min(topK, len(searchResult.Items))),
		TotalCount: 0,
	}
	if !countMatches {
		response.TotalCount = safeconv.Int64ToInt(searchResult.Total)
	}
	for _, hit := range searchResult.Items {
		if !matchesFilter(hit.Value, request.Filter) {
			continue
		}
		if countMatches {
			response.TotalCount++
		}
		if len(response.Results) == topK {
			continue
		}
		response.Results = append(response.Results, toSearchResult(hit.Value, hit.Score, request))
	}

	return response, searchResult.Candidates, nil
}

// toSearchResult converts a ranked document into a search result, copying the vector and
// metadata only when the request asks for them.
//
// Takes document (llm_dto.VectorDocument) which is the ranked document.
// Takes score (float64) which is the provider's similarity score.
// Takes request (*llm_dto.VectorSearchRequest) which selects the optional fields.
//
// Returns llm_dto.VectorSearchResult which is the converted result.
func toSearchResult(document llm_dto.VectorDocument, score float64, request *llm_dto.VectorSearchRequest) llm_dto.VectorSearchResult {
	result := llm_dto.VectorSearchResult{
		ID:       document.ID,
		Content:  document.Content,
		Score:    float32(score),
		Metadata: nil,
		Vector:   nil,
	}
	if request.IncludeVectors {
		result.Vector = document.Vector
	}
	if request.IncludeMetadata {
		result.Metadata = document.Metadata
	}
	return result
}
