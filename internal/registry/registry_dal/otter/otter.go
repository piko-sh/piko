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

package otter

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_dal"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

const (
	// defaultCacheCapacity is the default number of items the cache can store.
	defaultCacheCapacity = 100_000

	// maxTransactionTimeout is the maximum duration a RunAtomic transaction
	// may hold the mutex before the context is cancelled.
	maxTransactionTimeout = 30 * time.Second

	// tagKeySeparator joins tag keys and values in the tag index (e.g.
	// "category:blog").
	tagKeySeparator = ":"
)

var (
	log = logger_domain.GetLogger("piko/internal/registry/registry_dal/otter")

	_ registry_dal.RegistryDALWithTx = (*DAL)(nil)

	_ registry_domain.RegistryInspector = (*DAL)(nil)
)

// blobRef tracks a blob's reference count alongside its metadata.
type blobRef struct {
	// blob holds the metadata for the referenced blob.
	blob registry_domain.BlobReference

	// refCount tracks how many times this blob is referenced.
	refCount int
}

// DAL provides in-memory storage for registry artefacts using otter cache.
// It implements RegistryDALWithTx and RegistryInspector.
type DAL struct {
	// artefacts stores artefact metadata by artefact ID. Uses the cache
	// hexagon's ProviderPort for optional WAL persistence.
	artefacts cache_domain.ProviderPort[string, *registry_dto.ArtefactMeta]

	// tagIndex maps "tagKey:tagValue" strings to sets of artefact IDs for
	// tag-based search. Uses the cache hexagon's TagIndex for fast exact-match
	// lookups.
	tagIndex *provider_otter.TagIndex[string]

	// variantKeyIndex maps variant and chunk storage keys to their parent artefact
	// ID.
	variantKeyIndex map[string]string

	// blobRefs maps storage keys to their reference counts for content
	// deduplication.
	blobRefs map[string]*blobRef

	// gcHints holds garbage collection hints for orphaned blobs awaiting removal.
	gcHints []registry_dto.GCHint

	// ownsCache indicates whether this DAL is responsible for closing the cache.
	ownsCache bool

	// mu guards access to indexes, blobRefs, and gcHints.
	mu sync.RWMutex
}

// Config holds settings for the otter-based registry DAL.
type Config struct {
	// Capacity is the maximum number of items to store.
	// Defaults to 100,000 if zero or negative.
	Capacity int64
}

// Option configures the DAL during construction.
type Option func(*DAL)

// HealthCheck verifies the DAL is operational.
//
// Returns error which is always nil for in-memory storage.
func (*DAL) HealthCheck(_ context.Context) error {
	return nil
}

// Close releases resources held by the DAL.
//
// Returns error which is always nil for in-memory storage.
func (d *DAL) Close() error {
	if d.ownsCache {
		return d.artefacts.Close(context.Background())
	}
	return nil
}

// RunAtomic executes fn within a transaction with rollback support.
//
// All cache mutations are journalled so they can be undone if fn
// returns an error. Non-cache state (blobRefs, gcHints,
// variantKeyIndex) is snapshotted and restored on rollback.
//
// Takes fn (func) which receives a transactional MetadataStore
// scoped to this transaction.
//
// Returns error when fn returns an error (after rolling back all
// mutations).
//
// Panics if fn panics; the transaction is rolled back before the
// panic is re-raised.
//
// Safe for concurrent use; acquires a write lock for the entire
// duration.
func (d *DAL) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore registry_domain.MetadataStore) error) error {
	ctx, cancel := context.WithTimeoutCause(ctx, maxTransactionTimeout,
		fmt.Errorf("transaction exceeded maximum duration of %s", maxTransactionTimeout))
	defer cancel()

	d.mu.Lock()
	defer d.mu.Unlock()

	transactionCache := cache_domain.BeginTransaction(ctx, d.artefacts)
	tx := &otterTransactionDAL{
		parent:              d,
		transactionCache:    transactionCache,
		blobRefSnapshots:    make(map[string]*blobRef),
		newBlobRefKeys:      make(map[string]struct{}),
		variantKeySnapshots: make(map[string]string),
		newVariantKeys:      make(map[string]struct{}),
		gcHintsSnapshot:     make([]registry_dto.GCHint, len(d.gcHints)),
	}
	copy(tx.gcHintsSnapshot, d.gcHints)

	rollbackCtx := context.WithoutCancel(ctx)

	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				tx.rollback(rollbackCtx)
				panic(r)
			}
		}()
		err = fn(ctx, tx)
	}()
	if err != nil {
		tx.rollback(rollbackCtx)
		return err
	}

	if commitErr := transactionCache.Commit(ctx); commitErr != nil {
		tx.rollback(rollbackCtx)
		return fmt.Errorf("committing transaction: %w", commitErr)
	}
	return nil
}

// otterTransactionDAL is a transaction-scoped MetadataStore that
// wraps the parent DAL.
//
// It uses a journalled cache for artefact mutations and snapshots
// non-cache state (blobRefs, gcHints, variantKeyIndex) for
// rollback. All methods skip mutex acquisition since the parent's
// mu is already held by RunAtomic.
type otterTransactionDAL struct {
	// parent is the owning DAL whose state is being mutated.
	parent *DAL

	// transactionCache journals artefact cache mutations for
	// rollback.
	transactionCache cache_domain.TransactionCache[string, *registry_dto.ArtefactMeta]

	// blobRefSnapshots stores the old blobRef value (or nil if absent) for each
	// key that was mutated. Only the first mutation per key is recorded.
	blobRefSnapshots map[string]*blobRef

	// newBlobRefKeys tracks keys that were created during the transaction (did
	// not exist before). On rollback these are deleted.
	newBlobRefKeys map[string]struct{}

	// variantKeySnapshots stores the old variantKeyIndex value (or "" if absent)
	// for each key that was mutated.
	variantKeySnapshots map[string]string

	// newVariantKeys tracks keys that were created during the transaction.
	newVariantKeys map[string]struct{}

	// gcHintsSnapshot captures the gcHints slice at transaction start.
	gcHintsSnapshot []registry_dto.GCHint
}

// GetArtefact retrieves an artefact from the transaction cache.
//
// Takes artefactID (string) which identifies the artefact.
//
// Returns *ArtefactMeta which is the artefact.
// Returns error when the artefact is not found.
func (tx *otterTransactionDAL) GetArtefact(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	artefact, found, _ := tx.transactionCache.GetIfPresent(ctx, artefactID)
	if !found {
		return nil, registry_domain.ErrArtefactNotFound
	}
	return artefact, nil
}

// GetMultipleArtefacts fetches several artefacts from the
// transaction cache.
//
// Takes artefactIDs ([]string) which lists the IDs to fetch.
//
// Returns []*ArtefactMeta which contains the found
// artefacts.
// Returns error which is always nil.
func (tx *otterTransactionDAL) GetMultipleArtefacts(ctx context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error) {
	results := make([]*registry_dto.ArtefactMeta, 0, len(artefactIDs))
	for _, id := range artefactIDs {
		artefact, found, _ := tx.transactionCache.GetIfPresent(ctx, id)
		if !found {
			continue
		}
		results = append(results, artefact)
	}
	return results, nil
}

// ListAllArtefactIDs returns all artefact IDs from the transaction
// cache.
//
// Returns []string which contains all artefact IDs.
// Returns error which is always nil.
func (tx *otterTransactionDAL) ListAllArtefactIDs(_ context.Context) ([]string, error) {
	ids := make([]string, 0, tx.transactionCache.EstimatedSize())
	for key := range tx.transactionCache.Keys() {
		ids = append(ids, key)
	}
	return ids, nil
}

// SearchArtefacts finds artefacts matching the given tag query.
//
// Takes query (registry_domain.SearchQuery) which specifies the
// search terms.
//
// Returns []*ArtefactMeta which contains the matching
// artefacts.
// Returns error which is always nil.
func (tx *otterTransactionDAL) SearchArtefacts(ctx context.Context, query registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
	if len(query.SimpleTagQuery) == 0 {
		return []*registry_dto.ArtefactMeta{}, nil
	}

	matchingIDs := tx.parent.intersectTagMatches(query.SimpleTagQuery)
	if len(matchingIDs) == 0 {
		return []*registry_dto.ArtefactMeta{}, nil
	}

	results := make([]*registry_dto.ArtefactMeta, 0, len(matchingIDs))
	for id := range matchingIDs {
		if artefact, found, _ := tx.transactionCache.GetIfPresent(ctx, id); found {
			results = append(results, artefact)
		}
	}
	return results, nil
}

// SearchArtefactsByTagValues searches for artefacts with a specific
// tag key.
//
// Takes tagKey (string) which specifies the tag key to match.
// Takes tagValues ([]string) which lists acceptable values.
//
// Returns []*ArtefactMeta which contains the matching
// artefacts.
// Returns error which is always nil.
func (tx *otterTransactionDAL) SearchArtefactsByTagValues(ctx context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error) {
	if len(tagValues) == 0 {
		return []*registry_dto.ArtefactMeta{}, nil
	}

	matchingIDs := make(map[string]struct{})
	for _, tagValue := range tagValues {
		indexKey := tagKey + tagKeySeparator + tagValue
		for id := range tx.parent.tagIndex.Get(indexKey) {
			matchingIDs[id] = struct{}{}
		}
	}

	results := make([]*registry_dto.ArtefactMeta, 0, len(matchingIDs))
	for id := range matchingIDs {
		if artefact, found, _ := tx.transactionCache.GetIfPresent(ctx, id); found {
			results = append(results, artefact)
		}
	}
	return results, nil
}

// FindArtefactByVariantStorageKey finds an artefact by variant
// storage key.
//
// Takes storageKey (string) which identifies the variant.
//
// Returns *ArtefactMeta which is the artefact.
// Returns error when no artefact has that storage key.
func (tx *otterTransactionDAL) FindArtefactByVariantStorageKey(ctx context.Context, storageKey string) (*registry_dto.ArtefactMeta, error) {
	artefactID, found := tx.parent.variantKeyIndex[storageKey]
	if !found {
		return nil, registry_domain.ErrArtefactNotFound
	}

	artefact, found, _ := tx.transactionCache.GetIfPresent(ctx, artefactID)
	if !found {
		return nil, registry_domain.ErrArtefactNotFound
	}
	return artefact, nil
}

// PopGCHints retrieves and removes garbage collection hints.
//
// Takes limit (int) which caps the number of hints returned.
//
// Returns []GCHint which contains the removed hints.
// Returns error which is always nil.
func (tx *otterTransactionDAL) PopGCHints(_ context.Context, limit int) ([]registry_dto.GCHint, error) {
	if len(tx.parent.gcHints) == 0 {
		return []registry_dto.GCHint{}, nil
	}

	count := min(limit, len(tx.parent.gcHints))
	hints := make([]registry_dto.GCHint, count)
	copy(hints, tx.parent.gcHints[:count])
	tx.parent.gcHints = tx.parent.gcHints[count:]
	return hints, nil
}

// AtomicUpdate runs a batch of actions using the transaction cache.
//
// Takes actions ([]registry_dto.AtomicAction) which lists the
// operations to perform.
//
// Returns error when any action fails.
func (tx *otterTransactionDAL) AtomicUpdate(ctx context.Context, actions []registry_dto.AtomicAction) error {
	for _, action := range actions {
		switch action.Type {
		case registry_dto.ActionTypeUpsertArtefact:
			if action.Artefact == nil {
				return errors.New("upsert action missing artefact")
			}
			tx.upsertArtefactLocked(ctx, action.Artefact)

		case registry_dto.ActionTypeDeleteArtefact:
			if action.ArtefactID == "" {
				return errors.New("delete action missing artefact ID")
			}
			tx.deleteArtefactLocked(ctx, action.ArtefactID)

		case registry_dto.ActionTypeAddGCHints:
			tx.parent.gcHints = append(tx.parent.gcHints, action.GCHints...)

		default:
			return fmt.Errorf("unknown action type: %s", action.Type)
		}
	}
	return nil
}

// IncrementBlobRefCount increments the reference count,
// snapshotting first.
//
// Takes blob (registry_domain.BlobReference) which identifies the
// blob.
//
// Returns int which is the count after the increment.
// Returns error which is always nil.
func (tx *otterTransactionDAL) IncrementBlobRefCount(_ context.Context, blob registry_domain.BlobReference) (int, error) {
	tx.snapshotBlobRef(blob.StorageKey)

	ref, exists := tx.parent.blobRefs[blob.StorageKey]
	if !exists {
		ref = &blobRef{
			blob:     blob,
			refCount: 0,
		}
		tx.parent.blobRefs[blob.StorageKey] = ref
	}

	ref.refCount++
	ref.blob.LastReferencedAt = time.Now()
	return ref.refCount, nil
}

// DecrementBlobRefCount decrements the reference count,
// snapshotting first.
//
// Takes storageKey (string) which identifies the blob.
//
// Returns int which is the updated reference count.
// Returns bool which is true when the count reaches zero.
// Returns error when the blob does not exist.
func (tx *otterTransactionDAL) DecrementBlobRefCount(_ context.Context, storageKey string) (int, bool, error) {
	tx.snapshotBlobRef(storageKey)

	ref, exists := tx.parent.blobRefs[storageKey]
	if !exists {
		return 0, false, registry_domain.ErrBlobReferenceNotFound
	}

	ref.refCount--
	shouldDelete := ref.refCount <= 0

	if shouldDelete {
		delete(tx.parent.blobRefs, storageKey)
	}

	return ref.refCount, shouldDelete, nil
}

// GetBlobRefCount returns the current reference count for a blob.
//
// Takes storageKey (string) which identifies the blob.
//
// Returns int which is the current count (0 if absent).
// Returns error which is always nil.
func (tx *otterTransactionDAL) GetBlobRefCount(_ context.Context, storageKey string) (int, error) {
	if ref, exists := tx.parent.blobRefs[storageKey]; exists {
		return ref.refCount, nil
	}
	return 0, nil
}

// RunAtomic is not supported within an existing transaction.
//
// Returns error which is always ErrNestedTransactionUnsupported.
func (*otterTransactionDAL) RunAtomic(_ context.Context, _ func(ctx context.Context, transactionStore registry_domain.MetadataStore) error) error {
	return cache_domain.ErrNestedTransactionUnsupported
}

// Close is a no-op for the transaction DAL.
//
// Returns error which is always nil.
func (*otterTransactionDAL) Close() error {
	return nil
}

// snapshotBlobRef records the old value for a blob ref key before
// mutation.
//
// Takes key (string) which identifies the blob ref to snapshot.
func (tx *otterTransactionDAL) snapshotBlobRef(key string) {
	if _, already := tx.blobRefSnapshots[key]; already {
		return
	}
	old, exists := tx.parent.blobRefs[key]
	if exists {
		cp := &blobRef{
			blob:     old.blob,
			refCount: old.refCount,
		}
		tx.blobRefSnapshots[key] = cp
	} else {
		tx.blobRefSnapshots[key] = nil
		tx.newBlobRefKeys[key] = struct{}{}
	}
}

// snapshotVariantKey records the old value for a variant key index
// entry.
//
// Takes key (string) which identifies the variant key to snapshot.
func (tx *otterTransactionDAL) snapshotVariantKey(key string) {
	if _, already := tx.variantKeySnapshots[key]; already {
		return
	}
	old, exists := tx.parent.variantKeyIndex[key]
	if exists {
		tx.variantKeySnapshots[key] = old
	} else {
		tx.variantKeySnapshots[key] = ""
		tx.newVariantKeys[key] = struct{}{}
	}
}

// rollback restores all state to its pre-transaction values.
func (tx *otterTransactionDAL) rollback(ctx context.Context) {
	_ = tx.transactionCache.Rollback(ctx)

	for key, oldRef := range tx.blobRefSnapshots {
		if oldRef == nil {
			delete(tx.parent.blobRefs, key)
		} else {
			tx.parent.blobRefs[key] = oldRef
		}
	}
	for key := range tx.newBlobRefKeys {
		if _, snapshotted := tx.blobRefSnapshots[key]; !snapshotted {
			delete(tx.parent.blobRefs, key)
		}
	}

	tx.parent.gcHints = tx.gcHintsSnapshot

	for key, oldVal := range tx.variantKeySnapshots {
		if oldVal == "" {
			delete(tx.parent.variantKeyIndex, key)
		} else {
			tx.parent.variantKeyIndex[key] = oldVal
		}
	}
	for key := range tx.newVariantKeys {
		if _, snapshotted := tx.variantKeySnapshots[key]; !snapshotted {
			delete(tx.parent.variantKeyIndex, key)
		}
	}

	tx.parent.tagIndex = provider_otter.NewTagIndex[string]()
	for _, artefact := range tx.parent.artefacts.All() {
		tx.parent.addArtefactIndexesLocked(artefact)
	}
}

// upsertArtefactLocked inserts or updates an artefact in the
// transaction cache.
//
// Takes artefact (*registry_dto.ArtefactMeta) which is the
// artefact to store.
func (tx *otterTransactionDAL) upsertArtefactLocked(ctx context.Context, artefact *registry_dto.ArtefactMeta) {
	if old, found, _ := tx.transactionCache.GetIfPresent(ctx, artefact.ID); found {
		tx.removeArtefactIndexesLocked(old)
	}

	_ = tx.transactionCache.Set(ctx, artefact.ID, artefact)

	tx.addArtefactIndexesLocked(artefact)
}

// deleteArtefactLocked removes an artefact from the transaction
// cache.
//
// Takes artefactID (string) which identifies the artefact to
// remove.
func (tx *otterTransactionDAL) deleteArtefactLocked(ctx context.Context, artefactID string) {
	artefact, found, _ := tx.transactionCache.GetIfPresent(ctx, artefactID)
	if !found {
		return
	}

	tx.removeArtefactIndexesLocked(artefact)
	_ = tx.transactionCache.Invalidate(ctx, artefactID)
}

// addArtefactIndexesLocked updates indexes, snapshotting variant
// keys.
//
// Takes artefact (*registry_dto.ArtefactMeta) which is the
// artefact to index.
func (tx *otterTransactionDAL) addArtefactIndexesLocked(artefact *registry_dto.ArtefactMeta) {
	for i := range artefact.ActualVariants {
		if artefact.ActualVariants[i].StorageKey != "" {
			tx.snapshotVariantKey(artefact.ActualVariants[i].StorageKey)
			tx.parent.variantKeyIndex[artefact.ActualVariants[i].StorageKey] = artefact.ID
		}
		for j := range artefact.ActualVariants[i].Chunks {
			if artefact.ActualVariants[i].Chunks[j].StorageKey != "" {
				tx.snapshotVariantKey(artefact.ActualVariants[i].Chunks[j].StorageKey)
				tx.parent.variantKeyIndex[artefact.ActualVariants[i].Chunks[j].StorageKey] = artefact.ID
			}
		}
	}

	for i := range artefact.DesiredProfiles {
		for tagKey, tagValue := range artefact.DesiredProfiles[i].Profile.ResultingTags.All() {
			indexKey := tagKey + tagKeySeparator + tagValue
			tx.parent.tagIndex.AddSingle(indexKey, artefact.ID)
		}
	}

	for i := range artefact.ActualVariants {
		for tagKey, tagValue := range artefact.ActualVariants[i].MetadataTags.All() {
			indexKey := tagKey + tagKeySeparator + tagValue
			tx.parent.tagIndex.AddSingle(indexKey, artefact.ID)
		}
	}
}

// removeArtefactIndexesLocked removes indexes, snapshotting variant
// keys.
//
// Takes artefact (*registry_dto.ArtefactMeta) which is the
// artefact whose indexes should be removed.
func (tx *otterTransactionDAL) removeArtefactIndexesLocked(artefact *registry_dto.ArtefactMeta) {
	for i := range artefact.ActualVariants {
		if artefact.ActualVariants[i].StorageKey != "" {
			tx.snapshotVariantKey(artefact.ActualVariants[i].StorageKey)
			delete(tx.parent.variantKeyIndex, artefact.ActualVariants[i].StorageKey)
		}
		for j := range artefact.ActualVariants[i].Chunks {
			if artefact.ActualVariants[i].Chunks[j].StorageKey != "" {
				tx.snapshotVariantKey(artefact.ActualVariants[i].Chunks[j].StorageKey)
				delete(tx.parent.variantKeyIndex, artefact.ActualVariants[i].Chunks[j].StorageKey)
			}
		}
	}

	for i := range artefact.DesiredProfiles {
		for tagKey, tagValue := range artefact.DesiredProfiles[i].Profile.ResultingTags.All() {
			indexKey := tagKey + tagKeySeparator + tagValue
			tx.parent.tagIndex.RemoveSingle(indexKey, artefact.ID)
		}
	}

	for i := range artefact.ActualVariants {
		for tagKey, tagValue := range artefact.ActualVariants[i].MetadataTags.All() {
			indexKey := tagKey + tagKeySeparator + tagValue
			tx.parent.tagIndex.RemoveSingle(indexKey, artefact.ID)
		}
	}
}

// GetArtefact retrieves a single artefact by ID.
//
// Takes artefactID (string) which identifies the artefact to retrieve.
//
// Returns *ArtefactMeta which contains the artefact metadata.
// Returns error when the artefact is not found.
func (d *DAL) GetArtefact(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	artefact, found, _ := d.artefacts.GetIfPresent(ctx, artefactID)
	if !found {
		return nil, registry_domain.ErrArtefactNotFound
	}
	return artefact, nil
}

// GetMultipleArtefacts fetches several artefacts by their IDs.
//
// Takes artefactIDs ([]string) which lists the IDs to fetch.
//
// Returns []*ArtefactMeta which contains the found artefacts.
// Returns error when an artefact cannot be found.
func (d *DAL) GetMultipleArtefacts(ctx context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error) {
	results := make([]*registry_dto.ArtefactMeta, 0, len(artefactIDs))
	for _, id := range artefactIDs {
		artefact, found, _ := d.artefacts.GetIfPresent(ctx, id)
		if !found {
			continue
		}
		results = append(results, artefact)
	}
	return results, nil
}

// ListAllArtefactIDs returns all artefact IDs in the store.
//
// Returns []string which contains all artefact IDs.
// Returns error which is always nil.
//
// Safe for concurrent use.
func (d *DAL) ListAllArtefactIDs(_ context.Context) ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ids := make([]string, 0, d.artefacts.EstimatedSize())
	for key := range d.artefacts.Keys() {
		ids = append(ids, key)
	}
	return ids, nil
}

// SearchArtefacts finds artefacts that match the given tag query.
//
// Takes query (registry_domain.SearchQuery) which specifies the search terms.
//
// Returns []*ArtefactMeta which contains the matching
// artefacts.
// Returns error when the search fails.
//
// Safe for concurrent use. Uses a read lock to protect
// access to the tag index and artefacts cache.
func (d *DAL) SearchArtefacts(ctx context.Context, query registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
	_, l := logger_domain.From(ctx, log)

	if query.RawRediSearchQuery != "" {
		l.Warn("RawRediSearchQuery not supported by otter DAL, ignoring")
	}

	if len(query.SimpleTagQuery) == 0 {
		return []*registry_dto.ArtefactMeta{}, nil
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	matchingIDs := d.intersectTagMatches(query.SimpleTagQuery)
	if len(matchingIDs) == 0 {
		return []*registry_dto.ArtefactMeta{}, nil
	}

	return d.collectArtefactsByIDs(ctx, matchingIDs), nil
}

// SearchArtefactsByTagValues searches for artefacts with a specific tag key
// and any of the given values.
//
// Takes tagKey (string) which specifies the tag key to match.
// Takes tagValues ([]string) which lists the acceptable values.
//
// Returns []*ArtefactMeta which contains the matching
// artefacts.
// Returns error which is always nil.
//
// Safe for concurrent use; holds a read lock during the
// search.
func (d *DAL) SearchArtefactsByTagValues(ctx context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error) {
	if len(tagValues) == 0 {
		return []*registry_dto.ArtefactMeta{}, nil
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	matchingIDs := make(map[string]struct{})
	for _, tagValue := range tagValues {
		indexKey := tagKey + tagKeySeparator + tagValue
		for id := range d.tagIndex.Get(indexKey) {
			matchingIDs[id] = struct{}{}
		}
	}

	results := make([]*registry_dto.ArtefactMeta, 0, len(matchingIDs))
	for id := range matchingIDs {
		if artefact, found, _ := d.artefacts.GetIfPresent(ctx, id); found {
			results = append(results, artefact)
		}
	}

	return results, nil
}

// FindArtefactByVariantStorageKey finds an artefact by a variant's storage key.
//
// Takes storageKey (string) which identifies the variant to search for.
//
// Returns *ArtefactMeta which contains the artefact
// metadata.
// Returns error when no artefact has a variant with that
// storage key.
//
// Safe for concurrent use. Uses a read lock to access the
// variant key index.
func (d *DAL) FindArtefactByVariantStorageKey(ctx context.Context, storageKey string) (*registry_dto.ArtefactMeta, error) {
	d.mu.RLock()
	artefactID, found := d.variantKeyIndex[storageKey]
	d.mu.RUnlock()

	if !found {
		return nil, registry_domain.ErrArtefactNotFound
	}

	artefact, found, _ := d.artefacts.GetIfPresent(ctx, artefactID)
	if !found {
		return nil, registry_domain.ErrArtefactNotFound
	}

	return artefact, nil
}

// PopGCHints retrieves and removes garbage collection hints from the store.
//
// Takes limit (int) which sets the maximum number of hints to return.
//
// Returns []GCHint which contains the removed hints.
// Returns error which is always nil.
//
// Safe for concurrent use. Protected by a mutex lock.
func (d *DAL) PopGCHints(_ context.Context, limit int) ([]registry_dto.GCHint, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.gcHints) == 0 {
		return []registry_dto.GCHint{}, nil
	}

	count := min(limit, len(d.gcHints))
	hints := make([]registry_dto.GCHint, count)
	copy(hints, d.gcHints[:count])
	d.gcHints = d.gcHints[count:]

	return hints, nil
}

// AtomicUpdate runs a batch of actions within a single lock.
//
// Takes actions ([]registry_dto.AtomicAction) which lists the operations.
//
// Returns error when any action fails.
//
// Safe for concurrent use; the method holds a mutex for the entire batch.
func (d *DAL) AtomicUpdate(ctx context.Context, actions []registry_dto.AtomicAction) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, action := range actions {
		switch action.Type {
		case registry_dto.ActionTypeUpsertArtefact:
			if action.Artefact == nil {
				return errors.New("upsert action missing artefact")
			}
			d.upsertArtefactLocked(ctx, action.Artefact)

		case registry_dto.ActionTypeDeleteArtefact:
			if action.ArtefactID == "" {
				return errors.New("delete action missing artefact ID")
			}
			d.deleteArtefactLocked(ctx, action.ArtefactID)

		case registry_dto.ActionTypeAddGCHints:
			d.gcHints = append(d.gcHints, action.GCHints...)

		default:
			return fmt.Errorf("unknown action type: %s", action.Type)
		}
	}

	return nil
}

// IncrementBlobRefCount atomically increments the reference count for a blob.
//
// Takes blob (registry_domain.BlobReference) which identifies the blob.
//
// Returns int which is the reference count after the increment.
// Returns error which is always nil.
//
// Safe for concurrent use; protected by a mutex.
func (d *DAL) IncrementBlobRefCount(_ context.Context, blob registry_domain.BlobReference) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	ref, exists := d.blobRefs[blob.StorageKey]
	if !exists {
		ref = &blobRef{
			blob:     blob,
			refCount: 0,
		}
		d.blobRefs[blob.StorageKey] = ref
	}

	ref.refCount++
	ref.blob.LastReferencedAt = time.Now()

	return ref.refCount, nil
}

// DecrementBlobRefCount atomically decrements the reference count for a blob.
//
// Takes storageKey (string) which identifies the blob.
//
// Returns int which is the updated reference count.
// Returns bool which is true when the count reaches zero.
// Returns error when the blob does not exist.
//
// Safe for concurrent use; protected by a mutex.
func (d *DAL) DecrementBlobRefCount(_ context.Context, storageKey string) (int, bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	ref, exists := d.blobRefs[storageKey]
	if !exists {
		return 0, false, registry_domain.ErrBlobReferenceNotFound
	}

	ref.refCount--
	shouldDelete := ref.refCount <= 0

	if shouldDelete {
		delete(d.blobRefs, storageKey)
	}

	return ref.refCount, shouldDelete, nil
}

// GetBlobRefCount returns the current reference count for a blob.
//
// Takes storageKey (string) which identifies the blob.
//
// Returns int which is the current count (0 if blob doesn't exist).
// Returns error which is always nil.
//
// Safe for concurrent use.
func (d *DAL) GetBlobRefCount(_ context.Context, storageKey string) (int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if ref, exists := d.blobRefs[storageKey]; exists {
		return ref.refCount, nil
	}
	return 0, nil
}

// ListArtefactSummary returns artefact counts grouped by status.
//
// Returns []registry_domain.ArtefactSummary which contains counts per status.
// Returns error which is always nil.
//
// Safe for concurrent use. Uses a read lock to access the artefact data.
func (d *DAL) ListArtefactSummary(_ context.Context) ([]registry_domain.ArtefactSummary, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	statusCounts := make(map[string]int64)
	for _, artefact := range d.artefacts.All() {
		statusCounts[string(artefact.Status)]++
	}

	results := make([]registry_domain.ArtefactSummary, 0, len(statusCounts))
	for status, count := range statusCounts {
		results = append(results, registry_domain.ArtefactSummary{
			Status: status,
			Count:  count,
		})
	}

	return results, nil
}

// ListVariantSummary returns variant counts grouped by status.
//
// Returns []registry_domain.VariantSummary which contains counts per status.
// Returns error which is always nil.
//
// Safe for concurrent use. Uses a read lock on the internal data store.
func (d *DAL) ListVariantSummary(_ context.Context) ([]registry_domain.VariantSummary, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	statusCounts := make(map[string]int64)
	for _, artefact := range d.artefacts.All() {
		for i := range artefact.ActualVariants {
			statusCounts[string(artefact.ActualVariants[i].Status)]++
		}
	}

	results := make([]registry_domain.VariantSummary, 0, len(statusCounts))
	for status, count := range statusCounts {
		results = append(results, registry_domain.VariantSummary{
			Status: status,
			Count:  count,
		})
	}

	return results, nil
}

// ListRecentArtefacts returns the most recently updated artefacts.
//
// Takes limit (int32) which specifies the maximum number to return.
//
// Returns []registry_domain.ArtefactListItem which contains the artefact data.
// Returns error which is always nil.
//
// Safe for concurrent use. Protected by a read lock.
func (d *DAL) ListRecentArtefacts(_ context.Context, limit int32) ([]registry_domain.ArtefactListItem, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	type sortableItem struct {
		artefact *registry_dto.ArtefactMeta
	}
	items := make([]sortableItem, 0, d.artefacts.EstimatedSize())
	for _, artefact := range d.artefacts.All() {
		items = append(items, sortableItem{artefact: artefact})
	}

	for i := range len(items) - 1 {
		for j := i + 1; j < len(items); j++ {
			if items[j].artefact.UpdatedAt.After(items[i].artefact.UpdatedAt) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	count := min(int(limit), len(items))
	results := make([]registry_domain.ArtefactListItem, count)
	for i := range count {
		artefact := items[i].artefact
		var totalSize int64
		for vi := range artefact.ActualVariants {
			totalSize += artefact.ActualVariants[vi].SizeBytes
		}

		results[i] = registry_domain.ArtefactListItem{
			ID:           artefact.ID,
			SourcePath:   artefact.SourcePath,
			Status:       string(artefact.Status),
			VariantCount: int64(len(artefact.ActualVariants)),
			TotalSize:    totalSize,
			CreatedAt:    artefact.CreatedAt.Unix(),
			UpdatedAt:    artefact.UpdatedAt.Unix(),
		}
	}

	return results, nil
}

// RebuildIndexes rebuilds all secondary indexes from the primary cache data.
// Call this after WAL recovery to restore tagIndex and variantKeyIndex.
//
// Safe for concurrent use; acquires the DAL mutex.
//
// Note: blobRefs and gcHints are not recovered - these are ephemeral data that
// reset on restart. Blob reference counts reset to zero, which may leave
// orphaned blobs (blob cleanup is idempotent so this is safe).
func (d *DAL) RebuildIndexes(ctx context.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.tagIndex = provider_otter.NewTagIndex[string]()
	d.variantKeyIndex = make(map[string]string)

	for _, artefact := range d.artefacts.All() {
		d.addArtefactIndexesLocked(artefact)
	}

	_, l := logger_domain.From(ctx, log)
	l.Internal("Registry indexes rebuilt from cache",
		logger_domain.Int("artefact_count", d.artefacts.EstimatedSize()))
}

// intersectTagMatches returns the set of artefact IDs matching ALL tag
// key-value pairs via set intersection.
//
// Takes tags (map[string]string) which specifies the tag key-value pairs to
// match against.
//
// Returns map[string]struct{} which contains the IDs that match all tags, or
// nil when no matches are found.
func (d *DAL) intersectTagMatches(tags map[string]string) map[string]struct{} {
	var matchingIDs map[string]struct{}

	for tagKey, tagValue := range tags {
		ids := d.tagIndex.Get(tagKey + tagKeySeparator + tagValue)

		if matchingIDs == nil {
			matchingIDs = make(map[string]struct{}, len(ids))
			for id := range ids {
				matchingIDs[id] = struct{}{}
			}
			continue
		}

		for id := range matchingIDs {
			if _, ok := ids[id]; !ok {
				delete(matchingIDs, id)
			}
		}

		if len(matchingIDs) == 0 {
			return nil
		}
	}

	return matchingIDs
}

// collectArtefactsByIDs looks up artefacts from the cache for each ID in the
// set.
//
// Takes ids (map[string]struct{}) which specifies the set of artefact IDs to
// look up.
//
// Returns []*ArtefactMeta which contains the cached artefacts
// found for the given IDs.
func (d *DAL) collectArtefactsByIDs(ctx context.Context, ids map[string]struct{}) []*registry_dto.ArtefactMeta {
	results := make([]*registry_dto.ArtefactMeta, 0, len(ids))
	for id := range ids {
		if artefact, found, _ := d.artefacts.GetIfPresent(ctx, id); found {
			results = append(results, artefact)
		}
	}
	return results
}

// upsertArtefactLocked inserts or updates an artefact. Caller must hold mu.
//
// Takes artefact (*registry_dto.ArtefactMeta) which is the artefact to store.
func (d *DAL) upsertArtefactLocked(ctx context.Context, artefact *registry_dto.ArtefactMeta) {
	if old, found, _ := d.artefacts.GetIfPresent(ctx, artefact.ID); found {
		d.removeArtefactIndexesLocked(old)
	}

	_ = d.artefacts.Set(ctx, artefact.ID, artefact)

	d.addArtefactIndexesLocked(artefact)
}

// deleteArtefactLocked removes an artefact. Caller must hold mu.
//
// Takes artefactID (string) which identifies the artefact to remove.
func (d *DAL) deleteArtefactLocked(ctx context.Context, artefactID string) {
	artefact, found, _ := d.artefacts.GetIfPresent(ctx, artefactID)
	if !found {
		return
	}

	d.removeArtefactIndexesLocked(artefact)
	_ = d.artefacts.Invalidate(ctx, artefactID)
}

// addArtefactIndexesLocked updates indexes for an artefact. Caller must hold
// mu.
//
// Takes artefact (*registry_dto.ArtefactMeta) which is the artefact to index.
func (d *DAL) addArtefactIndexesLocked(artefact *registry_dto.ArtefactMeta) {
	for i := range artefact.ActualVariants {
		if artefact.ActualVariants[i].StorageKey != "" {
			d.variantKeyIndex[artefact.ActualVariants[i].StorageKey] = artefact.ID
		}
		for j := range artefact.ActualVariants[i].Chunks {
			if artefact.ActualVariants[i].Chunks[j].StorageKey != "" {
				d.variantKeyIndex[artefact.ActualVariants[i].Chunks[j].StorageKey] = artefact.ID
			}
		}
	}

	for i := range artefact.DesiredProfiles {
		for tagKey, tagValue := range artefact.DesiredProfiles[i].Profile.ResultingTags.All() {
			indexKey := tagKey + tagKeySeparator + tagValue
			d.tagIndex.AddSingle(indexKey, artefact.ID)
		}
	}

	for i := range artefact.ActualVariants {
		for tagKey, tagValue := range artefact.ActualVariants[i].MetadataTags.All() {
			indexKey := tagKey + tagKeySeparator + tagValue
			d.tagIndex.AddSingle(indexKey, artefact.ID)
		}
	}
}

// removeArtefactIndexesLocked removes indexes for an artefact.
// Caller must hold mu.
//
// Takes artefact (*registry_dto.ArtefactMeta) which is the artefact whose
// indexes should be removed.
func (d *DAL) removeArtefactIndexesLocked(artefact *registry_dto.ArtefactMeta) {
	for i := range artefact.ActualVariants {
		if artefact.ActualVariants[i].StorageKey != "" {
			delete(d.variantKeyIndex, artefact.ActualVariants[i].StorageKey)
		}
		for j := range artefact.ActualVariants[i].Chunks {
			if artefact.ActualVariants[i].Chunks[j].StorageKey != "" {
				delete(d.variantKeyIndex, artefact.ActualVariants[i].Chunks[j].StorageKey)
			}
		}
	}

	for i := range artefact.DesiredProfiles {
		for tagKey, tagValue := range artefact.DesiredProfiles[i].Profile.ResultingTags.All() {
			indexKey := tagKey + tagKeySeparator + tagValue
			d.tagIndex.RemoveSingle(indexKey, artefact.ID)
		}
	}

	for i := range artefact.ActualVariants {
		for tagKey, tagValue := range artefact.ActualVariants[i].MetadataTags.All() {
			indexKey := tagKey + tagKeySeparator + tagValue
			d.tagIndex.RemoveSingle(indexKey, artefact.ID)
		}
	}
}

// WithCache injects an externally configured cache instance.
//
// This enables WAL persistence when the cache is created with
// PersistenceConfig. When provided, the DAL will not close the cache on
// shutdown. The caller is responsible for cache lifecycle management.
//
// Takes cache (cache_domain.ProviderPort) which is the cache to use.
//
// Returns Option which configures the DAL to use the provided cache.
func WithCache(cache cache_domain.ProviderPort[string, *registry_dto.ArtefactMeta]) Option {
	return func(d *DAL) {
		d.artefacts = cache
		d.ownsCache = false
	}
}

// NewOtterDAL creates a new in-memory registry DAL using otter cache.
//
// Takes config (Config) which specifies cache settings.
// Takes opts (...Option) which configures optional features like cache
// injection.
//
// Returns registry_dal.RegistryDALWithTx which is the configured DAL.
// Returns error when the cache cannot be created.
func NewOtterDAL(config Config, opts ...Option) (registry_dal.RegistryDALWithTx, error) {
	dal := &DAL{
		tagIndex:        provider_otter.NewTagIndex[string](),
		variantKeyIndex: make(map[string]string),
		blobRefs:        make(map[string]*blobRef),
		gcHints:         make([]registry_dto.GCHint, 0),
		ownsCache:       true,
		mu:              sync.RWMutex{},
	}

	for _, opt := range opts {
		opt(dal)
	}

	if dal.artefacts == nil {
		capacity := config.Capacity
		if capacity <= 0 {
			capacity = defaultCacheCapacity
		}

		cacheOpts := cache_dto.Options[string, *registry_dto.ArtefactMeta]{
			MaximumSize: int(capacity),
		}

		cache, err := provider_otter.OtterProviderFactory(cacheOpts)
		if err != nil {
			return nil, fmt.Errorf("creating otter cache: %w", err)
		}
		dal.artefacts = cache
	}

	return dal, nil
}
