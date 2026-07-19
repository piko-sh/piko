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

package snapshot

import (
	"cmp"
	"context"
	"slices"

	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

const (
	// tagKeySeparator joins a tag key and value into a single index key. The NUL byte cannot
	// appear in a tag key or value, so it is an unambiguous separator.
	tagKeySeparator = "\x00"
)

// Store is an immutable, read-only registry store built from a fixed artefact set.
//
// It is safe for concurrent use without locking because it is never mutated after
// construction.
type Store struct {
	// byID maps an artefact ID to its record.
	byID map[string]*registry_dto.ArtefactMeta

	// byStorageKey maps a variant or chunk storage key to its owning artefact ID.
	byStorageKey map[string]string

	// byTag maps a "tagKey<sep>tagValue" index key to the set of artefact IDs carrying it.
	byTag map[string]map[string]struct{}

	// ids is the sorted list of artefact IDs, so listing is deterministic.
	ids []string
}

// compile-time proofs that Store satisfies the read-only store and inspector contracts,
// so a base-only deployment's artefacts still appear in the monitoring TUI.
var (
	_ registry_domain.MetadataStore = (*Store)(nil)

	_ registry_domain.RegistryInspector = (*Store)(nil)
)

// New builds an immutable store from the given artefacts.
//
// The artefacts are indexed by ID, by every variant and chunk storage key, and by every
// tag on their profiles and variants, mirroring the indexes the writable backends
// maintain so a union over this base and an overlay answers the same queries. The input
// slice is not retained; each artefact is cloned into the store.
//
// Takes artefacts ([]*registry_dto.ArtefactMeta) which are the base-layer records.
//
// Returns *Store which is the immutable base store.
func New(artefacts []*registry_dto.ArtefactMeta) *Store {
	store := &Store{
		byID:         make(map[string]*registry_dto.ArtefactMeta, len(artefacts)),
		byStorageKey: make(map[string]string),
		byTag:        make(map[string]map[string]struct{}),
		ids:          make([]string, 0, len(artefacts)),
	}

	for _, artefact := range artefacts {
		if artefact == nil {
			continue
		}
		store.index(artefact.Clone())
	}
	slices.Sort(store.ids)
	return store
}

// OwnsArtefact reports whether the base holds an artefact with the given ID, without
// cloning.
//
// The union store calls it to decide, without an overlay round trip, whether an artefact
// exists in the binary.
//
// Takes artefactID (string) which is the ID to test.
//
// Returns bool which is true when the base holds it.
func (s *Store) OwnsArtefact(artefactID string) bool {
	_, ok := s.byID[artefactID]
	return ok
}

// OwnsStorageKey reports whether the base owns a variant or chunk stored at the given
// key.
//
// The union store calls it so a blob the binary ships is never treated as orphanable and
// a ref-count operation on it never reaches the overlay.
//
// Takes storageKey (string) which is the key to test.
//
// Returns bool which is true when the base owns it.
func (s *Store) OwnsStorageKey(storageKey string) bool {
	_, ok := s.byStorageKey[storageKey]
	return ok
}

// ArtefactIDs returns every ID the base holds, sorted.
//
// Returns []string which is a copy of the base's ID set.
func (s *Store) ArtefactIDs() []string {
	return slices.Clone(s.ids)
}

// GetArtefact returns a clone of the base artefact, or ErrArtefactNotFound.
//
// Takes artefactID (string) which is the ID to fetch.
//
// Returns *registry_dto.ArtefactMeta which is an independent clone.
// Returns error which is ErrArtefactNotFound when absent.
func (s *Store) GetArtefact(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	artefact, ok := s.byID[artefactID]
	if !ok {
		return nil, registry_domain.ErrArtefactNotFound
	}
	return artefact.Clone(), nil
}

// GetMultipleArtefacts returns clones of the present artefacts, skipping the absent ones.
//
// Takes artefactIDs ([]string) which are the IDs to fetch.
//
// Returns []*registry_dto.ArtefactMeta which are the present artefacts, cloned.
// Returns error which is always nil.
func (s *Store) GetMultipleArtefacts(_ context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error) {
	results := make([]*registry_dto.ArtefactMeta, 0, len(artefactIDs))
	for _, id := range artefactIDs {
		if artefact, ok := s.byID[id]; ok {
			results = append(results, artefact.Clone())
		}
	}
	return results, nil
}

// ListAllArtefactIDs returns every base artefact ID, sorted.
//
// Returns []string which is a copy of the base's ID set.
// Returns error which is always nil.
func (s *Store) ListAllArtefactIDs(_ context.Context) ([]string, error) {
	return s.ArtefactIDs(), nil
}

// SearchArtefacts returns the artefacts matching a simple tag query.
//
// Only SimpleTagQuery is supported; a RawRediSearchQuery yields ErrSearchUnsupported,
// matching the writable backends that also lack a full-text engine.
//
// Takes query (registry_domain.SearchQuery) which describes the filter.
//
// Returns []*registry_dto.ArtefactMeta which match every tag in the query, cloned.
// Returns error which is ErrSearchUnsupported for an unsupported query shape.
func (s *Store) SearchArtefacts(_ context.Context, query registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
	if query.RawRediSearchQuery != "" {
		return nil, registry_domain.ErrSearchUnsupported
	}
	if len(query.SimpleTagQuery) == 0 {
		return nil, nil
	}

	var matched map[string]struct{}
	for tagKey, tagValue := range query.SimpleTagQuery {
		ids := s.byTag[tagKey+tagKeySeparator+tagValue]
		if matched == nil {
			matched = make(map[string]struct{}, len(ids))
			for id := range ids {
				matched[id] = struct{}{}
			}
			continue
		}
		for id := range matched {
			if _, ok := ids[id]; !ok {
				delete(matched, id)
			}
		}
	}
	return s.cloneByIDs(matched), nil
}

// SearchArtefactsByTagValues returns the artefacts carrying any of the given values on a
// key.
//
// Takes tagKey (string) which is the tag name to match.
// Takes tagValues ([]string) which are the values to match against.
//
// Returns []*registry_dto.ArtefactMeta which carry any of the values, cloned.
// Returns error which is always nil.
func (s *Store) SearchArtefactsByTagValues(_ context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error) {
	matched := make(map[string]struct{})
	for _, value := range tagValues {
		for id := range s.byTag[tagKey+tagKeySeparator+value] {
			matched[id] = struct{}{}
		}
	}
	return s.cloneByIDs(matched), nil
}

// FindArtefactByVariantStorageKey returns the artefact owning a variant or chunk stored
// at the given key.
//
// Takes storageKey (string) which is the key to resolve.
//
// Returns *registry_dto.ArtefactMeta which owns the key, cloned.
// Returns error which is ErrArtefactNotFound when no artefact owns the key.
func (s *Store) FindArtefactByVariantStorageKey(_ context.Context, storageKey string) (*registry_dto.ArtefactMeta, error) {
	artefactID, ok := s.byStorageKey[storageKey]
	if !ok {
		return nil, registry_domain.ErrArtefactNotFound
	}
	artefact, ok := s.byID[artefactID]
	if !ok {
		return nil, registry_domain.ErrArtefactNotFound
	}
	return artefact.Clone(), nil
}

// GetBlobRefCount reports one for a blob the base owns, otherwise zero.
//
// A blob the binary ships is referenced by the binary itself, so its count is one; an
// unknown key is zero, which is a count, not an error.
//
// Takes storageKey (string) which is the key to look up.
//
// Returns int which is one for an owned key, zero otherwise.
// Returns error which is always nil.
func (s *Store) GetBlobRefCount(_ context.Context, storageKey string) (int, error) {
	if s.OwnsStorageKey(storageKey) {
		return 1, nil
	}
	return 0, nil
}

// PopGCHints returns nothing: base blobs live in the binary and can never be orphaned.
//
// Returns []registry_dto.GCHint which is always empty.
// Returns error which is always nil.
func (*Store) PopGCHints(_ context.Context, _ int) ([]registry_dto.GCHint, error) {
	return nil, nil
}

// AtomicUpdate is unsupported: the base is read-only.
//
// Returns error which is always ErrReadOnlyStore.
func (*Store) AtomicUpdate(_ context.Context, _ []registry_dto.AtomicAction) error {
	return registry_domain.ErrReadOnlyStore
}

// IncrementBlobRefCount is unsupported: the base is read-only.
//
// Returns int which is always zero.
// Returns error which is always ErrReadOnlyStore.
func (*Store) IncrementBlobRefCount(_ context.Context, _ registry_domain.BlobReference) (int, error) {
	return 0, registry_domain.ErrReadOnlyStore
}

// DecrementBlobRefCount is unsupported: the base is read-only.
//
// Returns int which is always zero.
// Returns bool which is always false.
// Returns error which is always ErrReadOnlyStore.
func (*Store) DecrementBlobRefCount(_ context.Context, _ string) (int, bool, error) {
	return 0, false, registry_domain.ErrReadOnlyStore
}

// RunAtomic is unsupported: the base is read-only.
//
// Returns error which is always ErrReadOnlyStore.
func (*Store) RunAtomic(_ context.Context, _ func(ctx context.Context, transactionStore registry_domain.MetadataStore) error) error {
	return registry_domain.ErrReadOnlyStore
}

// Close releases nothing: the base is in-memory.
//
// Returns error which is always nil.
func (*Store) Close() error {
	return nil
}

// ListArtefactSummary counts the base artefacts by status.
//
// Returns []registry_domain.ArtefactSummary which is the count per status.
// Returns error which is always nil.
func (s *Store) ListArtefactSummary(_ context.Context) ([]registry_domain.ArtefactSummary, error) {
	counts := make(map[string]int64)
	for _, artefact := range s.byID {
		counts[string(artefact.Status)]++
	}
	summaries := make([]registry_domain.ArtefactSummary, 0, len(counts))
	for status, count := range counts {
		summaries = append(summaries, registry_domain.ArtefactSummary{Status: status, Count: count})
	}
	return summaries, nil
}

// ListVariantSummary counts the base variants by status.
//
// Returns []registry_domain.VariantSummary which is the count per status.
// Returns error which is always nil.
func (s *Store) ListVariantSummary(_ context.Context) ([]registry_domain.VariantSummary, error) {
	counts := make(map[string]int64)
	for _, artefact := range s.byID {
		for i := range artefact.ActualVariants {
			counts[string(artefact.ActualVariants[i].Status)]++
		}
	}
	summaries := make([]registry_domain.VariantSummary, 0, len(counts))
	for status, count := range counts {
		summaries = append(summaries, registry_domain.VariantSummary{Status: status, Count: count})
	}
	return summaries, nil
}

// ListRecentArtefacts returns the most recently updated base artefacts, newest first.
//
// Takes limit (int) which caps the number returned; a non-positive limit returns all.
//
// Returns []registry_domain.ArtefactListItem which describes each recent artefact.
// Returns error which is always nil.
func (s *Store) ListRecentArtefacts(_ context.Context, limit int) ([]registry_domain.ArtefactListItem, error) {
	items := make([]registry_domain.ArtefactListItem, 0, len(s.byID))
	for _, artefact := range s.byID {
		var totalSize int64
		for i := range artefact.ActualVariants {
			totalSize += artefact.ActualVariants[i].SizeBytes
		}
		items = append(items, registry_domain.ArtefactListItem{
			ID:           artefact.ID,
			SourcePath:   artefact.SourcePath,
			Status:       string(artefact.Status),
			VariantCount: int64(len(artefact.ActualVariants)),
			TotalSize:    totalSize,
			CreatedAt:    artefact.CreatedAt.Unix(),
			UpdatedAt:    artefact.UpdatedAt.Unix(),
		})
	}
	slices.SortFunc(items, func(a, b registry_domain.ArtefactListItem) int {
		return cmp.Compare(b.UpdatedAt, a.UpdatedAt)
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

// index records one artefact into every index. It is called only during construction.
//
// Takes artefact (*registry_dto.ArtefactMeta) which is the already-cloned record to
// index.
func (s *Store) index(artefact *registry_dto.ArtefactMeta) {
	if _, exists := s.byID[artefact.ID]; !exists {
		s.ids = append(s.ids, artefact.ID)
	}
	s.byID[artefact.ID] = artefact

	for i := range artefact.ActualVariants {
		variant := &artefact.ActualVariants[i]
		if variant.StorageKey != "" {
			s.byStorageKey[variant.StorageKey] = artefact.ID
		}
		for j := range variant.Chunks {
			if variant.Chunks[j].StorageKey != "" {
				s.byStorageKey[variant.Chunks[j].StorageKey] = artefact.ID
			}
		}
		for tagKey, tagValue := range variant.MetadataTags.All() {
			s.addTag(tagKey, tagValue, artefact.ID)
		}
	}

	for i := range artefact.DesiredProfiles {
		for tagKey, tagValue := range artefact.DesiredProfiles[i].Profile.ResultingTags.All() {
			s.addTag(tagKey, tagValue, artefact.ID)
		}
	}
}

// addTag records that an artefact carries a tag value under a tag key.
//
// Takes tagKey (string) which is the tag name.
// Takes tagValue (string) which is the tag value.
// Takes artefactID (string) which carries the tag.
func (s *Store) addTag(tagKey, tagValue, artefactID string) {
	indexKey := tagKey + tagKeySeparator + tagValue
	ids, ok := s.byTag[indexKey]
	if !ok {
		ids = make(map[string]struct{})
		s.byTag[indexKey] = ids
	}
	ids[artefactID] = struct{}{}
}

// cloneByIDs returns cloned artefacts for a set of IDs.
//
// Takes ids (map[string]struct{}) which is the set of IDs to materialise.
//
// Returns []*registry_dto.ArtefactMeta which are the cloned matches.
func (s *Store) cloneByIDs(ids map[string]struct{}) []*registry_dto.ArtefactMeta {
	results := make([]*registry_dto.ArtefactMeta, 0, len(ids))
	for id := range ids {
		if artefact, ok := s.byID[id]; ok {
			results = append(results, artefact.Clone())
		}
	}
	return results
}
