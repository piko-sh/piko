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

package cache_provider_valkey_cluster

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/valkey-io/valkey-go"
	"golang.org/x/sync/singleflight"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
)

const (
	// scanBatchSize is the number of keys to fetch in each Valkey SCAN call.
	scanBatchSize = 100

	// logKeyField is the attribute key used when logging Valkey cache keys.
	logKeyField = "key"

	// logKeyNode is the attribute key used when logging cluster node addresses.
	logKeyNode = "node"

	// maxTransactionCommands is the maximum number of commands in a
	// MULTI/EXEC transaction (MULTI + action + EXEC).
	maxTransactionCommands = 3

	// errFmtEncodeKey is the format string used when key encoding fails.
	errFmtEncodeKey = "failed to encode key: %w"
)

// ValkeyClusterAdapter implements the ProviderPort using a Valkey Cluster
// client. It supports generics by encoding keys to strings and using a
// type-driven EncodingRegistry for values.
//
// IMPORTANT CLUSTER CONSTRAINTS:
//   - Multi-key operations (MGET, MSET) only work if all keys hash to the same
//     slot
//   - Tag operations use hash tags {...} to ensure same-slot placement
//   - WATCH/MULTI/EXEC transactions are scoped to single nodes via hash tags
//   - Some operations may be slower due to cross-node coordination
type ValkeyClusterAdapter[K comparable, V any] struct {
	// expiryCalculator calculates expiry durations for each key; optional.
	expiryCalculator cache.ExpiryCalculator[K, V]

	// refreshCalculator calculates when entries become eligible for background
	// refresh; optional.
	refreshCalculator cache.RefreshCalculator[K, V]

	// registry stores the encoding registry used to encode values.
	registry *cache.EncodingRegistry

	// client is the Valkey Cluster client used for cluster operations.
	client valkey.Client

	// keyRegistry handles complex key types; defaults to fmt.Sprintf if nil.
	keyRegistry *cache.EncodingRegistry

	// schema defines the search schema for this cache; nil means search is off.
	schema *cache.SearchSchema

	// sf deduplicates concurrent loads for the same key.
	sf singleflight.Group

	// namespace is the prefix added to all keys in Valkey.
	namespace string

	// indexName is the Valkey Search index name for this namespace.
	indexName string

	// ttl is the default time-to-live for cache entries.
	ttl time.Duration

	// operationTimeout is the maximum duration for Valkey operations; 0 means no
	// limit.
	operationTimeout time.Duration

	// atomicOperationTimeout is the maximum duration for WATCH/MULTI/EXEC
	// operations.
	atomicOperationTimeout time.Duration

	// bulkOperationTimeout is the timeout for bulk operations such as MGET, MSET,
	// and pipelines.
	bulkOperationTimeout time.Duration

	// flushTimeout is the timeout for InvalidateAll flush operations.
	flushTimeout time.Duration

	// searchTimeout is the time limit for FT.SEARCH operations.
	searchTimeout time.Duration

	// maxComputeRetries is the maximum number of retries for optimistic locking
	// in Compute methods.
	maxComputeRetries int

	// If true and no namespace is set, InvalidateAll uses FLUSHDB.
	// If false, InvalidateAll is blocked without a namespace for safety.
	allowUnsafeFLUSHDB bool

	// indexCreated indicates whether the search index has been created.
	indexCreated bool
}

var _ cache.ProviderPort[any, any] = (*ValkeyClusterAdapter[any, any])(nil)

// encodeKey converts a key of type K to a namespace-prefixed Valkey key string
// using the shared encoding logic in cache_domain.
//
// Takes key (K) which is the cache key to encode.
//
// Returns string which is the encoded Valkey key, with namespace prefix if set.
// Returns error when no encoder is registered for the key type or when
// marshalling fails.
func (a *ValkeyClusterAdapter[K, V]) encodeKey(key K) (string, error) {
	return cache_domain.EncodeKey(key, a.namespace, a.keyRegistry)
}

// decodeKey converts a Valkey key string back to a key of type K using the
// shared decoding logic in cache_domain.
//
// Takes keyString (string) which is the Valkey key to decode.
//
// Returns K which is the decoded key value.
// Returns error when the namespace prefix is missing, decoding fails, or no
// encoder is registered for the key type.
func (a *ValkeyClusterAdapter[K, V]) decodeKey(keyString string) (K, error) {
	return cache_domain.DecodeKey[K](keyString, a.namespace, a.keyRegistry)
}

// GetIfPresent retrieves a value from the cache if it exists, without blocking
// or loading. When SearchSchema is configured, reads from JSON storage.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to retrieve.
//
// Returns V which is the cached value, or the zero value if not found.
// Returns bool which indicates whether the key was present in the cache.
// Returns error when the operation fails (e.g. network error).
func (a *ValkeyClusterAdapter[K, V]) GetIfPresent(ctx context.Context, key K) (V, bool, error) {
	if err := ctx.Err(); err != nil {
		return *new(V), false, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("valkey cluster GetIfPresent exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	if a.needsJSONStorage() {
		v, ok := a.getJSONValue(timeoutCtx, keyString)
		return v, ok, nil
	}

	value, err := a.client.Do(timeoutCtx, a.client.B().Get().Key(keyString).Build()).AsBytes()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return *new(V), false, nil
		}
		return *new(V), false, fmt.Errorf("valkey cluster get failed for key %s: %w", keyString, err)
	}

	var v V
	encoder, err := a.registry.GetByType(reflect.TypeOf(v))
	if err != nil {
		return *new(V), false, fmt.Errorf("failed to find encoder for type: %w", err)
	}

	unmarshalled, err := encoder.UnmarshalAny(value)
	if err != nil {
		return *new(V), false, fmt.Errorf("failed to unmarshal value from valkey cluster: %w", err)
	}

	result, ok := unmarshalled.(V)
	if !ok {
		return *new(V), false, fmt.Errorf("type assertion failed after unmarshal for key %s", keyString)
	}

	return result, true, nil
}

// Get retrieves a value from the cache, loading it via the provided loader
// if not present.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to retrieve.
// Takes loader (Loader[K, V]) which loads the value on cache miss.
//
// Returns V which is the cached or newly loaded value.
// Returns error when key encoding fails, the loader fails, or type assertion
// fails.
func (a *ValkeyClusterAdapter[K, V]) Get(ctx context.Context, key K, loader cache.Loader[K, V]) (V, error) {
	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), fmt.Errorf(errFmtEncodeKey, err)
	}

	result, err, _ := a.sf.Do(keyString, func() (any, error) {
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
			return nil, fmt.Errorf("failed to store loaded value: %w", setErr)
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
// invalidation. When a SearchSchema is configured, values are stored as JSON
// for Valkey Search indexing.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes tags (...string) which associate the key with groups for bulk removal.
//
// Returns error when the operation fails.
func (a *ValkeyClusterAdapter[K, V]) Set(ctx context.Context, key K, value V, tags ...string) error {
	ctx, l := logger.From(ctx, log)

	if err := ctx.Err(); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("valkey cluster Set exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return fmt.Errorf(errFmtEncodeKey, err)
	}

	ttl := a.ttl
	if a.expiryCalculator != nil {
		entry := cache.Entry[K, V]{
			Key:               key,
			Value:             value,
			Weight:            0,
			ExpiresAtNano:     0,
			RefreshableAtNano: 0,
			SnapshotAtNano:    time.Now().UnixNano(),
		}
		ttl = a.expiryCalculator.ExpireAfterCreate(entry)
	}

	if a.needsJSONStorage() {
		a.indexDocument(timeoutCtx, keyString, value)
		if err := a.client.Do(timeoutCtx, a.client.B().Expire().Key(keyString).Seconds(int64(ttl.Seconds())).Build()).Error(); err != nil {
			l.Warn("Failed to set TTL on JSON document", logger.String(logKeyField, keyString), logger.Error(err))
		}
	} else {
		encoder, err := a.registry.Get(value)
		if err != nil {
			return fmt.Errorf("failed to find encoder for value: %w", err)
		}

		valBytes, err := encoder.MarshalAny(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value for valkey cluster: %w", err)
		}

		if err := a.client.Do(timeoutCtx, a.client.B().Set().Key(keyString).Value(string(valBytes)).Px(ttl).Build()).Error(); err != nil {
			return fmt.Errorf("valkey cluster set failed: %w", err)
		}
	}

	if err := addTagsToKey(timeoutCtx, a.client, a.namespace, keyString, tags); err != nil {
		l.Warn("Failed to add tags to key", logger.String(logKeyField, keyString), logger.Error(err))
	}

	return nil
}

// SetWithTTL stores a key-value pair with a specific time-to-live duration.
//
// Takes key (K) which is the cache key to store.
// Takes value (V) which is the value to cache.
// Takes ttl (time.Duration) which specifies how long the entry remains valid.
// Takes tags (...string) which are optional tags to associate with the key.
//
// Returns error when encoding fails, marshalling fails, or the Valkey operation
// fails.
func (a *ValkeyClusterAdapter[K, V]) SetWithTTL(ctx context.Context, key K, value V, ttl time.Duration, tags ...string) error {
	ctx, l := logger.From(ctx, log)

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("valkey cluster SetWithTTL exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return fmt.Errorf(errFmtEncodeKey, err)
	}

	encoder, err := a.registry.Get(value)
	if err != nil {
		l.Warn("Failed to find encoder for value", logger.String(logKeyField, keyString), logger.Error(err))
		return fmt.Errorf("encoder not found: %w", err)
	}

	valBytes, err := encoder.MarshalAny(value)
	if err != nil {
		l.Warn("Failed to marshal value for Valkey Cluster", logger.String(logKeyField, keyString), logger.Error(err))
		return fmt.Errorf("marshal failed: %w", err)
	}

	if err := a.client.Do(timeoutCtx, a.client.B().Set().Key(keyString).Value(string(valBytes)).Px(ttl).Build()).Error(); err != nil {
		l.Warn("Valkey Cluster Set with TTL failed", logger.String(logKeyField, keyString), logger.Error(err))
		return fmt.Errorf("valkey set failed: %w", err)
	}

	if err := addTagsToKey(timeoutCtx, a.client, a.namespace, keyString, tags); err != nil {
		l.Warn("Failed to add tags to key", logger.String(logKeyField, keyString), logger.Error(err))
	}

	return nil
}

// prepareBulkSetItem encodes a key-value pair and calculates its TTL.
//
// Takes key (K) which is the cache key to encode.
// Takes value (V) which is the value to marshal for storage.
// Takes defaultTTL (time.Duration) which is the fallback TTL when no expiry
// calculator is configured.
//
// Returns string which is the encoded key.
// Returns []byte which is the marshalled value.
// Returns time.Duration which is the calculated TTL for the entry.
// Returns bool which is true on success, or false if encoding or marshalling
// fails.
func (a *ValkeyClusterAdapter[K, V]) prepareBulkSetItem(ctx context.Context, key K, value V, defaultTTL time.Duration) (string, []byte, time.Duration, bool) {
	_, l := logger.From(ctx, log)

	keyString, err := a.encodeKey(key)
	if err != nil {
		l.Warn("Failed to encode key in BulkSet, skipping", logger.Error(err))
		return "", nil, 0, false
	}

	encoder, err := a.registry.Get(value)
	if err != nil {
		l.Warn("Failed to find encoder for value in BulkSet",
			logger.String(logKeyField, keyString), logger.Error(err))
		return "", nil, 0, false
	}

	valBytes, err := encoder.MarshalAny(value)
	if err != nil {
		l.Warn("Failed to marshal value for Valkey Cluster in BulkSet",
			logger.String(logKeyField, keyString), logger.Error(err))
		return "", nil, 0, false
	}

	entryTTL := defaultTTL
	if a.expiryCalculator != nil {
		entry := cache.Entry[K, V]{
			Key: key, Value: value, SnapshotAtNano: time.Now().UnixNano(),
		}
		entryTTL = a.expiryCalculator.ExpireAfterCreate(entry)
	}

	return keyString, valBytes, entryTTL, true
}

// BulkSet stores multiple key-value pairs in the cache using DoMulti for
// efficiency.
//
// Takes items (map[K]V) which contains the key-value pairs to store.
// Takes tags (...string) which specifies optional tags to associate with each
// key.
//
// Returns error when the pipeline execution fails.
func (a *ValkeyClusterAdapter[K, V]) BulkSet(ctx context.Context, items map[K]V, tags ...string) error {
	ctx, l := logger.From(ctx, log)

	if len(items) == 0 {
		return nil
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.bulkOperationTimeout, fmt.Errorf("valkey cluster BulkSet exceeded %s timeout", a.bulkOperationTimeout))
	defer cancel()

	cmds := make(valkey.Commands, 0, len(items)*(1+len(tags)))

	for key, value := range items {
		keyString, valBytes, entryTTL, ok := a.prepareBulkSetItem(ctx, key, value, a.ttl)
		if !ok {
			continue
		}

		cmds = append(cmds, a.client.B().Set().Key(keyString).Value(string(valBytes)).Px(entryTTL).Build())

		for _, tag := range tags {
			cmds = append(cmds, a.client.B().Sadd().Key(tagPrefix+clusterHashTag(a.namespace+tag)).Member(keyString).Build())
		}

		if len(tags) > 0 {
			keyTagsKey := keyTagsPrefix + keyString
			cmds = append(cmds, a.client.B().Sadd().Key(keyTagsKey).Member(tags...).Build())
		}
	}

	for _, response := range a.client.DoMulti(timeoutCtx, cmds...) {
		if err := response.Error(); err != nil {
			l.Warn("Failed to execute BulkSet pipeline",
				logger.Int("item_count", len(items)), logger.Error(err))
			return fmt.Errorf("bulk set pipeline failed: %w", err)
		}
	}

	return nil
}

// Invalidate removes a key from the cache and cleans up its tag links.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which specifies the cache key to remove.
//
// Returns error when the operation fails.
func (a *ValkeyClusterAdapter[K, V]) Invalidate(ctx context.Context, key K) error {
	ctx, l := logger.From(ctx, log)

	if err := ctx.Err(); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("valkey cluster Invalidate exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return fmt.Errorf(errFmtEncodeKey, err)
	}

	if err := removeKeyFromTags(timeoutCtx, a.client, a.namespace, keyString); err != nil {
		l.Warn("Failed to remove key from tag sets", logger.String(logKeyField, keyString), logger.Error(err))
	}

	if err := a.client.Do(timeoutCtx, a.client.B().Del().Key(keyString).Build()).Error(); err != nil {
		return fmt.Errorf("valkey cluster del failed for key %s: %w", keyString, err)
	}

	return nil
}

// decodeValue decodes bytes into a value of type V using the shared decoding
// logic in cache_domain.
//
// Takes valBytes ([]byte) which contains the encoded data to decode.
//
// Returns V which is the decoded value.
// Returns error when the encoder cannot be found, unmarshalling fails, or type
// assertion fails.
func (a *ValkeyClusterAdapter[K, V]) decodeValue(valBytes []byte) (V, error) {
	return cache_domain.DecodeValue[V](valBytes, a.registry)
}

// encodeValue encodes a value of type V to bytes using the shared encoding
// logic in cache_domain.
//
// Takes value (V) which is the value to encode.
//
// Returns []byte which contains the encoded value.
// Returns error when no encoder is found for the value type or encoding fails.
func (a *ValkeyClusterAdapter[K, V]) encodeValue(value V) ([]byte, error) {
	return cache_domain.EncodeValue(value, a.registry)
}

// bulkEncodeKeys encodes a slice of keys and returns the encoded strings with a
// reverse lookup map.
//
// Takes keys ([]K) which specifies the keys to encode.
//
// Returns []string which contains the encoded string forms of the keys.
// Returns map[string]K which maps each encoded string back to its original key.
func (a *ValkeyClusterAdapter[K, V]) bulkEncodeKeys(ctx context.Context, keys []K) ([]string, map[string]K) {
	_, l := logger.From(ctx, log)

	keyStrs := make([]string, 0, len(keys))
	keyMap := make(map[string]K, len(keys))
	for _, key := range keys {
		keyString, err := a.encodeKey(key)
		if err != nil {
			l.Warn("Failed to encode key in BulkGet, skipping", logger.Error(err))
			continue
		}
		keyStrs = append(keyStrs, keyString)
		keyMap[keyString] = key
	}
	return keyStrs, keyMap
}

// processBulkGetResultBytes handles a single GET result from a Valkey pipeline.
//
// Takes valBytes ([]byte) which is the raw value bytes from the GET response.
// Takes keyString (string) which identifies the key for logging purposes.
//
// Returns V which is the unmarshalled value on success, or the zero value on
// failure.
// Returns bool which is true on success, or false to indicate a cache miss.
func (a *ValkeyClusterAdapter[K, V]) processBulkGetResultBytes(ctx context.Context, valBytes []byte, keyString string) (V, bool) {
	_, l := logger.From(ctx, log)

	var zero V

	encoder, err := a.registry.GetByType(reflect.TypeOf(zero))
	if err != nil {
		l.Warn("Failed to find encoder for type", logger.String(logKeyField, keyString), logger.Error(err))
		return zero, false
	}

	unmarshalled, err := encoder.UnmarshalAny(valBytes)
	if err != nil {
		l.Warn("Failed to unmarshal value from Valkey Cluster GET", logger.String(logKeyField, keyString), logger.Error(err))
		return zero, false
	}

	result, ok := unmarshalled.(V)
	if !ok {
		l.Warn("Type assertion failed after unmarshal", logger.String(logKeyField, keyString))
		return zero, false
	}

	return result, true
}

// storeLoadedValues stores loaded values to Valkey via DoMulti and updates
// the results map.
//
// Takes loaded (map[K]V) which contains the key-value pairs to store.
// Takes results (map[K]V) which receives successfully stored entries.
func (a *ValkeyClusterAdapter[K, V]) storeLoadedValues(ctx context.Context, loaded map[K]V, results map[K]V) {
	ctx, l := logger.From(ctx, log)

	cmds := make(valkey.Commands, 0, len(loaded))
	for k, v := range loaded {
		keyString, encodeErr := a.encodeKey(k)
		if encodeErr != nil {
			l.Warn("Failed to encode key for loaded value, skipping", logger.Error(encodeErr))
			continue
		}

		encoder, marshalErr := a.registry.Get(v)
		if marshalErr != nil {
			l.Warn("Failed to find encoder for loaded value", logger.String(logKeyField, keyString), logger.Error(marshalErr))
			continue
		}

		valBytes, marshalErr := encoder.MarshalAny(v)
		if marshalErr != nil {
			l.Warn("Failed to marshal loaded value for Valkey Cluster", logger.String(logKeyField, keyString), logger.Error(marshalErr))
			continue
		}
		cmds = append(cmds, a.client.B().Set().Key(keyString).Value(string(valBytes)).Px(a.ttl).Build())
		results[k] = v
	}

	for _, response := range a.client.DoMulti(ctx, cmds...) {
		if err := response.Error(); err != nil {
			l.Warn("Failed to execute SET pipeline after bulk load", logger.Error(err))
		}
	}
}

// BulkGet fetches multiple keys in a single operation.
//
// Takes keys ([]K) which specifies the cache keys to retrieve.
// Takes bulkLoader (BulkLoader) which loads values for any keys not found in
// the cache.
//
// Returns map[K]V which contains the retrieved values keyed by their original
// keys.
// Returns error when the Valkey MGET operation fails or the bulk loader fails.
//
// CLUSTER WARNING: MGET only works efficiently if all keys hash to the same
// slot. In a cluster deployment with randomly distributed keys, this will
// result in multiple round-trips to different nodes, reducing performance.
// Consider using hash tags in your key design if bulk operations are critical.
func (a *ValkeyClusterAdapter[K, V]) BulkGet(ctx context.Context, keys []K, bulkLoader cache.BulkLoader[K, V]) (map[K]V, error) {
	if len(keys) == 0 {
		return make(map[K]V), nil
	}

	results := make(map[K]V, len(keys))
	keyStrs, keyMap := a.bulkEncodeKeys(ctx, keys)
	if len(keyStrs) == 0 {
		return results, nil
	}

	getCmds := make(valkey.Commands, len(keyStrs))
	for i, keyString := range keyStrs {
		getCmds[i] = a.client.B().Get().Key(keyString).Build()
	}
	resps := a.client.DoMulti(ctx, getCmds...)

	var misses []K
	for i, response := range resps {
		keyString := keyStrs[i]
		originalKey := keyMap[keyString]

		valBytes, err := response.AsBytes()
		if err != nil {
			misses = append(misses, originalKey)
			continue
		}

		if result, ok := a.processBulkGetResultBytes(ctx, valBytes, keyString); ok {
			results[originalKey] = result
		} else {
			misses = append(misses, originalKey)
		}
	}

	if len(misses) > 0 {
		loaded, err := bulkLoader.BulkLoad(ctx, misses)
		if err != nil {
			return results, fmt.Errorf("bulk loader failed: %w", err)
		}
		if len(loaded) > 0 {
			a.storeLoadedValues(ctx, loaded, results)
		}
	}

	return results, nil
}

// InvalidateByTags removes all cache entries associated with the given tags.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes tags (...string) which specifies the tags whose entries should be
// removed.
//
// Returns int which is the number of entries that were invalidated.
// Returns error when the operation fails.
func (a *ValkeyClusterAdapter[K, V]) InvalidateByTags(ctx context.Context, tags ...string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.bulkOperationTimeout, fmt.Errorf("valkey cluster InvalidateByTags exceeded %s timeout", a.bulkOperationTimeout))
	defer cancel()

	count, err := performTagInvalidation(timeoutCtx, a.client, a.namespace, tags)
	if err != nil {
		return 0, fmt.Errorf("failed to invalidate by tags: %w", err)
	}
	return count, nil
}

// flushClusterUnsafe runs FLUSHDB on all master nodes in the Valkey Cluster.
// This deletes all keys on all nodes and should only be used in unsafe mode.
//
// Returns error when FLUSHDB fails on any cluster node.
func (a *ValkeyClusterAdapter[K, V]) flushClusterUnsafe(ctx context.Context) error {
	ctx, l := logger.From(ctx, log)

	l.Warn("Executing FLUSHDB on Valkey Cluster. ALL keys on ALL nodes will be deleted.",
		logger.Bool("unsafe_mode", true))

	var errs []error
	for addr, nodeClient := range a.client.Nodes() {
		if err := nodeClient.Do(ctx, nodeClient.B().Flushdb().Build()).Error(); err != nil {
			l.Error("Valkey Cluster FLUSHDB failed on node",
				logger.String(logKeyNode, addr),
				logger.Error(err))
			errs = append(errs, fmt.Errorf("FLUSHDB failed on node %s: %w", addr, err))
		}
	}

	return errors.Join(errs...)
}

// deleteBatch deletes a batch of keys individually via DoMulti and returns
// the number deleted. Each key is deleted with a separate DEL command to avoid
// cross-slot panics in cluster mode.
//
// Takes batch ([]string) which contains the keys to delete.
//
// Returns int which is the count of keys deleted, or zero on error.
func (a *ValkeyClusterAdapter[K, V]) deleteBatch(ctx context.Context, batch []string) int {
	cmds := make(valkey.Commands, len(batch))
	for i, key := range batch {
		cmds[i] = a.client.B().Del().Key(key).Build()
	}
	var deleted int
	for _, response := range a.client.DoMulti(ctx, cmds...) {
		n, err := response.AsInt64()
		if err == nil && n > 0 {
			deleted += int(n)
		}
	}
	return deleted
}

// invalidateByNamespace scans and deletes all keys that match the namespace
// pattern across the Valkey cluster.
func (a *ValkeyClusterAdapter[K, V]) invalidateByNamespace(ctx context.Context) {
	ctx, l := logger.From(ctx, log)

	scanPattern := a.namespace + "*"
	deletedCount := 0

	l.Internal("Invalidating all keys in namespace across cluster",
		logger.String("namespace", a.namespace),
		logger.String("pattern", scanPattern))

	for addr, nodeClient := range a.client.Nodes() {
		var cursor uint64
		for {
			response := nodeClient.Do(ctx, nodeClient.B().Scan().Cursor(cursor).Match(scanPattern).Count(scanBatchSize).Build())
			scanEntry, err := response.AsScanEntry()
			if err != nil {
				l.Warn("SCAN failed on cluster node",
					logger.String(logKeyNode, addr),
					logger.Error(err))
				break
			}

			if len(scanEntry.Elements) > 0 {
				deletedCount += a.deleteBatch(ctx, scanEntry.Elements)
			}

			cursor = scanEntry.Cursor
			if cursor == 0 {
				break
			}
		}
	}

	l.Internal("InvalidateAll completed",
		logger.String("namespace", a.namespace),
		logger.Int("keys_deleted", deletedCount))
}

// InvalidateAll flushes all data from the cluster.
// If search is enabled, also drops the Valkey Search index.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
//
// Returns error when the operation fails.
//
// CLUSTER NOTE: FLUSHDB in cluster mode flushes all databases on all nodes.
// This is an EXTREMELY destructive operation that affects the entire cluster.
func (a *ValkeyClusterAdapter[K, V]) InvalidateAll(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.flushTimeout, fmt.Errorf("valkey cluster InvalidateAll exceeded %s timeout", a.flushTimeout))
	defer cancel()

	a.dropIndex(timeoutCtx)

	if a.namespace == "" && a.allowUnsafeFLUSHDB {
		return a.flushClusterUnsafe(timeoutCtx)
	}

	if a.namespace == "" {
		return errors.New("InvalidateAll blocked: no namespace configured and AllowUnsafeFLUSHDB is false")
	}

	a.invalidateByNamespace(timeoutCtx)
	return nil
}

// BulkRefresh asynchronously refreshes multiple cache entries using the bulk
// loader.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes keys ([]K) which specifies the cache keys to refresh.
// Takes bulkLoader (BulkLoader) which loads the values for the given keys.
//
// Safe for concurrent use. Spawns a goroutine that loads
// values and updates the cache. The goroutine runs
// independently and logs warnings on failure.
func (a *ValkeyClusterAdapter[K, V]) BulkRefresh(ctx context.Context, keys []K, bulkLoader cache.BulkLoader[K, V]) {
	ctx, l := logger.From(ctx, log)

	go func() {
		defer goroutine.RecoverPanic(ctx, "cache.valkeyClusterBulkRefresh")
		loaded, err := bulkLoader.BulkLoad(ctx, keys)
		if err != nil {
			l.Warn("Bulk refresh failed", logger.Error(err))
			return
		}
		for k, v := range loaded {
			if setErr := a.Set(ctx, k, v); setErr != nil {
				l.Warn("Failed to set refreshed value", logger.Error(setErr))
			}
		}
	}()
}

// Refresh asynchronously refreshes a single cache entry using the
// provided loader.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to refresh.
// Takes loader (Loader[K, V]) which loads the fresh value for the
// given key.
//
// Returns <-chan cache.LoadResult[V] which receives the load outcome
// once the background goroutine completes.
//
// Safe for concurrent use. Spawns a goroutine that loads the value and updates
// the cache. The channel is closed after the result is sent.
func (a *ValkeyClusterAdapter[K, V]) Refresh(ctx context.Context, key K, loader cache.Loader[K, V]) <-chan cache.LoadResult[V] {
	ctx, l := logger.From(ctx, log)

	resultChan := make(chan cache.LoadResult[V], 1)
	go func() {
		defer close(resultChan)
		defer goroutine.RecoverPanic(ctx, "cache.valkeyClusterRefresh")
		value, err := loader.Load(ctx, key)
		if err == nil {
			if setErr := a.Set(ctx, key, value); setErr != nil {
				l.Warn("Failed to set refreshed value", logger.Error(setErr))
			}
		}
		resultChan <- cache.LoadResult[V]{Value: value, Err: err}
	}()
	return resultChan
}

// Search performs full-text search across indexed TEXT fields. Valkey Search
// does not support full-text TEXT fields; returns ErrSearchNotSupported for
// text queries.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes query (string) which is the search query to match against TEXT fields.
// Takes opts (*cache.SearchOptions) which contains filters, pagination, and
// sorting.
//
// Returns SearchResult containing matching items and total count.
// Returns error if search is not supported or the search fails.
func (a *ValkeyClusterAdapter[K, V]) Search(ctx context.Context, query string, opts *cache.SearchOptions) (cache.SearchResult[K, V], error) {
	if a.schema == nil {
		return cache.SearchResult[K, V]{}, fmt.Errorf(
			"%w: configure a SearchSchema via Searchable() to enable search",
			cache.ErrSearchNotSupported,
		)
	}

	if query != "" {
		return cache.SearchResult[K, V]{}, fmt.Errorf(
			"%w: Valkey Search does not support full-text TEXT fields; use Query() with TAG/NUMERIC filters instead",
			cache.ErrSearchNotSupported,
		)
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.searchTimeout, fmt.Errorf("valkey cluster Search exceeded %s timeout", a.searchTimeout))
	defer cancel()

	return a.searchWithValkeySearch(timeoutCtx, query, opts)
}

// Query performs structured filtering, sorting, and pagination without
// full-text search.
// Returns ErrSearchNotSupported if no SearchSchema was configured.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes opts (*cache.QueryOptions) which contains filters, pagination, and
// sorting.
//
// Returns SearchResult containing matching items and total count.
// Returns error if search is not supported or the query fails.
func (a *ValkeyClusterAdapter[K, V]) Query(ctx context.Context, opts *cache.QueryOptions) (cache.SearchResult[K, V], error) {
	if a.schema == nil {
		return cache.SearchResult[K, V]{}, fmt.Errorf(
			"%w: configure a SearchSchema via Searchable() to enable query",
			cache.ErrSearchNotSupported,
		)
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.searchTimeout, fmt.Errorf("valkey cluster Query exceeded %s timeout", a.searchTimeout))
	defer cancel()

	return a.queryWithValkeySearch(timeoutCtx, opts)
}

// SupportsSearch returns true if a SearchSchema was configured for this cache.
//
// Returns bool indicating whether search operations are available.
func (a *ValkeyClusterAdapter[K, V]) SupportsSearch() bool {
	return a.schema != nil
}

// GetSchema returns the configured search schema for this cache.
//
// Returns *cache.SearchSchema which is the schema, or nil if not searchable.
func (a *ValkeyClusterAdapter[K, V]) GetSchema() *cache.SearchSchema {
	return a.schema
}
