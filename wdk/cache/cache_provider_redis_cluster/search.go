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

package cache_provider_redis_cluster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
)

const (
	// DefaultSearchResultLimit is the default number of results returned by search
	// queries when no explicit limit is specified.
	DefaultSearchResultLimit = 10

	// redisJSONPath is the root path selector for Redis JSON commands.
	redisJSONPath = "$"

	// redisLogKeyField is the attribute key for logging Redis keys in search
	// operations.
	redisLogKeyField = "key"
)

// indexCreationMu protects concurrent index creation attempts.
var indexCreationMu sync.Mutex

// ensureIndexExists creates the RediSearch index if it does not already exist.
// This is called lazily on the first search operation.
//
// Returns error when the index cannot be created or search is not supported.
//
// Safe for concurrent use. Uses a mutex to ensure only one goroutine creates
// the index.
func (a *RedisClusterAdapter[K, V]) ensureIndexExists(ctx context.Context) error {
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

	_, err := a.client.Do(ctx, "FT.INFO", a.indexName).Result()
	if err == nil {
		a.indexCreated = true
		l.Internal("RediSearch index already exists", logger.String("index", a.indexName))
		return nil
	}

	if err := a.createIndex(ctx); err != nil {
		return err
	}

	a.indexCreated = true
	return nil
}

// createIndex creates the RediSearch index using FT.CREATE.
//
// Returns error when the index creation command fails.
func (a *RedisClusterAdapter[K, V]) createIndex(ctx context.Context) error {
	ctx, l := logger.From(ctx, log)

	arguments := []any{
		"FT.CREATE", a.indexName,
		"ON", "JSON",
		"PREFIX", "1", a.namespace,
		"SCHEMA",
	}

	for _, field := range a.schema.Fields {
		jsonPath := "$." + field.Name
		alias := field.Name

		arguments = append(arguments, jsonPath, "AS", alias)

		switch field.Type {
		case cache.FieldTypeText:
			arguments = append(arguments, "TEXT")
			if field.Weight != 0 && field.Weight != 1.0 {
				arguments = append(arguments, "WEIGHT", field.Weight)
			}
		case cache.FieldTypeTag:
			arguments = append(arguments, "TAG")
		case cache.FieldTypeNumeric:
			arguments = append(arguments, "NUMERIC")
		case cache.FieldTypeGeo:
			arguments = append(arguments, "GEO")
		}

		if field.Sortable {
			arguments = append(arguments, "SORTABLE")
		}
	}

	if err := a.client.Do(ctx, arguments...).Err(); err != nil {
		return fmt.Errorf("failed to create RediSearch index %s: %w", a.indexName, err)
	}

	l.Internal("Created RediSearch index",
		logger.String("index", a.indexName),
		logger.Int("fields", len(a.schema.Fields)))

	return nil
}

// buildSearchQuery constructs the FT.SEARCH query string from options.
//
// Takes textQuery (string) which is the text search term to include.
// Takes filters ([]cache.Filter) which are the filter clauses to apply.
//
// Returns string which is the complete query, or "*" if no parts are given.
func (a *RedisClusterAdapter[K, V]) buildSearchQuery(textQuery string, filters []cache.Filter) string {
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

// buildFilterClause converts a Filter to a RediSearch query clause.
//
// Takes f (cache.Filter) which specifies the filter operation and values.
//
// Returns string which is the RediSearch query clause, or empty if the filter
// operation is not supported or has insufficient values.
func (*RedisClusterAdapter[K, V]) buildFilterClause(f cache.Filter) string {
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
// Takes query (string) which specifies the RediSearch query to execute.
// Takes opts (*cache.SearchOptions) which provides pagination and sorting
// options.
//
// Returns []any which contains the raw result documents from the search.
// Returns int64 which is the total count of matching documents.
// Returns error when the search command fails or the result format is
// unexpected.
func (a *RedisClusterAdapter[K, V]) executeSearch(ctx context.Context, query string, opts *cache.SearchOptions) ([]any, int64, error) {
	arguments := []any{"FT.SEARCH", a.indexName, query}

	limit := DefaultSearchResultLimit
	offset := 0
	if opts != nil {
		if opts.Limit > 0 {
			limit = opts.Limit
		}
		offset = opts.Offset
	}
	arguments = append(arguments, "LIMIT", offset, limit)

	if opts != nil && opts.SortBy != "" {
		arguments = append(arguments, "SORTBY", opts.SortBy)
		if opts.SortOrder == cache.SortDesc {
			arguments = append(arguments, "DESC")
		} else {
			arguments = append(arguments, "ASC")
		}
	}

	arguments = append(arguments, "RETURN", "1", redisJSONPath)

	result, err := a.client.Do(ctx, arguments...).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("FT.SEARCH failed: %w", err)
	}

	results, ok := result.([]any)
	if !ok || len(results) == 0 {
		return nil, 0, nil
	}

	total, ok := results[0].(int64)
	if !ok {
		return nil, 0, errors.New("unexpected result format: total count not int64")
	}

	return results[1:], total, nil
}

// parseSearchResults converts raw FT.SEARCH results to SearchResult.
//
// Takes rawResults ([]any) which contains the document pairs from
// the FT.SEARCH response.
// Takes total (int64) which is the total number of matching
// documents reported by Redis.
// Takes opts (*cache.SearchOptions) which provides the pagination
// settings for the result.
//
// Returns the search result containing the parsed hits with
// pagination metadata, and an error when result parsing fails.
func (a *RedisClusterAdapter[K, V]) parseSearchResults(ctx context.Context, rawResults []any, total int64, opts *cache.SearchOptions) (cache.SearchResult[K, V], error) {
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

// parseSearchHit parses a single search hit from the raw results at the given
// index.
//
// Takes rawResults ([]any) which contains the raw FT.SEARCH
// result pairs.
// Takes i (int) which is the index of the key element in the
// results slice.
//
// Returns cache.SearchHit[K, V] which contains the decoded key
// and unmarshalled value.
// Returns bool which indicates whether parsing succeeded.
func (a *RedisClusterAdapter[K, V]) parseSearchHit(ctx context.Context, rawResults []any, i int) (cache.SearchHit[K, V], bool) {
	_, l := logger.From(ctx, log)

	var zero cache.SearchHit[K, V]

	keyString, ok := rawResults[i].(string)
	if !ok {
		return zero, false
	}

	key, err := a.decodeKey(keyString)
	if err != nil {
		l.Trace("Failed to decode key from search result",
			logger.String(redisLogKeyField, keyString),
			logger.Error(err))
		return zero, false
	}

	if i+1 >= len(rawResults) {
		return zero, false
	}

	docData, ok := rawResults[i+1].([]any)
	if !ok || len(docData) < 2 {
		return zero, false
	}

	jsonString := extractJSONFromDocData(docData)
	if jsonString == "" {
		return zero, false
	}

	var value V
	if err := json.Unmarshal([]byte(jsonString), &value); err != nil {
		l.Trace("Failed to unmarshal search result value",
			logger.String(redisLogKeyField, keyString),
			logger.Error(err))
		return zero, false
	}

	return cache.SearchHit[K, V]{Key: key, Value: value}, true
}

// setJSONValue stores a value as JSON for RediSearch indexing.
//
// Takes keyString (string) which is the Redis key to store the value under.
// Takes value (V) which is the value to marshal and store as JSON.
// Takes ttl (int) which is the time-to-live in seconds; zero means no expiry.
//
// Returns error when marshalling fails or the JSON.SET command fails.
func (a *RedisClusterAdapter[K, V]) setJSONValue(ctx context.Context, keyString string, value V, ttl int) error {
	ctx, l := logger.From(ctx, log)

	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value to JSON: %w", err)
	}

	arguments := []any{"JSON.SET", keyString, redisJSONPath, string(jsonBytes)}
	if err := a.client.Do(ctx, arguments...).Err(); err != nil {
		return fmt.Errorf("JSON.SET failed: %w", err)
	}

	if ttl > 0 {
		if err := a.client.Expire(ctx, keyString, a.ttl).Err(); err != nil {
			l.Warn("Failed to set TTL on JSON document",
				logger.String(redisLogKeyField, keyString),
				logger.Error(err))
		}
	}

	return nil
}

// indexDocument stores a document for RediSearch indexing.
// This is called from Set when a search schema is configured.
//
// Takes keyString (string) which is the Redis key for the document.
// Takes value (V) which is the value to index.
func (a *RedisClusterAdapter[K, V]) indexDocument(ctx context.Context, keyString string, value V) {
	ctx, l := logger.From(ctx, log)

	if a.schema == nil {
		return
	}

	if err := a.ensureIndexExists(ctx); err != nil {
		l.Warn("Failed to ensure search index exists",
			logger.String(redisLogKeyField, keyString),
			logger.Error(err))
		return
	}

	ttlSeconds := int(a.ttl.Seconds())
	if err := a.setJSONValue(ctx, keyString, value, ttlSeconds); err != nil {
		l.Warn("Failed to index document",
			logger.String(redisLogKeyField, keyString),
			logger.Error(err))
	}
}

// searchWithRediSearch performs a full-text search using FT.SEARCH.
//
// Takes query (string) which is the text query to search for.
// Takes opts (*cache.SearchOptions) which provides filters,
// pagination, and sorting options.
//
// Returns cache.SearchResult[K, V] which contains the matched
// documents with pagination metadata.
// Returns error when index creation or the search command fails.
func (a *RedisClusterAdapter[K, V]) searchWithRediSearch(ctx context.Context, query string, opts *cache.SearchOptions) (cache.SearchResult[K, V], error) {
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

// queryWithRediSearch performs a structured query using FT.SEARCH.
//
// Takes opts (*cache.QueryOptions) which provides filters,
// pagination, and sorting options.
//
// Returns cache.SearchResult[K, V] which contains the matched
// documents with pagination metadata.
// Returns error when index creation or the search command fails.
func (a *RedisClusterAdapter[K, V]) queryWithRediSearch(ctx context.Context, opts *cache.QueryOptions) (cache.SearchResult[K, V], error) {
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

// dropIndex removes the RediSearch index. Called during InvalidateAll if search
// is enabled.
func (a *RedisClusterAdapter[K, V]) dropIndex(ctx context.Context) {
	ctx, l := logger.From(ctx, log)

	if a.schema == nil || !a.indexCreated {
		return
	}

	if err := a.client.Do(ctx, "FT.DROPINDEX", a.indexName).Err(); err != nil {
		if !isUnknownIndexError(err) {
			l.Warn("Failed to drop search index",
				logger.String("index", a.indexName),
				logger.Error(err))
		}
	}

	a.indexCreated = false
}

// needsJSONStorage reports whether search is enabled and values should be
// stored as JSON.
//
// Returns bool which is true when a schema is configured.
func (a *RedisClusterAdapter[K, V]) needsJSONStorage() bool {
	return a.schema != nil
}

// getJSONValue retrieves a JSON-stored value from Redis.
//
// Takes keyString (string) which specifies the Redis key to retrieve.
//
// Returns V which is the deserialised value from the JSON document.
// Returns bool which indicates whether the value was found and valid.
// Returns error when the operation fails.
func (a *RedisClusterAdapter[K, V]) getJSONValue(ctx context.Context, keyString string) (V, bool, error) {
	var zero V

	result, err := a.client.Do(ctx, "JSON.GET", keyString, redisJSONPath).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return zero, false, nil
		}
		return zero, false, fmt.Errorf("JSON.GET failed: %w", err)
	}

	jsonString, ok := result.(string)
	if !ok {
		return zero, false, fmt.Errorf("unexpected result type from JSON.GET: %T", result)
	}

	var values []V
	if err := json.Unmarshal([]byte(jsonString), &values); err != nil {
		return zero, false, fmt.Errorf("failed to unmarshal JSON value: %w", err)
	}

	if len(values) == 0 {
		return zero, false, nil
	}

	return values[0], true, nil
}

// isUnknownIndexError reports whether the error indicates
// an unknown RediSearch index.
//
// Takes err (error) which is the error to check for an unknown index message.
//
// Returns bool which is true when the error message contains "Unknown".
func isUnknownIndexError(err error) bool {
	return strings.Contains(err.Error(), "Unknown")
}

// escapeTagValue escapes special characters in tag values for RediSearch.
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
// Takes docData ([]any) which contains field name and value pairs.
//
// Returns string which is the JSON value if found, or an empty string if not.
func extractJSONFromDocData(docData []any) string {
	for j := 0; j < len(docData); j += 2 {
		fieldName, ok := docData[j].(string)
		if !ok || fieldName != redisJSONPath {
			continue
		}
		if j+1 < len(docData) {
			if value, ok := docData[j+1].(string); ok {
				return value
			}
		}
	}
	return ""
}
