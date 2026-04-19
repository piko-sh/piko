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

package union

import (
	"cmp"
	"context"
	"errors"
	"maps"
	"slices"

	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

// Base is the read-only layer's contract: it answers reads and inspector queries, and
// reports what it owns, so the union can serve base content without an overlay round
// trip, keep the overlay a delta, and show base artefacts in the monitoring TUI.
type Base interface {
	registry_domain.MetadataStore

	registry_domain.RegistryInspector

	// OwnsArtefact reports whether the base holds an artefact with the given ID.
	OwnsArtefact(artefactID string) bool

	// OwnsStorageKey reports whether the base owns a variant or chunk stored at the key.
	OwnsStorageKey(storageKey string) bool

	// ArtefactIDs returns every ID the base holds.
	ArtefactIDs() []string
}

// Store presents a base layer and a writable overlay as one MetadataStore.
type Store struct {
	// base is the read-only layer, or nil when the binary ships no base.
	base Base

	// overlay is the writable layer, or nil for a read-only deployment.
	overlay registry_domain.MetadataStore
}

// compile-time proof that Store satisfies the store, inspector and locking contracts.
var (
	_ registry_domain.MetadataStore = (*Store)(nil)

	_ registry_domain.RegistryInspector = (*Store)(nil)

	_ registry_domain.ArtefactLocker = (*Store)(nil)
)

// New composes a base and an overlay into one store.
//
// With base nil it returns the overlay untouched, so a non-embedded deployment pays
// nothing for the union. With overlay nil the store is read-only and every write returns
// ErrReadOnlyStore. At least one layer must be present.
//
// Takes base (Base) which is the read-only layer, or nil.
// Takes overlay (registry_domain.MetadataStore) which is the writable layer, or nil.
//
// Returns registry_domain.MetadataStore which is the composed store.
func New(base Base, overlay registry_domain.MetadataStore) registry_domain.MetadataStore {
	if base == nil {
		return overlay
	}
	return &Store{base: base, overlay: overlay}
}

// GetArtefact merges the base and overlay records for the ID.
//
// It fetches the artefact from each layer that has it and unions their variants with
// MergeLayers, so a caller sees build variants from the base and runtime variants from
// the overlay as one artefact. An overlay read error other than not-found is returned; a
// base hit with an overlay miss returns the base alone.
//
// Takes artefactID (string) which is the ID to fetch.
//
// Returns *registry_dto.ArtefactMeta which is the merged artefact.
// Returns error which is ErrArtefactNotFound when neither layer has it.
func (s *Store) GetArtefact(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	baseArtefact, err := s.baseArtefact(ctx, artefactID)
	if err != nil {
		return nil, err
	}

	var overlayArtefact *registry_dto.ArtefactMeta
	if s.overlay != nil {
		got, overlayErr := s.overlay.GetArtefact(ctx, artefactID)
		if overlayErr != nil && !isNotFound(overlayErr) {
			return nil, overlayErr
		}
		overlayArtefact = got
	}

	merged := registry_dto.MergeLayers(baseArtefact, overlayLayers(overlayArtefact))
	if merged == nil {
		return nil, registry_domain.ErrArtefactNotFound
	}
	return merged, nil
}

// GetArtefactForUpdate merges the base and overlay records for the ID, taking the
// overlay's row lock so a read-modify-write serialises against concurrent writers.
//
// Takes artefactID (string) which is the ID to fetch.
//
// Returns *registry_dto.ArtefactMeta which is the merged artefact.
// Returns error which is ErrArtefactNotFound when neither layer has it.
func (s *Store) GetArtefactForUpdate(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	baseArtefact, err := s.baseArtefact(ctx, artefactID)
	if err != nil {
		return nil, err
	}

	var overlayArtefact *registry_dto.ArtefactMeta
	if s.overlay != nil {
		got, overlayErr := registry_domain.ReadArtefactForLockedUpdate(ctx, s.overlay, artefactID)
		if overlayErr != nil && !isNotFound(overlayErr) {
			return nil, overlayErr
		}
		overlayArtefact = got
	}

	merged := registry_dto.MergeLayers(baseArtefact, overlayLayers(overlayArtefact))
	if merged == nil {
		return nil, registry_domain.ErrArtefactNotFound
	}
	return merged, nil
}

// GetMultipleArtefacts merges each requested artefact across layers.
//
// It fetches all IDs from each layer in a single batched call, then merges per ID, so the
// cost is two store reads rather than one per ID. Requested IDs neither layer has are
// omitted, and a repeated ID yields at most one result.
//
// Takes artefactIDs ([]string) which are the IDs to fetch.
//
// Returns []*registry_dto.ArtefactMeta which are the merged, present artefacts.
// Returns error when a layer read fails.
func (s *Store) GetMultipleArtefacts(ctx context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error) {
	baseArtefacts, err := s.base.GetMultipleArtefacts(ctx, artefactIDs)
	if err != nil {
		return nil, err
	}
	baseByID := indexArtefactsByID(baseArtefacts)

	var overlayByID map[string]*registry_dto.ArtefactMeta
	if s.overlay != nil {
		overlayArtefacts, overlayErr := s.overlay.GetMultipleArtefacts(ctx, artefactIDs)
		if overlayErr != nil {
			return nil, overlayErr
		}
		overlayByID = indexArtefactsByID(overlayArtefacts)
	}

	results := make([]*registry_dto.ArtefactMeta, 0, len(artefactIDs))
	emitted := make(map[string]struct{}, len(artefactIDs))
	for _, id := range artefactIDs {
		if _, done := emitted[id]; done {
			continue
		}
		merged := registry_dto.MergeLayers(baseByID[id], overlayLayers(overlayByID[id]))
		if merged == nil {
			continue
		}
		emitted[id] = struct{}{}
		results = append(results, merged)
	}
	return results, nil
}

// ListAllArtefactIDs returns the deduplicated, sorted union of both layers' IDs.
//
// Returns []string which is every ID across both layers, once each.
// Returns error when the overlay listing fails.
func (s *Store) ListAllArtefactIDs(ctx context.Context) ([]string, error) {
	seen := make(map[string]struct{})
	for _, id := range s.base.ArtefactIDs() {
		seen[id] = struct{}{}
	}
	if s.overlay != nil {
		overlayIDs, err := s.overlay.ListAllArtefactIDs(ctx)
		if err != nil {
			return nil, err
		}
		for _, id := range overlayIDs {
			seen[id] = struct{}{}
		}
	}

	return slices.Sorted(maps.Keys(seen)), nil
}

// SearchArtefacts merges the search results of both layers by artefact ID.
//
// Takes query (registry_domain.SearchQuery) which describes the filter.
//
// Returns []*registry_dto.ArtefactMeta which match, merged across layers.
// Returns error which is ErrSearchUnsupported only when both layers reject the query.
func (s *Store) SearchArtefacts(ctx context.Context, query registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
	baseResults, baseErr := s.base.SearchArtefacts(ctx, query)
	if baseErr != nil && !isSearchUnsupported(baseErr) {
		return nil, baseErr
	}

	var overlayResults []*registry_dto.ArtefactMeta
	overlayErr := registry_domain.ErrSearchUnsupported
	if s.overlay != nil {
		overlayResults, overlayErr = s.overlay.SearchArtefacts(ctx, query)
		if overlayErr != nil && !isSearchUnsupported(overlayErr) {
			return nil, overlayErr
		}
	}

	if isSearchUnsupported(baseErr) && isSearchUnsupported(overlayErr) {
		return nil, registry_domain.ErrSearchUnsupported
	}
	return s.mergeMatches(ctx, baseResults, overlayResults)
}

// SearchArtefactsByTagValues merges the tag-search results of both layers by artefact ID.
//
// Takes tagKey (string) which is the tag name to match.
// Takes tagValues ([]string) which are the values to match against.
//
// Returns []*registry_dto.ArtefactMeta which carry any value, merged across layers.
// Returns error when a layer search fails.
func (s *Store) SearchArtefactsByTagValues(ctx context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error) {
	baseResults, err := s.base.SearchArtefactsByTagValues(ctx, tagKey, tagValues)
	if err != nil {
		return nil, err
	}
	var overlayResults []*registry_dto.ArtefactMeta
	if s.overlay != nil {
		overlayResults, err = s.overlay.SearchArtefactsByTagValues(ctx, tagKey, tagValues)
		if err != nil {
			return nil, err
		}
	}
	return s.mergeMatches(ctx, baseResults, overlayResults)
}

// FindArtefactByVariantStorageKey resolves a storage key against the base first, then the
// overlay.
//
// A base-owned key resolves to the merged artefact for its owner, so the result carries
// any overlay additions too. Otherwise the overlay is consulted, and its hit is likewise
// returned merged, since a runtime blob may belong to an artefact the base also has.
//
// Takes storageKey (string) which is the key to resolve.
//
// Returns *registry_dto.ArtefactMeta which owns the key, merged.
// Returns error which is ErrArtefactNotFound when no layer owns the key.
func (s *Store) FindArtefactByVariantStorageKey(ctx context.Context, storageKey string) (*registry_dto.ArtefactMeta, error) {
	if s.base.OwnsStorageKey(storageKey) {
		owner, err := s.base.FindArtefactByVariantStorageKey(ctx, storageKey)
		if err != nil {
			return nil, err
		}
		return s.GetArtefact(ctx, owner.ID)
	}
	if s.overlay == nil {
		return nil, registry_domain.ErrArtefactNotFound
	}
	owner, err := s.overlay.FindArtefactByVariantStorageKey(ctx, storageKey)
	if err != nil {
		return nil, err
	}
	return s.GetArtefact(ctx, owner.ID)
}

// PopGCHints reads GC hints from the overlay only: base blobs live in the binary and can
// never be orphaned.
//
// Takes limit (int) which caps the hints returned.
//
// Returns []registry_dto.GCHint which are the overlay's hints, or none when there is no
// overlay.
// Returns error when the overlay read fails.
func (s *Store) PopGCHints(ctx context.Context, limit int) ([]registry_dto.GCHint, error) {
	if s.overlay == nil {
		return nil, nil
	}
	return s.overlay.PopGCHints(ctx, limit)
}

// AtomicUpdate applies a batch to the overlay, stripping base-provided records from each
// upsert so the overlay stays a delta.
//
// Takes actions ([]registry_dto.AtomicAction) which describe the batch.
//
// Returns error which is ErrReadOnlyStore when there is no overlay.
func (s *Store) AtomicUpdate(ctx context.Context, actions []registry_dto.AtomicAction) error {
	if s.overlay == nil {
		return registry_domain.ErrReadOnlyStore
	}
	delta, err := s.deltaActions(ctx, actions)
	if err != nil {
		return err
	}
	return s.overlay.AtomicUpdate(ctx, delta)
}

// IncrementBlobRefCount counts a blob the base owns as always referenced, otherwise
// defers to the overlay.
//
// Takes blob (registry_domain.BlobReference) which identifies the blob.
//
// Returns int which is the reference count.
// Returns error which is ErrReadOnlyStore when a non-base blob is counted with no
// overlay.
func (s *Store) IncrementBlobRefCount(ctx context.Context, blob registry_domain.BlobReference) (int, error) {
	if s.base.OwnsStorageKey(blob.StorageKey) {
		return 1, nil
	}
	if s.overlay == nil {
		return 0, registry_domain.ErrReadOnlyStore
	}
	return s.overlay.IncrementBlobRefCount(ctx, blob)
}

// DecrementBlobRefCount treats a base-owned blob as never deletable, otherwise defers to
// the overlay.
//
// Takes storageKey (string) which identifies the blob.
//
// Returns int which is the reference count.
// Returns bool which is true when the blob should be deleted, never for a base blob.
// Returns error which is ErrReadOnlyStore when a non-base blob is decremented with no
// overlay.
func (s *Store) DecrementBlobRefCount(ctx context.Context, storageKey string) (int, bool, error) {
	if s.base.OwnsStorageKey(storageKey) {
		return 1, false, nil
	}
	if s.overlay == nil {
		return 0, false, registry_domain.ErrReadOnlyStore
	}
	return s.overlay.DecrementBlobRefCount(ctx, storageKey)
}

// GetBlobRefCount reports one for a base-owned blob, otherwise the overlay's count.
//
// Takes storageKey (string) which identifies the blob.
//
// Returns int which is the reference count.
// Returns error when the overlay read fails.
func (s *Store) GetBlobRefCount(ctx context.Context, storageKey string) (int, error) {
	if s.base.OwnsStorageKey(storageKey) {
		return 1, nil
	}
	if s.overlay == nil {
		return 0, nil
	}
	return s.overlay.GetBlobRefCount(ctx, storageKey)
}

// RunAtomic runs fn in an overlay transaction, presenting fn a union over the base and
// the transaction-scoped overlay.
//
// The transaction store fn receives merges the immutable base with the open overlay
// transaction, so a read-modify-write that adds a runtime variant to a base artefact sees
// the merged artefact and does not fail as not-found. The base participates for free
// because it is immutable.
//
// Takes fn (func) which performs the transactional work.
//
// Returns error which is ErrReadOnlyStore when there is no overlay, or fn's error.
func (s *Store) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore registry_domain.MetadataStore) error) error {
	if s.overlay == nil {
		return registry_domain.ErrReadOnlyStore
	}
	return s.overlay.RunAtomic(ctx, func(txCtx context.Context, txOverlay registry_domain.MetadataStore) error {
		return fn(txCtx, &Store{base: s.base, overlay: txOverlay})
	})
}

// Close closes the overlay: the base is in-memory.
//
// Returns error when the overlay close fails.
func (s *Store) Close() error {
	if s.overlay == nil {
		return nil
	}
	return s.overlay.Close()
}

// ListArtefactSummary sums the base and overlay artefact-status counts.
//
// An artefact present in both layers is counted in each, so the histogram can
// double-count a base artefact that also has an overlay delta. The overlay delta
// population is small by design, and an exact count would need an ID-exclusion the
// backends cannot express cheaply.
//
// Returns []registry_domain.ArtefactSummary which is the combined count per status.
// Returns error when either layer's summary fails.
func (s *Store) ListArtefactSummary(ctx context.Context) ([]registry_domain.ArtefactSummary, error) {
	return mergeStatusSummaries(ctx, s.overlayInspector(),
		s.base.ListArtefactSummary,
		registry_domain.RegistryInspector.ListArtefactSummary,
		func(entry registry_domain.ArtefactSummary) string { return entry.Status },
		func(entry registry_domain.ArtefactSummary) int64 { return entry.Count },
		func(status string, count int64) registry_domain.ArtefactSummary {
			return registry_domain.ArtefactSummary{Status: status, Count: count}
		})
}

// ListVariantSummary sums the base and overlay variant-status counts.
//
// The overlay is a strict variant delta, so no variant is counted in both layers and the
// sum is exact.
//
// Returns []registry_domain.VariantSummary which is the combined count per status.
// Returns error when either layer's summary fails.
func (s *Store) ListVariantSummary(ctx context.Context) ([]registry_domain.VariantSummary, error) {
	return mergeStatusSummaries(ctx, s.overlayInspector(),
		s.base.ListVariantSummary,
		registry_domain.RegistryInspector.ListVariantSummary,
		func(entry registry_domain.VariantSummary) string { return entry.Status },
		func(entry registry_domain.VariantSummary) int64 { return entry.Count },
		func(status string, count int64) registry_domain.VariantSummary {
			return registry_domain.VariantSummary{Status: status, Count: count}
		})
}

// ListRecentArtefacts merges the most recent artefacts across layers, newest first.
//
// It asks each layer for its recent artefacts, merges by ID keeping the newest entry,
// sorts by update time and truncates to the limit.
//
// Takes limit (int) which caps the number returned.
//
// Returns []registry_domain.ArtefactListItem which describe the most recent artefacts.
// Returns error when either layer's listing fails.
func (s *Store) ListRecentArtefacts(ctx context.Context, limit int) ([]registry_domain.ArtefactListItem, error) {
	items := make(map[string]registry_domain.ArtefactListItem)
	baseItems, err := s.base.ListRecentArtefacts(ctx, limit)
	if err != nil {
		return nil, err
	}
	for _, item := range baseItems {
		items[item.ID] = item
	}
	if inspector, ok := s.overlay.(registry_domain.RegistryInspector); ok {
		overlayItems, overlayErr := inspector.ListRecentArtefacts(ctx, limit)
		if overlayErr != nil {
			return nil, overlayErr
		}
		for _, item := range overlayItems {
			if existing, seen := items[item.ID]; !seen || item.UpdatedAt > existing.UpdatedAt {
				items[item.ID] = item
			}
		}
	}

	merged := slices.Collect(maps.Values(items))
	slices.SortFunc(merged, func(a, b registry_domain.ArtefactListItem) int {
		return cmp.Compare(b.UpdatedAt, a.UpdatedAt)
	})
	if limit > 0 && len(merged) > limit {
		merged = merged[:limit]
	}
	return merged, nil
}

// baseArtefact returns the base's artefact for the ID, or nil when the base does not have
// it.
//
// A base that claims ownership but then fails the read is an inconsistency, so the read
// error is returned rather than swallowed: a caller writing the overlay delta must not
// silently treat a base failure as "the base provides nothing" and copy base-owned
// records into the overlay.
//
// Takes artefactID (string) which is the ID to fetch.
//
// Returns *registry_dto.ArtefactMeta which is the base artefact, or nil when the base
// lacks it.
// Returns error when the base owns the ID but the read fails.
func (s *Store) baseArtefact(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	if !s.base.OwnsArtefact(artefactID) {
		return nil, nil
	}
	return s.base.GetArtefact(ctx, artefactID)
}

// deltaActions rewrites each upsert action to carry only the records the base does not
// provide, leaving other action types untouched.
//
// Takes actions ([]registry_dto.AtomicAction) which is the incoming batch.
//
// Returns []registry_dto.AtomicAction which is the delta batch for the overlay.
// Returns error when a base read fails, so a base failure never lets base-owned records
// be written into the overlay delta.
func (s *Store) deltaActions(ctx context.Context, actions []registry_dto.AtomicAction) ([]registry_dto.AtomicAction, error) {
	out := make([]registry_dto.AtomicAction, 0, len(actions))
	for _, action := range actions {
		if action.Type != registry_dto.ActionTypeUpsertArtefact || action.Artefact == nil {
			out = append(out, action)
			continue
		}
		baseArtefact, err := s.baseArtefact(ctx, action.Artefact.ID)
		if err != nil {
			return nil, err
		}
		delta, _ := registry_dto.ArtefactDelta(baseArtefact, action.Artefact)
		out = append(out, registry_dto.AtomicAction{
			Type:       registry_dto.ActionTypeUpsertArtefact,
			ArtefactID: action.ArtefactID,
			Artefact:   delta,
		})
	}
	return out, nil
}

// mergeMatches unions two search result sets by artefact ID, merging records that appear
// in both.
//
// Takes baseResults ([]*registry_dto.ArtefactMeta) which are the base matches.
// Takes overlayResults ([]*registry_dto.ArtefactMeta) which are the overlay matches.
//
// Returns []*registry_dto.ArtefactMeta which are the merged matches.
// Returns error when the batched merge read fails.
func (s *Store) mergeMatches(ctx context.Context, baseResults, overlayResults []*registry_dto.ArtefactMeta) ([]*registry_dto.ArtefactMeta, error) {
	seen := make(map[string]struct{})
	ids := make([]string, 0, len(baseResults)+len(overlayResults))
	for _, result := range slices.Concat(baseResults, overlayResults) {
		if _, done := seen[result.ID]; done {
			continue
		}
		seen[result.ID] = struct{}{}
		ids = append(ids, result.ID)
	}
	return s.GetMultipleArtefacts(ctx, ids)
}

// overlayLayers wraps a possibly-nil overlay artefact into the slice MergeLayers expects.
//
// Takes overlay (*registry_dto.ArtefactMeta) which is the overlay artefact, or nil.
//
// Returns []*registry_dto.ArtefactMeta which is empty when overlay is nil.
func overlayLayers(overlay *registry_dto.ArtefactMeta) []*registry_dto.ArtefactMeta {
	if overlay == nil {
		return nil
	}
	return []*registry_dto.ArtefactMeta{overlay}
}

// indexArtefactsByID keys a slice of artefacts by ID, skipping nil entries, so a batched
// layer read can be merged per requested ID without a per-ID store round trip.
//
// Takes artefacts ([]*registry_dto.ArtefactMeta) which is the layer's batch result.
//
// Returns map[string]*registry_dto.ArtefactMeta keyed by artefact ID.
func indexArtefactsByID(artefacts []*registry_dto.ArtefactMeta) map[string]*registry_dto.ArtefactMeta {
	byID := make(map[string]*registry_dto.ArtefactMeta, len(artefacts))
	for _, artefact := range artefacts {
		if artefact != nil {
			byID[artefact.ID] = artefact
		}
	}
	return byID
}

// overlayInspector returns the overlay as a RegistryInspector, or nil when the overlay is
// absent or does not answer inspector queries.
//
// Returns registry_domain.RegistryInspector which is the overlay inspector, or nil.
func (s *Store) overlayInspector() registry_domain.RegistryInspector {
	if inspector, ok := s.overlay.(registry_domain.RegistryInspector); ok {
		return inspector
	}
	return nil
}

// mergeStatusSummaries reads a status-count summary from the base and, when an overlay
// inspector is present, the overlay, then folds them into one slice summing counts per
// status. It lets the artefact and variant summary merges share one implementation over
// their distinct entry types.
//
// Takes inspector (registry_domain.RegistryInspector) which is the overlay inspector, or
// nil.
// Takes fetchBase (func(context.Context) ([]T, error)) which reads the base layer's
// entries.
// Takes fetchOverlay (func(registry_domain.RegistryInspector, context.Context) ([]T,
// error)) which reads the overlay layer's entries when an inspector is present.
// Takes status (func(T) string) which reads an entry's status.
// Takes count (func(T) int64) which reads an entry's count.
// Takes build (func(string, int64) T) which constructs a combined entry.
//
// Returns []T which holds one entry per distinct status with counts summed.
// Returns error when either layer's read fails.
func mergeStatusSummaries[T any](
	ctx context.Context,
	inspector registry_domain.RegistryInspector,
	fetchBase func(context.Context) ([]T, error),
	fetchOverlay func(registry_domain.RegistryInspector, context.Context) ([]T, error),
	status func(T) string,
	count func(T) int64,
	build func(string, int64) T,
) ([]T, error) {
	baseEntries, err := fetchBase(ctx)
	if err != nil {
		return nil, err
	}
	var overlayEntries []T
	if inspector != nil {
		overlayEntries, err = fetchOverlay(inspector, ctx)
		if err != nil {
			return nil, err
		}
	}
	counts := make(map[string]int64)
	for _, entry := range baseEntries {
		counts[status(entry)] += count(entry)
	}
	for _, entry := range overlayEntries {
		counts[status(entry)] += count(entry)
	}
	out := make([]T, 0, len(counts))
	for statusKey, total := range counts {
		out = append(out, build(statusKey, total))
	}
	return out, nil
}

// isNotFound reports whether an error is an artefact-not-found error.
//
// Takes err (error) which is the error to classify.
//
// Returns bool which is true for ErrArtefactNotFound.
func isNotFound(err error) bool {
	return errors.Is(err, registry_domain.ErrArtefactNotFound)
}

// isSearchUnsupported reports whether an error signals an unsupported search.
//
// Takes err (error) which is the error to classify.
//
// Returns bool which is true for ErrSearchUnsupported.
func isSearchUnsupported(err error) bool {
	return errors.Is(err, registry_domain.ErrSearchUnsupported)
}
