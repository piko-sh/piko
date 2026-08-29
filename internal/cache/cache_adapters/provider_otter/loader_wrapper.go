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

package provider_otter

import (
	"context"
	"errors"
	"fmt"

	"github.com/maypok86/otter/v2"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// wrappedLoader adapts a caller's Loader to the semantics otter expects.
type wrappedLoader[K comparable, V any] struct {
	// inner is the caller's loader.
	inner cache_dto.Loader[K, V]

	// adapter owns the search index the loaded value must be added to.
	adapter *OtterAdapter[K, V]
}

// Load fetches a value and translates the outcome for otter.
//
// Takes key (K) which identifies the value to load.
//
// Returns V which is the loaded value, present even when the value is not stored.
// Returns error which is nil, a wrapped otter.ErrNotFound, or the loader's own failure.
func (w wrappedLoader[K, V]) Load(ctx context.Context, key K) (V, error) {
	value, err := w.inner.Load(ctx, key)

	return w.translate(ctx, key, value, err, false)
}

// Reload refreshes a value and translates the outcome exactly as Load does.
//
// Takes key (K) which identifies the value to reload.
// Takes oldValue (V) which is the currently cached value.
//
// Returns V which is the reloaded value.
// Returns error which is nil, a wrapped otter.ErrNotFound, or the loader's own failure.
func (w wrappedLoader[K, V]) Reload(ctx context.Context, key K, oldValue V) (V, error) {
	value, err := w.inner.Reload(ctx, key, oldValue)

	return w.translate(ctx, key, value, err, true)
}

// translate converts a loader outcome into the form otter acts on, and indexes an
// admitted value.
//
// Takes key (K) which identifies the entry.
// Takes value (V) which is what the loader produced.
// Takes err (error) which is what the loader reported.
// Takes isReload (bool) which reports that an entry already exists for this key, so a
// refusal must leave it alone instead of deleting it.
//
// Returns V which is the value to hand back.
// Returns error which is the translated error.
func (w wrappedLoader[K, V]) translate(ctx context.Context, key K, value V, err error, isReload bool) (V, error) {
	switch {
	case err == nil:
		return w.admit(ctx, key, value, isReload)
	case errors.Is(err, cache_dto.ErrDoNotStore):
		return value, fmt.Errorf("%w: %w", otter.ErrNotFound, cache_dto.ErrDoNotStore)
	case errors.Is(err, cache_dto.ErrNotFound):
		var zero V

		return zero, fmt.Errorf("%w: %w", otter.ErrNotFound, cache_dto.ErrNotFound)
	default:
		return value, err
	}
}

// admit applies the per-entry ceiling to a freshly loaded value.
//
// Takes key (K) which identifies the entry.
// Takes value (V) which is the loaded value.
// Takes isReload (bool) which reports that an entry already exists for this key.
//
// Returns V which is the loaded value, handed back whether or not it was stored.
// Returns error which is the translated refusal, or nil when the value was admitted.
func (w wrappedLoader[K, V]) admit(ctx context.Context, key K, value V, isReload bool) (V, error) {
	if !w.adapter.exceedsEntryCeiling(key, value) {
		w.adapter.indexDocument(key, value)

		return value, nil
	}

	_, l := logger_domain.From(ctx, log)
	l.Warn("Loaded value refused: it exceeds the cache's MaxEntryWeight",
		logger_domain.Bool("reload", isReload))

	w.adapter.reportRejection(key, value)

	if isReload {
		return value, fmt.Errorf("%w: refusing to replace the cached entry", cache_domain.ErrEntryTooLarge)
	}

	return value, fmt.Errorf("%w: %w", otter.ErrNotFound, cache_domain.ErrEntryTooLarge)
}

// exceedsEntryCeiling reports whether a loaded value is too heavy to admit.
//
// The read-through path never passes through Set, so the domain's admission gate cannot
// see it. Without this check a ceiling would bind on explicit writes and silently not on
// loads, which is the harder case to notice.
//
// Takes key (K) which identifies the entry.
// Takes value (V) which is the loaded value.
//
// Returns bool which is true when the value is above the ceiling.
func (a *OtterAdapter[K, V]) exceedsEntryCeiling(key K, value V) bool {
	if a.maxEntryWeight == 0 || a.weigher == nil {
		return false
	}

	return a.weigher(key, value) > a.maxEntryWeight
}

// reportRejection notifies the deletion callbacks that a value was refused admission, so
// the read path and the write path report a refusal the same way.
//
// Takes key (K) which identifies the refused entry.
// Takes value (V) which is the refused value.
func (a *OtterAdapter[K, V]) reportRejection(key K, value V) {
	event := cache_dto.DeletionEvent[K, V]{Key: key, Value: value, Cause: cache_dto.CauseRejected}

	if a.onDeletion != nil {
		a.onDeletion(event)
	}
}

// wrappedBulkLoader applies the same admission and indexing rules to a bulk load that
// wrappedLoader applies to a single one.
//
// Without it the bulk paths keep the defects the single-key path was fixed for: entries
// arriving through BulkGet are absent from the search index, and the per-entry ceiling
// does not bind on them.
type wrappedBulkLoader[K comparable, V any] struct {
	// inner is the caller's bulk loader.
	inner cache_dto.BulkLoader[K, V]

	// adapter owns the search index and the ceiling the loaded values are checked against.
	adapter *OtterAdapter[K, V]
}

// BulkLoad fetches many values and admits the ones that fit under the ceiling.
//
// Takes keys ([]K) which are the keys to load.
//
// Returns map[K]V which contains only the admitted pairs.
// Returns error which is the loader's own failure.
func (w wrappedBulkLoader[K, V]) BulkLoad(ctx context.Context, keys []K) (map[K]V, error) {
	loaded, err := w.inner.BulkLoad(ctx, keys)
	if err != nil {
		return nil, err
	}

	return w.admitAll(ctx, loaded), nil
}

// BulkReload reloads many values and admits the ones that fit under the ceiling.
//
// Takes keys ([]K) which are the keys to reload.
// Takes oldValues ([]V) which are the cached values being replaced.
//
// Returns map[K]V which contains only the admitted pairs.
// Returns error which is the loader's own failure.
func (w wrappedBulkLoader[K, V]) BulkReload(ctx context.Context, keys []K, oldValues []V) (map[K]V, error) {
	loaded, err := w.inner.BulkReload(ctx, keys, oldValues)
	if err != nil {
		return nil, err
	}

	return w.admitAll(ctx, loaded), nil
}

// admitAll indexes every value that fits under the ceiling and drops the rest.
//
// A refused key is omitted from the returned map, which is how a bulk loader declines a
// key, so the cache stores nothing for it.
//
// Takes loaded (map[K]V) which are the pairs the loader produced.
//
// Returns map[K]V which contains the admitted pairs.
func (w wrappedBulkLoader[K, V]) admitAll(ctx context.Context, loaded map[K]V) map[K]V {
	admitted := make(map[K]V, len(loaded))

	for key, value := range loaded {
		if w.adapter.exceedsEntryCeiling(key, value) {
			_, l := logger_domain.From(ctx, log)
			l.Warn("Bulk loaded value refused: it exceeds the cache's MaxEntryWeight")
			w.adapter.reportRejection(key, value)

			continue
		}

		w.adapter.indexDocument(key, value)
		admitted[key] = value
	}

	return admitted
}

// wrapBulkLoader adapts a caller's bulk loader, or returns nil when there is none.
//
// Takes bulkLoader (cache_dto.BulkLoader[K, V]) which is the caller's bulk loader.
//
// Returns cache_dto.BulkLoader[K, V] which is the adapted loader.
func (a *OtterAdapter[K, V]) wrapBulkLoader(bulkLoader cache_dto.BulkLoader[K, V]) cache_dto.BulkLoader[K, V] {
	if bulkLoader == nil {
		return nil
	}

	return wrappedBulkLoader[K, V]{inner: bulkLoader, adapter: a}
}

// wrapLoader adapts a caller's loader, or returns nil when there is none.
//
// Takes loader (cache_dto.Loader[K, V]) which is the caller's loader.
//
// Returns cache_dto.Loader[K, V] which is the adapted loader.
func (a *OtterAdapter[K, V]) wrapLoader(loader cache_dto.Loader[K, V]) cache_dto.Loader[K, V] {
	if loader == nil {
		return nil
	}

	return wrappedLoader[K, V]{inner: loader, adapter: a}
}

// unwrapLoadOutcome converts a translated load result back into the caller's terms.
//
// Takes value (V) which is what the load produced.
// Takes err (error) which is the translated error.
//
// Returns V which is the value to return to the caller.
// Returns error which is nil for a declined value, ErrNotFound for an absence, or the
// loader's own failure.
func unwrapLoadOutcome[V any](value V, err error) (V, error) {
	switch {
	case err == nil:
		return value, nil
	case errors.Is(err, cache_dto.ErrDoNotStore), errors.Is(err, cache_domain.ErrEntryTooLarge):
		return value, nil
	case errors.Is(err, cache_dto.ErrNotFound):
		var zero V

		return zero, cache_dto.ErrNotFound
	default:
		return value, err
	}
}
