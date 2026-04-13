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
	"iter"
	"strings"
	"sync/atomic"
	"time"

	"cloud.google.com/go/firestore"
	"golang.org/x/sync/singleflight"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_search"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
)

const (
	// fieldValue is the Firestore document field for the encoded value.
	fieldValue = "__value"

	// fieldTags is the Firestore document field for tag strings.
	fieldTags = "__tags"

	// fieldTTL is the Firestore document field for the TTL timestamp.
	fieldTTL = "__ttl"

	// fieldCreated is the Firestore document field for the creation timestamp.
	fieldCreated = "__created"

	// fieldUpdated is the Firestore document field for the last-update timestamp.
	fieldUpdated = "__updated"

	// fieldVersion is the Firestore document field for the optimistic concurrency
	// version counter.
	fieldVersion = "__version"

	// logKeyField is the attribute key used when logging Firestore cache keys.
	logKeyField = "key"

	// errMessageEncodeKey is the warning message logged when key encoding fails.
	errMessageEncodeKey = "Failed to encode key"

	// errFmtEncodeKey is the format string used when key encoding fails.
	errFmtEncodeKey = "failed to encode key: %w"

	// maxArrayContainsAny is the Firestore limit for array-contains-any queries.
	maxArrayContainsAny = 30

	// searchFieldPrefix is the prefix added to search field names when stored as
	// top-level Firestore document fields, to avoid collisions with internal
	// fields.
	searchFieldPrefix = "sf_"
)

// FirestoreAdapter implements the ProviderPort using a Firestore client. It
// supports generics by encoding keys to document IDs and using a type-driven
// EncodingRegistry for values.
type FirestoreAdapter[K comparable, V any] struct {
	// expiryCalculator sets the expiry time for each key; optional.
	expiryCalculator cache.ExpiryCalculator[K, V]

	// refreshCalculator calculates when entries become ready for background
	// refresh; optional.
	refreshCalculator cache.RefreshCalculator[K, V]

	// registry encodes values before they are stored.
	registry *cache.EncodingRegistry

	// client is the Firestore client for storage operations.
	client *firestore.Client

	// keyRegistry stores encoders for complex key types; nil uses fmt.Sprintf.
	keyRegistry *cache.EncodingRegistry

	// collection is the entries subcollection reference for this namespace.
	collection *firestore.CollectionRef

	// schema is the search schema for this cache; nil means search is disabled.
	schema *cache.SearchSchema

	// fieldExtractor extracts field values from cached values for search indexing.
	fieldExtractor *cache_search.FieldExtractor[V]

	// sf deduplicates concurrent loads for the same key.
	sf singleflight.Group

	// namespace is the namespace identifier for this cache instance.
	namespace string

	// ttl is the default time-to-live for cache entries.
	ttl time.Duration

	// operationTimeout is the time limit for a single Firestore operation.
	operationTimeout time.Duration

	// atomicOperationTimeout is the time limit for transaction operations.
	atomicOperationTimeout time.Duration

	// bulkOperationTimeout is the maximum time for bulk operations like
	// BulkGet and BulkSet.
	bulkOperationTimeout time.Duration

	// flushTimeout is the time limit for InvalidateAll operations.
	flushTimeout time.Duration

	// searchTimeout is the time limit for search operations.
	searchTimeout time.Duration

	// maxComputeRetries is the maximum number of retry attempts for
	// transactions in Compute methods.
	maxComputeRetries int

	// batchSize is the maximum number of documents to process in a single
	// batch write or bulk read.
	batchSize int

	// enableTTLClientCheck controls whether client-side TTL expiry checks
	// are performed on reads.
	enableTTLClientCheck bool

	// hits tracks the number of cache hits for local statistics.
	hits atomic.Uint64

	// misses tracks the number of cache misses for local statistics.
	misses atomic.Uint64
}

// _ checks that FirestoreAdapter implements the ProviderPort interface.
var _ cache.ProviderPort[any, any] = (*FirestoreAdapter[any, any])(nil)

// docRef returns a DocumentRef for the given key by encoding it to a document
// ID within the entries subcollection.
//
// Takes key (K) which is the cache key to resolve to a document reference.
//
// Returns *firestore.DocumentRef which is the document reference.
// Returns error when key encoding fails.
func (a *FirestoreAdapter[K, V]) docRef(key K) (*firestore.DocumentRef, error) {
	docID, err := a.encodeKey(key)
	if err != nil {
		return nil, err
	}
	return a.collection.Doc(docID), nil
}

// encodeKey converts a key of type K to a Firestore document ID string.
// Unlike Redis, the namespace is already encoded in the collection path, so
// the key is encoded without a namespace prefix.
//
// Firestore document IDs must not contain '/'. When a slash is present in the
// encoded key it is percent-encoded as %2F and reversed on decode.
//
// Takes key (K) which is the cache key to encode.
//
// Returns string which is the encoded document ID.
// Returns error when no encoder is registered for the key type or when
// marshalling fails.
func (a *FirestoreAdapter[K, V]) encodeKey(key K) (string, error) {
	encoded, err := cache_domain.EncodeKey(key, "", a.keyRegistry)
	if err != nil {
		return "", err
	}

	encoded = strings.ReplaceAll(encoded, "%", "%25")
	encoded = strings.ReplaceAll(encoded, "/", "%2F")
	return encoded, nil
}

// decodeKey converts a Firestore document ID string back to a key of type K.
//
// Takes keyString (string) which is the document ID to decode.
//
// Returns K which is the decoded key value.
// Returns error when decoding fails or no encoder is registered for the key
// type.
func (a *FirestoreAdapter[K, V]) decodeKey(keyString string) (K, error) {
	keyString = strings.ReplaceAll(keyString, "%2F", "/")
	keyString = strings.ReplaceAll(keyString, "%25", "%")
	return cache_domain.DecodeKey[K](keyString, "", a.keyRegistry)
}

// encodeValue encodes a value of type V to bytes using the registry.
//
// Takes value (V) which is the value to encode.
//
// Returns []byte which contains the encoded value.
// Returns error when no encoder is found for the value type or encoding fails.
func (a *FirestoreAdapter[K, V]) encodeValue(value V) ([]byte, error) {
	return cache_domain.EncodeValue(value, a.registry)
}

// decodeValue decodes bytes into a value of type V using the registry.
//
// Takes valBytes ([]byte) which contains the encoded data to decode.
//
// Returns V which is the decoded value.
// Returns error when the encoder cannot be found, unmarshalling fails, or type
// assertion fails.
func (a *FirestoreAdapter[K, V]) decodeValue(valBytes []byte) (V, error) {
	return cache_domain.DecodeValue[V](valBytes, a.registry)
}

// buildDocumentData creates the Firestore document map for a cache entry.
//
// Takes value (V) which is the value to store.
// Takes tags ([]string) which are the tags to associate with the entry.
// Takes ttl (time.Duration) which is the time-to-live for this entry.
//
// Returns map[string]any containing the document data.
// Returns error when value encoding fails.
func (a *FirestoreAdapter[K, V]) buildDocumentData(value V, tags []string, ttl time.Duration) (map[string]any, error) {
	valBytes, err := a.encodeValue(value)
	if err != nil {
		return nil, fmt.Errorf("failed to encode value: %w", err)
	}

	now := time.Now()
	data := map[string]any{
		fieldValue:   valBytes,
		fieldTags:    tags,
		fieldTTL:     now.Add(ttl),
		fieldCreated: now,
		fieldUpdated: now,
		fieldVersion: int64(1),
	}

	if a.schema != nil && a.fieldExtractor != nil {
		a.addSearchFields(data, value)
	}

	return data, nil
}

// addSearchFields extracts searchable field values from the cached value and
// stores them as top-level Firestore document fields with a "sf_" prefix. This
// allows native Firestore Where/OrderBy queries on these fields.
//
// Takes data (map[string]any) which is the document map to populate.
// Takes value (V) which is the cached value to extract fields from.
func (a *FirestoreAdapter[K, V]) addSearchFields(data map[string]any, value V) {
	for _, field := range a.schema.Fields {
		switch field.Type {
		case cache.FieldTypeTag:
			if v, ok := a.fieldExtractor.ExtractAny(value, field.Name); ok {
				data[searchFieldPrefix+field.Name] = cache_search.ToString(v)
			}
		case cache.FieldTypeNumeric:
			if v, ok := a.fieldExtractor.ExtractNumericValue(value, field.Name); ok {
				data[searchFieldPrefix+field.Name] = v
			}
		case cache.FieldTypeText:
			texts := a.fieldExtractor.ExtractTextFields(value)
			if len(texts) > 0 {
				data[searchFieldPrefix+field.Name] = strings.Join(texts, " ")
			}
		case cache.FieldTypeVector:
			if v, ok := a.fieldExtractor.ExtractVectorValue(value, field.Name); ok {
				data[searchFieldPrefix+field.Name] = v
			}
		}
	}
}

// isExpired checks whether a document's TTL timestamp has passed.
//
// Takes data (map[string]any) containing the document data.
//
// Returns bool which is true if the document is expired.
func (a *FirestoreAdapter[K, V]) isExpired(data map[string]any) bool {
	if !a.enableTTLClientCheck {
		return false
	}

	ttlVal, ok := data[fieldTTL]
	if !ok {
		return false
	}

	ttlTime, ok := ttlVal.(time.Time)
	if !ok {
		return false
	}

	return ttlTime.Before(time.Now())
}

// extractValue extracts and decodes the value from a Firestore document
// snapshot.
//
// Takes snap (*firestore.DocumentSnapshot) which is the document to read.
//
// Returns V which is the decoded value.
// Returns bool which is true if the value was successfully extracted.
// Returns error when decoding fails.
func (a *FirestoreAdapter[K, V]) extractValue(snap *firestore.DocumentSnapshot) (V, bool, error) {
	data := snap.Data()
	if data == nil {
		return *new(V), false, nil
	}

	if a.isExpired(data) {
		return *new(V), false, nil
	}

	valRaw, ok := data[fieldValue]
	if !ok {
		return *new(V), false, nil
	}

	valBytes, ok := valRaw.([]byte)
	if !ok {
		return *new(V), false, fmt.Errorf("unexpected value type in document %q: expected []byte, got %T", snap.Ref.ID, valRaw)
	}

	value, err := a.decodeValue(valBytes)
	if err != nil {
		return *new(V), false, fmt.Errorf("failed to decode value from document %q: %w", snap.Ref.ID, err)
	}

	return value, true, nil
}

// GetIfPresent retrieves a value from the cache if it exists, without blocking
// or loading.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to retrieve.
//
// Returns V which is the cached value, or zero value if not found.
// Returns bool which indicates whether the key was present in the cache.
// Returns error when the operation fails (e.g. network error).
func (a *FirestoreAdapter[K, V]) GetIfPresent(ctx context.Context, key K) (V, bool, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("firestore GetIfPresent exceeded %s timeout", a.operationTimeout))
	defer cancel()

	ref, err := a.docRef(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	snap, err := ref.Get(timeoutCtx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			a.misses.Add(1)
			return *new(V), false, nil
		}
		return *new(V), false, fmt.Errorf("firestore get failed for key %q: %w", ref.ID, err)
	}

	value, ok, decodeErr := a.extractValue(snap)
	if decodeErr != nil {
		return *new(V), false, decodeErr
	}

	if !ok {
		a.misses.Add(1)
		return *new(V), false, nil
	}

	a.hits.Add(1)
	return value, true, nil
}

// Get retrieves a value from the cache, loading it via the provided loader if
// not present.
//
// Takes key (K) which identifies the cached value to retrieve.
// Takes loader (Loader[K, V]) which loads the value if not already cached.
//
// Returns V which is the cached or newly loaded value.
// Returns error when key encoding fails, the loader fails, or type assertion
// fails.
func (a *FirestoreAdapter[K, V]) Get(ctx context.Context, key K, loader cache.Loader[K, V]) (V, error) {
	ctx, l := logger.From(ctx, log)
	docID, err := a.encodeKey(key)
	if err != nil {
		return *new(V), fmt.Errorf(errFmtEncodeKey, err)
	}

	result, err, _ := a.sf.Do(docID, func() (any, error) {
		if v, ok, getErr := a.GetIfPresent(ctx, key); getErr != nil {
			return nil, getErr
		} else if ok {
			return v, nil
		}

		loadedVal, loadErr := loader.Load(ctx, key)
		if loadErr != nil {
			return nil, loadErr
		}

		if setErr := a.Set(ctx, key, loadedVal); setErr != nil {
			l.Warn("Failed to cache loaded value", logger.Error(setErr))
		}
		return loadedVal, nil
	})

	if err != nil {
		return *new(V), err
	}
	value, ok := result.(V)
	if !ok {
		return *new(V), fmt.Errorf("type assertion failed: expected %T, got %T", *new(V), result)
	}
	return value, nil
}

// Set stores a key-value pair in the cache with optional tags for grouped
// invalidation.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes tags (...string) which provide optional grouping for bulk invalidation.
//
// Returns error when the operation fails.
func (a *FirestoreAdapter[K, V]) Set(ctx context.Context, key K, value V, tags ...string) error {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("firestore Set exceeded %s timeout", a.operationTimeout))
	defer cancel()

	ref, err := a.docRef(key)
	if err != nil {
		return fmt.Errorf(errFmtEncodeKey, err)
	}

	ttl := a.ttl
	if a.expiryCalculator != nil {
		entry := cache.Entry[K, V]{
			Key:            key,
			Value:          value,
			SnapshotAtNano: time.Now().UnixNano(),
		}
		ttl = a.expiryCalculator.ExpireAfterCreate(entry)
	}

	docData, err := a.buildDocumentData(value, tags, ttl)
	if err != nil {
		return err
	}

	if _, err := ref.Set(timeoutCtx, docData); err != nil {
		return fmt.Errorf("firestore set failed for key %q: %w", ref.ID, err)
	}

	return nil
}

// SetWithTTL stores a key-value pair with a custom expiry time for this entry.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes ttl (time.Duration) which sets how long the entry stays valid.
// Takes tags (...string) which links labels to the entry.
//
// Returns error when encoding, marshalling, or the Firestore operation fails.
func (a *FirestoreAdapter[K, V]) SetWithTTL(ctx context.Context, key K, value V, ttl time.Duration, tags ...string) error {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("firestore SetWithTTL exceeded %s timeout", a.operationTimeout))
	defer cancel()

	ref, err := a.docRef(key)
	if err != nil {
		return fmt.Errorf(errFmtEncodeKey, err)
	}

	docData, err := a.buildDocumentData(value, tags, ttl)
	if err != nil {
		return err
	}

	if _, err := ref.Set(timeoutCtx, docData); err != nil {
		return fmt.Errorf("firestore set with TTL failed for key %q: %w", ref.ID, err)
	}

	return nil
}

// Invalidate removes a key from the cache. Tags are stored inline in the
// document so they are automatically removed with the document.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to remove.
//
// Returns error when the operation fails.
func (a *FirestoreAdapter[K, V]) Invalidate(ctx context.Context, key K) error {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("firestore Invalidate exceeded %s timeout", a.operationTimeout))
	defer cancel()

	ref, err := a.docRef(key)
	if err != nil {
		return fmt.Errorf(errFmtEncodeKey, err)
	}

	if _, err := ref.Delete(timeoutCtx); err != nil {
		return fmt.Errorf("firestore delete failed for key %q: %w", ref.ID, err)
	}

	return nil
}

// SetExpiresAfter updates the time-to-live for an existing key by updating the
// __ttl field.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to update.
// Takes expiresAfter (time.Duration) which specifies the new time-to-live.
//
// Returns error when the operation fails.
func (a *FirestoreAdapter[K, V]) SetExpiresAfter(ctx context.Context, key K, expiresAfter time.Duration) error {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("firestore SetExpiresAfter exceeded %s timeout", a.operationTimeout))
	defer cancel()

	ref, err := a.docRef(key)
	if err != nil {
		return fmt.Errorf(errFmtEncodeKey, err)
	}

	newTTLTime := time.Now().Add(expiresAfter)
	if _, err := ref.Update(timeoutCtx, []firestore.Update{
		{Path: fieldTTL, Value: newTTLTime},
		{Path: fieldUpdated, Value: time.Now()},
	}); err != nil {
		return fmt.Errorf("firestore update TTL failed for key %q: %w", ref.ID, err)
	}

	return nil
}

// decodeDocument extracts and decodes both the key and value from a Firestore
// document snapshot.
//
// Returns K which is the decoded key.
// Returns V which is the decoded value.
// Returns bool which is true when both key and value were decoded successfully.
func (a *FirestoreAdapter[K, V]) decodeDocument(ctx context.Context, snap *firestore.DocumentSnapshot) (K, V, bool) {
	value, ok, decodeErr := a.extractValue(snap)
	if decodeErr != nil {
		_, l := logger.From(ctx, log)
		l.Trace("Failed to decode value during iteration",
			logger.String(logKeyField, snap.Ref.ID),
			logger.Error(decodeErr))
		return *new(K), *new(V), false
	}
	if !ok {
		return *new(K), *new(V), false
	}

	key, err := a.decodeKey(snap.Ref.ID)
	if err != nil {
		_, l := logger.From(ctx, log)
		l.Trace("Failed to decode key during iteration",
			logger.String(logKeyField, snap.Ref.ID),
			logger.Error(err))
		return *new(K), *new(V), false
	}

	return key, value, true
}

// All returns an iterator over all key-value pairs in the cache namespace.
//
// Returns iter.Seq2[K, V] which yields each key-value pair found in the
// namespace via Firestore document iteration.
func (a *FirestoreAdapter[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		ctx := context.Background()
		docIter := a.collection.Documents(ctx)
		defer docIter.Stop()

		for {
			snap, err := docIter.Next()
			if errors.Is(err, iterator.Done) {
				return
			}
			if err != nil {
				_, l := logger.From(ctx, log)
				l.Warn("Error iterating documents in All()", logger.Error(err))
				return
			}

			key, value, ok := a.decodeDocument(ctx, snap)
			if !ok {
				continue
			}

			if !yield(key, value) {
				return
			}
		}
	}
}

// Keys returns an iterator over all keys in the cache namespace.
//
// Returns iter.Seq[K] which yields each key found in the namespace.
func (a *FirestoreAdapter[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		for k := range a.All() {
			if !yield(k) {
				return
			}
		}
	}
}

// Values returns an iterator over all values in the cache namespace.
//
// Returns iter.Seq[V] which yields each value found in the namespace.
func (a *FirestoreAdapter[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, v := range a.All() {
			if !yield(v) {
				return
			}
		}
	}
}

// GetEntry retrieves the full entry metadata for a key including TTL
// information.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to retrieve.
//
// Returns Entry[K, V] which contains the value and metadata such as expiry
// time.
// Returns bool which indicates whether the key was found.
// Returns error when the operation fails.
func (a *FirestoreAdapter[K, V]) GetEntry(ctx context.Context, key K) (cache.Entry[K, V], bool, error) {
	return a.ProbeEntry(ctx, key)
}

// ProbeEntry retrieves entry metadata without affecting access patterns or
// TTL.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to probe.
//
// Returns Entry[K, V] which contains the value and metadata such as expiry
// time.
// Returns bool which indicates whether the key was found.
// Returns error when the operation fails.
func (a *FirestoreAdapter[K, V]) ProbeEntry(ctx context.Context, key K) (cache.Entry[K, V], bool, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("firestore ProbeEntry exceeded %s timeout", a.operationTimeout))
	defer cancel()

	ref, err := a.docRef(key)
	if err != nil {
		return cache.Entry[K, V]{}, false, fmt.Errorf(errFmtEncodeKey, err)
	}

	snap, err := ref.Get(timeoutCtx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return cache.Entry[K, V]{}, false, nil
		}
		return cache.Entry[K, V]{}, false, fmt.Errorf("firestore get failed for key %q: %w", ref.ID, err)
	}

	value, ok, decodeErr := a.extractValue(snap)
	if decodeErr != nil {
		return cache.Entry[K, V]{}, false, decodeErr
	}
	if !ok {
		return cache.Entry[K, V]{}, false, nil
	}

	data := snap.Data()
	var expiresAtNano int64
	if ttlVal, exists := data[fieldTTL]; exists {
		if ttlTime, timeOK := ttlVal.(time.Time); timeOK {
			expiresAtNano = ttlTime.UnixNano()
		}
	}

	entry := cache.Entry[K, V]{
		Key:               key,
		Value:             value,
		Weight:            0,
		ExpiresAtNano:     expiresAtNano,
		RefreshableAtNano: 0,
		SnapshotAtNano:    time.Now().UnixNano(),
	}

	return entry, true, nil
}

// EstimatedSize returns the approximate number of entries in the Firestore
// namespace by iterating over all document references.
//
// Returns int which is the count of documents, or zero if the query fails.
func (a *FirestoreAdapter[K, V]) EstimatedSize() int {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("firestore EstimatedSize exceeded %s timeout", a.operationTimeout))
	defer cancel()

	return a.estimatedSizeByIteration(ctx)
}

// estimatedSizeByIteration counts documents by iterating the collection. This
// is used as the primary counting mechanism since the Firestore aggregation API
// surface varies across SDK versions.
//
// Returns int which is the document count.
func (a *FirestoreAdapter[K, V]) estimatedSizeByIteration(ctx context.Context) int {
	_, l := logger.From(ctx, log)

	docIter := a.collection.Documents(ctx)
	defer docIter.Stop()

	count := 0
	for {
		_, err := docIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			l.Warn("Error counting documents in EstimatedSize", logger.Error(err))
			break
		}
		count++
	}

	return count
}

// Stats returns cache statistics from local atomic counters.
//
// Returns cache.Stats which contains hit and miss counts.
func (a *FirestoreAdapter[K, V]) Stats() cache.Stats {
	return cache.Stats{
		Hits:   a.hits.Load(),
		Misses: a.misses.Load(),
	}
}

// Close releases resources held by this adapter. The Firestore client
// lifecycle is managed by the provider, so this is a no-op.
//
// Returns error (always nil).
func (*FirestoreAdapter[K, V]) Close(_ context.Context) error {
	return nil
}

// GetMaximum returns 0 as Firestore does not have a configurable maximum
// capacity.
//
// Returns uint64 which is always 0.
func (*FirestoreAdapter[K, V]) GetMaximum() uint64 {
	return 0
}

// SetMaximum is not supported by the Firestore provider.
//
// Firestore does not have a maximum capacity concept.
func (*FirestoreAdapter[K, V]) SetMaximum(_ uint64) {
	_, l := logger.From(context.Background(), log)
	l.Warn("SetMaximum is not supported by the Firestore provider and will have no effect.")
}

// WeightedSize returns 0 as Firestore does not expose storage usage per
// collection.
//
// Returns uint64 which is always 0.
func (*FirestoreAdapter[K, V]) WeightedSize() uint64 {
	return 0
}

// Refresh asynchronously refreshes a single cache entry using the provided
// loader.
//
// Takes key (K) which identifies the cache entry to refresh.
// Takes loader (Loader[K, V]) which loads the fresh value for the given key.
//
// Returns <-chan LoadResult[V] which receives the loaded value or error once
// the background goroutine completes.
//
// Safe for concurrent use. Spawns a goroutine that loads the value and updates
// the cache. The returned channel is closed when the goroutine finishes.
func (a *FirestoreAdapter[K, V]) Refresh(ctx context.Context, key K, loader cache.Loader[K, V]) <-chan cache.LoadResult[V] {
	ctx, l := logger.From(ctx, log)
	resultChan := make(chan cache.LoadResult[V], 1)
	go func() {
		defer close(resultChan)
		defer goroutine.RecoverPanic(ctx, "cache.firestoreRefresh")
		value, err := loader.Load(ctx, key)
		if err == nil {
			if setErr := a.Set(ctx, key, value); setErr != nil {
				l.Warn("Failed to set value during refresh", logger.Error(setErr))
			}
		}
		resultChan <- cache.LoadResult[V]{Value: value, Err: err}
	}()
	return resultChan
}

// BulkRefresh updates several cache entries in the background using the bulk
// loader.
//
// Takes keys ([]K) which specifies the cache keys to refresh.
// Takes bulkLoader (BulkLoader) which loads values for the given keys.
//
// Safe for concurrent use. Starts a goroutine that runs the bulk loader and
// updates the cache. The goroutine finishes when all keys are loaded and
// stored.
func (a *FirestoreAdapter[K, V]) BulkRefresh(ctx context.Context, keys []K, bulkLoader cache.BulkLoader[K, V]) {
	ctx, l := logger.From(ctx, log)
	go func() {
		defer goroutine.RecoverPanic(ctx, "cache.firestoreBulkRefresh")
		loaded, err := bulkLoader.BulkLoad(ctx, keys)
		if err != nil {
			l.Warn("Bulk refresh failed", logger.Error(err))
			return
		}
		for k, v := range loaded {
			if setErr := a.Set(ctx, k, v); setErr != nil {
				l.Warn("Failed to set value during bulk refresh", logger.Error(setErr))
			}
		}
	}()
}

// SetRefreshableAfter is a no-op as Firestore does not natively support
// refresh scheduling.
//
// Returns error (always nil for this no-op implementation).
func (*FirestoreAdapter[K, V]) SetRefreshableAfter(ctx context.Context, _ K, _ time.Duration) error {
	_, l := logger.From(ctx, log)
	l.Internal("SetRefreshableAfter is not natively supported by the Firestore provider.")
	return nil
}
