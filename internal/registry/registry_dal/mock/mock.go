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

package mock

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"piko.sh/piko/internal/registry/registry_dal"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

var _ registry_dal.RegistryDALWithTx = (*RegistryDAL)(nil)

// defaultGCHintLimit is the default limit for GC hint queries when no limit is
// specified.
const defaultGCHintLimit = 100

// Behaviour defines settings that control how mock methods act during tests.
type Behaviour struct {
	// Error is the error to return instead of normal behaviour; nil means success.
	Error error

	// PanicMessage is the message to use when panicking; empty uses a default.
	PanicMessage string

	// Delay is the time to wait before running the method; 0 means no delay.
	Delay time.Duration

	// ShouldPanic indicates whether to trigger a panic during execution.
	ShouldPanic bool
}

// CallRecord stores details of a method call for test verification.
type CallRecord struct {
	// Timestamp is when the request or response occurred.
	Timestamp time.Time

	// Method is the name of the DAL method that was called.
	Method string

	// Args contains the arguments passed to the mock function call.
	Args []any
}

// RegistryDAL is a mock implementation of the RegistryDALWithTx interface.
// It stores artefacts and blob references in memory, and supports transaction
// simulation for testing purposes.
type RegistryDAL struct {
	// artefacts stores artefact metadata keyed by artefact ID.
	artefacts map[string]*registry_dto.ArtefactMeta

	// blobReferences maps storage keys to their blob reference entries.
	blobReferences map[string]*blobRefEntry

	// artefactsByStorageKey maps variant storage keys to artefact IDs.
	artefactsByStorageKey map[string]string

	// tagIndex maps tag keys to their values and the artefact IDs that have them.
	tagIndex map[string]map[string][]string

	// behaviours maps method names to their test behaviour settings.
	behaviours map[string]*Behaviour

	// gcHints stores garbage collection hints waiting to be processed.
	gcHints []registry_dto.GCHint

	// calls stores all recorded method calls for test verification.
	calls []CallRecord

	// mu guards concurrent access to behaviours and calls.
	mu sync.RWMutex

	// inTransaction indicates whether a simulated transaction is active.
	inTransaction bool
}

// blobRefEntry stores a blob reference and tracks how many times it is used.
type blobRefEntry struct {
	// Blob is the reference to the blob content in the registry.
	Blob registry_domain.BlobReference

	// RefCount is the number of references to this blob.
	RefCount int
}

// NewRegistryDAL creates a new mock data access layer instance.
//
// Returns *RegistryDAL which is ready for use in tests.
func NewRegistryDAL() *RegistryDAL {
	return &RegistryDAL{
		artefacts:             make(map[string]*registry_dto.ArtefactMeta),
		blobReferences:        make(map[string]*blobRefEntry),
		gcHints:               make([]registry_dto.GCHint, 0),
		artefactsByStorageKey: make(map[string]string),
		tagIndex:              make(map[string]map[string][]string),
		behaviours:            make(map[string]*Behaviour),
		calls:                 make([]CallRecord, 0),
	}
}

// SetBehaviour sets the mock behaviour for a specific method.
//
// Takes method (string) which is the name of the method to set up.
// Takes behaviour (*Behaviour) which defines how the method should respond.
//
// Safe for concurrent use.
func (m *RegistryDAL) SetBehaviour(method string, behaviour *Behaviour) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.behaviours[method] = behaviour
}

// ClearBehaviours removes all configured behaviours.
//
// Safe for concurrent use.
func (m *RegistryDAL) ClearBehaviours() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.behaviours = make(map[string]*Behaviour)
}

// GetCalls returns all recorded method calls.
//
// Returns []CallRecord which contains a copy of all calls made to the mock.
//
// Safe for concurrent use.
func (m *RegistryDAL) GetCalls() []CallRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	calls := make([]CallRecord, len(m.calls))
	copy(calls, m.calls)
	return calls
}

// ClearCalls removes all recorded calls.
//
// Safe for concurrent use.
func (m *RegistryDAL) ClearCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = make([]CallRecord, 0)
}

// GetCallCount returns the number of times a method was called.
//
// Takes method (string) which specifies the method name to count.
//
// Returns int which is the number of recorded calls for the given method.
//
// Safe for concurrent use.
func (m *RegistryDAL) GetCallCount(method string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, call := range m.calls {
		if call.Method == method {
			count++
		}
	}
	return count
}

// ClearData removes all stored data from the registry.
//
// Safe for concurrent use.
func (m *RegistryDAL) ClearData() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.artefacts = make(map[string]*registry_dto.ArtefactMeta)
	m.blobReferences = make(map[string]*blobRefEntry)
	m.gcHints = make([]registry_dto.GCHint, 0)
	m.artefactsByStorageKey = make(map[string]string)
	m.tagIndex = make(map[string]map[string][]string)
}

// HealthCheck implements registry_dal.RegistryDAL.
//
// Returns error when the configured behaviour for HealthCheck returns one.
//
// Safe for concurrent use.
func (m *RegistryDAL) HealthCheck(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordCall("HealthCheck")
	return m.executeBehaviour("HealthCheck")
}

// Close implements registry_dal.RegistryDAL.
//
// Returns error when the underlying close operation fails.
//
// Safe for concurrent use.
func (m *RegistryDAL) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordCall("Close")
	return m.executeBehaviour("Close")
}

// RunAtomic executes fn within a transaction.
//
// Takes fn (func(...)) which is the function to run inside the
// transaction, receiving a MetadataStore scoped to that transaction.
//
// Returns error when the transaction function fails or when configured
// mock behaviour returns an error.
func (m *RegistryDAL) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore registry_domain.MetadataStore) error) error {
	return m.withTransaction(ctx, func(ctx context.Context, transactionDAL registry_dal.RegistryDAL) error {
		store, ok := transactionDAL.(registry_domain.MetadataStore)
		if !ok {
			return errors.New("transaction DAL does not implement MetadataStore")
		}
		return fn(ctx, store)
	})
}

// IsInTransaction returns whether the mock is currently in a transaction.
//
// Returns bool which is true if a transaction is active, false otherwise.
//
// Safe for concurrent use.
func (m *RegistryDAL) IsInTransaction() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.inTransaction
}

// GetArtefact retrieves artefact metadata by ID. Implements
// registry_dal.MetadataDAL.
//
// Takes artefactID (string) which identifies the artefact to retrieve.
//
// Returns *registry_dto.ArtefactMeta which is a copy of the artefact metadata.
// Returns error when the artefact is not found or a set behaviour triggers an
// error.
//
// Safe for concurrent use; protected by a read lock.
func (m *RegistryDAL) GetArtefact(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.recordCall("GetArtefact", artefactID)
	if err := m.executeBehaviour("GetArtefact"); err != nil {
		return nil, err
	}

	artefact, exists := m.artefacts[artefactID]
	if !exists {
		return nil, registry_domain.ErrArtefactNotFound
	}

	return new(*artefact), nil
}

// GetMultipleArtefacts retrieves metadata for the specified artefact IDs.
// Implements registry_dal.MetadataDAL.
//
// Takes artefactIDs ([]string) which specifies the IDs of artefacts to fetch.
//
// Returns []*registry_dto.ArtefactMeta which contains the found artefacts.
// Returns error when a configured behaviour error is triggered.
//
// Safe for concurrent use; protected by a read lock.
func (m *RegistryDAL) GetMultipleArtefacts(_ context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.recordCall("GetMultipleArtefacts", artefactIDs)
	if err := m.executeBehaviour("GetMultipleArtefacts"); err != nil {
		return nil, err
	}

	results := make([]*registry_dto.ArtefactMeta, 0, len(artefactIDs))
	for _, id := range artefactIDs {
		if artefact, exists := m.artefacts[id]; exists {
			results = append(results, new(*artefact))
		}
	}

	return results, nil
}

// ListAllArtefactIDs implements registry_dal.MetadataDAL.
//
// Returns []string which contains all artefact IDs in the registry.
// Returns error when the set behaviour triggers an error.
//
// Safe for concurrent use; protected by a read lock.
func (m *RegistryDAL) ListAllArtefactIDs(_ context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.recordCall("ListAllArtefactIDs")
	if err := m.executeBehaviour("ListAllArtefactIDs"); err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(m.artefacts))
	for id := range m.artefacts {
		ids = append(ids, id)
	}

	return ids, nil
}

// SearchArtefacts searches for artefacts matching the given query criteria.
// Implements registry_dal.MetadataDAL.
//
// Takes query (registry_domain.SearchQuery) which specifies the search
// criteria including tag filters.
//
// Returns []*registry_dto.ArtefactMeta which contains the matching artefacts.
// Returns error when the query is empty or uses unsupported RediSearch syntax.
//
// Safe for concurrent use.
func (m *RegistryDAL) SearchArtefacts(_ context.Context, query registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.recordCall("SearchArtefacts", query)
	if err := m.executeBehaviour("SearchArtefacts"); err != nil {
		return nil, err
	}

	if query.RawRediSearchQuery != "" {
		return nil, registry_dal.ErrSearchUnsupported
	}

	if len(query.SimpleTagQuery) == 0 {
		return nil, errors.New("search query is empty")
	}

	matchingIDs := m.findMatchingIDsByTagIntersection(query.SimpleTagQuery)
	return m.collectArtefactsByIDs(matchingIDs), nil
}

// SearchArtefactsByTagValues implements registry_dal.MetadataDAL.
//
// Takes tagKey (string) which specifies the tag key to search for.
// Takes tagValues ([]string) which specifies the tag values to match against.
//
// Returns []*registry_dto.ArtefactMeta which contains copies of matching
// artefacts.
// Returns error when the configured behaviour produces an error.
//
// Safe for concurrent use; uses a read lock during the search.
func (m *RegistryDAL) SearchArtefactsByTagValues(_ context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.recordCall("SearchArtefactsByTagValues", tagKey, tagValues)
	if err := m.executeBehaviour("SearchArtefactsByTagValues"); err != nil {
		return nil, err
	}

	if len(tagValues) == 0 {
		return []*registry_dto.ArtefactMeta{}, nil
	}

	matchingIDs := make(map[string]struct{})

	valueMap, keyExists := m.tagIndex[tagKey]
	if !keyExists {
		return []*registry_dto.ArtefactMeta{}, nil
	}

	for _, tagValue := range tagValues {
		if ids, valueExists := valueMap[tagValue]; valueExists {
			for _, id := range ids {
				matchingIDs[id] = struct{}{}
			}
		}
	}

	results := make([]*registry_dto.ArtefactMeta, 0, len(matchingIDs))
	for id := range matchingIDs {
		if artefact, exists := m.artefacts[id]; exists {
			results = append(results, new(*artefact))
		}
	}

	return results, nil
}

// FindArtefactByVariantStorageKey retrieves artefact metadata by its variant
// storage key. Implements registry_dal.MetadataDAL.
//
// Takes storageKey (string) which identifies the variant's storage location.
//
// Returns *registry_dto.ArtefactMeta which contains the artefact metadata.
// Returns error when the storage key is not found or the artefact is missing.
//
// Safe for concurrent use; protected by a read lock.
func (m *RegistryDAL) FindArtefactByVariantStorageKey(_ context.Context, storageKey string) (*registry_dto.ArtefactMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.recordCall("FindArtefactByVariantStorageKey", storageKey)
	if err := m.executeBehaviour("FindArtefactByVariantStorageKey"); err != nil {
		return nil, err
	}

	artefactID, exists := m.artefactsByStorageKey[storageKey]
	if !exists {
		return nil, registry_domain.ErrArtefactNotFound
	}

	artefact, exists := m.artefacts[artefactID]
	if !exists {
		return nil, registry_domain.ErrArtefactNotFound
	}

	return new(*artefact), nil
}

// PopGCHints retrieves and removes garbage collection hints from the store.
// Implements registry_dal.MetadataDAL.
//
// Takes limit (int) which specifies the maximum number of hints to return.
// When limit is zero or negative, a default limit is used.
//
// Returns []registry_dto.GCHint which contains the removed hints.
// Returns error when the configured behaviour fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *RegistryDAL) PopGCHints(_ context.Context, limit int) ([]registry_dto.GCHint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("PopGCHints", limit)
	if err := m.executeBehaviour("PopGCHints"); err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = defaultGCHintLimit
	}

	if len(m.gcHints) == 0 {
		return []registry_dto.GCHint{}, nil
	}

	popCount := min(limit, len(m.gcHints))

	hints := make([]registry_dto.GCHint, popCount)
	copy(hints, m.gcHints[:popCount])
	m.gcHints = m.gcHints[popCount:]

	return hints, nil
}

// AtomicUpdate applies a set of actions as a single atomic operation.
// Implements registry_dal.MetadataDAL.
//
// Takes actions ([]registry_dto.AtomicAction) which specifies the operations
// to perform atomically.
//
// Returns error when an action type is unrecognised or a configured behaviour
// fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *RegistryDAL) AtomicUpdate(_ context.Context, actions []registry_dto.AtomicAction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("AtomicUpdate", actions)
	if err := m.executeBehaviour("AtomicUpdate"); err != nil {
		return err
	}

	for _, action := range actions {
		switch action.Type {
		case registry_dto.ActionTypeUpsertArtefact:
			m.upsertArtefactInternal(action.Artefact)
		case registry_dto.ActionTypeDeleteArtefact:
			m.deleteArtefactInternal(action.ArtefactID)
		case registry_dto.ActionTypeAddGCHints:
			m.gcHints = append(m.gcHints, action.GCHints...)
		default:
			return fmt.Errorf("unrecognised atomic action type: %s", action.Type)
		}
	}

	return nil
}

// IncrementBlobRefCount implements registry_dal.MetadataDAL.
//
// Takes blob (registry_domain.BlobReference) which identifies the blob to
// increment.
//
// Returns int which is the new reference count after incrementing.
// Returns error when the configured behaviour produces an error.
//
// Safe for concurrent use; protected by a mutex.
func (m *RegistryDAL) IncrementBlobRefCount(_ context.Context, blob registry_domain.BlobReference) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("IncrementBlobRefCount", blob)
	if err := m.executeBehaviour("IncrementBlobRefCount"); err != nil {
		return 0, err
	}

	entry, exists := m.blobReferences[blob.StorageKey]
	if !exists {
		entry = &blobRefEntry{
			Blob:     blob,
			RefCount: 0,
		}
		m.blobReferences[blob.StorageKey] = entry
	}

	entry.RefCount++
	return entry.RefCount, nil
}

// DecrementBlobRefCount decrements the reference count for a blob and removes
// it when the count reaches zero. Implements registry_dal.MetadataDAL.
//
// Takes storageKey (string) which identifies the blob to decrement.
//
// Returns int which is the new reference count, or zero if the blob was
// removed.
// Returns bool which is true when the blob was removed due to zero references.
// Returns error when the blob reference is not found.
//
// Safe for concurrent use; protected by a mutex.
func (m *RegistryDAL) DecrementBlobRefCount(_ context.Context, storageKey string) (int, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("DecrementBlobRefCount", storageKey)
	if err := m.executeBehaviour("DecrementBlobRefCount"); err != nil {
		return 0, false, err
	}

	entry, exists := m.blobReferences[storageKey]
	if !exists {
		return 0, false, registry_domain.ErrBlobReferenceNotFound
	}

	entry.RefCount--
	if entry.RefCount <= 0 {
		delete(m.blobReferences, storageKey)
		return 0, true, nil
	}

	return entry.RefCount, false, nil
}

// GetBlobRefCount returns the reference count for a blob. Implements
// registry_dal.MetadataDAL.
//
// Takes storageKey (string) which identifies the blob to look up.
//
// Returns int which is the reference count, or zero if the blob does not
// exist.
// Returns error when the configured mock behaviour fails.
//
// Safe for concurrent use; protected by a read lock.
func (m *RegistryDAL) GetBlobRefCount(_ context.Context, storageKey string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.recordCall("GetBlobRefCount", storageKey)
	if err := m.executeBehaviour("GetBlobRefCount"); err != nil {
		return 0, err
	}

	entry, exists := m.blobReferences[storageKey]
	if !exists {
		return 0, nil
	}

	return entry.RefCount, nil
}

// SetArtefact is a test helper to directly set an artefact in the mock.
//
// Takes artefact (*registry_dto.ArtefactMeta) which is the artefact to store.
//
// Safe for concurrent use.
func (m *RegistryDAL) SetArtefact(artefact *registry_dto.ArtefactMeta) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upsertArtefactInternal(artefact)
}

// SetBlobRefCount sets a blob reference count directly for testing.
//
// Takes storageKey (string) which identifies the blob in storage.
// Takes blob (registry_domain.BlobReference) which is the blob to store.
// Takes refCount (int) which is the reference count to set.
//
// Safe for concurrent use; protected by a mutex.
func (m *RegistryDAL) SetBlobRefCount(storageKey string, blob registry_domain.BlobReference, refCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blobReferences[storageKey] = &blobRefEntry{
		Blob:     blob,
		RefCount: refCount,
	}
}

// AddGCHint adds a GC hint directly to the registry.
// This is a test helper.
//
// Takes hint (registry_dto.GCHint) which specifies the garbage collection hint
// to add.
//
// Safe for concurrent use.
func (m *RegistryDAL) AddGCHint(hint registry_dto.GCHint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gcHints = append(m.gcHints, hint)
}

// GetArtefactCount returns the number of artefacts stored in the mock.
//
// Returns int which is the count of stored artefacts.
//
// Safe for concurrent use.
func (m *RegistryDAL) GetArtefactCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.artefacts)
}

// GetGCHintCount returns the number of GC hints in the mock.
//
// Returns int which is the count of GC hints currently stored.
//
// Safe for concurrent use.
func (m *RegistryDAL) GetGCHintCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.gcHints)
}

// withTransaction is an internal helper used by RunAtomic.
//
// Takes operation (func(...)) which is the function to execute within
// the transaction. The function receives a transaction-isolated copy
// of the DAL.
//
// Returns error when the transaction function fails or when configured
// mock behaviour returns an error.
//
// Safe for concurrent use. Holds a mutex lock for the duration of the transaction.
func (m *RegistryDAL) withTransaction(ctx context.Context, operation func(ctx context.Context, dal registry_dal.RegistryDAL) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("withTransaction", operation)
	if err := m.executeBehaviour("withTransaction"); err != nil {
		return err
	}

	txMock := m.createTransactionCopy()

	err := operation(ctx, txMock)
	if err != nil {
		return err
	}

	m.commitTransaction(txMock)

	return nil
}

// recordCall logs a method call for later verification.
//
// Takes method (string) which specifies the name of the method being called.
// Takes arguments (...any) which provides the arguments passed to the method.
func (m *RegistryDAL) recordCall(method string, arguments ...any) {
	m.calls = append(m.calls, CallRecord{
		Method:    method,
		Args:      arguments,
		Timestamp: time.Now(),
	})
}

// executeBehaviour checks for configured behaviour and executes it.
//
// Takes method (string) which identifies the method whose behaviour to execute.
//
// Returns error when the configured behaviour specifies an error to return.
//
// Panics when the configured behaviour has ShouldPanic set to true.
func (m *RegistryDAL) executeBehaviour(method string) error {
	if behaviour, exists := m.behaviours[method]; exists {
		if behaviour.Delay > 0 {
			time.Sleep(behaviour.Delay)
		}

		if behaviour.ShouldPanic {
			message := behaviour.PanicMessage
			if message == "" {
				message = fmt.Sprintf("Mock panic in %s", method)
			}
			panic(message)
		}

		if behaviour.Error != nil {
			return behaviour.Error
		}
	}
	return nil
}

// createTransactionCopy creates a deep copy of the mock for transaction
// isolation.
//
// Returns *RegistryDAL which is the isolated copy with inTransaction set to
// true.
func (m *RegistryDAL) createTransactionCopy() *RegistryDAL {
	txMock := &RegistryDAL{
		artefacts:             make(map[string]*registry_dto.ArtefactMeta),
		blobReferences:        make(map[string]*blobRefEntry),
		gcHints:               make([]registry_dto.GCHint, len(m.gcHints)),
		artefactsByStorageKey: make(map[string]string),
		tagIndex:              make(map[string]map[string][]string),
		behaviours:            make(map[string]*Behaviour),
		calls:                 make([]CallRecord, 0),
		inTransaction:         true,
	}

	for k, v := range m.artefacts {
		txMock.artefacts[k] = new(*v)
	}

	for k, v := range m.blobReferences {
		txMock.blobReferences[k] = new(*v)
	}

	copy(txMock.gcHints, m.gcHints)

	maps.Copy(txMock.artefactsByStorageKey, m.artefactsByStorageKey)

	for tagKey, valueMap := range m.tagIndex {
		txMock.tagIndex[tagKey] = make(map[string][]string)
		for tagValue, ids := range valueMap {
			txMock.tagIndex[tagKey][tagValue] = append([]string{}, ids...)
		}
	}

	for k, v := range m.behaviours {
		txMock.behaviours[k] = new(*v)
	}

	return txMock
}

// commitTransaction applies changes from a transaction back to the main mock.
//
// Takes txMock (*RegistryDAL) which provides the transaction state to commit.
func (m *RegistryDAL) commitTransaction(txMock *RegistryDAL) {
	m.artefacts = txMock.artefacts
	m.blobReferences = txMock.blobReferences
	m.gcHints = txMock.gcHints
	m.artefactsByStorageKey = txMock.artefactsByStorageKey
	m.tagIndex = txMock.tagIndex

	m.calls = append(m.calls, txMock.calls...)
}

// findMatchingIDsByTagIntersection finds artefact IDs that match all tag
// key-value pairs.
//
// Takes tagQuery (map[string]string) which specifies the tag key-value pairs
// to match against.
//
// Returns map[string]struct{} which contains the matching artefact IDs, or an
// empty map if no matches are found.
func (m *RegistryDAL) findMatchingIDsByTagIntersection(tagQuery map[string]string) map[string]struct{} {
	var matchingIDs map[string]struct{}

	for tagKey, tagValue := range tagQuery {
		ids := m.getIDsForTag(tagKey, tagValue)
		if ids == nil {
			return map[string]struct{}{}
		}

		matchingIDs = m.intersectIDSets(matchingIDs, ids)
		if len(matchingIDs) == 0 {
			return map[string]struct{}{}
		}
	}

	return matchingIDs
}

// getIDsForTag returns the artefact IDs for a tag key-value pair.
//
// Takes tagKey (string) which specifies the tag key to look up.
// Takes tagValue (string) which specifies the tag value to match.
//
// Returns []string which contains the matching artefact IDs, or nil if the
// tag key or value does not exist.
func (m *RegistryDAL) getIDsForTag(tagKey, tagValue string) []string {
	valueMap, keyExists := m.tagIndex[tagKey]
	if !keyExists {
		return nil
	}
	ids, valueExists := valueMap[tagValue]
	if !valueExists {
		return nil
	}
	return ids
}

// intersectIDSets finds the common IDs between a set and a slice.
// If existing is nil, creates a new set from the ids slice.
//
// Takes existing (map[string]struct{}) which is the current set to check
// against, or nil to create a new set.
// Takes ids ([]string) which contains the IDs to match or add.
//
// Returns map[string]struct{} which contains only IDs found in both inputs,
// or all IDs if existing was nil.
func (*RegistryDAL) intersectIDSets(existing map[string]struct{}, ids []string) map[string]struct{} {
	if existing == nil {
		newSet := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			newSet[id] = struct{}{}
		}
		return newSet
	}

	newSet := make(map[string]struct{})
	for _, id := range ids {
		if _, exists := existing[id]; exists {
			newSet[id] = struct{}{}
		}
	}
	return newSet
}

// collectArtefactsByIDs collects artefact metadata for the given IDs.
//
// Takes ids (map[string]struct{}) which specifies the artefact IDs to collect.
//
// Returns []*registry_dto.ArtefactMeta which contains copies of the matching
// artefacts.
func (m *RegistryDAL) collectArtefactsByIDs(ids map[string]struct{}) []*registry_dto.ArtefactMeta {
	results := make([]*registry_dto.ArtefactMeta, 0, len(ids))
	for id := range ids {
		if artefact, exists := m.artefacts[id]; exists {
			results = append(results, new(*artefact))
		}
	}
	return results
}

// upsertArtefactInternal stores or updates an artefact without locking.
//
// Takes artefact (*registry_dto.ArtefactMeta) which is the artefact to store.
func (m *RegistryDAL) upsertArtefactInternal(artefact *registry_dto.ArtefactMeta) {
	if existing, exists := m.artefacts[artefact.ID]; exists {
		m.removeArtefactIndexEntries(existing)
	}

	artCopy := *artefact
	m.artefacts[artefact.ID] = &artCopy

	m.addArtefactIndexEntries(&artCopy)
}

// deleteArtefactInternal removes an artefact without getting a lock.
//
// Takes artefactID (string) which identifies the artefact to remove.
func (m *RegistryDAL) deleteArtefactInternal(artefactID string) {
	if existing, exists := m.artefacts[artefactID]; exists {
		m.removeArtefactIndexEntries(existing)
	}
	delete(m.artefacts, artefactID)
}

// addArtefactIndexEntries adds index entries for an artefact.
//
// Takes artefact (*registry_dto.ArtefactMeta) which contains the artefact
// metadata to be indexed.
func (m *RegistryDAL) addArtefactIndexEntries(artefact *registry_dto.ArtefactMeta) {
	for i := range artefact.ActualVariants {
		v := &artefact.ActualVariants[i]
		m.artefactsByStorageKey[v.StorageKey] = artefact.ID

		for key, value := range v.MetadataTags.All() {
			if m.tagIndex[key] == nil {
				m.tagIndex[key] = make(map[string][]string)
			}
			m.tagIndex[key][value] = append(m.tagIndex[key][value], artefact.ID)
		}
	}
}

// removeArtefactIndexEntries removes index entries for an artefact.
//
// Takes artefact (*registry_dto.ArtefactMeta) which specifies the artefact
// whose index entries should be removed.
func (m *RegistryDAL) removeArtefactIndexEntries(artefact *registry_dto.ArtefactMeta) {
	for i := range artefact.ActualVariants {
		v := &artefact.ActualVariants[i]
		delete(m.artefactsByStorageKey, v.StorageKey)

		for key, value := range v.MetadataTags.All() {
			m.removeIDFromTagIndex(key, value, artefact.ID)
		}
	}
}

// removeIDFromTagIndex removes an artefact ID from a specific tag key-value
// pair in the index.
//
// Takes tagKey (string) which identifies the tag category.
// Takes tagValue (string) which specifies the tag value within the category.
// Takes artefactID (string) which is the ID to remove from the index.
func (m *RegistryDAL) removeIDFromTagIndex(tagKey, tagValue, artefactID string) {
	valueMap, keyExists := m.tagIndex[tagKey]
	if !keyExists {
		return
	}

	ids := valueMap[tagValue]
	newIDs := filterOutID(ids, artefactID)

	if len(newIDs) == 0 {
		delete(valueMap, tagValue)
	} else {
		valueMap[tagValue] = newIDs
	}
}

// filterOutID returns a new slice with the given ID removed.
//
// Takes ids ([]string) which is the slice of IDs to filter.
// Takes excludeID (string) which is the ID to remove.
//
// Returns []string which is a new slice with all IDs except the excluded one.
func filterOutID(ids []string, excludeID string) []string {
	newIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != excludeID {
			newIDs = append(newIDs, id)
		}
	}
	return newIDs
}
