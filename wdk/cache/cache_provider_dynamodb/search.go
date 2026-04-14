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

package cache_provider_dynamodb

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
)

const (
	// searchFieldPrefix is prepended to field names for DynamoDB search
	// attributes, avoiding collisions with core attributes (pk, sk, val, etc.).
	searchFieldPrefix = "sf_"

	// filterValueSuffix is the default suffix for single-value filter
	// placeholders.
	filterValueSuffix = "0"

	// intFormatVerb is the format string for integer values in DynamoDB
	// attribute conversions.
	intFormatVerb = "%d"

	// gsiFilterValuePlaceholder is the expression attribute value placeholder
	// for GSI key condition values.
	gsiFilterValuePlaceholder = ":fv"
)

// SupportsSearch returns true if a search schema is configured.
//
// Returns bool which is true when search operations are available.
func (a *DynamoDBAdapter[K, V]) SupportsSearch() bool {
	return a.schema != nil
}

// GetSchema returns the search schema for this cache.
//
// Returns *cache.SearchSchema which describes searchable fields, or nil.
func (a *DynamoDBAdapter[K, V]) GetSchema() *cache.SearchSchema {
	return a.schema
}

// Search returns ErrSearchNotSupported because DynamoDB does not support
// full-text search without server-side indexing.
//
// Takes opts (*cache.SearchOptions) which specifies pagination parameters.
//
// Returns cache.SearchResult[K, V] which contains empty results with pagination
// metadata.
// Returns error which is always ErrSearchNotSupported.
func (*DynamoDBAdapter[K, V]) Search(_ context.Context, _ string, opts *cache.SearchOptions) (cache.SearchResult[K, V], error) {
	return cache.SearchResult[K, V]{
			Offset: opts.Offset,
			Limit:  opts.Limit,
		}, fmt.Errorf(
			"%w: DynamoDB provider does not support full-text search",
			cache.ErrSearchNotSupported,
		)
}

// Query performs structured filtering, sorting, and pagination without
// full-text search.
//
// Takes opts (*cache.QueryOptions) which specifies filters, sorting, and
// pagination.
//
// Returns cache.SearchResult[K, V] which contains matched entries.
// Returns error when no schema is configured (ErrSearchNotSupported).
func (a *DynamoDBAdapter[K, V]) Query(ctx context.Context, opts *cache.QueryOptions) (cache.SearchResult[K, V], error) {
	if a.schema == nil {
		return cache.SearchResult[K, V]{}, fmt.Errorf(
			"%w: DynamoDB provider requires a SearchSchema for query operations",
			cache.ErrSearchNotSupported,
		)
	}

	if len(opts.Vector) > 0 {
		return cache.SearchResult[K, V]{
				Offset: opts.Offset,
				Limit:  opts.Limit,
			}, fmt.Errorf(
				"%w: DynamoDB provider does not support vector queries",
				cache.ErrSearchNotSupported,
			)
	}

	if result, ok := a.tryGSIQuery(ctx, opts); ok {
		return result, nil
	}

	candidateKeys := a.getAllKeys(ctx)

	if len(candidateKeys) == 0 {
		return cache.SearchResult[K, V]{
			Items:  nil,
			Total:  0,
			Offset: opts.Offset,
			Limit:  opts.Limit,
		}, nil
	}

	filteredKeys := a.applyFilters(ctx, candidateKeys, opts.Filters)
	sortedKeys := a.sortKeys(filteredKeys, opts.SortBy, opts.SortOrder)

	return a.buildSearchResult(ctx, sortedKeys, opts.Offset, opts.Limit)
}

// getAllKeys returns all keys stored in the DynamoDB namespace by scanning the
// table.
//
// Returns []K which contains all non-expired keys in the namespace.
func (a *DynamoDBAdapter[K, V]) getAllKeys(ctx context.Context) []K {
	_, l := logger.From(ctx, log)
	var keys []K

	paginator := dynamodb.NewScanPaginator(a.client, &dynamodb.ScanInput{
		TableName:            aws.String(a.tableName),
		FilterExpression:     aws.String("#ns = :ns AND #sk = :sk"),
		ProjectionExpression: aws.String("#pk"),
		ExpressionAttributeNames: map[string]string{
			"#ns": attrNamespace,
			"#sk": attrSK,
			"#pk": attrPK,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":ns": &types.AttributeValueMemberS{Value: a.namespace},
			":sk": &types.AttributeValueMemberS{Value: skData},
		},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			l.Warn("Failed to scan for keys during search", logger.Error(err))
			return keys
		}

		for _, item := range page.Items {
			pkAttr, ok := item[attrPK].(*types.AttributeValueMemberS)
			if !ok {
				continue
			}
			key, err := a.decodeKey(pkAttr.Value)
			if err != nil {
				continue
			}
			keys = append(keys, key)
		}
	}

	return keys
}

// applyFilters filters candidate keys by checking each value against the
// provided filter conditions using the field extractor.
//
// Takes keys ([]K) which specifies the keys to filter.
// Takes filters ([]cache.Filter) which provides the filter conditions.
//
// Returns []K which contains the keys that match all filter conditions.
func (a *DynamoDBAdapter[K, V]) applyFilters(ctx context.Context, keys []K, filters []cache.Filter) []K {
	if len(filters) == 0 || a.fieldExtractor == nil {
		return keys
	}

	result := make([]K, 0, len(keys))
	for _, key := range keys {
		value, ok, err := a.GetIfPresent(ctx, key)
		if err != nil || !ok {
			continue
		}

		if cache_domain.MatchesAllFilters(a.fieldExtractor, value, filters) {
			result = append(result, key)
		}
	}
	return result
}

// sortKeys sorts keys by the specified field and order using field extraction.
//
// Takes keys ([]K) which contains the keys to sort.
// Takes sortBy (string) which specifies the field name to sort by.
// Takes sortOrder (cache.SortOrder) which specifies ascending or descending.
//
// Returns []K which contains the sorted keys, or the original keys if sortBy
// is empty.
func (a *DynamoDBAdapter[K, V]) sortKeys(keys []K, sortBy string, sortOrder cache.SortOrder) []K {
	if sortBy == "" || len(keys) == 0 {
		return keys
	}

	ascending := sortOrder == cache.SortAsc
	return a.sortKeysByField(keys, sortBy, ascending)
}

// sortKeysByField sorts keys by extracting field values and comparing them.
//
// Takes keys ([]K) which contains the keys to sort.
// Takes fieldName (string) which specifies the field to extract for comparison.
// Takes ascending (bool) which determines the sort direction.
//
// Returns []K which contains the sorted keys.
func (a *DynamoDBAdapter[K, V]) sortKeysByField(keys []K, fieldName string, ascending bool) []K {
	if a.fieldExtractor == nil {
		return keys
	}

	type keyWithValue struct {
		key   K
		value any
	}

	items := make([]keyWithValue, 0, len(keys))
	for _, key := range keys {
		value, ok, err := a.GetIfPresent(context.Background(), key)
		if err != nil || !ok {
			continue
		}
		fieldValue, _ := a.fieldExtractor.ExtractAny(value, fieldName)
		items = append(items, keyWithValue{key: key, value: fieldValue})
	}

	slices.SortFunc(items, func(itemA, itemB keyWithValue) int {
		comparison := cache_domain.CompareNumeric(itemA.value, itemB.value)
		if !ascending {
			comparison = -comparison
		}
		return comparison
	})

	result := make([]K, 0, len(items))
	for _, item := range items {
		result = append(result, item.key)
	}
	return result
}

// buildSearchResult creates a SearchResult with pagination applied and flat
// scoring (1.0 for all hits).
//
// Takes keys ([]K) which contains the matched keys in display order.
// Takes offset (int) which is the pagination offset.
// Takes limit (int) which is the maximum number of results.
//
// Returns cache.SearchResult[K, V] which contains the paginated results.
// Returns error when a value cannot be retrieved for a matched key.
func (a *DynamoDBAdapter[K, V]) buildSearchResult(ctx context.Context, keys []K, offset, limit int) (cache.SearchResult[K, V], error) {
	return a.buildSearchResultWithScores(ctx, keys, nil, offset, limit)
}

// buildSearchResultWithScores creates a SearchResult with pagination applied.
// When scores is non-nil, each hit uses the score from the map; otherwise a
// flat score of 1.0 is used.
//
// Takes keys ([]K) which contains the matched keys in display order.
// Takes scores (map[K]float64) which maps keys to relevance scores (may be
// nil).
// Takes offset (int) which is the pagination offset.
// Takes limit (int) which is the maximum number of results.
//
// Returns cache.SearchResult[K, V] with scored items.
func (a *DynamoDBAdapter[K, V]) buildSearchResultWithScores(ctx context.Context, keys []K, scores map[K]float64, offset, limit int) (cache.SearchResult[K, V], error) {
	total := int64(len(keys))
	keys, limit = cache_domain.ApplyPagination(keys, offset, limit)

	items := make([]cache.SearchHit[K, V], 0, len(keys))
	for _, key := range keys {
		value, ok, err := a.GetIfPresent(ctx, key)
		if err != nil || !ok {
			continue
		}

		score := 1.0
		if scores != nil {
			if scoredValue, exists := scores[key]; exists {
				score = scoredValue
			}
		}

		items = append(items, cache.SearchHit[K, V]{
			Key:   key,
			Value: value,
			Score: score,
		})
	}

	return cache.SearchResult[K, V]{
		Items:  items,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}, nil
}

// buildFilterExpression constructs a DynamoDB FilterExpression from cache
// filters. Each filter targets a search field attribute (sf_{fieldname}) and
// the conditions are combined with AND.
//
// Takes filters ([]cache.Filter) which specifies the filter conditions.
//
// Returns string which is the filter expression string.
// Returns map[string]string which maps expression attribute name placeholders
// to actual attribute names.
// Returns map[string]types.AttributeValue which maps expression value
// placeholders to attribute values.
func buildFilterExpression(filters []cache.Filter) (string, map[string]string, map[string]types.AttributeValue) {
	if len(filters) == 0 {
		return "", nil, nil
	}

	expressionNames := make(map[string]string)
	expressionValues := make(map[string]types.AttributeValue)
	var conditions []string

	for filterIndex, filter := range filters {
		nameAlias := fmt.Sprintf("#sf_%d", filterIndex)
		expressionNames[nameAlias] = searchFieldPrefix + filter.Field

		condition := buildSingleFilterCondition(filter, filterIndex, nameAlias, expressionValues)
		if condition != "" {
			conditions = append(conditions, condition)
		}
	}

	if len(conditions) == 0 {
		return "", nil, nil
	}

	return strings.Join(conditions, " AND "), expressionNames, expressionValues
}

// buildSingleFilterCondition produces a DynamoDB condition string for a single
// filter and populates the expression values map.
//
// Takes filter (cache.Filter) which is the filter condition.
// Takes filterIndex (int) which is the index used for unique placeholder names.
// Takes nameAlias (string) which is the expression attribute name alias.
// Takes expressionValues (map[string]types.AttributeValue) which collects value
// placeholders.
//
// Returns string which is the condition clause, or empty if unsupported.
func buildSingleFilterCondition(filter cache.Filter, filterIndex int, nameAlias string, expressionValues map[string]types.AttributeValue) string {
	valueAlias := func(suffix string) string {
		return fmt.Sprintf(":sf_%d_%s", filterIndex, suffix)
	}

	switch filter.Operation {
	case cache.FilterOpEq:
		return buildComparisonCondition(nameAlias, "=", valueAlias, filter.Value, expressionValues)
	case cache.FilterOpNe:
		return buildComparisonCondition(nameAlias, "<>", valueAlias, filter.Value, expressionValues)
	case cache.FilterOpGt:
		return buildComparisonCondition(nameAlias, ">", valueAlias, filter.Value, expressionValues)
	case cache.FilterOpGe:
		return buildComparisonCondition(nameAlias, ">=", valueAlias, filter.Value, expressionValues)
	case cache.FilterOpLt:
		return buildComparisonCondition(nameAlias, "<", valueAlias, filter.Value, expressionValues)
	case cache.FilterOpLe:
		return buildComparisonCondition(nameAlias, "<=", valueAlias, filter.Value, expressionValues)
	case cache.FilterOpIn:
		return buildInCondition(nameAlias, valueAlias, filter.Values, expressionValues)
	case cache.FilterOpBetween:
		return buildBetweenCondition(nameAlias, valueAlias, filter.Values, expressionValues)
	case cache.FilterOpPrefix:
		return buildPrefixCondition(nameAlias, valueAlias, filter.Value, expressionValues)
	default:
		return ""
	}
}

// buildComparisonCondition builds a simple binary comparison condition (=, <>,
// >, >=, <, <=).
//
// Takes nameAlias (string) which is the expression attribute name placeholder.
// Takes operator (string) which is the comparison operator.
// Takes valueAlias (func(string) string) which generates value placeholder
// names.
// Takes value (any) which is the filter value to compare against.
// Takes expressionValues (map[string]types.AttributeValue) which collects value
// placeholders.
//
// Returns string which is the condition clause.
func buildComparisonCondition(nameAlias, operator string, valueAlias func(string) string, value any, expressionValues map[string]types.AttributeValue) string {
	alias := valueAlias(filterValueSuffix)
	expressionValues[alias] = toAttributeValue(value)
	return fmt.Sprintf("%s %s %s", nameAlias, operator, alias)
}

// buildInCondition builds a DynamoDB IN condition for set membership.
//
// Takes nameAlias (string) which is the expression attribute name placeholder.
// Takes valueAlias (func(string) string) which generates value placeholder
// names.
// Takes values ([]any) which are the set membership values.
// Takes expressionValues (map[string]types.AttributeValue) which collects value
// placeholders.
//
// Returns string which is the IN condition clause, or empty if values is empty.
func buildInCondition(nameAlias string, valueAlias func(string) string, values []any, expressionValues map[string]types.AttributeValue) string {
	inParts := make([]string, 0, len(values))
	for valueIndex, value := range values {
		alias := valueAlias(strconv.Itoa(valueIndex))
		expressionValues[alias] = toAttributeValue(value)
		inParts = append(inParts, alias)
	}
	if len(inParts) == 0 {
		return ""
	}
	return fmt.Sprintf("%s IN (%s)", nameAlias, strings.Join(inParts, ", "))
}

// buildBetweenCondition builds a DynamoDB BETWEEN condition for range queries.
//
// Takes nameAlias (string) which is the expression attribute name placeholder.
// Takes valueAlias (func(string) string) which generates value placeholder
// names.
// Takes values ([]any) which must contain exactly two boundary values.
// Takes expressionValues (map[string]types.AttributeValue) which collects value
// placeholders.
//
// Returns string which is the BETWEEN clause, or empty if values does not
// contain exactly two elements.
func buildBetweenCondition(nameAlias string, valueAlias func(string) string, values []any, expressionValues map[string]types.AttributeValue) string {
	if len(values) != 2 {
		return ""
	}
	loAlias := valueAlias("lo")
	hiAlias := valueAlias("hi")
	expressionValues[loAlias] = toAttributeValue(values[0])
	expressionValues[hiAlias] = toAttributeValue(values[1])
	return fmt.Sprintf("%s BETWEEN %s AND %s", nameAlias, loAlias, hiAlias)
}

// buildPrefixCondition builds a DynamoDB begins_with condition for prefix
// matching.
//
// Takes nameAlias (string) which is the expression attribute name placeholder.
// Takes valueAlias (func(string) string) which generates value placeholder
// names.
// Takes value (any) which is the prefix string to match.
// Takes expressionValues (map[string]types.AttributeValue) which collects value
// placeholders.
//
// Returns string which is the begins_with condition clause.
func buildPrefixCondition(nameAlias string, valueAlias func(string) string, value any, expressionValues map[string]types.AttributeValue) string {
	alias := valueAlias(filterValueSuffix)
	expressionValues[alias] = toAttributeValue(value)
	return fmt.Sprintf("begins_with(%s, %s)", nameAlias, alias)
}

// toAttributeValue converts a Go value to a DynamoDB AttributeValue. Numeric
// types are stored as N, strings as S, and everything else is coerced to S via
// cache_domain.ToString.
//
// Takes value (any) which is the value to convert.
//
// Returns types.AttributeValue which is the DynamoDB representation.
func toAttributeValue(value any) types.AttributeValue {
	switch typedValue := value.(type) {
	case int:
		return &types.AttributeValueMemberN{Value: fmt.Sprintf(intFormatVerb, typedValue)}
	case int8:
		return &types.AttributeValueMemberN{Value: fmt.Sprintf(intFormatVerb, typedValue)}
	case int16:
		return &types.AttributeValueMemberN{Value: fmt.Sprintf(intFormatVerb, typedValue)}
	case int32:
		return &types.AttributeValueMemberN{Value: fmt.Sprintf(intFormatVerb, typedValue)}
	case int64:
		return &types.AttributeValueMemberN{Value: fmt.Sprintf(intFormatVerb, typedValue)}
	case uint:
		return &types.AttributeValueMemberN{Value: fmt.Sprintf(intFormatVerb, typedValue)}
	case uint8:
		return &types.AttributeValueMemberN{Value: fmt.Sprintf(intFormatVerb, typedValue)}
	case uint16:
		return &types.AttributeValueMemberN{Value: fmt.Sprintf(intFormatVerb, typedValue)}
	case uint32:
		return &types.AttributeValueMemberN{Value: fmt.Sprintf(intFormatVerb, typedValue)}
	case uint64:
		return &types.AttributeValueMemberN{Value: fmt.Sprintf(intFormatVerb, typedValue)}
	case float32:
		return &types.AttributeValueMemberN{Value: fmt.Sprintf("%g", typedValue)}
	case float64:
		return &types.AttributeValueMemberN{Value: fmt.Sprintf("%g", typedValue)}
	case string:
		return &types.AttributeValueMemberS{Value: typedValue}
	default:
		return &types.AttributeValueMemberS{Value: cache_domain.ToString(value)}
	}
}

// extractSearchAttributes uses the field extractor to build a map of DynamoDB
// attribute values for each search field defined in the schema.
//
// Takes value (V) which is the cached value to extract fields from.
//
// Returns map[string]types.AttributeValue which maps sf_{fieldname} to their
// DynamoDB values.
func (a *DynamoDBAdapter[K, V]) extractSearchAttributes(value V) map[string]types.AttributeValue {
	if a.fieldExtractor == nil || a.schema == nil {
		return nil
	}

	attributes := make(map[string]types.AttributeValue)
	for _, field := range a.schema.Fields {
		extracted, ok := a.fieldExtractor.ExtractAny(value, field.Name)
		if !ok {
			continue
		}
		attrName := searchFieldPrefix + field.Name
		attributes[attrName] = toAttributeValue(extracted)
	}

	if len(attributes) == 0 {
		return nil
	}
	return attributes
}

// tryGSIQuery attempts to use a DynamoDB GSI for the primary filter field,
// returning (result, true) when a GSI was used or (zero, false) to fall back
// to the Scan path.
//
// Takes opts (*cache.QueryOptions) which specifies filters, sorting, and
// pagination.
//
// Returns cache.SearchResult[K, V] which contains the query results when a GSI
// was used.
// Returns bool which is true when a GSI query was successfully performed.
func (a *DynamoDBAdapter[K, V]) tryGSIQuery(ctx context.Context, opts *cache.QueryOptions) (cache.SearchResult[K, V], bool) {
	if len(a.gsiFields) == 0 || len(opts.Filters) == 0 {
		return cache.SearchResult[K, V]{}, false
	}

	var gsiFilter cache.Filter
	var gsiName string
	var remainingFilters []cache.Filter
	found := false

	for _, f := range opts.Filters {
		if !found {
			if name, ok := a.gsiFields[f.Field]; ok {
				gsiFilter = f
				gsiName = name
				found = true
				continue
			}
		}
		remainingFilters = append(remainingFilters, f)
	}

	if !found {
		return cache.SearchResult[K, V]{}, false
	}

	keys := a.queryGSI(ctx, gsiName, gsiFilter)
	if keys == nil {
		return cache.SearchResult[K, V]{}, false
	}

	filteredKeys := a.applyFilters(ctx, keys, remainingFilters)
	sortedKeys := a.sortKeys(filteredKeys, opts.SortBy, opts.SortOrder)

	result, err := a.buildSearchResult(ctx, sortedKeys, opts.Offset, opts.Limit)
	if err != nil {
		return cache.SearchResult[K, V]{}, false
	}
	return result, true
}

// queryGSI queries a field GSI using the filter's operation and value.
//
// Takes gsiName (string) which is the name of the GSI to query.
// Takes filter (cache.Filter) which specifies the field condition.
//
// Returns []K which contains the matching keys, or nil on failure.
func (a *DynamoDBAdapter[K, V]) queryGSI(ctx context.Context, gsiName string, filter cache.Filter) []K {
	_, l := logger.From(ctx, log)

	attrName := searchFieldPrefix + filter.Field
	keyCondition, attrNames, attrValues := buildGSIKeyCondition(a.namespace, attrName, filter)
	if keyCondition == "" {
		return nil
	}

	paginator := dynamodb.NewQueryPaginator(a.client, &dynamodb.QueryInput{
		TableName:                 aws.String(a.tableName),
		IndexName:                 aws.String(gsiName),
		KeyConditionExpression:    aws.String(keyCondition),
		ExpressionAttributeNames:  attrNames,
		ExpressionAttributeValues: attrValues,
	})

	var keys []K
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			l.Warn("GSI query failed, falling back to Scan",
				logger.String("gsi", gsiName), logger.Error(err))
			return nil
		}
		for _, item := range page.Items {
			pkAttr, ok := item[attrPK].(*types.AttributeValueMemberS)
			if !ok {
				continue
			}
			key, err := a.decodeKey(pkAttr.Value)
			if err != nil {
				continue
			}
			keys = append(keys, key)
		}
	}

	return keys
}

// buildGSIKeyCondition builds a KeyConditionExpression for a GSI query from a
// filter, using the namespace as partition key and the filter field as sort key.
//
// Takes namespace (string) which is the partition key value.
// Takes attrName (string) which is the sort key attribute name.
// Takes filter (cache.Filter) which specifies the operation and value.
//
// Returns string which is the key condition expression, or empty if unsupported.
// Returns map[string]string which maps expression attribute name placeholders.
// Returns map[string]types.AttributeValue which maps value placeholders.
func buildGSIKeyCondition(
	namespace string,
	attrName string,
	filter cache.Filter,
) (string, map[string]string, map[string]types.AttributeValue) {
	attrNames := map[string]string{
		"#ns":    attrNamespace,
		"#field": attrName,
	}
	attrValues := map[string]types.AttributeValue{
		":ns": &types.AttributeValueMemberS{Value: namespace},
	}

	pkCondition := "#ns = :ns"
	var skCondition string

	switch filter.Operation {
	case cache.FilterOpEq:
		skCondition = "#field = :fv"
		attrValues[gsiFilterValuePlaceholder] = toAttributeValue(filter.Value)

	case cache.FilterOpGt:
		skCondition = "#field > :fv"
		attrValues[gsiFilterValuePlaceholder] = toAttributeValue(filter.Value)

	case cache.FilterOpGe:
		skCondition = "#field >= :fv"
		attrValues[gsiFilterValuePlaceholder] = toAttributeValue(filter.Value)

	case cache.FilterOpLt:
		skCondition = "#field < :fv"
		attrValues[gsiFilterValuePlaceholder] = toAttributeValue(filter.Value)

	case cache.FilterOpLe:
		skCondition = "#field <= :fv"
		attrValues[gsiFilterValuePlaceholder] = toAttributeValue(filter.Value)

	case cache.FilterOpBetween:
		if len(filter.Values) >= 2 {
			skCondition = "#field BETWEEN :fv_lo AND :fv_hi"
			attrValues[":fv_lo"] = toAttributeValue(filter.Values[0])
			attrValues[":fv_hi"] = toAttributeValue(filter.Values[1])
		}

	case cache.FilterOpPrefix:
		skCondition = "begins_with(#field, :fv)"
		attrValues[gsiFilterValuePlaceholder] = toAttributeValue(filter.Value)

	default:

		return "", nil, nil
	}

	if skCondition == "" {
		return "", nil, nil
	}

	return pkCondition + " AND " + skCondition, attrNames, attrValues
}
