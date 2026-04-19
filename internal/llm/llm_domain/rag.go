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
	"piko.sh/piko/internal/logger/logger_domain"
)

// ragConfig holds the configuration for automatic RAG context injection.
// It is stored on the builder and resolved lazily during Do().
type ragConfig struct {
	// filter specifies metadata criteria that documents must match.
	filter map[string]any

	// minScore filters out results below this similarity threshold.
	minScore *float32

	// rewriter transforms the query before embedding; nil uses the query as-is.
	// Multiple returned queries trigger multi-query expansion with deduplication.
	rewriter QueryRewriterFunc

	// query is the explicit query text for embedding. If empty, the last user
	// message content is used.
	query string

	// embeddingProvider is the name of the embedding provider to use. If empty,
	// the default embedding provider is used.
	embeddingProvider string

	// embeddingModel is the model to use for embedding. If empty, the default
	// model is used.
	embeddingModel string

	// namespace is the vector store namespace to search.
	namespace string

	// topK is the maximum number of results to return.
	topK int

	// enableHybridSearch enables combined vector + text search. When true, the
	// query text is passed to the vector store alongside the embedding vector,
	// enabling Reciprocal Rank Fusion of semantic and lexical results.
	enableHybridSearch bool
}

// RAGOption is a functional option for configuring automatic RAG behaviour.
type RAGOption func(*ragConfig)

// RAG enables automatic retrieval-augmented generation for this request.
// During Do(), the builder will embed the query text, search the vector store
// in the given namespace, and inject the results as context.
//
// Takes namespace (string) which is the vector store namespace to search.
// Takes topK (int) which is the maximum number of documents to retrieve.
// Takes opts (...RAGOption) which are optional functional options for advanced
// configuration such as [WithRAGQueryRewriter], [WithRAGMinScore], or
// [WithRAGFilter].
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) RAG(namespace string, topK int, opts ...RAGOption) *CompletionBuilder {
	b.ragConfig = &ragConfig{
		namespace: namespace,
		topK:      topK,
	}
	for _, opt := range opts {
		opt(b.ragConfig)
	}
	return b
}

// resolveRAGContext performs the embed-and-search workflow when RAG is
// configured. It stores the results on b.vectorContext so that the existing
// injectVectorContext method formats and prepends them.
//
// When a query rewriter is configured, the original query is rewritten (or
// expanded into multiple queries) before embedding. Multi-query results are
// deduplicated by document ID, keeping the highest score.
//
// Degrades gracefully: if any step fails (no query text, no embedding service, no
// vector store, embedding error, rewriter error, no results), a debug log is emitted
// and the completion proceeds without RAG context.
func (b *CompletionBuilder) resolveRAGContext(ctx context.Context) {
	if b.ragConfig == nil || b.ragResolved {
		return
	}
	b.ragResolved = true

	ctx, l := logger_domain.From(ctx, log)

	query := b.ragConfig.query
	if query == "" {
		query = b.lastUserMessageContent()
	}
	if query == "" {
		l.Debug("RAG skipped: no query text available")
		return
	}
	b.originalQuery = query

	queries := b.resolveRAGQueries(ctx, query)

	embResp, ok := b.embedRAGQueries(ctx, queries)
	if !ok {
		return
	}

	if len(queries) == 1 {
		b.resolveSingleQueryRAG(ctx, queries[0], embResp)
		return
	}

	b.resolveMultiQueryRAG(ctx, queries, embResp)
}

// resolveRAGQueries applies the query rewriter if configured and returns the
// list of queries to embed.
//
// Takes ctx (context.Context) which carries the logger context.
// Takes query (string) which is the original query text.
//
// Returns []string which contains the queries to embed.
func (b *CompletionBuilder) resolveRAGQueries(ctx context.Context, query string) []string {
	queries := []string{query}
	if b.ragConfig.rewriter == nil {
		return queries
	}

	_, l := logger_domain.From(ctx, log)
	rewritten, err := b.ragConfig.rewriter(ctx, query)
	if err != nil {
		l.Debug("RAG query rewriter failed, using original query",
			logger_domain.Error(err),
		)
		return queries
	}
	if len(rewritten) > 0 {
		b.rewrittenQueries = rewritten
		return rewritten
	}
	return queries
}

// embedRAGQueries embeds the queries and validates the prerequisites. Returns
// false when RAG should be skipped.
//
// Takes ctx (context.Context) which carries the logger context.
// Takes queries ([]string) which contains the queries to embed.
//
// Returns *llm_dto.EmbeddingResponse which contains the embedding results.
// Returns bool which is true when embedding succeeded and RAG should continue.
func (b *CompletionBuilder) embedRAGQueries(ctx context.Context, queries []string) (*llm_dto.EmbeddingResponse, bool) {
	_, l := logger_domain.From(ctx, log)

	if b.service.embeddingService == nil {
		l.Debug("RAG skipped: no embedding service configured")
		return nil, false
	}
	if b.service.vectorStore == nil {
		l.Debug("RAG skipped: no vector store configured")
		return nil, false
	}

	embReq := &llm_dto.EmbeddingRequest{
		Input: queries,
		Model: b.ragConfig.embeddingModel,
	}
	embResp, err := b.service.embeddingService.Embed(ctx, b.ragConfig.embeddingProvider, embReq)
	if err != nil {
		l.Debug("RAG skipped: embedding failed", logger_domain.Error(err))
		return nil, false
	}
	if len(embResp.Embeddings) == 0 {
		l.Debug("RAG skipped: empty embedding returned")
		return nil, false
	}
	return embResp, true
}

// resolveSingleQueryRAG performs vector search for a single query and stores
// results on the builder.
//
// Takes ctx (context.Context) which carries the logger context.
// Takes query (string) which is the query text.
// Takes embResp (*llm_dto.EmbeddingResponse) which contains the embedding.
func (b *CompletionBuilder) resolveSingleQueryRAG(ctx context.Context, query string, embResp *llm_dto.EmbeddingResponse) {
	_, l := logger_domain.From(ctx, log)

	if len(embResp.Embeddings[0].Vector) == 0 {
		l.Debug("RAG skipped: empty embedding vector")
		return
	}

	searchReq := &llm_dto.VectorSearchRequest{
		Namespace:       b.ragConfig.namespace,
		Vector:          embResp.Embeddings[0].Vector,
		TopK:            b.ragConfig.topK,
		MinScore:        b.ragConfig.minScore,
		Filter:          b.ragConfig.filter,
		IncludeMetadata: true,
	}
	if b.ragConfig.enableHybridSearch {
		searchReq.TextQuery = query
	}

	searchResp, err := b.service.vectorStore.Search(ctx, searchReq)
	if err != nil {
		l.Debug("RAG skipped: vector search failed", logger_domain.Error(err))
		return
	}
	if !searchResp.HasResults() {
		l.Debug("RAG skipped: no results found")
		return
	}
	b.vectorContext = searchResp.Results
}

// resolveMultiQueryRAG performs vector search for each expanded query,
// deduplicates results, and stores them on the builder.
//
// Takes ctx (context.Context) which carries the logger context.
// Takes queries ([]string) which contains the expanded queries.
// Takes embResp (*llm_dto.EmbeddingResponse) which contains the embeddings.
func (b *CompletionBuilder) resolveMultiQueryRAG(ctx context.Context, queries []string, embResp *llm_dto.EmbeddingResponse) {
	_, l := logger_domain.From(ctx, log)

	resultSets := make([][]llm_dto.VectorSearchResult, 0, len(embResp.Embeddings))
	for i, emb := range embResp.Embeddings {
		if len(emb.Vector) == 0 {
			l.Debug("RAG: empty embedding for expanded query, skipping",
				logger_domain.Int("query_index", i),
			)
			continue
		}
		searchReq := &llm_dto.VectorSearchRequest{
			Namespace:       b.ragConfig.namespace,
			Vector:          emb.Vector,
			TopK:            b.ragConfig.topK,
			MinScore:        b.ragConfig.minScore,
			Filter:          b.ragConfig.filter,
			IncludeMetadata: true,
		}
		if b.ragConfig.enableHybridSearch && i < len(queries) {
			searchReq.TextQuery = queries[i]
		}
		searchResp, err := b.service.vectorStore.Search(ctx, searchReq)
		if err != nil {
			l.Debug("RAG: vector search failed for expanded query",
				logger_domain.Int("query_index", i),
				logger_domain.Error(err),
			)
			continue
		}
		if searchResp.HasResults() {
			resultSets = append(resultSets, searchResp.Results)
		}
	}

	if len(resultSets) == 0 {
		l.Debug("RAG skipped: no results from any expanded query")
		return
	}

	b.vectorContext = mergeSearchResults(resultSets, b.ragConfig.topK)
}

// lastUserMessageContent returns the content of the last user message in the
// request.
//
// Returns string which is the message content, or empty if there are no user
// messages.
func (b *CompletionBuilder) lastUserMessageContent() string {
	for i := len(b.request.Messages) - 1; i >= 0; i-- {
		if b.request.Messages[i].Role == llm_dto.RoleUser {
			return b.request.Messages[i].Content
		}
	}
	return ""
}

// WithRAGQuery sets an explicit query string for the embedding lookup. If not
// set, the content of the last user message is used.
//
// Takes query (string) which is the query text to embed.
//
// Returns RAGOption which applies the query to the RAG configuration.
func WithRAGQuery(query string) RAGOption {
	return func(c *ragConfig) {
		c.query = query
	}
}

// WithRAGMinScore sets a minimum similarity score threshold. Results below
// this score are excluded.
//
// Takes score (float32) which is the minimum similarity score.
//
// Returns RAGOption which applies the filter to the RAG configuration.
func WithRAGMinScore(score float32) RAGOption {
	return func(c *ragConfig) {
		c.minScore = &score
	}
}

// WithRAGFilter sets metadata filter criteria for the vector search. Only
// documents matching the filter are returned.
//
// Takes filter (map[string]any) which specifies metadata criteria.
//
// Returns RAGOption which applies the filter to the RAG configuration.
func WithRAGFilter(filter map[string]any) RAGOption {
	return func(c *ragConfig) {
		c.filter = filter
	}
}

// WithRAGEmbeddingProvider sets the embedding provider to use for the RAG
// query. If not set, the default embedding provider is used.
//
// Takes provider (string) which identifies the embedding provider.
//
// Returns RAGOption which applies the provider to the RAG configuration.
func WithRAGEmbeddingProvider(provider string) RAGOption {
	return func(c *ragConfig) {
		c.embeddingProvider = provider
	}
}

// WithRAGEmbeddingModel sets the embedding model to use for the RAG query.
// If not set, the default embedding model is used.
//
// Takes model (string) which identifies the embedding model.
//
// Returns RAGOption which applies the model to the RAG configuration.
func WithRAGEmbeddingModel(model string) RAGOption {
	return func(c *ragConfig) {
		c.embeddingModel = model
	}
}

// WithRAGQueryRewriter sets a query rewriter function for the RAG
// pipeline, passing the original query through the rewriter before
// embedding and, when multiple queries are returned, embedding and
// searching each independently with results deduplicated by document
// ID (highest score wins) and sorted by score.
//
// Takes rewriter (QueryRewriterFunc) which transforms the query.
//
// Returns RAGOption which applies the rewriter to the RAG configuration.
func WithRAGQueryRewriter(rewriter QueryRewriterFunc) RAGOption {
	return func(c *ragConfig) {
		c.rewriter = rewriter
	}
}

// WithRAGHybridSearch enables combined vector and text search for RAG
// retrieval. When enabled, the query text is passed to the vector store
// alongside the embedding vector, allowing the store to combine semantic
// similarity with lexical text matching using Reciprocal Rank Fusion (RRF).
//
// This requires the underlying vector store's cache to have TEXT fields
// configured in its search schema, typically for the document Content field.
// Use [cache_linguistics.NewTextAnalyser] to configure linguistic text
// analysis on the cache schema.
//
// Returns RAGOption which enables hybrid search on the RAG configuration.
func WithRAGHybridSearch() RAGOption {
	return func(c *ragConfig) {
		c.enableHybridSearch = true
	}
}
