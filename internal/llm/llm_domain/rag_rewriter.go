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
	"cmp"
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/llm/llm_dto"
)

const (
	// defaultRewriterMaxQueries is the default maximum number of queries to generate.
	defaultRewriterMaxQueries = 1

	// defaultRewriterMaxTokens is the default token limit for rewritten queries.
	defaultRewriterMaxTokens = 200

	// defaultSingleRewritePrompt is the default prompt for generating a single
	// optimised search query as JSON. Override with [WithRewriterPrompt] to
	// provide domain-specific examples.
	defaultSingleRewritePrompt = `Task: rewrite the user's search query into a single precise search query optimised for vector similarity search.

Rules:
- Output ONLY valid JSON in this exact format: {"queries": ["your rewritten query"]}
- Do NOT answer the question
- Do NOT add explanations, greetings, or commentary
- Use specific technical terms that documentation would contain

Examples:
User: how do I set up the thing
Output: {"queries": ["installation setup configuration guide"]}
User: what's the cache thing
Output: {"queries": ["cache system configuration and usage"]}
User: it keeps crashing when I deploy
Output: {"queries": ["deployment error troubleshooting"]}

Now rewrite the user's query below.`

	// defaultMultiQueryPromptTemplate instructs the model to produce multiple
	// diverse queries as JSON, with the %d verb replaced by the desired count.
	//
	// Uses a structured format with examples to prevent weak models from
	// generating conversational responses. Override with [WithRewriterPrompt] for
	// domain-specific examples.
	defaultMultiQueryPromptTemplate = `Task: rewrite the user's search query into exactly %d different search queries optimised for vector similarity search. 
Each query should capture a different aspect of what the user is looking for.

Rules:
- Output ONLY valid JSON in this exact format: {"queries": ["query1", "query2", "query3"]}
- Do NOT answer the question
- Do NOT add explanations, greetings, numbering, or commentary
- Use specific technical terms that documentation would contain

Example input: how do I set up the thing with auth
Example output: {"queries": ["installation setup configuration guide", "authentication login security setup", "getting started tutorial"]}

Now generate %d queries for the user's query below.`
)

// QueryRewriterFunc rewrites a user query into one or more search queries
// before embedding. Returning a single element performs simple rewriting;
// returning multiple elements triggers multi-query expansion with result
// deduplication.
//
// A nil or empty slice means "use the original query unchanged". An error
// causes graceful degradation to the original query.
type QueryRewriterFunc func(ctx context.Context, query string) ([]string, error)

// QueryRewriterOption is a functional option for configuring the built-in
// LLM-based query rewriter created by [LLMQueryRewriter].
type QueryRewriterOption func(*queryRewriterConfig)

// queryRewriterConfig holds settings for LLM-based query rewriting.
type queryRewriterConfig struct {
	// model is the LLM model to use for rewriting. Empty uses the provider
	// default.
	model string

	// provider is the LLM provider name. Empty uses the service default.
	provider string

	// prompt is the system prompt. Empty uses the built-in default.
	prompt string

	// maxQueries is the maximum number of expanded queries to generate, defaulting
	// to 1 (simple rewrite) with larger values enabling multi-query expansion.
	maxQueries int

	// maxTokens limits the rewriter completion output. Defaults to 200.
	maxTokens int
}

// numberPrefixRe matches common numbering patterns LLMs prepend to list items.
var numberPrefixRe = regexp.MustCompile(`^\d+[\.\)]\s*`)

// rewriterJSONResponse is the expected JSON structure from the rewriter.
type rewriterJSONResponse struct {
	// Queries contains the rewritten search queries.
	Queries []string `json:"queries"`
}

// WithRewriterModel sets the LLM model for the built-in query rewriter.
//
// Takes model (string) which identifies the model to use.
//
// Returns QueryRewriterOption which applies the model setting.
func WithRewriterModel(model string) QueryRewriterOption {
	return func(c *queryRewriterConfig) { c.model = model }
}

// WithRewriterProvider sets the LLM provider for the built-in query rewriter.
//
// Takes provider (string) which identifies the provider to use.
//
// Returns QueryRewriterOption which applies the provider setting.
func WithRewriterProvider(provider string) QueryRewriterOption {
	return func(c *queryRewriterConfig) { c.provider = provider }
}

// WithRewriterPrompt sets a custom system prompt for the built-in query
// rewriter. When set, this overrides the default single-rewrite or
// multi-query expansion prompt.
//
// Takes prompt (string) which is the system prompt text.
//
// Returns QueryRewriterOption which applies the prompt.
func WithRewriterPrompt(prompt string) QueryRewriterOption {
	return func(c *queryRewriterConfig) { c.prompt = prompt }
}

// WithRewriterMaxQueries sets the maximum number of expanded queries
// the rewriter should generate, where 1 (default) produces a single
// rewritten query and values greater than 1 enable multi-query
// expansion with independent embedding, searching, and result merging.
//
// Takes n (int) which is the maximum query count.
//
// Returns QueryRewriterOption which applies the limit.
func WithRewriterMaxQueries(n int) QueryRewriterOption {
	return func(c *queryRewriterConfig) { c.maxQueries = n }
}

// WithRewriterMaxTokens sets the maximum number of tokens for the rewriter
// completion output. Defaults to 200.
//
// Takes n (int) which is the token limit.
//
// Returns QueryRewriterOption which applies the limit.
func WithRewriterMaxTokens(n int) QueryRewriterOption {
	return func(c *queryRewriterConfig) { c.maxTokens = n }
}

// LLMQueryRewriter creates a [QueryRewriterFunc] that uses the LLM service to
// rewrite or expand a query for improved vector search retrieval. The rewriter
// makes a completion call with a system prompt instructing the model to
// reformulate the query.
//
// When maxQueries is 1 (default), the model returns a single improved query.
// When maxQueries > 1, the model returns multiple diverse query expansions.
//
// Takes service (Service) which provides the LLM completion capability.
// Takes opts (...QueryRewriterOption) which configure the rewriter behaviour.
//
// Returns QueryRewriterFunc which rewrites queries using the LLM.
func LLMQueryRewriter(service Service, opts ...QueryRewriterOption) QueryRewriterFunc {
	config := queryRewriterConfig{
		maxQueries: defaultRewriterMaxQueries,
		maxTokens:  defaultRewriterMaxTokens,
	}
	for _, opt := range opts {
		opt(&config)
	}

	prompt := config.prompt
	if prompt == "" {
		if config.maxQueries <= 1 {
			prompt = defaultSingleRewritePrompt
		} else {
			prompt = fmt.Sprintf(defaultMultiQueryPromptTemplate, config.maxQueries, config.maxQueries)
		}
	}

	maxQueries := max(config.maxQueries, 1)

	return func(ctx context.Context, query string) ([]string, error) {
		builder := service.NewCompletion().
			System(prompt).
			User(query).
			MaxTokens(config.maxTokens).
			JSONResponse()

		if config.model != "" {
			builder = builder.Model(config.model)
		}
		if config.provider != "" {
			builder = builder.Provider(config.provider)
		}

		response, err := builder.Do(ctx)
		if err != nil {
			return nil, fmt.Errorf("query rewriter completion: %w", err)
		}

		content := response.Content()
		if content == "" {
			return nil, nil
		}

		queries := parseJSONRewriterResponse(content, maxQueries)
		if len(queries) == 0 {
			queries = parseRewriterResponse(content, maxQueries)
		}
		return queries, nil
	}
}

// parseJSONRewriterResponse attempts to parse LLM output as JSON containing a
// "queries" array.
//
// Takes content (string) which is the raw LLM output to parse.
// Takes maxQueries (int) which limits the number of queries returned.
//
// Returns []string which contains the parsed queries, or nil if parsing fails
// to signal the caller to fall back to text-based parsing.
func parseJSONRewriterResponse(content string, maxQueries int) []string {
	var response rewriterJSONResponse
	if err := json.Unmarshal([]byte(content), &response); err != nil {
		return nil
	}

	queries := make([]string, 0, len(response.Queries))
	for _, q := range response.Queries {
		trimmed := strings.TrimSpace(q)
		if trimmed != "" {
			queries = append(queries, trimmed)
		}
	}

	if len(queries) > maxQueries {
		queries = queries[:maxQueries]
	}

	return queries
}

// parseRewriterResponse splits LLM output into individual queries.
//
// It strips numbering prefixes, empty lines, and conversational filler that
// small models sometimes emit despite the prompt.
//
// Takes content (string) which is the raw LLM response to parse.
// Takes maxQueries (int) which limits the maximum number of queries returned.
//
// Returns []string which contains the cleaned queries, capped at maxQueries.
func parseRewriterResponse(content string, maxQueries int) []string {
	lines := strings.Split(content, "\n")
	queries := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		trimmed = numberPrefixRe.ReplaceAllString(trimmed, "")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed == "" {
			continue
		}
		if isRewriterJunk(trimmed) {
			continue
		}
		queries = append(queries, trimmed)
	}

	if len(queries) > maxQueries {
		queries = queries[:maxQueries]
	}

	return queries
}

// isRewriterJunk reports whether a line from the rewriter output is
// conversational filler or a formatting artefact rather than a real search
// query.
//
// Takes line (string) which is a single line from the rewriter output.
//
// Returns bool which is true when the line is junk that should be filtered.
//
// Small models (e.g. tinyllama) frequently emit these despite explicit
// prompt instructions.
func isRewriterJunk(line string) bool {
	lower := strings.ToLower(line)

	if strings.HasPrefix(lower, "```") || strings.HasPrefix(lower, "~~~") {
		return true
	}

	for _, prefix := range []string{
		"sure", "here's", "here is", "of course", "certainly",
		"i'd", "i would", "let me", "the following", "below",
		"note:", "note that", "example:", "output:",
	} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}

	return false
}

// mergeSearchResults deduplicates vector search results from multiple query
// expansions, keeping the highest similarity score for each document ID.
//
// Takes resultSets ([][]llm_dto.VectorSearchResult) which contains the search
// results from multiple query expansions to merge.
// Takes topK (int) which specifies the maximum number of results to return.
//
// Returns []llm_dto.VectorSearchResult which contains the merged results
// sorted by score descending and truncated to topK.
func mergeSearchResults(resultSets [][]llm_dto.VectorSearchResult, topK int) []llm_dto.VectorSearchResult {
	seen := make(map[string]llm_dto.VectorSearchResult)

	for _, results := range resultSets {
		for _, r := range results {
			if existing, ok := seen[r.ID]; !ok || r.Score > existing.Score {
				seen[r.ID] = r
			}
		}
	}

	merged := make([]llm_dto.VectorSearchResult, 0, len(seen))
	for _, r := range seen {
		merged = append(merged, r)
	}

	slices.SortFunc(merged, func(a, b llm_dto.VectorSearchResult) int {
		return cmp.Compare(b.Score, a.Score)
	})

	if len(merged) > topK {
		merged = merged[:topK]
	}

	return merged
}
