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

package cache_provider_firestore

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/wdk/cache"
)

// SupportsSearch returns true if a search schema is configured for this
// Firestore adapter.
//
// Returns bool which is true when search operations are available.
func (a *FirestoreAdapter[K, V]) SupportsSearch() bool {
	return a.schema != nil
}

// GetSchema returns the search schema for this cache.
//
// Returns *cache.SearchSchema which describes searchable fields, or nil when
// search is not configured.
func (a *FirestoreAdapter[K, V]) GetSchema() *cache.SearchSchema {
	return a.schema
}

// Query performs structured filtering, sorting, and pagination using native
// Firestore server-side queries. Filter conditions are translated to Firestore
// Where clauses, and sorting uses Firestore OrderBy.
//
// Takes opts (*cache.QueryOptions) which specifies filters, sorting, and
// pagination.
//
// Returns cache.SearchResult[K, V] which contains matched entries.
// Returns error when no schema is configured (ErrSearchNotSupported), when a
// vector query is requested (ErrSearchNotSupported), or when the Firestore
// query fails.
func (a *FirestoreAdapter[K, V]) Query(ctx context.Context, opts *cache.QueryOptions) (cache.SearchResult[K, V], error) {
	if a.schema == nil {
		return cache.SearchResult[K, V]{}, fmt.Errorf(
			"%w: Firestore query requires a search schema",
			cache.ErrSearchNotSupported,
		)
	}

	if len(opts.Vector) > 0 {
		return cache.SearchResult[K, V]{}, fmt.Errorf(
			"%w: Firestore provider does not support vector queries",
			cache.ErrSearchNotSupported,
		)
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.searchTimeout,
		fmt.Errorf("firestore Query exceeded %s timeout", a.searchTimeout))
	defer cancel()

	query := a.buildFirestoreQuery(opts.Filters, opts.SortBy, opts.SortOrder, opts.Limit, opts.Offset)

	items, total, err := a.executeQuery(timeoutCtx, &query)
	if err != nil {
		return cache.SearchResult[K, V]{}, fmt.Errorf("executing Firestore query: %w", err)
	}

	return cache.SearchResult[K, V]{
		Items:  items,
		Total:  total,
		Offset: opts.Offset,
		Limit:  resolveLimit(opts.Limit),
	}, nil
}

// Search returns ErrSearchNotSupported because the Firestore provider delegates
// structured queries to Query and does not support client-side full-text or
// vector search.
//
// Takes query (string) which is the search query text (unused).
// Takes opts (*cache.SearchOptions) which configures the search (unused).
//
// Returns cache.SearchResult[K, V] which is always empty.
// Returns error which is always ErrSearchNotSupported.
func (*FirestoreAdapter[K, V]) Search(_ context.Context, _ string, _ *cache.SearchOptions) (cache.SearchResult[K, V], error) {
	return cache.SearchResult[K, V]{}, fmt.Errorf(
		"%w: Firestore provider does not support text search; use Query for structured filtering",
		cache.ErrSearchNotSupported,
	)
}

// buildFirestoreQuery constructs a Firestore query from filter options, sort
// settings, and pagination parameters. Filters are mapped to native Firestore
// Where clauses using the "sf_" field prefix.
//
// Takes filters ([]cache.Filter) which specifies conditions.
// Takes sortBy (string) which specifies the field to sort by.
// Takes sortOrder (cache.SortOrder) which specifies the sort direction.
// Takes limit (int) which is the maximum number of results.
// Takes offset (int) which is the pagination offset.
//
// Returns firestore.Query which is the constructed query chain.
func (a *FirestoreAdapter[K, V]) buildFirestoreQuery(
	filters []cache.Filter,
	sortBy string,
	sortOrder cache.SortOrder,
	limit, offset int,
) firestore.Query {
	query := a.collection.Query

	for _, filter := range filters {
		fieldName := searchFieldPrefix + filter.Field
		query = *applyFilterToQuery(&query, fieldName, filter)
	}

	if sortBy != "" {
		direction := firestore.Asc
		if sortOrder == cache.SortDesc {
			direction = firestore.Desc
		}
		query = query.OrderBy(searchFieldPrefix+sortBy, direction)
	}

	resolvedLimit := resolveLimit(limit)
	query = query.Limit(resolvedLimit)

	if offset > 0 {
		query = query.Offset(offset)
	}

	return query
}

// applyFilterToQuery translates a single cache filter to Firestore Where
// clause(s) and appends them to the query.
//
// Takes query (*firestore.Query) which is the query to extend.
// Takes fieldName (string) which is the prefixed Firestore field name.
// Takes filter (cache.Filter) which specifies the filter condition.
//
// Returns *firestore.Query with the filter applied.
func applyFilterToQuery(query *firestore.Query, fieldName string, filter cache.Filter) *firestore.Query {
	var result firestore.Query
	switch filter.Operation {
	case cache.FilterOpEq:
		result = query.Where(fieldName, "==", filter.Value)
	case cache.FilterOpNe:
		result = query.Where(fieldName, "!=", filter.Value)
	case cache.FilterOpGt:
		result = query.Where(fieldName, ">", filter.Value)
	case cache.FilterOpGe:
		result = query.Where(fieldName, ">=", filter.Value)
	case cache.FilterOpLt:
		result = query.Where(fieldName, "<", filter.Value)
	case cache.FilterOpLe:
		result = query.Where(fieldName, "<=", filter.Value)
	case cache.FilterOpIn:
		result = query.Where(fieldName, "in", filter.Values)
	case cache.FilterOpBetween:
		if len(filter.Values) != 2 {
			return query
		}
		lower := query.Where(fieldName, ">=", filter.Values[0])
		result = lower.Where(fieldName, "<=", filter.Values[1])
	case cache.FilterOpPrefix:
		prefix := cache_domain.ToString(filter.Value)
		lower := query.Where(fieldName, ">=", prefix)
		result = lower.Where(fieldName, "<", prefix+"\uffff")
	default:
		return query
	}
	return &result
}

// executeQuery runs a Firestore query, iterates the results, and decodes each
// document into a SearchHit.
//
// Takes query (*firestore.Query) which is the Firestore query to run.
//
// Returns []cache.SearchHit[K, V] which contains the decoded results.
// Returns int64 which is the total number of results.
// Returns error when iteration or decoding fails.
func (a *FirestoreAdapter[K, V]) executeQuery(ctx context.Context, query *firestore.Query) ([]cache.SearchHit[K, V], int64, error) {
	iter := query.Documents(ctx)
	defer iter.Stop()

	var items []cache.SearchHit[K, V]

	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("firestore query iteration failed: %w", err)
		}

		key, value, ok := a.decodeDocument(ctx, snap)
		if !ok {
			continue
		}

		items = append(items, cache.SearchHit[K, V]{
			Key:   key,
			Value: value,
			Score: 1.0,
		})
	}

	return items, int64(len(items)), nil
}

// resolveLimit returns the requested limit or the default search limit.
//
// Takes limit (int) which is the caller-supplied limit.
//
// Returns int which is the resolved limit.
func resolveLimit(limit int) int {
	if limit <= 0 {
		return cache_domain.DefaultSearchLimit
	}
	return limit
}
