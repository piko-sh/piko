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

package cache_provider_redis

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"reflect"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// scanBatchSize is the number of keys to fetch in each Redis SCAN call.
	scanBatchSize = 100

	// logKeyField is the attribute key used when logging Redis cache keys.
	logKeyField = "key"

	// scanAllPattern is the wildcard pattern used in Redis SCAN commands to
	// match all keys.
	scanAllPattern = "*"

	// errMessageEncodeKey is the warning message logged when key encoding fails.
	errMessageEncodeKey = "Failed to encode key"

	// errFmtEncodeKey is the format string used when key encoding fails.
	errFmtEncodeKey = "failed to encode key: %w"
)

// RedisAdapter implements the ProviderPort using a Redis client. It supports
// generics by encoding keys to strings and using a type-driven EncodingRegistry
// for values.
type RedisAdapter[K comparable, V any] struct {
	// expiryCalculator sets the expiry time for each key; optional.
	expiryCalculator cache.ExpiryCalculator[K, V]

	// refreshCalculator calculates when entries become ready for background
	// refresh; optional.
	refreshCalculator cache.RefreshCalculator[K, V]

	// registry encodes values before they are stored.
	registry *cache.EncodingRegistry

	// client is the Redis client for storage operations.
	client *redis.Client

	// keyRegistry stores encoders for complex key types; nil uses fmt.Sprintf.
	keyRegistry *cache.EncodingRegistry

	// schema is the search schema for this cache; nil means search is disabled.
	schema *cache.SearchSchema

	// sf deduplicates concurrent loads for the same key.
	sf singleflight.Group

	// namespace is the prefix added to all keys in Redis.
	namespace string

	// indexName is the RediSearch index name for this namespace.
	indexName string

	// ttl is the default time-to-live for cache entries.
	ttl time.Duration

	// operationTimeout is the time limit for a single Redis operation.
	operationTimeout time.Duration

	// atomicOperationTimeout is the time limit for WATCH/MULTI/EXEC operations.
	atomicOperationTimeout time.Duration

	// bulkOperationTimeout is the maximum time for bulk operations like MGET,
	// MSET, and pipelines.
	bulkOperationTimeout time.Duration

	// flushTimeout is the time limit for InvalidateAll operations.
	flushTimeout time.Duration

	// searchTimeout is the time limit for FT.SEARCH operations.
	searchTimeout time.Duration

	// maxComputeRetries is the maximum number of retry attempts for optimistic
	// locking in Compute methods.
	maxComputeRetries int

	// allowUnsafeFLUSHDB controls whether InvalidateAll may use FLUSHDB when no
	// namespace is set. If false, InvalidateAll is blocked without a namespace
	// for safety.
	allowUnsafeFLUSHDB bool

	// indexCreated indicates whether the search index has been created.
	indexCreated bool
}

var _ cache.ProviderPort[any, any] = (*RedisAdapter[any, any])(nil)

// encodeKey converts a key of type K to a Redis key string.
//
// Takes key (K) which is the cache key to encode.
//
// Returns string which is the encoded Redis key, with namespace prefix if set.
// Returns error when no encoder is registered for the key type or when
// marshalling fails.
func (a *RedisAdapter[K, V]) encodeKey(key K) (string, error) {
	return cache_domain.EncodeKey(key, a.namespace, a.keyRegistry)
}

// decodeKey converts a Redis key string back to a key of type K.
//
// Takes keyString (string) which is the Redis key to decode.
//
// Returns K which is the decoded key value.
// Returns error when the namespace prefix is missing, decoding fails, or no
// encoder is registered for the key type.
func (a *RedisAdapter[K, V]) decodeKey(keyString string) (K, error) {
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
// Returns V which is the cached value, or zero value if not found.
// Returns bool which indicates whether the key was present in the cache.
// Returns error when the operation fails (e.g. network error).
func (a *RedisAdapter[K, V]) GetIfPresent(ctx context.Context, key K) (V, bool, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("redis GetIfPresent exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	if a.needsJSONStorage() {
		return a.getJSONValue(timeoutCtx, keyString)
	}

	value, err := a.client.Get(timeoutCtx, keyString).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return *new(V), false, nil
		}
		return *new(V), false, fmt.Errorf("redis get failed: %w", err)
	}

	var v V
	encoder, err := a.registry.GetByType(reflect.TypeOf(v))
	if err != nil {
		return *new(V), false, fmt.Errorf("failed to find encoder for type: %w", err)
	}

	unmarshalled, err := encoder.UnmarshalAny(value)
	if err != nil {
		return *new(V), false, fmt.Errorf("failed to unmarshal value from Redis: %w", err)
	}

	result, ok := unmarshalled.(V)
	if !ok {
		return *new(V), false, fmt.Errorf("type assertion failed after unmarshal for key %q", keyString)
	}

	return result, true, nil
}

// Get retrieves a value from the cache, loading it via the provided loader
// if not present.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cached value to retrieve.
// Takes loader (Loader[K, V]) which loads the value if not already cached.
//
// Returns V which is the cached or newly loaded value.
// Returns error when key encoding fails, the loader fails, or type assertion
// fails.
func (a *RedisAdapter[K, V]) Get(ctx context.Context, key K, loader cache.Loader[K, V]) (V, error) {
	ctx, l := logger.From(ctx, log)
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
// invalidation. When a SearchSchema is configured, values are stored as JSON
// for RediSearch indexing.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes tags (...string) which provide optional grouping for bulk invalidation.
//
// Returns error when the operation fails.
func (a *RedisAdapter[K, V]) Set(ctx context.Context, key K, value V, tags ...string) error {
	ctx, l := logger.From(ctx, log)
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("redis Set exceeded %s timeout", a.operationTimeout))
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
		if err := a.client.Expire(timeoutCtx, keyString, ttl).Err(); err != nil {
			l.Warn("Failed to set TTL on JSON document", logger.String(logKeyField, keyString), logger.Error(err))
		}
	} else {
		encoder, err := a.registry.Get(value)
		if err != nil {
			return fmt.Errorf("failed to find encoder for value: %w", err)
		}

		valBytes, err := encoder.MarshalAny(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value for Redis: %w", err)
		}

		if err := a.client.Set(timeoutCtx, keyString, valBytes, ttl).Err(); err != nil {
			return fmt.Errorf("redis set failed: %w", err)
		}
	}

	if err := addTagsToKey(timeoutCtx, a.client, a.namespace, keyString, tags); err != nil {
		l.Warn("Failed to add tags to key", logger.String(logKeyField, keyString), logger.Error(err))
	}

	return nil
}

// SetWithTTL stores a key-value pair with a set time-to-live duration.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes ttl (time.Duration) which sets how long the entry stays valid.
// Takes tags (...string) which links labels to the entry.
//
// Returns error when encoding, marshalling, or the Redis operation fails.
func (a *RedisAdapter[K, V]) SetWithTTL(ctx context.Context, key K, value V, ttl time.Duration, tags ...string) error {
	ctx, l := logger.From(ctx, log)
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("redis SetWithTTL exceeded %s timeout", a.operationTimeout))
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
		l.Warn("Failed to marshal value for Redis", logger.String(logKeyField, keyString), logger.Error(err))
		return fmt.Errorf("marshal failed: %w", err)
	}

	if err := a.client.Set(timeoutCtx, keyString, valBytes, ttl).Err(); err != nil {
		l.Warn("Redis Set with TTL failed", logger.String(logKeyField, keyString), logger.Error(err))
		return fmt.Errorf("redis set failed: %w", err)
	}

	if err := addTagsToKey(timeoutCtx, a.client, a.namespace, keyString, tags); err != nil {
		l.Warn("Failed to add tags to key", logger.String(logKeyField, keyString), logger.Error(err))
	}

	return nil
}

// Invalidate removes a key from the cache and cleans up its tag links.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to remove.
//
// Returns error when the operation fails.
func (a *RedisAdapter[K, V]) Invalidate(ctx context.Context, key K) error {
	ctx, l := logger.From(ctx, log)
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("redis Invalidate exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return fmt.Errorf(errFmtEncodeKey, err)
	}

	if err := removeKeyFromTags(timeoutCtx, a.client, a.namespace, keyString); err != nil {
		l.Warn("Failed to remove key from tag sets", logger.String(logKeyField, keyString), logger.Error(err))
	}

	if err := a.client.Del(timeoutCtx, keyString).Err(); err != nil {
		return fmt.Errorf("redis del failed: %w", err)
	}

	return nil
}

// decodeValue decodes bytes into a value of type V using the registry.
//
// Takes valBytes ([]byte) which contains the encoded data to decode.
//
// Returns V which is the decoded value.
// Returns error when the encoder cannot be found, unmarshalling fails, or type
// assertion fails.
func (a *RedisAdapter[K, V]) decodeValue(valBytes []byte) (V, error) {
	return cache_domain.DecodeValue[V](valBytes, a.registry)
}

// encodeValue encodes a value of type V to bytes using the registry.
//
// Takes value (V) which is the value to encode.
//
// Returns []byte which contains the encoded value.
// Returns error when no encoder is found for the value type or encoding fails.
func (a *RedisAdapter[K, V]) encodeValue(value V) ([]byte, error) {
	return cache_domain.EncodeValue(value, a.registry)
}

// executeComputeAction executes the compute action within a Redis pipeline.
//
// Takes pipe (redis.Pipeliner) which is the pipeline to queue commands on.
// Takes keyString (string) which is the cache key to operate on.
// Takes newValue (V) which is the value to set when the action is Set.
// Takes action (cache.ComputeAction) which specifies the operation to
// perform.
// Takes found (bool) which indicates whether the key exists in the cache.
//
// Returns error when encoding the value fails.
func (a *RedisAdapter[K, V]) executeComputeAction(ctx context.Context, pipe redis.Pipeliner, keyString string, newValue V, action cache.ComputeAction, found bool) error {
	switch action {
	case cache.ComputeActionSet:
		valBytes, err := a.encodeValue(newValue)
		if err != nil {
			return err
		}
		pipe.Set(ctx, keyString, valBytes, a.ttl)

	case cache.ComputeActionDelete:
		if found {
			pipe.Del(ctx, keyString)
		}

	case cache.ComputeActionNoop:
	}
	return nil
}

// handleComputeRetryResult processes the result of a compute transaction
// attempt.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which is the cache key being computed.
// Takes keyString (string) which is the string representation of the key for
// logging.
// Takes err (error) which is the error from the transaction attempt, or nil on
// success.
// Takes attempt (int) which is the current retry attempt number.
//
// Returns bool which indicates whether the operation should be retried.
// Returns V which is the computed value if found.
// Returns bool which indicates whether a valid value was found.
// Returns error when the post-transaction read fails.
func (a *RedisAdapter[K, V]) handleComputeRetryResult(ctx context.Context, key K, keyString string, err error, attempt int) (bool, V, bool, error) {
	ctx, l := logger.From(ctx, log)
	var zero V

	if err == nil {
		finalValue, ok, getErr := a.GetIfPresent(ctx, key)
		if getErr != nil {
			return false, zero, false, getErr
		}
		if ok {
			return false, finalValue, true, nil
		}
		return false, zero, false, nil
	}

	if errors.Is(err, redis.TxFailedErr) {
		l.Trace("Compute transaction failed, retrying",
			logger.String(logKeyField, keyString),
			logger.Int("attempt", attempt+1))
		return true, zero, false, nil
	}

	return false, zero, false, err
}

// Compute atomically updates a cache entry using a compute function with
// optimistic locking. Computes and writes the new value in one round trip.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(...)) which calculates the new value based on the
// current value and whether it exists.
//
// Returns V which is the computed value, or zero value if the operation fails.
// Returns bool which indicates whether the operation succeeded.
// Returns error when the operation fails.
func (a *RedisAdapter[K, V]) Compute(ctx context.Context, key K, computeFunction func(oldValue V, found bool) (newValue V, action cache.ComputeAction)) (V, bool, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("redis Compute exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	for attempt := range a.maxComputeRetries {
		err := a.client.Watch(timeoutCtx, func(tx *redis.Tx) error {
			oldValue, found, getErr := a.fetchValueInWatch(timeoutCtx, tx, keyString)
			if getErr != nil {
				return getErr
			}

			newValue, action := computeFunction(oldValue, found)

			_, txErr := tx.TxPipelined(timeoutCtx, func(pipe redis.Pipeliner) error {
				return a.executeComputeAction(timeoutCtx, pipe, keyString, newValue, action, found)
			})
			return txErr
		}, keyString)

		shouldRetry, value, found, retryErr := a.handleComputeRetryResult(ctx, key, keyString, err, attempt)
		if retryErr != nil {
			return *new(V), false, retryErr
		}
		if !shouldRetry {
			return value, found, nil
		}
	}

	return *new(V), false, fmt.Errorf("compute max retries exceeded (%d) for key %q", a.maxComputeRetries, keyString)
}

// fetchValueInWatch fetches and decodes a value within a Redis watch context.
//
// Takes tx (*redis.Tx) which is the transaction for the watch operation.
// Takes keyString (string) which is the key to fetch.
//
// Returns V which is the decoded value if found.
// Returns bool which is true if the key exists.
// Returns error when the Redis get or decoding fails.
func (a *RedisAdapter[K, V]) fetchValueInWatch(ctx context.Context, tx *redis.Tx, keyString string) (V, bool, error) {
	var zero V
	valBytes, err := tx.Get(ctx, keyString).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return zero, false, nil
		}
		return zero, false, fmt.Errorf("redis get failed: %w", err)
	}

	value, err := a.decodeValue(valBytes)
	if err != nil {
		return zero, false, err
	}
	return value, true, nil
}

// computeIfAbsentWatchFunction runs the WATCH callback for ComputeIfAbsent.
//
// Takes tx (*redis.Tx) which provides the transaction context for the
// operation.
// Takes keyString (string) which specifies the key to check and set if missing.
// Takes computeFunction (func() V) which creates the value if the key does not
// exist.
// Takes didCompute (*bool) which is set to true if the value was computed.
//
// Returns error when the key check fails or the transaction cannot complete.
func (a *RedisAdapter[K, V]) computeIfAbsentWatchFunction(ctx context.Context, tx *redis.Tx, keyString string, computeFunction func() V, didCompute *bool) error {
	exists, err := tx.Exists(ctx, keyString).Result()
	if err != nil {
		return fmt.Errorf("redis exists check failed: %w", err)
	}
	if exists > 0 {
		return nil
	}

	newValue := computeFunction()
	*didCompute = true
	_, txErr := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		valBytes, err := a.encodeValue(newValue)
		if err != nil {
			return err
		}
		pipe.Set(ctx, keyString, valBytes, a.ttl)
		return nil
	})
	return txErr
}

// handleComputeIfAbsentResult processes the result of a ComputeIfAbsent
// transaction attempt.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which is the cache key being computed.
// Takes keyString (string) which is the string representation of key for logging.
// Takes err (error) which is the result of the transaction attempt.
// Takes didCompute (bool) which tracks whether the compute function was called.
//
// Returns value (V) which is the computed or cached value if successful.
// Returns computed (bool) which indicates whether the value was newly computed.
// Returns shouldRetry (bool) which indicates whether the operation should be
// retried due to a transaction conflict.
// Returns error when the post-transaction read fails.
func (a *RedisAdapter[K, V]) handleComputeIfAbsentResult(ctx context.Context, key K, keyString string, err error, didCompute bool) (value V, computed bool, shouldRetry bool, retErr error) {
	ctx, l := logger.From(ctx, log)
	var zero V

	if err == nil {
		finalValue, ok, getErr := a.GetIfPresent(ctx, key)
		if getErr != nil {
			return zero, false, false, getErr
		}
		if ok {
			return finalValue, didCompute, false, nil
		}
		return zero, false, false, nil
	}

	if errors.Is(err, redis.TxFailedErr) {
		l.Trace("ComputeIfAbsent transaction failed, retrying",
			logger.String(logKeyField, keyString))
		return zero, false, true, nil
	}

	return zero, false, false, err
}

// ComputeIfAbsent atomically computes and stores a value only if the key is
// not present.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to check or create.
// Takes computeFunction (func() V) which generates the value if the key is absent.
//
// Returns V which is the existing or newly computed value.
// Returns bool which indicates whether a value was successfully retrieved or
// computed.
// Returns error when the operation fails.
func (a *RedisAdapter[K, V]) ComputeIfAbsent(ctx context.Context, key K, computeFunction func() V) (V, bool, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("redis ComputeIfAbsent exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	for range a.maxComputeRetries {
		didCompute := false
		err := a.client.Watch(timeoutCtx, func(tx *redis.Tx) error {
			return a.computeIfAbsentWatchFunction(timeoutCtx, tx, keyString, computeFunction, &didCompute)
		}, keyString)

		value, computed, shouldRetry, retryErr := a.handleComputeIfAbsentResult(ctx, key, keyString, err, didCompute)
		if retryErr != nil {
			return *new(V), false, retryErr
		}
		if !shouldRetry {
			return value, computed, nil
		}
	}

	return *new(V), false, fmt.Errorf("compute if absent max retries exceeded (%d) for key %q", a.maxComputeRetries, keyString)
}

// executeComputeActionPresent executes the compute action for ComputeIfPresent,
// always deleting when action is delete.
//
// Takes pipe (redis.Pipeliner) which queues the Redis commands.
// Takes keyString (string) which is the cache key to operate on.
// Takes newValue (V) which is the value to set when action is set.
// Takes action (cache.ComputeAction) which determines the operation to run.
//
// Returns error when the value cannot be encoded.
func (a *RedisAdapter[K, V]) executeComputeActionPresent(ctx context.Context, pipe redis.Pipeliner, keyString string, newValue V, action cache.ComputeAction) error {
	switch action {
	case cache.ComputeActionSet:
		valBytes, err := a.encodeValue(newValue)
		if err != nil {
			return err
		}
		pipe.Set(ctx, keyString, valBytes, a.ttl)
	case cache.ComputeActionDelete:
		pipe.Del(ctx, keyString)
	case cache.ComputeActionNoop:
	}
	return nil
}

// computeIfPresentWatchFunction runs the WATCH callback for ComputeIfPresent.
//
// Takes tx (*redis.Tx) which is the Redis transaction for the watch operation.
// Takes keyString (string) which is the cache key to operate on.
// Takes computeFunction (func(oldValue V) (V, cache.ComputeAction))
// which works out the new value from the existing value.
//
// Returns error when the fetch or transaction fails.
func (a *RedisAdapter[K, V]) computeIfPresentWatchFunction(ctx context.Context, tx *redis.Tx, keyString string, computeFunction func(oldValue V) (V, cache.ComputeAction)) error {
	oldValue, found, err := a.fetchValueInWatch(ctx, tx, keyString)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	newValue, action := computeFunction(oldValue)
	_, txErr := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		return a.executeComputeActionPresent(ctx, pipe, keyString, newValue, action)
	})
	return txErr
}

// handleComputeIfPresentResult processes the result of a ComputeIfPresent
// transaction attempt.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which is the cache key being computed.
// Takes keyString (string) which is the string form of the key for logging.
// Takes err (error) which is the error from the transaction attempt.
// Takes attempt (int) which is the current retry attempt number.
//
// Returns value (V) which is the computed value if found.
// Returns found (bool) which indicates whether the value was present.
// Returns shouldRetry (bool) which indicates whether the transaction should be
// retried due to a conflict.
// Returns error when the post-transaction read fails.
func (a *RedisAdapter[K, V]) handleComputeIfPresentResult(ctx context.Context, key K, keyString string, err error, attempt int) (value V, found bool, shouldRetry bool, retErr error) {
	ctx, l := logger.From(ctx, log)
	var zero V

	if err == nil {
		finalValue, ok, getErr := a.GetIfPresent(ctx, key)
		if getErr != nil {
			return zero, false, false, getErr
		}
		if ok {
			return finalValue, true, false, nil
		}
		return zero, false, false, nil
	}

	if errors.Is(err, redis.TxFailedErr) {
		l.Trace("ComputeIfPresent transaction failed, retrying",
			logger.String(logKeyField, keyString),
			logger.Int("attempt", attempt+1))
		return zero, false, true, nil
	}

	return zero, false, false, err
}

// ComputeIfPresent atomically updates a value only if the key exists in the
// cache.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(...)) which receives the current value
// and returns the new value along with an action indicating whether
// to update or remove.
//
// Returns V which is the resulting value after computation, or the zero value
// if the key was not found or the operation failed.
// Returns bool which is true if the key existed and the computation succeeded.
// Returns error when the operation fails.
func (a *RedisAdapter[K, V]) ComputeIfPresent(ctx context.Context, key K, computeFunction func(oldValue V) (newValue V, action cache.ComputeAction)) (V, bool, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("redis ComputeIfPresent exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	for attempt := range a.maxComputeRetries {
		err := a.client.Watch(timeoutCtx, func(tx *redis.Tx) error {
			return a.computeIfPresentWatchFunction(timeoutCtx, tx, keyString, computeFunction)
		}, keyString)

		value, found, shouldRetry, retryErr := a.handleComputeIfPresentResult(ctx, key, keyString, err, attempt)
		if retryErr != nil {
			return *new(V), false, retryErr
		}
		if !shouldRetry {
			return value, found, nil
		}
	}

	return *new(V), false, fmt.Errorf("compute if present max retries exceeded (%d) for key %q", a.maxComputeRetries, keyString)
}

// executeComputeActionWithTTL executes the compute action within a Redis
// pipeline with optional custom TTL.
//
// Takes pipe (redis.Pipeliner) which is the pipeline to queue commands on.
// Takes keyString (string) which is the cache key to operate on.
// Takes newValue (V) which is the value to set when the action is Set.
// Takes action (cache.ComputeAction) which specifies the operation to perform.
// Takes found (bool) which indicates whether the key exists in the cache.
// Takes ttl (time.Duration) which is the custom TTL; zero uses the default.
//
// Returns error when encoding the value fails.
func (a *RedisAdapter[K, V]) executeComputeActionWithTTL(ctx context.Context, pipe redis.Pipeliner, keyString string, newValue V, action cache.ComputeAction, found bool, ttl time.Duration) error {
	switch action {
	case cache.ComputeActionSet:
		valBytes, err := a.encodeValue(newValue)
		if err != nil {
			return err
		}
		effectiveTTL := a.ttl
		if ttl > 0 {
			effectiveTTL = ttl
		}
		pipe.Set(ctx, keyString, valBytes, effectiveTTL)
	case cache.ComputeActionDelete:
		if found {
			pipe.Del(ctx, keyString)
		}
	case cache.ComputeActionNoop:
	}
	return nil
}

// ComputeWithTTL atomically computes a new value with per-call TTL control.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(...)) which receives the old value and found flag,
// returning a ComputeResult containing the new value, action, and optional TTL.
//
// Returns V which is the resulting value after the operation.
// Returns bool which indicates whether a value is now present.
// Returns error when the operation fails.
func (a *RedisAdapter[K, V]) ComputeWithTTL(ctx context.Context, key K, computeFunction func(oldValue V, found bool) cache.ComputeResult[V]) (V, bool, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("redis ComputeWithTTL exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	for attempt := range a.maxComputeRetries {
		err := a.client.Watch(timeoutCtx, func(tx *redis.Tx) error {
			oldValue, found, getErr := a.fetchValueInWatch(timeoutCtx, tx, keyString)
			if getErr != nil {
				return getErr
			}

			result := computeFunction(oldValue, found)

			_, txErr := tx.TxPipelined(timeoutCtx, func(pipe redis.Pipeliner) error {
				return a.executeComputeActionWithTTL(timeoutCtx, pipe, keyString, result.Value, result.Action, found, result.TTL)
			})
			return txErr
		}, keyString)

		shouldRetry, value, found, retryErr := a.handleComputeRetryResult(ctx, key, keyString, err, attempt)
		if retryErr != nil {
			return *new(V), false, retryErr
		}
		if !shouldRetry {
			return value, found, nil
		}
	}

	return *new(V), false, fmt.Errorf("compute with TTL max retries exceeded (%d) for key %q", a.maxComputeRetries, keyString)
}

// bulkEncodeKeys encodes a slice of keys and returns the encoded string keys
// and a reverse map for decoding. Keys that fail to encode are skipped.
//
// Takes keys ([]K) which contains the keys to encode.
//
// Returns []string which contains the encoded keys.
// Returns map[string]K which maps encoded strings back to their original keys.
func (a *RedisAdapter[K, V]) bulkEncodeKeys(ctx context.Context, keys []K) ([]string, map[string]K) {
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

// processBulkGetResult processes a single MGET result and returns the value
// if successful.
//
// Takes value (any) which is the raw value returned from Redis MGET.
// Takes keyString (string) which identifies the cache key for logging.
//
// Returns V which is the unmarshalled value of the generic type.
// Returns bool which indicates whether the processing succeeded.
func (a *RedisAdapter[K, V]) processBulkGetResult(ctx context.Context, value any, keyString string) (V, bool) {
	_, l := logger.From(ctx, log)

	var zero V

	valString, ok := value.(string)
	if !ok {
		l.Warn("Unexpected value type from Redis MGET", logger.String(logKeyField, keyString))
		return zero, false
	}

	encoder, err := a.registry.GetByType(reflect.TypeOf(zero))
	if err != nil {
		l.Warn("Failed to find encoder for type", logger.String(logKeyField, keyString), logger.Error(err))
		return zero, false
	}

	unmarshalled, err := encoder.UnmarshalAny([]byte(valString))
	if err != nil {
		l.Warn("Failed to unmarshal value from Redis MGET", logger.String(logKeyField, keyString), logger.Error(err))
		return zero, false
	}

	result, ok := unmarshalled.(V)
	if !ok {
		l.Warn("Type assertion failed after unmarshal", logger.String(logKeyField, keyString))
		return zero, false
	}

	return result, true
}

// storeLoadedValues stores loaded values to Redis using a pipeline and updates
// the results map.
//
// Takes loaded (map[K]V) which holds the values fetched from the loader.
// Takes results (map[K]V) which is updated with entries that were stored.
func (a *RedisAdapter[K, V]) storeLoadedValues(ctx context.Context, loaded map[K]V, results map[K]V) {
	ctx, l := logger.From(ctx, log)
	pipe := a.client.Pipeline()
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
			l.Warn("Failed to marshal loaded value for Redis", logger.String(logKeyField, keyString), logger.Error(marshalErr))
			continue
		}
		pipe.Set(ctx, keyString, valBytes, a.ttl)
		results[k] = v
	}

	if _, err := pipe.Exec(ctx); err != nil {
		l.Warn("Failed to execute SET pipeline after bulk load", logger.Error(err))
	}
}

// BulkGet retrieves multiple values from the cache, loading missing ones
// via the bulk loader.
//
// Takes keys ([]K) which specifies the cache keys to retrieve.
// Takes bulkLoader (BulkLoader[K, V]) which loads values for any cache misses.
//
// Returns map[K]V which contains the retrieved and loaded values.
// Returns error when the Redis MGET operation or bulk loader fails.
func (a *RedisAdapter[K, V]) BulkGet(ctx context.Context, keys []K, bulkLoader cache.BulkLoader[K, V]) (map[K]V, error) {
	ctx, l := logger.From(ctx, log)
	if len(keys) == 0 {
		return make(map[K]V), nil
	}

	results := make(map[K]V, len(keys))
	keyStrs, keyMap := a.bulkEncodeKeys(ctx, keys)
	if len(keyStrs) == 0 {
		return results, nil
	}

	vals, err := a.client.MGet(ctx, keyStrs...).Result()
	if err != nil {
		l.Warn("Redis MGET failed", logger.Int("key_count", len(keys)), logger.Error(err))
		return results, fmt.Errorf("redis MGET failed: %w", err)
	}

	var misses []K
	for i, value := range vals {
		keyString := keyStrs[i]
		originalKey := keyMap[keyString]

		if value == nil {
			misses = append(misses, originalKey)
			continue
		}

		if result, ok := a.processBulkGetResult(ctx, value, keyString); ok {
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

// prepareBulkSetItem encodes a key-value pair and calculates its TTL.
//
// Takes key (K) which is the cache key to encode.
// Takes value (V) which is the value to marshal for storage.
// Takes defaultTTL (time.Duration) which is the fallback TTL when no expiry
// calculator is configured.
//
// Returns string which is the encoded key string.
// Returns []byte which is the marshalled value bytes.
// Returns time.Duration which is the TTL to use for this entry.
// Returns bool which is false when encoding or marshalling fails.
func (a *RedisAdapter[K, V]) prepareBulkSetItem(ctx context.Context, key K, value V, defaultTTL time.Duration) (string, []byte, time.Duration, bool) {
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
		l.Warn("Failed to marshal value for Redis in BulkSet",
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

// BulkSet stores multiple key-value pairs in the cache using a pipeline for
// efficiency.
//
// Takes items (map[K]V) which contains the key-value pairs to store.
// Takes tags (...string) which specifies optional tags to associate with the
// keys.
//
// Returns error when the pipeline execution fails.
func (a *RedisAdapter[K, V]) BulkSet(ctx context.Context, items map[K]V, tags ...string) error {
	ctx, l := logger.From(ctx, log)
	if len(items) == 0 {
		return nil
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.bulkOperationTimeout, fmt.Errorf("redis BulkSet exceeded %s timeout", a.bulkOperationTimeout))
	defer cancel()

	if a.needsJSONStorage() {
		return a.bulkSetJSON(timeoutCtx, items, tags...)
	}

	pipe := a.client.Pipeline()

	for key, value := range items {
		if timeoutCtx.Err() != nil {
			return timeoutCtx.Err()
		}

		keyString, valBytes, entryTTL, ok := a.prepareBulkSetItem(ctx, key, value, a.ttl)
		if !ok {
			continue
		}

		pipe.Set(timeoutCtx, keyString, valBytes, entryTTL)

		for _, tag := range tags {
			pipe.SAdd(timeoutCtx, a.namespace+tagPrefix+tag, keyString)
		}

		if len(tags) > 0 {
			keyTagsKey := keyTagsPrefix + keyString
			tagArgs := make([]any, len(tags))
			for i, tag := range tags {
				tagArgs[i] = tag
			}
			pipe.SAdd(timeoutCtx, keyTagsKey, tagArgs...)
		}
	}

	if _, err := pipe.Exec(timeoutCtx); err != nil {
		l.Warn("Failed to execute BulkSet pipeline",
			logger.Int("item_count", len(items)), logger.Error(err))
		return fmt.Errorf("bulk set pipeline failed: %w", err)
	}

	return nil
}

// bulkSetJSON stores multiple items using JSON.SET for search-indexed
// namespaces.
//
// Takes items (map[K]V) which contains the key-value pairs to store.
// Takes tags (...string) which specifies optional tags to associate with keys.
//
// Returns error when storage operations fail.
func (a *RedisAdapter[K, V]) bulkSetJSON(ctx context.Context, items map[K]V, tags ...string) error {
	ctx, l := logger.From(ctx, log)
	for key, value := range items {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		keyString, err := a.encodeKey(key)
		if err != nil {
			l.Warn(errMessageEncodeKey, logger.Error(err))
			continue
		}

		a.indexDocument(ctx, keyString, value)

		if err := addTagsToKey(ctx, a.client, a.namespace, keyString, tags); err != nil {
			l.Warn("Failed to add tags to key", logger.String(logKeyField, keyString), logger.Error(err))
		}
	}

	return nil
}

// InvalidateByTags removes all cache entries linked to the given tags.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes tags (...string) which specifies the tags whose entries to remove.
//
// Returns int which is the number of entries removed.
// Returns error when the operation fails.
func (a *RedisAdapter[K, V]) InvalidateByTags(ctx context.Context, tags ...string) (int, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.bulkOperationTimeout, fmt.Errorf("redis InvalidateByTags exceeded %s timeout", a.bulkOperationTimeout))
	defer cancel()

	count, err := performTagInvalidation(timeoutCtx, a.client, a.namespace, tags)
	if err != nil {
		return 0, fmt.Errorf("failed to invalidate by tags: %w", err)
	}
	return count, nil
}

// flushUnsafe runs FLUSHDB on the Redis instance, deleting all keys in the
// current database.
//
// Returns error when the FLUSHDB command fails.
func (a *RedisAdapter[K, V]) flushUnsafe(ctx context.Context) error {
	ctx, l := logger.From(ctx, log)
	l.Warn("Executing FLUSHDB on Redis. ALL keys in the current database will be deleted.",
		logger.Bool("unsafe_mode", true))
	if err := a.client.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("redis FLUSHDB failed: %w", err)
	}
	return nil
}

// deleteBatch deletes a batch of keys and returns the number deleted.
//
// Takes batch ([]string) which contains the cache keys to delete.
//
// Returns int which is the number of keys deleted.
func (a *RedisAdapter[K, V]) deleteBatch(ctx context.Context, batch []string) int {
	ctx, l := logger.From(ctx, log)
	deleted, err := a.client.Del(ctx, batch...).Result()
	if err != nil {
		l.Warn("Failed to delete batch of keys",
			logger.Int("batch_size", len(batch)), logger.Error(err))
		return 0
	}
	return int(deleted)
}

// invalidateByNamespace scans for and deletes all keys that match the
// namespace pattern.
//
// Returns error when the SCAN iteration fails.
func (a *RedisAdapter[K, V]) invalidateByNamespace(ctx context.Context) error {
	ctx, l := logger.From(ctx, log)
	scanPattern := a.namespace + scanAllPattern
	deletedCount := 0

	l.Internal("Invalidating all keys in namespace",
		logger.String("namespace", a.namespace),
		logger.String("pattern", scanPattern))

	scanIterator := a.client.Scan(ctx, 0, scanPattern, scanBatchSize).Iterator()
	batch := make([]string, 0, scanBatchSize)

	for scanIterator.Next(ctx) {
		batch = append(batch, scanIterator.Val())
		if len(batch) >= scanBatchSize {
			deletedCount += a.deleteBatch(ctx, batch)
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		deletedCount += a.deleteBatch(ctx, batch)
	}

	if err := scanIterator.Err(); err != nil {
		return fmt.Errorf("error during SCAN iteration in InvalidateAll: %w", err)
	}

	l.Internal("InvalidateAll completed",
		logger.String("namespace", a.namespace),
		logger.Int("keys_deleted", deletedCount))

	return nil
}

// InvalidateAll removes all cache entries within the set namespace.
// If search is enabled, it also drops the RediSearch index.
//
// When no namespace is set and AllowUnsafeFLUSHDB is false, the operation
// is blocked and an error is returned.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
//
// Returns error when the operation fails.
func (a *RedisAdapter[K, V]) InvalidateAll(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.flushTimeout, fmt.Errorf("redis InvalidateAll exceeded %s timeout", a.flushTimeout))
	defer cancel()

	a.dropIndex(timeoutCtx)

	if a.namespace == "" && a.allowUnsafeFLUSHDB {
		return a.flushUnsafe(timeoutCtx)
	}

	if a.namespace == "" {
		return errors.New("InvalidateAll blocked: no namespace configured and AllowUnsafeFLUSHDB is false")
	}

	return a.invalidateByNamespace(timeoutCtx)
}

// BulkRefresh updates several cache entries in the background using the
// bulk loader.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes keys ([]K) which specifies the cache keys to refresh.
// Takes bulkLoader (BulkLoader) which loads values for the given keys.
//
// Safe for concurrent use. Starts a goroutine that runs
// the bulk loader and updates the cache. The goroutine
// finishes when all keys are loaded and stored.
func (a *RedisAdapter[K, V]) BulkRefresh(ctx context.Context, keys []K, bulkLoader cache.BulkLoader[K, V]) {
	ctx, l := logger.From(ctx, log)
	go func() {
		defer goroutine.RecoverPanic(ctx, "cache.redisBulkRefresh")
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

// Refresh asynchronously refreshes a single cache entry using the provided
// loader.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to refresh.
// Takes loader (Loader[K, V]) which loads the fresh value for the
// given key.
//
// Returns <-chan LoadResult[V] which receives the loaded value or error once
// the background goroutine completes.
//
// Safe for concurrent use. Spawns a goroutine that loads the value and updates
// the cache. The returned channel is closed when the goroutine finishes.
func (a *RedisAdapter[K, V]) Refresh(ctx context.Context, key K, loader cache.Loader[K, V]) <-chan cache.LoadResult[V] {
	ctx, l := logger.From(ctx, log)
	resultChan := make(chan cache.LoadResult[V], 1)
	go func() {
		defer close(resultChan)
		defer goroutine.RecoverPanic(ctx, "cache.redisRefresh")
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

// All returns an iterator over all key-value pairs in the cache namespace.
//
// Returns iter.Seq2[K, V] which yields each key-value pair found in the
// namespace via Redis SCAN.
func (a *RedisAdapter[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		ctx := context.Background()

		scanPattern := a.allScanPattern()
		scanIterator := a.client.Scan(ctx, 0, scanPattern, 100).Iterator()
		for scanIterator.Next(ctx) {
			if !a.yieldScannedEntry(ctx, scanIterator.Val(), yield) {
				return
			}
		}
	}
}

// allScanPattern returns the Redis SCAN pattern for iterating
// all keys in the adapter's namespace.
//
// Returns string which is the wildcard pattern scoped to the
// configured namespace.
func (a *RedisAdapter[K, V]) allScanPattern() string {
	if a.namespace != "" {
		return a.namespace + scanAllPattern
	}
	return scanAllPattern
}

// yieldScannedEntry decodes a single scanned key, fetches its
// value, and yields it to the iterator consumer.
//
// Takes keyString (string) which is the raw Redis key to decode
// and look up.
// Takes yield (func(K, V) bool) which is the iterator callback
// that receives the decoded key and its value.
//
// Returns bool which is false when the consumer stopped
// iteration early, or true if processing should continue.
func (a *RedisAdapter[K, V]) yieldScannedEntry(ctx context.Context, keyString string, yield func(K, V) bool) bool {
	_, l := logger.From(ctx, log)

	key, err := a.decodeKey(keyString)
	if err != nil {
		l.Trace("Failed to decode key during iteration",
			logger.String(logKeyField, keyString),
			logger.Error(err))
		return true
	}

	value, ok, getErr := a.GetIfPresent(ctx, key)
	if getErr != nil {
		l.Trace("Failed to get value during iteration",
			logger.String(logKeyField, keyString),
			logger.Error(getErr))
		return true
	}
	if ok {
		return yield(key, value)
	}
	return true
}

// Keys returns an iterator over all keys in the cache namespace.
//
// Returns iter.Seq[K] which yields each key found in the namespace.
func (a *RedisAdapter[K, V]) Keys() iter.Seq[K] {
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
func (a *RedisAdapter[K, V]) Values() iter.Seq[V] {
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
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to retrieve.
//
// Returns Entry[K, V] which contains the value and metadata such
// as expiry time.
// Returns bool which indicates whether the key was found.
// Returns error when the operation fails.
func (a *RedisAdapter[K, V]) GetEntry(ctx context.Context, key K) (cache.Entry[K, V], bool, error) {
	return a.ProbeEntry(ctx, key)
}

// ProbeEntry retrieves entry metadata without affecting access
// patterns or TTL.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to probe.
//
// Returns Entry[K, V] which contains the value and metadata such
// as expiry time.
// Returns bool which indicates whether the key was found.
// Returns error when the operation fails.
func (a *RedisAdapter[K, V]) ProbeEntry(ctx context.Context, key K) (cache.Entry[K, V], bool, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("redis ProbeEntry exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return cache.Entry[K, V]{}, false, fmt.Errorf(errFmtEncodeKey, err)
	}

	pipe := a.client.Pipeline()
	getCmd := pipe.Get(timeoutCtx, keyString)
	ttlCmd := pipe.TTL(timeoutCtx, keyString)
	if _, err = pipe.Exec(timeoutCtx); err != nil {
		if errors.Is(err, redis.Nil) {
			return cache.Entry[K, V]{}, false, nil
		}
		return cache.Entry[K, V]{}, false, fmt.Errorf("redis pipeline exec failed: %w", err)
	}

	valBytes, err := getCmd.Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return cache.Entry[K, V]{}, false, nil
		}
		return cache.Entry[K, V]{}, false, fmt.Errorf("redis get failed: %w", err)
	}

	var v V
	encoder, err := a.registry.GetByType(reflect.TypeOf(v))
	if err != nil {
		return cache.Entry[K, V]{}, false, fmt.Errorf("failed to find encoder for type: %w", err)
	}

	unmarshalled, err := encoder.UnmarshalAny(valBytes)
	if err != nil {
		return cache.Entry[K, V]{}, false, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	value, ok := unmarshalled.(V)
	if !ok {
		return cache.Entry[K, V]{}, false, fmt.Errorf("type assertion failed after unmarshal for key %q", keyString)
	}

	ttl, _ := ttlCmd.Result()

	var expiresAtNano int64
	if ttl > 0 {
		expiresAtNano = time.Now().Add(ttl).UnixNano()
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

// EstimatedSize returns the approximate number of keys in the Redis database.
//
// Returns int which is the count of keys, or zero if the query fails.
func (a *RedisAdapter[K, V]) EstimatedSize() int {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("redis EstimatedSize exceeded %s timeout", a.operationTimeout))
	defer cancel()
	_, l := logger.From(ctx, log)

	if a.namespace == "" {
		size, err := a.client.DBSize(ctx).Result()
		if err != nil {
			l.Warn("Failed to get DBSize from Redis", logger.Error(err))
			return 0
		}
		return int(size)
	}

	var count int
	scanPattern := a.namespace + scanAllPattern
	scanIterator := a.client.Scan(ctx, 0, scanPattern, scanBatchSize).Iterator()
	for scanIterator.Next(ctx) {
		count++
	}
	if err := scanIterator.Err(); err != nil {
		l.Warn("Failed to scan keys for EstimatedSize", logger.Error(err))
	}
	return count
}

// Stats returns cache statistics from the Redis server.
//
// Returns cache.Stats which contains hit and miss counts from the server's
// INFO command.
func (a *RedisAdapter[K, V]) Stats() cache.Stats {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("redis Stats exceeded %s timeout", a.operationTimeout))
	defer cancel()
	_, l := logger.From(ctx, log)

	info, err := a.client.Info(ctx, "stats").Result()
	if err != nil {
		l.Warn("Failed to get INFO from Redis", logger.Error(err))
		return cache.Stats{}
	}

	var hits, misses int64
	for line := range strings.SplitSeq(info, "\r\n") {
		if strings.HasPrefix(line, "keyspace_hits:") {
			if _, err := fmt.Sscanf(line, "keyspace_hits:%d", &hits); err != nil {
				l.Trace("Failed to parse keyspace_hits", logger.String("line", line), logger.Error(err))
			}
		}
		if strings.HasPrefix(line, "keyspace_misses:") {
			if _, err := fmt.Sscanf(line, "keyspace_misses:%d", &misses); err != nil {
				l.Trace("Failed to parse keyspace_misses", logger.String("line", line), logger.Error(err))
			}
		}
	}

	var hitsUint, missesUint uint64
	if hits > 0 {
		hitsUint = safeconv.Int64ToUint64(hits)
	}
	if misses > 0 {
		missesUint = safeconv.Int64ToUint64(misses)
	}
	return cache.Stats{
		Hits:             hitsUint,
		Misses:           missesUint,
		Evictions:        0,
		LoadSuccessCount: 0,
		LoadFailureCount: 0,
		TotalLoadTime:    0,
	}
}

// Close releases the Redis client connection.
//
// Takes ctx (context.Context) for cancellation and timeout.
//
// Returns error when the client cannot be closed cleanly.
func (a *RedisAdapter[K, V]) Close(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.client.Close(); err != nil {
		return fmt.Errorf("error closing Redis client: %w", err)
	}
	return nil
}

// SetExpiresAfter updates the time-to-live for an existing key using the Redis
// EXPIRE command.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to update.
// Takes expiresAfter (time.Duration) which specifies the new time-to-live.
//
// Returns error when the operation fails.
func (a *RedisAdapter[K, V]) SetExpiresAfter(ctx context.Context, key K, expiresAfter time.Duration) error {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("redis SetExpiresAfter exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return fmt.Errorf(errFmtEncodeKey, err)
	}

	if err := a.client.PExpire(timeoutCtx, keyString, expiresAfter).Err(); err != nil {
		return fmt.Errorf("redis PExpire failed: %w", err)
	}

	return nil
}

// GetMaximum returns the Redis maxmemory configuration value.
//
// Returns uint64 which is the maximum memory in bytes, or 0 if the value
// cannot be retrieved or parsed.
func (a *RedisAdapter[K, V]) GetMaximum() uint64 {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("redis GetMaximum exceeded %s timeout", a.operationTimeout))
	defer cancel()
	_, l := logger.From(ctx, log)

	configResult, err := a.client.ConfigGet(ctx, "maxmemory").Result()
	if err != nil {
		return 0
	}
	if maxString, ok := configResult["maxmemory"]; ok {
		var maxMemory uint64
		if _, err := fmt.Sscanf(maxString, "%d", &maxMemory); err != nil {
			l.Trace("Failed to parse maxmemory", logger.String("value", maxString), logger.Error(err))
			return 0
		}
		return maxMemory
	}
	return 0
}

// SetMaximum is not supported by the Redis provider.
//
// Redis manages maximum capacity at the server level.
func (*RedisAdapter[K, V]) SetMaximum(_ uint64) {
	_, l := logger.From(context.Background(), log)
	l.Warn("SetMaximum is not supported by the Redis provider and will have no effect.")
}

// WeightedSize returns the memory usage in bytes from the Redis used_memory
// statistic.
//
// Returns uint64 which is the memory usage in bytes, or zero if the statistic
// cannot be read.
func (a *RedisAdapter[K, V]) WeightedSize() uint64 {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("redis WeightedSize exceeded %s timeout", a.operationTimeout))
	defer cancel()
	_, l := logger.From(ctx, log)

	info, err := a.client.Info(ctx, "memory").Result()
	if err != nil {
		return 0
	}
	for line := range strings.SplitSeq(info, "\r\n") {
		if strings.HasPrefix(line, "used_memory:") {
			var used uint64
			if _, err := fmt.Sscanf(line, "used_memory:%d", &used); err != nil {
				l.Trace("Failed to parse used_memory", logger.String("line", line), logger.Error(err))
				return 0
			}
			return used
		}
	}
	return 0
}

// SetRefreshableAfter is a no-op as Redis does not natively support refresh
// scheduling.
//
// Takes ctx (context.Context) for cancellation and timeout.
//
// Returns error (always nil for this no-op implementation).
func (*RedisAdapter[K, V]) SetRefreshableAfter(ctx context.Context, _ K, _ time.Duration) error {
	_, l := logger.From(ctx, log)
	l.Internal("SetRefreshableAfter is not natively supported by the Redis provider.")
	return nil
}
