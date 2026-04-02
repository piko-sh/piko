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

package registry_domain

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

// GetArtefact retrieves a single artefact by its ID.
//
// Uses cache and singleflight to reduce repeated lookups.
//
// Takes artefactID (string) which identifies the artefact to retrieve.
//
// Returns *registry_dto.ArtefactMeta which contains the artefact metadata.
// Returns error when the artefact cannot be found or loaded.
func (s *registryService) GetArtefact(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	startTime := time.Now()
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, _ := l.Span(ctx, "RegistryService.GetArtefact",
		logger_domain.String(logKeyArtefactID, artefactID),
	)
	defer func() {
		span.End()
		registryServiceGetArtefactDuration.Record(ctx, float64(time.Since(startTime).Milliseconds()))
	}()

	if cached, found := s.tryGetFromCache(ctx, artefactID); found {
		return cached, nil
	}

	return s.loadArtefactFromStore(ctx, span, artefactID)
}

// tryGetFromCache attempts to retrieve an artefact from cache and records
// metrics.
//
// Takes artefactID (string) which identifies the artefact to retrieve.
//
// Returns *registry_dto.ArtefactMeta which is the cached artefact metadata.
// Returns bool which indicates whether the artefact was found in cache.
func (s *registryService) tryGetFromCache(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, bool) {
	if s.cache == nil {
		registryServiceCacheMissCount.Add(ctx, 1)
		return nil, false
	}

	cachedArt, err := s.cache.Get(ctx, artefactID)
	if err == nil {
		registryServiceCacheHitCount.Add(ctx, 1)
		return cachedArt, true
	}

	if !errors.Is(err, ErrCacheMiss) {
		registryServiceCacheMissCount.Add(ctx, 1)
		return nil, false
	}

	registryServiceCacheMissCount.Add(ctx, 1)
	return nil, false
}

// loadArtefactFromStore loads an artefact from the metadata store using
// singleflight to prevent duplicate requests for the same artefact.
//
// Takes span (trace.Span) which records tracing data for the operation.
// Takes artefactID (string) which identifies the artefact to load.
//
// Returns *registry_dto.ArtefactMeta which contains the loaded artefact data.
// Returns error when the artefact cannot be found or the database fails.
func (s *registryService) loadArtefactFromStore(
	ctx context.Context,
	span trace.Span,
	artefactID string,
) (*registry_dto.ArtefactMeta, error) {
	ctx, l := logger_domain.From(ctx, log)
	sfKey := "artefact:" + artefactID

	result, err, _ := s.loader.Do(sfKey, func() (any, error) {
		l.Trace("Retrieving artefact from metadata store")
		persistentArt, dbErr := s.metaStore.GetArtefact(ctx, artefactID)
		if dbErr != nil {
			s.logArtefactLoadError(ctx, span, dbErr, artefactID)
			return nil, dbErr
		}

		if s.cache != nil {
			s.cache.Set(ctx, persistentArt)
		}
		return persistentArt, nil
	})

	if err != nil {
		return nil, fmt.Errorf("loading artefact %q from store: %w", artefactID, err)
	}

	artefact, ok := result.(*registry_dto.ArtefactMeta)
	if !ok {
		return nil, fmt.Errorf("unexpected type from singleflight loader: %T", result)
	}

	l.Trace("Artefact retrieved successfully",
		logger_domain.Int("variantCount", len(artefact.ActualVariants)),
		logger_domain.Int("profileCount", len(artefact.DesiredProfiles)))
	return artefact, nil
}

// logArtefactLoadError logs the right error message when an artefact fails to
// load.
//
// Takes span (trace.Span) which provides the tracing context for error
// reporting.
// Takes err (error) which is the error that happened during loading.
// Takes artefactID (string) which identifies the artefact that failed to load.
func (*registryService) logArtefactLoadError(ctx context.Context, span trace.Span, err error, artefactID string) {
	ctx, l := logger_domain.From(ctx, log)
	if errors.Is(err, ErrArtefactNotFound) {
		l.Trace("Artefact not found", logger_domain.String(logKeyArtefactID, artefactID))
	} else {
		l.ReportError(span, err, "Failed to retrieve artefact")
	}
}

// GetMultipleArtefacts retrieves multiple artefacts by IDs, using cache for
// hits and batch-fetching misses.
//
// Takes artefactIDs ([]string) which specifies the artefact IDs to retrieve.
//
// Returns []*registry_dto.ArtefactMeta which contains the retrieved artefacts
// in the same order as the input IDs.
// Returns error when loading from cache or store fails.
func (s *registryService) GetMultipleArtefacts(ctx context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error) {
	startTime := time.Now()
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "RegistryService.GetMultipleArtefacts",
		logger_domain.Int("requestedCount", len(artefactIDs)),
	)
	defer func() {
		span.End()
		registryServiceGetMultipleArtefactsDuration.Record(ctx, float64(time.Since(startTime).Milliseconds()))
	}()

	if len(artefactIDs) == 0 {
		l.Warn("Empty artefact ID list provided")
		return []*registry_dto.ArtefactMeta{}, nil
	}

	if s.cache == nil {
		return s.loadMultipleFromStoreWithoutCache(ctx, span, artefactIDs)
	}

	hits, err := s.loadMultipleWithCache(ctx, span, artefactIDs)
	if err != nil {
		return nil, fmt.Errorf("loading multiple artefacts with cache: %w", err)
	}

	orderedResults := orderArtefactsByIDs(hits, artefactIDs)

	l.Trace("Retrieved artefacts successfully",
		logger_domain.Int("foundCount", len(orderedResults)),
		logger_domain.Int("requestedCount", len(artefactIDs)))
	return orderedResults, nil
}

// loadMultipleFromStoreWithoutCache loads artefacts straight from the store
// when no cache is set up.
//
// Takes span (trace.Span) which provides tracing context.
// Takes artefactIDs ([]string) which lists the artefacts to fetch.
//
// Returns []*registry_dto.ArtefactMeta which contains the fetched artefacts.
// Returns error when the metadata store cannot fetch the artefacts.
func (s *registryService) loadMultipleFromStoreWithoutCache(
	ctx context.Context,
	span trace.Span,
	artefactIDs []string,
) ([]*registry_dto.ArtefactMeta, error) {
	ctx, l := logger_domain.From(ctx, log)
	registryServiceCacheMissCount.Add(ctx, int64(len(artefactIDs)))
	l.Trace("Retrieving multiple artefacts from metadata store")

	artefacts, err := s.metaStore.GetMultipleArtefacts(ctx, artefactIDs)
	if err != nil {
		l.ReportError(span, err, "Failed to retrieve multiple artefacts")
		return nil, fmt.Errorf("retrieving multiple artefacts from store: %w", err)
	}
	return artefacts, nil
}

// loadMultipleWithCache loads artefacts using cache hits and fetches misses
// from the store.
//
// Takes span (trace.Span) which provides tracing context for the operation.
// Takes artefactIDs ([]string) which lists the artefact IDs to load.
//
// Returns []*registry_dto.ArtefactMeta which contains the loaded artefacts.
// Returns error when fetching missing artefacts from the store fails.
func (s *registryService) loadMultipleWithCache(
	ctx context.Context,
	span trace.Span,
	artefactIDs []string,
) ([]*registry_dto.ArtefactMeta, error) {
	hits, misses := s.cache.GetMultiple(ctx, artefactIDs)

	registryServiceCacheHitCount.Add(ctx, int64(len(hits)))
	registryServiceCacheMissCount.Add(ctx, int64(len(misses)))

	if len(misses) == 0 {
		return hits, nil
	}

	fetchedArts, err := s.fetchMissingArtefacts(ctx, span, misses)
	if err != nil {
		return nil, fmt.Errorf("fetching missing artefacts: %w", err)
	}

	return append(hits, fetchedArts...), nil
}

// fetchMissingArtefacts fetches artefacts not found in cache from the store
// via singleflight.
//
// Takes span (trace.Span) which receives tracing events for the operation.
// Takes misses ([]string) which contains the artefact identifiers to fetch.
//
// Returns []*registry_dto.ArtefactMeta which contains the fetched artefacts.
// Returns error when the store query fails or returns an unexpected type.
func (s *registryService) fetchMissingArtefacts(
	ctx context.Context,
	span trace.Span,
	misses []string,
) ([]*registry_dto.ArtefactMeta, error) {
	ctx, l := logger_domain.From(ctx, log)
	uniqueMisses := deduplicateAndSort(misses)
	sfKey := "artefacts-batch:" + strings.Join(uniqueMisses, ",")

	v, err, _ := s.loader.Do(sfKey, func() (any, error) {
		l.Trace("Singleflight: Performing batch DB query for artefacts",
			logger_domain.Int("missCount", len(uniqueMisses)))
		span.AddEvent("Executing batch DB query via singleflight")

		persistentArts, dbErr := s.metaStore.GetMultipleArtefacts(ctx, uniqueMisses)
		if dbErr != nil {
			return nil, dbErr
		}

		s.cache.SetMultiple(ctx, persistentArts)
		return persistentArts, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load missing artefacts from store: %w", err)
	}

	fetchedArts, ok := v.([]*registry_dto.ArtefactMeta)
	if !ok {
		return nil, fmt.Errorf("unexpected type from singleflight loader: %T", v)
	}
	return fetchedArts, nil
}

// ListAllArtefactIDs returns a list of all artefact IDs in the registry.
//
// Returns []string which contains all known artefact IDs.
// Returns error when the metadata store cannot be read.
func (s *registryService) ListAllArtefactIDs(ctx context.Context) ([]string, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "RegistryService.ListAllArtefactIDs")
	defer span.End()

	l.Trace("Retrieving all artefact IDs from metadata store")
	ids, err := s.metaStore.ListAllArtefactIDs(ctx)
	if err != nil {
		l.ReportError(span, err, "Failed to list artefact IDs")
		return nil, fmt.Errorf("listing all artefact IDs: %w", err)
	}

	l.Trace("Retrieved artefact IDs successfully", logger_domain.Int("count", len(ids)))
	return ids, nil
}

// SearchArtefacts searches for artefacts using a query.
//
// Takes query (SearchQuery) which specifies the search criteria.
//
// Returns []*registry_dto.ArtefactMeta which contains the matching artefacts.
// Returns error when the search fails.
func (s *registryService) SearchArtefacts(ctx context.Context, query SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "RegistryService.SearchArtefacts")
	defer span.End()

	s.logSearchQuery(ctx, query)

	artefacts, err := s.metaStore.SearchArtefacts(ctx, query)
	if err != nil {
		l.ReportError(span, err, "Search failed")
		return nil, fmt.Errorf("searching artefacts: %w", err)
	}

	if s.cache != nil && len(artefacts) > 0 {
		s.cache.SetMultiple(ctx, artefacts)
	}

	l.Trace("Search completed successfully", logger_domain.Int("resultCount", len(artefacts)))
	return artefacts, nil
}

// logSearchQuery logs debug information about the search query type.
//
// Takes query (SearchQuery) which holds the search settings to log.
func (*registryService) logSearchQuery(ctx context.Context, query SearchQuery) {
	ctx, l := logger_domain.From(ctx, log)
	if query.RawRediSearchQuery != "" {
		l.Trace("Performing raw search query", logger_domain.String("rawQuery", query.RawRediSearchQuery))
	} else if len(query.SimpleTagQuery) > 0 {
		l.Trace("Performing simple tag search", logger_domain.Int("tagCount", len(query.SimpleTagQuery)))
	} else {
		l.Warn("Empty search query")
	}
}

// SearchArtefactsByTagValues finds artefacts that match a tag key and any of
// the given tag values.
//
// Takes tagKey (string) which specifies the metadata tag to search by.
// Takes tagValues ([]string) which contains the values to match against.
//
// Returns []*registry_dto.ArtefactMeta which contains the matching artefacts.
// Returns error when the metadata store search fails.
func (s *registryService) SearchArtefactsByTagValues(ctx context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "RegistryService.SearchArtefactsByTagValues",
		logger_domain.String("tagKey", tagKey),
		logger_domain.Int("valueCount", len(tagValues)),
	)
	defer span.End()

	if len(tagValues) == 0 {
		return []*registry_dto.ArtefactMeta{}, nil
	}

	l.Trace("Searching for artefacts by tag values in metadata store")
	artefacts, err := s.metaStore.SearchArtefactsByTagValues(ctx, tagKey, tagValues)
	if err != nil {
		l.ReportError(span, err, "Failed to search artefacts by tag values")
		return nil, fmt.Errorf("searching artefacts by tag %q: %w", tagKey, err)
	}

	if s.cache != nil && len(artefacts) > 0 {
		s.cache.SetMultiple(ctx, artefacts)
	}

	l.Trace("Search by tag values completed successfully", logger_domain.Int("resultCount", len(artefacts)))
	return artefacts, nil
}

// FindArtefactByVariantStorageKey finds an artefact by one of its variant's
// storage key.
//
// Takes storageKey (string) which identifies the variant to look up.
//
// Returns *registry_dto.ArtefactMeta which contains the artefact metadata.
// Returns error when the artefact cannot be found or retrieval fails.
func (s *registryService) FindArtefactByVariantStorageKey(ctx context.Context, storageKey string) (*registry_dto.ArtefactMeta, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, _ := l.Span(ctx, "RegistryService.FindArtefactByVariantStorageKey",
		logger_domain.String(logKeyStorageKey, storageKey),
	)
	defer span.End()

	return s.loadArtefactByStorageKey(ctx, span, storageKey)
}

// loadArtefactByStorageKey loads an artefact by its storage key via
// singleflight.
//
// Takes span (trace.Span) which records tracing information for the lookup.
// Takes storageKey (string) which identifies the artefact variant in storage.
//
// Returns *registry_dto.ArtefactMeta which contains the artefact metadata.
// Returns error when the lookup fails or the singleflight returns an
// unexpected type.
func (s *registryService) loadArtefactByStorageKey(
	ctx context.Context,
	span trace.Span,
	storageKey string,
) (*registry_dto.ArtefactMeta, error) {
	ctx, l := logger_domain.From(ctx, log)
	sfKey := "storagekey:" + storageKey

	result, err, _ := s.loader.Do(sfKey, func() (any, error) {
		l.Trace("Looking up artefact by variant storage key in metastore")
		artefact, lookupErr := s.metaStore.FindArtefactByVariantStorageKey(ctx, storageKey)
		if lookupErr != nil {
			s.logStorageKeyLookupError(ctx, span, lookupErr)
			return nil, lookupErr
		}

		if s.cache != nil && artefact != nil {
			s.cache.Set(ctx, artefact)
		}
		return artefact, nil
	})

	if err != nil {
		return nil, fmt.Errorf("loading artefact by storage key %q: %w", storageKey, err)
	}

	artefact, ok := result.(*registry_dto.ArtefactMeta)
	if !ok {
		return nil, fmt.Errorf("unexpected type from singleflight loader: %T", result)
	}

	l.Trace("Found artefact by storage key",
		logger_domain.String(logKeyArtefactID, artefact.ID),
		logger_domain.Int("variantCount", len(artefact.ActualVariants)))
	return artefact, nil
}

// logStorageKeyLookupError logs an error that occurred during storage key
// lookup.
//
// Takes span (trace.Span) which provides the tracing context for error
// reporting.
// Takes err (error) which is the error to log.
func (*registryService) logStorageKeyLookupError(ctx context.Context, span trace.Span, err error) {
	ctx, l := logger_domain.From(ctx, log)
	if errors.Is(err, ErrArtefactNotFound) {
		l.Trace("No artefact found with the given storage key")
	} else {
		l.ReportError(span, err, "Failed to find artefact by storage key")
	}
}

// deduplicateAndSort removes duplicate strings from a slice and returns the
// unique values in sorted order. Uses a pooled map to reduce memory allocation.
//
// Takes items ([]string) which contains the strings to process.
//
// Returns []string which contains the unique values in sorted order.
func deduplicateAndSort(items []string) []string {
	seen := getDedupeMap()
	defer putDedupeMap(seen)

	unique := make([]string, 0, len(items))

	for _, item := range items {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{}
			unique = append(unique, item)
		}
	}

	slices.Sort(unique)
	return unique
}

// orderArtefactsByIDs orders artefacts to match the original ID request order.
// Uses a pooled map to reduce allocations.
//
// Takes artefacts ([]*registry_dto.ArtefactMeta) which are the artefacts to
// reorder.
// Takes requestedIDs ([]string) which specifies the desired order by ID.
//
// Returns []*registry_dto.ArtefactMeta which contains only the artefacts that
// match the requested IDs, in the order specified.
func orderArtefactsByIDs(artefacts []*registry_dto.ArtefactMeta, requestedIDs []string) []*registry_dto.ArtefactMeta {
	resultsMap := getOrderingMap()
	defer putOrderingMap(resultsMap)

	for _, art := range artefacts {
		resultsMap[art.ID] = art
	}

	ordered := make([]*registry_dto.ArtefactMeta, 0, len(requestedIDs))
	for _, id := range requestedIDs {
		if art, ok := resultsMap[id].(*registry_dto.ArtefactMeta); ok {
			ordered = append(ordered, art)
		}
	}
	return ordered
}
