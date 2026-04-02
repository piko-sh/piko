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

package cache_provider_valkey

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
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

	// searchFmtInt is the format specifier for converting integers to strings in
	// search commands.
	searchFmtInt = "%d"

	// searchKeywordAS is the AS keyword used in FT.CREATE schema definitions and
	// FT.SEARCH RETURN clauses.
	searchKeywordAS = "AS"
)

// indexCreationMu protects concurrent index creation attempts.
var indexCreationMu sync.Mutex

// ensureIndexExists creates the Valkey Search index if it does not already
// exist. This is called lazily on first search operation.
//
// Returns error when the index cannot be created or search is not supported.
//
// Safe for concurrent use. Uses double-checked locking to ensure the index is
// created only once.
func (a *ValkeyAdapter[K, V]) ensureIndexExists(ctx context.Context) error {
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

	err := a.client.Do(ctx, a.client.B().Arbitrary("FT.INFO").Keys(a.indexName).Build()).Error()
	if err == nil {
		a.indexCreated = true
		l.Internal("Valkey Search index already exists", logger.String(logKeyIndex, a.indexName))
		return nil
	}

	if err := a.createIndex(ctx); err != nil {
		return err
	}

	a.indexCreated = true
	return nil
}

// createIndex creates the Valkey Search index using FT.CREATE.
// Valkey Search only supports TAG and NUMERIC fields; TEXT and GEO fields
// are skipped with a warning.
//
// Returns error when the index creation fails.
func (a *ValkeyAdapter[K, V]) createIndex(ctx context.Context) error {
	_, l := logger.From(ctx, log)

	arguments := []string{
		"FT.CREATE", a.indexName,
		"ON", "JSON",
		"PREFIX", "1", a.namespace,
		"SCHEMA",
	}

	fieldCount := 0
	for _, field := range a.schema.Fields {
		jsonPath := "$." + field.Name
		alias := field.Name

		switch field.Type {
		case cache.FieldTypeText:
			l.Internal("Skipping TEXT field in Valkey Search index; Valkey Search does not support TEXT fields",
				logger.String("field", field.Name),
				logger.String(logKeyIndex, a.indexName))
			continue
		case cache.FieldTypeGeo:
			l.Internal("Skipping GEO field in Valkey Search index; Valkey Search does not support GEO fields",
				logger.String("field", field.Name),
				logger.String(logKeyIndex, a.indexName))
			continue
		case cache.FieldTypeTag:
			arguments = append(arguments, jsonPath, searchKeywordAS, alias, "TAG")
		case cache.FieldTypeNumeric:
			arguments = append(arguments, jsonPath, searchKeywordAS, alias, "NUMERIC")
		case cache.FieldTypeVector:
			arguments = append(arguments, jsonPath, searchKeywordAS, alias, "VECTOR", "HNSW", "6",
				"TYPE", "FLOAT32",
				"DIM", fmt.Sprintf(searchFmtInt, field.Dimension),
				"DISTANCE_METRIC", vectorDistanceMetric(field.DistanceMetric),
			)
		}

		if field.Sortable {
			arguments = append(arguments, "SORTABLE")
		}

		fieldCount++
	}

	command := a.client.B().Arbitrary(arguments[0])
	for _, arg := range arguments[1:] {
		command = command.Args(arg)
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
// Returns string which is the complete query string, or "*" if no terms are
// provided.
func (a *ValkeyAdapter[K, V]) buildSearchQuery(textQuery string, filters []cache.Filter) string {
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

// buildFilterClause converts a Filter to a search query clause.
//
// Takes f (cache.Filter) which specifies the filter to convert.
//
// Returns string which is the search query clause, or empty if the
// filter operation is not supported.
func (*ValkeyAdapter[K, V]) buildFilterClause(f cache.Filter) string {
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
// Takes query (string) which specifies the search query to execute.
// Takes opts (*cache.SearchOptions) which provides pagination and sorting
// options.
//
// Returns []valkey.ValkeyMessage which contains the raw search results.
// Returns int64 which is the total count of matching documents.
// Returns error when the search command fails or the result format is
// unexpected.
func (a *ValkeyAdapter[K, V]) executeSearch(ctx context.Context, query string, opts *cache.SearchOptions) ([]valkey.ValkeyMessage, int64, error) {
	limit := DefaultSearchResultLimit
	offset := 0
	if opts != nil {
		if opts.Limit > 0 {
			limit = opts.Limit
		}
		offset = opts.Offset
	}

	command := a.client.B().Arbitrary("FT.SEARCH").Keys(a.indexName).Args(query)
	command = command.Args("LIMIT", fmt.Sprintf(searchFmtInt, offset), fmt.Sprintf(searchFmtInt, limit))

	if opts != nil && opts.SortBy != "" {
		command = command.Args("SORTBY", opts.SortBy)
		if opts.SortOrder == cache.SortDesc {
			command = command.Args("DESC")
		} else {
			command = command.Args("ASC")
		}
	}

	command = command.Args("RETURN", "1", valkeyJSONPath)

	result, err := a.client.Do(ctx, command.Build()).ToArray()
	if err != nil {
		return nil, 0, fmt.Errorf("FT.SEARCH failed: %w", err)
	}

	if len(result) == 0 {
		return nil, 0, nil
	}

	total, err := result[0].AsInt64()
	if err != nil {
		return nil, 0, fmt.Errorf("unexpected result format: total count not int64: %w", err)
	}

	return result[1:], total, nil
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
func (a *ValkeyAdapter[K, V]) parseSearchResults(ctx context.Context, rawResults []valkey.ValkeyMessage, total int64, opts *cache.SearchOptions) (cache.SearchResult[K, V], error) {
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
func (a *ValkeyAdapter[K, V]) parseSearchHit(ctx context.Context, rawResults []valkey.ValkeyMessage, i int) (cache.SearchHit[K, V], bool) {
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
// Takes keyString (string) which is the key under which to store the value.
// Takes value (V) which is the value to marshal and store as JSON.
// Takes ttl (int) which is the time-to-live in seconds; zero means no expiry.
//
// Returns error when marshalling fails or the JSON.SET command fails.
func (a *ValkeyAdapter[K, V]) setJSONValue(ctx context.Context, keyString string, value V, ttl int) error {
	ctx, l := logger.From(ctx, log)

	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value to JSON: %w", err)
	}

	command := a.client.B().Arbitrary("JSON.SET").Keys(keyString).Args(valkeyJSONPath, string(jsonBytes))
	if err := a.client.Do(ctx, command.Build()).Error(); err != nil {
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
// This is called from Set when search schema is configured.
//
// Takes keyString (string) which is the cache key for the document.
// Takes value (V) which is the document to be indexed.
func (a *ValkeyAdapter[K, V]) indexDocument(ctx context.Context, keyString string, value V) {
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
// pagination, sorting, and optional vector parameters.
//
// Returns cache.SearchResult[K, V] which contains the matched items
// and total count.
// Returns error when the index cannot be ensured or the search
// command fails.
func (a *ValkeyAdapter[K, V]) searchWithValkeySearch(ctx context.Context, query string, opts *cache.SearchOptions) (cache.SearchResult[K, V], error) {
	if err := a.ensureIndexExists(ctx); err != nil {
		return cache.SearchResult[K, V]{}, err
	}

	if opts != nil && len(opts.Vector) > 0 {
		return a.vectorSearchWithValkeySearch(ctx, query, opts)
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
// sorting, and optional vector parameters.
//
// Returns cache.SearchResult[K, V] which contains the matched items
// and total count.
// Returns error when the index cannot be ensured or the search
// command fails.
func (a *ValkeyAdapter[K, V]) queryWithValkeySearch(ctx context.Context, opts *cache.QueryOptions) (cache.SearchResult[K, V], error) {
	if err := a.ensureIndexExists(ctx); err != nil {
		return cache.SearchResult[K, V]{}, err
	}

	searchOpts := &cache.SearchOptions{
		Limit:  DefaultSearchResultLimit,
		Offset: 0,
	}
	if opts != nil {
		searchOpts.Limit = opts.Limit
		searchOpts.Offset = opts.Offset
		searchOpts.SortBy = opts.SortBy
		searchOpts.SortOrder = opts.SortOrder
		searchOpts.Vector = opts.Vector
		searchOpts.VectorField = opts.VectorField
		searchOpts.MinScore = opts.MinScore
		searchOpts.TopK = opts.TopK
		searchOpts.Filters = opts.Filters
	}
	if searchOpts.Limit == 0 {
		searchOpts.Limit = DefaultSearchResultLimit
	}

	if len(searchOpts.Vector) > 0 {
		return a.vectorSearchWithValkeySearch(ctx, "", searchOpts)
	}

	var filters []cache.Filter
	if opts != nil {
		filters = opts.Filters
	}
	searchQuery := a.buildSearchQuery("", filters)

	rawResults, total, err := a.executeSearch(ctx, searchQuery, searchOpts)
	if err != nil {
		return cache.SearchResult[K, V]{}, err
	}

	return a.parseSearchResults(ctx, rawResults, total, searchOpts)
}

// dropIndex removes the search index. Called during InvalidateAll if search is
// enabled.
func (a *ValkeyAdapter[K, V]) dropIndex(ctx context.Context) {
	ctx, l := logger.From(ctx, log)

	if a.schema == nil || !a.indexCreated {
		return
	}

	command := a.client.B().Arbitrary("FT.DROPINDEX").Keys(a.indexName)
	if err := a.client.Do(ctx, command.Build()).Error(); err != nil {
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
// Returns bool which is true when a schema is configured for search.
func (a *ValkeyAdapter[K, V]) needsJSONStorage() bool {
	return a.schema != nil
}

// vectorSearchWithValkeySearch performs a vector similarity search using
// FT.SEARCH with KNN query syntax and DIALECT 2.
//
// Takes query (string) which is the optional text query to intersect with
// vector results.
// Takes opts (*cache.SearchOptions) which provides the vector, filters, and
// pagination options.
//
// Returns SearchResult[K, V] which contains the matched documents
// with similarity scores.
// Returns error when the search command fails.
func (a *ValkeyAdapter[K, V]) vectorSearchWithValkeySearch(ctx context.Context, query string, opts *cache.SearchOptions) (cache.SearchResult[K, V], error) {
	vectorField := opts.VectorField
	if vectorField == "" {
		vectorField = a.resolveVectorField()
	}
	if vectorField == "" {
		return cache.SearchResult[K, V]{}, errors.New("no vector field found in schema")
	}

	topK := opts.TopK
	if topK <= 0 {
		topK = opts.Limit
	}
	if topK <= 0 {
		topK = DefaultSearchResultLimit
	}

	baseQuery := a.buildSearchQuery(query, opts.Filters)
	knnQuery := fmt.Sprintf("(%s)=>[KNN %d @%s $vec]", baseQuery, topK, vectorField)
	scoreField := "__" + vectorField + "_score"

	command := a.client.B().Arbitrary("FT.SEARCH").Keys(a.indexName).Args(knnQuery)
	command = command.Args("PARAMS", "2", "vec", string(vectorToBlob(opts.Vector)))
	command = command.Args("LIMIT", fmt.Sprintf(searchFmtInt, opts.Offset), fmt.Sprintf(searchFmtInt, topK))
	command = command.Args("RETURN", "2", valkeyJSONPath, scoreField)
	command = command.Args("DIALECT", "2")

	result, err := a.client.Do(ctx, command.Build()).ToArray()
	if err != nil {
		return cache.SearchResult[K, V]{}, fmt.Errorf("FT.SEARCH vector query failed: %w", err)
	}

	if len(result) == 0 {
		return cache.SearchResult[K, V]{Items: make([]cache.SearchHit[K, V], 0)}, nil
	}

	total, err := result[0].AsInt64()
	if err != nil {
		return cache.SearchResult[K, V]{}, fmt.Errorf("unexpected result format: total count not int64: %w", err)
	}

	return a.parseVectorSearchResults(ctx, result[1:], total, opts, scoreField)
}

// parseVectorSearchResults converts raw FT.SEARCH vector results to
// SearchResult, extracting similarity scores from the distance field.
//
// Takes rawResults ([]valkey.ValkeyMessage) which contains the raw
// document pairs from the vector search response.
// Takes total (int64) which is the total count of matching documents
// reported by Valkey.
// Takes opts (*cache.SearchOptions) which provides pagination and
// minimum score filtering.
// Takes scoreField (string) which is the name of the distance score
// field in the result documents.
//
// Returns cache.SearchResult[K, V] which contains the parsed hits
// with similarity scores.
// Returns error which is currently always nil but reserved for future
// use.
func (a *ValkeyAdapter[K, V]) parseVectorSearchResults(
	ctx context.Context, rawResults []valkey.ValkeyMessage, total int64,
	opts *cache.SearchOptions, scoreField string,
) (cache.SearchResult[K, V], error) {
	result := cache.SearchResult[K, V]{
		Items:  make([]cache.SearchHit[K, V], 0),
		Total:  total,
		Offset: opts.Offset,
		Limit:  opts.Limit,
	}

	for i := 0; i < len(rawResults); i += 2 {
		hit, ok := a.parseVectorSearchHit(ctx, rawResults, i, scoreField)
		if !ok {
			continue
		}
		if opts.MinScore != nil && hit.Score < float64(*opts.MinScore) {
			continue
		}
		result.Items = append(result.Items, hit)
	}

	return result, nil
}

// parseVectorSearchHit parses a single vector search hit, extracting
// the document and its distance score.
//
// Takes rawResults ([]valkey.ValkeyMessage) which contains the full
// array of raw vector search result messages.
// Takes i (int) which is the index of the key element; the document
// data follows at i+1.
// Takes scoreField (string) which is the name of the distance score
// field to extract from the document data.
//
// Returns cache.SearchHit[K, V] which contains the decoded key,
// unmarshalled value, and similarity score.
// Returns bool which is true when parsing succeeded.
func (a *ValkeyAdapter[K, V]) parseVectorSearchHit(ctx context.Context, rawResults []valkey.ValkeyMessage, i int, scoreField string) (cache.SearchHit[K, V], bool) {
	_, l := logger.From(ctx, log)

	var zero cache.SearchHit[K, V]

	keyString, err := rawResults[i].ToString()
	if err != nil {
		return zero, false
	}

	key, err := a.decodeKey(keyString)
	if err != nil {
		l.Trace("Failed to decode key from vector search result",
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
		l.Trace("Failed to unmarshal vector search result value",
			logger.String(valkeyLogKeyField, keyString),
			logger.Error(err))
		return zero, false
	}

	score := extractVectorScore(docData, scoreField)

	return cache.SearchHit[K, V]{Key: key, Value: value, Score: score}, true
}

// resolveVectorField returns the name of the first vector field in the schema.
//
// Returns string which is the field name, or empty if no vector field exists.
func (a *ValkeyAdapter[K, V]) resolveVectorField() string {
	if a.schema == nil {
		return ""
	}
	for _, field := range a.schema.Fields {
		if field.Type == cache.FieldTypeVector {
			return field.Name
		}
	}
	return ""
}

// getJSONValue retrieves a JSON-stored value from Valkey.
//
// Takes keyString (string) which specifies the key to retrieve.
//
// Returns V which is the retrieved value.
// Returns bool which indicates whether the value was found.
func (a *ValkeyAdapter[K, V]) getJSONValue(ctx context.Context, keyString string) (V, bool) {
	var zero V

	command := a.client.B().Arbitrary("JSON.GET").Keys(keyString).Args(valkeyJSONPath)
	jsonString, err := a.client.Do(ctx, command.Build()).ToString()
	if err != nil {
		return zero, false
	}

	var values []V
	if err := json.Unmarshal([]byte(jsonString), &values); err != nil {
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

// escapeTagValue escapes special characters in tag values for search queries.
//
// Takes v (any) which is the value to escape.
//
// Returns string which is the escaped value safe for use in search queries.
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
// Takes docData ([]valkey.ValkeyMessage) which contains the document fields
// as key-value pairs.
//
// Returns string which is the JSON value if found, or an empty string if the
// field is not present or cannot be read.
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

// extractVectorScore extracts the distance score from document data and
// converts it to a similarity score. Valkey returns distance (lower is better),
// so we convert to similarity (higher is better) as 1 - distance.
//
// Takes docData ([]valkey.ValkeyMessage) which contains the document fields.
// Takes scoreField (string) which specifies the field name holding the score.
//
// Returns float64 which is the similarity score, or 0 if not found.
func extractVectorScore(docData []valkey.ValkeyMessage, scoreField string) float64 {
	for j := 0; j < len(docData); j += 2 {
		fieldName, err := docData[j].ToString()
		if err != nil || fieldName != scoreField {
			continue
		}
		if j+1 >= len(docData) {
			continue
		}
		value, err := docData[j+1].ToString()
		if err != nil {
			continue
		}
		distance, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return 1 - distance
		}
	}
	return 0
}

// vectorDistanceMetric converts a cache DTO distance metric string to the
// Valkey FT.CREATE DISTANCE_METRIC parameter value.
//
// Takes metric (string) which is the metric name from FieldSchema.
//
// Returns string which is the Valkey-format metric name.
func vectorDistanceMetric(metric string) string {
	switch metric {
	case "euclidean":
		return "L2"
	case "dot_product":
		return "IP"
	default:
		return "COSINE"
	}
}

// vectorToBlob serialises a float32 vector to a little-endian byte blob
// for use as a PARAMS argument in FT.SEARCH KNN queries.
//
// Takes v ([]float32) which is the vector to serialise.
//
// Returns []byte which is the raw little-endian float32 blob.
func vectorToBlob(v []float32) []byte {
	buffer := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(buffer[i*4:], math.Float32bits(f))
	}
	return buffer
}
