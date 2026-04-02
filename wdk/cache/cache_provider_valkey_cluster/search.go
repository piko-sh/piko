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

package cache_provider_valkey_cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/valkey-io/valkey-go"

	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
)

const (
	// DefaultSearchResultLimit is the default number of results returned by search
	// queries when no explicit limit is specified.
	DefaultSearchResultLimit = 10

	// valkeyJSONPath is the root path selector for Valkey JSON commands.
	valkeyJSONPath = "$"

	// valkeyLogKeyField is the attribute key for logging Valkey keys in search
	// operations.
	valkeyLogKeyField = "key"

	// logKeyIndex is the attribute key for logging the index name in search
	// operations.
	logKeyIndex = "index"
)

// indexCreationMu protects concurrent index creation attempts.
var indexCreationMu sync.Mutex

// ensureIndexExists creates the Valkey Search index if it does not already
// exist. This is called lazily on the first search operation.
//
// Returns error when the index cannot be created or search is not supported.
//
// Safe for concurrent use. Uses a mutex to ensure only one goroutine creates
// the index.
func (a *ValkeyClusterAdapter[K, V]) ensureIndexExists(ctx context.Context) error {
	ctx, l := logger.From(ctx, log)

	if a.schema == nil {
		return cache.ErrSearchNotSupported
	}

	if a.indexCreated {
		return nil
	}

	indexCreationMu.Lock()
	defer indexCreationMu.Unlock()

	if a.indexCreated {
		return nil
	}

	response := a.client.Do(ctx, a.client.B().Arbitrary("FT._LIST").Build())
	if indices, err := response.AsStrSlice(); err == nil {
		if slices.Contains(indices, a.indexName) {
			a.indexCreated = true
			l.Internal("Valkey Search index already exists", logger.String(logKeyIndex, a.indexName))
			return nil
		}
	}

	if err := a.createIndex(ctx); err != nil {
		return err
	}

	a.indexCreated = true
	return nil
}

// createIndex creates the Valkey Search index using FT.CREATE.
// TEXT and GEO fields are skipped as Valkey Search does not yet support them.
//
// Returns error when the index creation command fails.
func (a *ValkeyClusterAdapter[K, V]) createIndex(ctx context.Context) error {
	ctx, l := logger.From(ctx, log)

	command := a.client.B().Arbitrary("FT.CREATE").Keys(a.indexName).
		Args("ON", "JSON", "PREFIX", "1", a.namespace, "SCHEMA")

	fieldCount := 0
	for _, field := range a.schema.Fields {
		jsonPath := "$." + field.Name
		alias := field.Name

		switch field.Type {
		case cache.FieldTypeTag:
			command = command.Args(jsonPath, "AS", alias, "TAG")
			if field.Sortable {
				command = command.Args("SORTABLE")
			}
			fieldCount++
		case cache.FieldTypeNumeric:
			command = command.Args(jsonPath, "AS", alias, "NUMERIC")
			if field.Sortable {
				command = command.Args("SORTABLE")
			}
			fieldCount++
		case cache.FieldTypeText:
			l.Internal("Skipping TEXT field in Valkey Search index (not supported)",
				logger.String("field", field.Name),
				logger.String(logKeyIndex, a.indexName))
		case cache.FieldTypeGeo:
			l.Internal("Skipping GEO field in Valkey Search index (not supported)",
				logger.String("field", field.Name),
				logger.String(logKeyIndex, a.indexName))
		}
	}

	if fieldCount == 0 {
		return fmt.Errorf("no supported fields for Valkey Search index %s (TEXT and GEO fields are not supported)", a.indexName)
	}

	if err := a.client.Do(ctx, command.Build()).Error(); err != nil {
		return fmt.Errorf("failed to create Valkey Search index %s: %w", a.indexName, err)
	}

	l.Internal("Created Valkey Search index",
		logger.String(logKeyIndex, a.indexName),
		logger.Int("fields", fieldCount))

	return nil
}

// buildSearchQuery constructs the FT.SEARCH query string from options.
//
// Takes textQuery (string) which is the text search term to include.
// Takes filters ([]cache.Filter) which are the filter clauses to apply.
//
// Returns string which is the complete query, or "*" if no parts are given.
func (a *ValkeyClusterAdapter[K, V]) buildSearchQuery(textQuery string, filters []cache.Filter) string {
	parts := make([]string, 0, len(filters)+1)

	if textQuery != "" {
		parts = append(parts, textQuery)
	}

	for _, f := range filters {
		filterString := a.buildFilterClause(f)
		if filterString != "" {
			parts = append(parts, filterString)
		}
	}

	if len(parts) == 0 {
		return "*"
	}

	return strings.Join(parts, " ")
}

// buildFilterClause converts a Filter to a Valkey Search query clause.
//
// Takes f (cache.Filter) which specifies the filter operation and values.
//
// Returns string which is the Valkey Search query clause, or empty if the
// filter operation is not supported or has insufficient values.
func (*ValkeyClusterAdapter[K, V]) buildFilterClause(f cache.Filter) string {
	switch f.Operation {
	case cache.FilterOpEq:
		return fmt.Sprintf("@%s:{%v}", f.Field, escapeTagValue(f.Value))

	case cache.FilterOpNe:
		return fmt.Sprintf("-@%s:{%v}", f.Field, escapeTagValue(f.Value))

	case cache.FilterOpGt:
		return fmt.Sprintf("@%s:[(%v +inf]", f.Field, f.Value)

	case cache.FilterOpGe:
		return fmt.Sprintf("@%s:[%v +inf]", f.Field, f.Value)

	case cache.FilterOpLt:
		return fmt.Sprintf("@%s:[-inf (%v]", f.Field, f.Value)

	case cache.FilterOpLe:
		return fmt.Sprintf("@%s:[-inf %v]", f.Field, f.Value)

	case cache.FilterOpBetween:
		if len(f.Values) >= 2 {
			return fmt.Sprintf("@%s:[%v %v]", f.Field, f.Values[0], f.Values[1])
		}

	case cache.FilterOpIn:
		if len(f.Values) > 0 {
			escaped := make([]string, len(f.Values))
			for i, v := range f.Values {
				escaped[i] = escapeTagValue(v)
			}
			return fmt.Sprintf("@%s:{%s}", f.Field, strings.Join(escaped, "|"))
		}

	case cache.FilterOpPrefix:
		return fmt.Sprintf("@%s:%v*", f.Field, f.Value)
	}

	return ""
}

// executeSearch runs the FT.SEARCH command and returns raw results.
//
// Takes query (string) which specifies the Valkey Search query to execute.
// Takes opts (*cache.SearchOptions) which provides pagination and sorting
// options.
//
// Returns []valkey.ValkeyMessage which contains the raw result documents.
// Returns int64 which is the total count of matching documents.
// Returns error when the search command fails or the result format is
// unexpected.
func (a *ValkeyClusterAdapter[K, V]) executeSearch(ctx context.Context, query string, opts *cache.SearchOptions) ([]valkey.ValkeyMessage, int64, error) {
	limit := DefaultSearchResultLimit
	offset := 0
	if opts != nil {
		if opts.Limit > 0 {
			limit = opts.Limit
		}
		offset = opts.Offset
	}

	command := a.client.B().Arbitrary("FT.SEARCH").Keys(a.indexName).
		Args(query, "LIMIT", fmt.Sprintf("%d", offset), fmt.Sprintf("%d", limit))

	if opts != nil && opts.SortBy != "" {
		command = command.Args("SORTBY", opts.SortBy)
		if opts.SortOrder == cache.SortDesc {
			command = command.Args("DESC")
		} else {
			command = command.Args("ASC")
		}
	}

	command = command.Args("RETURN", "1", valkeyJSONPath)

	response := a.client.Do(ctx, command.Build())
	results, err := response.ToArray()
	if err != nil {
		return nil, 0, fmt.Errorf("FT.SEARCH failed: %w", err)
	}

	if len(results) == 0 {
		return nil, 0, nil
	}

	total, err := results[0].AsInt64()
	if err != nil {
		return nil, 0, fmt.Errorf("unexpected result format: total count not int64: %w", err)
	}

	return results[1:], total, nil
}

// parseSearchResults converts raw FT.SEARCH results to SearchResult.
//
// Takes rawResults ([]valkey.ValkeyMessage) which contains the raw
// document pairs from the FT.SEARCH response.
// Takes total (int64) which is the total number of matching documents
// reported by Valkey.
// Takes opts (*cache.SearchOptions) which provides the offset and
// limit for pagination metadata.
//
// Returns cache.SearchResult[K, V] which contains the parsed hits
// with pagination info.
// Returns error which is currently always nil but reserved for future
// use.
func (a *ValkeyClusterAdapter[K, V]) parseSearchResults(ctx context.Context, rawResults []valkey.ValkeyMessage, total int64, opts *cache.SearchOptions) (cache.SearchResult[K, V], error) {
	result := cache.SearchResult[K, V]{
		Items: make([]cache.SearchHit[K, V], 0),
		Total: total,
	}

	if opts != nil {
		result.Offset = opts.Offset
		result.Limit = opts.Limit
	}

	for i := 0; i < len(rawResults); i += 2 {
		hit, ok := a.parseSearchHit(ctx, rawResults, i)
		if ok {
			result.Items = append(result.Items, hit)
		}
	}

	return result, nil
}

// parseSearchHit parses a single search hit from the raw results at
// the given index.
//
// Takes rawResults ([]valkey.ValkeyMessage) which contains the full
// array of raw search result messages.
// Takes i (int) which is the index of the key element within
// rawResults; the document data follows at i+1.
//
// Returns cache.SearchHit[K, V] which contains the decoded key and
// unmarshalled value.
// Returns bool which is true when parsing succeeded.
func (a *ValkeyClusterAdapter[K, V]) parseSearchHit(ctx context.Context, rawResults []valkey.ValkeyMessage, i int) (cache.SearchHit[K, V], bool) {
	_, l := logger.From(ctx, log)

	var zero cache.SearchHit[K, V]

	keyString, err := rawResults[i].ToString()
	if err != nil {
		return zero, false
	}

	key, err := a.decodeKey(keyString)
	if err != nil {
		l.Trace("Failed to decode key from search result",
			logger.String(valkeyLogKeyField, keyString),
			logger.Error(err))
		return zero, false
	}

	if i+1 >= len(rawResults) {
		return zero, false
	}

	docData, err := rawResults[i+1].ToArray()
	if err != nil || len(docData) < 2 {
		return zero, false
	}

	jsonString := extractJSONFromDocData(docData)
	if jsonString == "" {
		return zero, false
	}

	var value V
	if err := json.Unmarshal([]byte(jsonString), &value); err != nil {
		l.Trace("Failed to unmarshal search result value",
			logger.String(valkeyLogKeyField, keyString),
			logger.Error(err))
		return zero, false
	}

	return cache.SearchHit[K, V]{Key: key, Value: value}, true
}

// setJSONValue stores a value as JSON for Valkey Search indexing.
//
// Takes keyString (string) which is the Valkey key to store the value under.
// Takes value (V) which is the value to marshal and store as JSON.
// Takes ttl (int) which is the time-to-live in seconds; zero means no expiry.
//
// Returns error when marshalling fails or the JSON.SET command fails.
func (a *ValkeyClusterAdapter[K, V]) setJSONValue(ctx context.Context, keyString string, value V, ttl int) error {
	ctx, l := logger.From(ctx, log)

	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value to JSON: %w", err)
	}

	command := a.client.B().Arbitrary("JSON.SET").Keys(keyString).Args(valkeyJSONPath, string(jsonBytes)).Build()
	if err := a.client.Do(ctx, command).Error(); err != nil {
		return fmt.Errorf("JSON.SET failed: %w", err)
	}

	if ttl > 0 {
		if err := a.client.Do(ctx, a.client.B().Expire().Key(keyString).Seconds(int64(ttl)).Build()).Error(); err != nil {
			l.Warn("Failed to set TTL on JSON document",
				logger.String(valkeyLogKeyField, keyString),
				logger.Error(err))
		}
	}

	return nil
}

// indexDocument stores a document for Valkey Search indexing.
// This is called from Set when a search schema is configured.
//
// Takes keyString (string) which is the Valkey key for the document.
// Takes value (V) which is the value to index.
func (a *ValkeyClusterAdapter[K, V]) indexDocument(ctx context.Context, keyString string, value V) {
	ctx, l := logger.From(ctx, log)

	if a.schema == nil {
		return
	}

	if err := a.ensureIndexExists(ctx); err != nil {
		l.Warn("Failed to ensure search index exists",
			logger.String(valkeyLogKeyField, keyString),
			logger.Error(err))
		return
	}

	ttlSeconds := int(a.ttl.Seconds())
	if err := a.setJSONValue(ctx, keyString, value, ttlSeconds); err != nil {
		l.Warn("Failed to index document",
			logger.String(valkeyLogKeyField, keyString),
			logger.Error(err))
	}
}

// searchWithValkeySearch performs a search using FT.SEARCH.
//
// Takes query (string) which is the text search term; may be empty
// for filter-only queries.
// Takes opts (*cache.SearchOptions) which provides filters,
// pagination, and sorting parameters.
//
// Returns cache.SearchResult[K, V] which contains the matched items
// and total count.
// Returns error when the index cannot be ensured or the search
// command fails.
func (a *ValkeyClusterAdapter[K, V]) searchWithValkeySearch(ctx context.Context, query string, opts *cache.SearchOptions) (cache.SearchResult[K, V], error) {
	if err := a.ensureIndexExists(ctx); err != nil {
		return cache.SearchResult[K, V]{}, err
	}

	var filters []cache.Filter
	if opts != nil {
		filters = opts.Filters
	}
	searchQuery := a.buildSearchQuery(query, filters)

	rawResults, total, err := a.executeSearch(ctx, searchQuery, opts)
	if err != nil {
		return cache.SearchResult[K, V]{}, err
	}

	return a.parseSearchResults(ctx, rawResults, total, opts)
}

// queryWithValkeySearch performs a structured query using FT.SEARCH.
//
// Takes opts (*cache.QueryOptions) which provides filters, pagination,
// and sorting parameters.
//
// Returns cache.SearchResult[K, V] which contains the matched items
// and total count.
// Returns error when the index cannot be ensured or the search
// command fails.
func (a *ValkeyClusterAdapter[K, V]) queryWithValkeySearch(ctx context.Context, opts *cache.QueryOptions) (cache.SearchResult[K, V], error) {
	if err := a.ensureIndexExists(ctx); err != nil {
		return cache.SearchResult[K, V]{}, err
	}

	var filters []cache.Filter
	if opts != nil {
		filters = opts.Filters
	}
	searchQuery := a.buildSearchQuery("", filters)

	searchOpts := &cache.SearchOptions{
		Limit:  DefaultSearchResultLimit,
		Offset: 0,
	}
	if opts != nil {
		searchOpts.Limit = opts.Limit
		searchOpts.Offset = opts.Offset
		searchOpts.SortBy = opts.SortBy
		searchOpts.SortOrder = opts.SortOrder
	}
	if searchOpts.Limit == 0 {
		searchOpts.Limit = DefaultSearchResultLimit
	}

	rawResults, total, err := a.executeSearch(ctx, searchQuery, searchOpts)
	if err != nil {
		return cache.SearchResult[K, V]{}, err
	}

	return a.parseSearchResults(ctx, rawResults, total, searchOpts)
}

// dropIndex removes the Valkey Search index. Called during InvalidateAll if
// search is enabled.
func (a *ValkeyClusterAdapter[K, V]) dropIndex(ctx context.Context) {
	ctx, l := logger.From(ctx, log)

	if a.schema == nil || !a.indexCreated {
		return
	}

	command := a.client.B().Arbitrary("FT.DROPINDEX").Keys(a.indexName).Build()
	if err := a.client.Do(ctx, command).Error(); err != nil {
		if !isUnknownIndexError(err) {
			l.Warn("Failed to drop search index",
				logger.String(logKeyIndex, a.indexName),
				logger.Error(err))
		}
	}

	a.indexCreated = false
}

// needsJSONStorage reports whether search is enabled and values should be
// stored as JSON.
//
// Returns bool which is true when a schema is configured.
func (a *ValkeyClusterAdapter[K, V]) needsJSONStorage() bool {
	return a.schema != nil
}

// getJSONValue retrieves a JSON-stored value from Valkey.
//
// Takes keyString (string) which specifies the Valkey key to retrieve.
//
// Returns V which is the deserialised value from the JSON document.
// Returns bool which indicates whether the value was found and valid.
func (a *ValkeyClusterAdapter[K, V]) getJSONValue(ctx context.Context, keyString string) (V, bool) {
	var zero V

	command := a.client.B().Arbitrary("JSON.GET").Keys(keyString).Args(valkeyJSONPath).Build()
	result, err := a.client.Do(ctx, command).ToString()
	if err != nil {
		return zero, false
	}

	var values []V
	if err := json.Unmarshal([]byte(result), &values); err != nil {
		return zero, false
	}

	if len(values) == 0 {
		return zero, false
	}

	return values[0], true
}

// isUnknownIndexError reports whether the error indicates
// an unknown Valkey search index.
//
// Takes err (error) which is the error to check for an unknown index message.
//
// Returns bool which is true when the error message contains "Unknown".
func isUnknownIndexError(err error) bool {
	return strings.Contains(err.Error(), "Unknown")
}

// escapeTagValue escapes special characters in tag values for Valkey Search.
//
// Takes v (any) which is the value to escape.
//
// Returns string which is the escaped value with backslash-prefixed special
// characters.
func escapeTagValue(v any) string {
	s := fmt.Sprintf("%v", v)
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, ".", "\\.")
	s = strings.ReplaceAll(s, "<", "\\<")
	s = strings.ReplaceAll(s, ">", "\\>")
	s = strings.ReplaceAll(s, "{", "\\{")
	s = strings.ReplaceAll(s, "}", "\\}")
	s = strings.ReplaceAll(s, "[", "\\[")
	s = strings.ReplaceAll(s, "]", "\\]")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, ":", "\\:")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, "!", "\\!")
	s = strings.ReplaceAll(s, "@", "\\@")
	s = strings.ReplaceAll(s, "#", "\\#")
	s = strings.ReplaceAll(s, "$", "\\$")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "^", "\\^")
	s = strings.ReplaceAll(s, "&", "\\&")
	s = strings.ReplaceAll(s, "*", "\\*")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	s = strings.ReplaceAll(s, "-", "\\-")
	s = strings.ReplaceAll(s, "+", "\\+")
	s = strings.ReplaceAll(s, "=", "\\=")
	s = strings.ReplaceAll(s, "~", "\\~")
	s = strings.ReplaceAll(s, " ", "\\ ")
	return s
}

// extractJSONFromDocData extracts the JSON string from document field data.
//
// Takes docData ([]valkey.ValkeyMessage) which contains field name and value
// pairs.
//
// Returns string which is the JSON value if found, or an empty string if not.
func extractJSONFromDocData(docData []valkey.ValkeyMessage) string {
	for j := 0; j < len(docData); j += 2 {
		fieldName, err := docData[j].ToString()
		if err != nil || fieldName != valkeyJSONPath {
			continue
		}
		if j+1 < len(docData) {
			if value, err := docData[j+1].ToString(); err == nil {
				return value
			}
		}
	}
	return ""
}
