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

package cache_provider_valkey

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"reflect"
	"strings"
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

	// logKeyMaxRetries is the attribute key for logging max retry counts.
	logKeyMaxRetries = "max_retries"

	// maxTransactionCommands is the capacity for MULTI/EXEC transaction
	// command slices (MULTI + action + EXEC).
	maxTransactionCommands = 3

	// errMessageEncodeKey is the warning message logged when key encoding fails.
	errMessageEncodeKey = "Failed to encode key"

	// errFmtEncodeKey is the format string used when key encoding fails.
	errFmtEncodeKey = "failed to encode key: %w"
)

// ValkeyAdapter implements the ProviderPort using a Valkey client. It encodes
// keys to strings and uses a type-driven EncodingRegistry for values.
type ValkeyAdapter[K comparable, V any] struct {
	// expiryCalculator sets the expiry time for each key; optional.
	expiryCalculator cache.ExpiryCalculator[K, V]

	// refreshCalculator calculates when entries become ready for background
	// refresh; optional.
	refreshCalculator cache.RefreshCalculator[K, V]

	// registry encodes values before they are stored.
	registry *cache.EncodingRegistry

	// client is the Valkey client for storage operations.
	client valkey.Client

	// keyRegistry stores encoders for complex key types; nil uses fmt.Sprintf.
	keyRegistry *cache.EncodingRegistry

	// schema is the search schema for this cache; nil means search is disabled.
	schema *cache.SearchSchema

	// sf deduplicates concurrent loads for the same key.
	sf singleflight.Group

	// namespace is the prefix added to all keys in Valkey.
	namespace string

	// indexName is the search index name for this namespace.
	indexName string

	// ttl is the default time-to-live for cache entries.
	ttl time.Duration

	// operationTimeout is the time limit for a single Valkey operation.
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

var _ cache.ProviderPort[any, any] = (*ValkeyAdapter[any, any])(nil)

// encodeKey converts a key of type K to a namespace-prefixed Valkey key string
// using the shared cache_domain encoding logic.
//
// Takes key (K) which is the cache key to encode.
//
// Returns string which is the encoded Valkey key, with namespace prefix if set.
// Returns error when no encoder is registered for the key type or when
// marshalling fails.
func (a *ValkeyAdapter[K, V]) encodeKey(key K) (string, error) {
	return cache_domain.EncodeKey(key, a.namespace, a.keyRegistry)
}

// decodeKey converts a Valkey key string back to a key of type K using the
// shared cache_domain decoding logic.
//
// Takes keyString (string) which is the Valkey key to decode.
//
// Returns K which is the decoded key value.
// Returns error when the namespace prefix is missing, decoding fails, or no
// encoder is registered for the key type.
func (a *ValkeyAdapter[K, V]) decodeKey(keyString string) (K, error) {
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
func (a *ValkeyAdapter[K, V]) GetIfPresent(ctx context.Context, key K) (V, bool, error) {
	if err := ctx.Err(); err != nil {
		return *new(V), false, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("valkey GetIfPresent exceeded %s timeout", a.operationTimeout))
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
		return *new(V), false, fmt.Errorf("valkey get failed for key %s: %w", keyString, err)
	}

	var v V
	encoder, err := a.registry.GetByType(reflect.TypeOf(v))
	if err != nil {
		return *new(V), false, fmt.Errorf("failed to find encoder for type: %w", err)
	}

	unmarshalled, err := encoder.UnmarshalAny(value)
	if err != nil {
		return *new(V), false, fmt.Errorf("failed to unmarshal value from Valkey: %w", err)
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
// Takes key (K) which identifies the cached value to retrieve.
// Takes loader (Loader[K, V]) which loads the value if not already cached.
//
// Returns V which is the cached or newly loaded value.
// Returns error when key encoding fails, the loader fails, or type assertion
// fails.
func (a *ValkeyAdapter[K, V]) Get(ctx context.Context, key K, loader cache.Loader[K, V]) (V, error) {
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
			return nil, setErr
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
// for search indexing.
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
func (a *ValkeyAdapter[K, V]) Set(ctx context.Context, key K, value V, tags ...string) error {
	ctx, l := logger.From(ctx, log)

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("valkey Set context cancelled: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("valkey Set exceeded %s timeout", a.operationTimeout))
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
			return fmt.Errorf("failed to marshal value for Valkey: %w", err)
		}

		if err := a.client.Do(timeoutCtx, a.client.B().Set().Key(keyString).Value(string(valBytes)).Ex(ttl).Build()).Error(); err != nil {
			return fmt.Errorf("valkey set failed for key %s: %w", keyString, err)
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
// Returns error when encoding, marshalling, or the Valkey operation fails.
func (a *ValkeyAdapter[K, V]) SetWithTTL(ctx context.Context, key K, value V, ttl time.Duration, tags ...string) error {
	ctx, l := logger.From(ctx, log)

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("valkey SetWithTTL exceeded %s timeout", a.operationTimeout))
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
		l.Warn("Failed to marshal value for Valkey", logger.String(logKeyField, keyString), logger.Error(err))
		return fmt.Errorf("marshal failed: %w", err)
	}

	if err := a.client.Do(timeoutCtx, a.client.B().Set().Key(keyString).Value(string(valBytes)).Px(ttl).Build()).Error(); err != nil {
		l.Warn("Valkey Set with TTL failed", logger.String(logKeyField, keyString), logger.Error(err))
		return fmt.Errorf("valkey set failed: %w", err)
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
func (a *ValkeyAdapter[K, V]) Invalidate(ctx context.Context, key K) error {
	ctx, l := logger.From(ctx, log)

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("valkey Invalidate context cancelled: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("valkey Invalidate exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return fmt.Errorf(errFmtEncodeKey, err)
	}

	if err := removeKeyFromTags(timeoutCtx, a.client, a.namespace, keyString); err != nil {
		l.Warn("Failed to remove key from tag sets", logger.String(logKeyField, keyString), logger.Error(err))
	}

	if err := a.client.Do(timeoutCtx, a.client.B().Del().Key(keyString).Build()).Error(); err != nil {
		return fmt.Errorf("valkey del failed for key %s: %w", keyString, err)
	}

	return nil
}

// decodeValue decodes bytes into a value of type V using the shared
// cache_domain decoding logic.
//
// Takes valBytes ([]byte) which contains the encoded data to decode.
//
// Returns V which is the decoded value.
// Returns error when the encoder cannot be found, unmarshalling fails, or type
// assertion fails.
func (a *ValkeyAdapter[K, V]) decodeValue(valBytes []byte) (V, error) {
	return cache_domain.DecodeValue[V](valBytes, a.registry)
}

// encodeValue encodes a value of type V to bytes using the shared
// cache_domain encoding logic.
//
// Takes value (V) which is the value to encode.
//
// Returns []byte which contains the encoded value.
// Returns error when no encoder is found for the value type or encoding fails.
func (a *ValkeyAdapter[K, V]) encodeValue(value V) ([]byte, error) {
	return cache_domain.EncodeValue(value, a.registry)
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
func (a *ValkeyAdapter[K, V]) Compute(ctx context.Context, key K, computeFunction func(oldValue V, found bool) (newValue V, action cache.ComputeAction)) (V, bool, error) {
	ctx, l := logger.From(ctx, log)

	if err := ctx.Err(); err != nil {
		return *new(V), false, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("valkey Compute exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	for attempt := range a.maxComputeRetries {
		err := a.client.Dedicated(func(c valkey.DedicatedClient) error {
			if err := c.Do(timeoutCtx, c.B().Watch().Key(keyString).Build()).Error(); err != nil {
				return fmt.Errorf("valkey WATCH failed for Compute key %s: %w", keyString, err)
			}

			oldValue, found, getErr := a.fetchValueInDedicated(timeoutCtx, c, keyString)
			if getErr != nil {
				return getErr
			}

			newValue, action := computeFunction(oldValue, found)

			return a.executeComputeInDedicated(timeoutCtx, c, keyString, newValue, action, found)
		})

		shouldRetry, value, found, retryErr := a.handleComputeRetryResult(ctx, key, keyString, err, attempt)
		if !shouldRetry {
			return value, found, retryErr
		}
	}

	l.Warn("Compute max retries exceeded",
		logger.String(logKeyField, keyString),
		logger.Int(logKeyMaxRetries, a.maxComputeRetries))
	return *new(V), false, fmt.Errorf("compute max retries exceeded for key %s", keyString)
}

// fetchValueInDedicated fetches and decodes a value within a dedicated Valkey
// connection.
//
// Takes c (valkey.DedicatedClient) which is the dedicated connection.
// Takes keyString (string) which is the key to fetch.
//
// Returns V which is the decoded value if found.
// Returns bool which is true if the key exists.
// Returns error when the Valkey get or decoding fails.
func (a *ValkeyAdapter[K, V]) fetchValueInDedicated(ctx context.Context, c valkey.DedicatedClient, keyString string) (V, bool, error) {
	var zero V
	valBytes, err := c.Do(ctx, c.B().Get().Key(keyString).Build()).AsBytes()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return zero, false, nil
		}
		return zero, false, fmt.Errorf("valkey get failed: %w", err)
	}

	value, err := a.decodeValue(valBytes)
	if err != nil {
		return zero, false, err
	}
	return value, true, nil
}

// executeComputeInDedicated executes a compute action within a dedicated
// connection using MULTI/EXEC.
//
// Takes c (valkey.DedicatedClient) which is the dedicated connection for the
// transaction.
// Takes keyString (string) which is the cache key to operate on.
// Takes newValue (V) which is the value to set when action is ComputeActionSet.
// Takes action (cache.ComputeAction) which specifies the operation to perform.
// Takes found (bool) which indicates whether the key existed before compute.
//
// Returns error when encoding fails or the transaction fails.
func (a *ValkeyAdapter[K, V]) executeComputeInDedicated(ctx context.Context, c valkey.DedicatedClient, keyString string, newValue V, action cache.ComputeAction, found bool) error {
	cmds := make(valkey.Commands, 0, maxTransactionCommands)
	cmds = append(cmds, c.B().Multi().Build())

	switch action {
	case cache.ComputeActionSet:
		valBytes, err := a.encodeValue(newValue)
		if err != nil {
			return fmt.Errorf("failed to encode value for compute on key %s: %w", keyString, err)
		}
		cmds = append(cmds, c.B().Set().Key(keyString).Value(string(valBytes)).Ex(a.ttl).Build())
	case cache.ComputeActionDelete:
		if found {
			cmds = append(cmds, c.B().Del().Key(keyString).Build())
		}
	case cache.ComputeActionNoop:
	}

	cmds = append(cmds, c.B().Exec().Build())

	results := c.DoMulti(ctx, cmds...)
	return results[len(results)-1].Error()
}

// handleComputeResult is the shared implementation for processing the result
// of a compute transaction attempt.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which is the cache key being computed.
// Takes keyString (string) which is the string representation of the key for
// logging.
// Takes opName (string) which is the operation name for log messages.
// Takes err (error) which is the error from the transaction attempt, or nil on
// success.
// Takes attempt (int) which is the current retry attempt number.
//
// Returns value (V) which is the computed value if found.
// Returns found (bool) which indicates whether a valid value was found.
// Returns shouldRetry (bool) which indicates whether the operation should be
// retried.
// Returns retErr (error) when the operation fails.
func (a *ValkeyAdapter[K, V]) handleComputeResult(ctx context.Context, key K, keyString, opName string, err error, attempt int) (value V, found bool, shouldRetry bool, retErr error) {
	ctx, l := logger.From(ctx, log)

	var zero V

	if err == nil {
		if finalValue, ok, getErr := a.GetIfPresent(ctx, key); getErr != nil {
			return zero, false, false, getErr
		} else if ok {
			return finalValue, true, false, nil
		}
		return zero, false, false, nil
	}

	if isTransactionConflict(err) {
		l.Trace(opName+" transaction failed, retrying",
			logger.String(logKeyField, keyString),
			logger.Int("attempt", attempt+1))
		return zero, false, true, nil
	}

	l.Warn(opName+" transaction error",
		logger.String(logKeyField, keyString),
		logger.Error(err))
	return zero, false, false, fmt.Errorf("%s transaction error: %w", opName, err)
}

// handleComputeRetryResult processes the result of a Compute transaction
// attempt. Delegates to handleComputeResult with reordered return values.
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
// Returns error when the operation fails.
func (a *ValkeyAdapter[K, V]) handleComputeRetryResult(ctx context.Context, key K, keyString string, err error, attempt int) (bool, V, bool, error) {
	value, found, shouldRetry, retErr := a.handleComputeResult(ctx, key, keyString, "Compute", err, attempt)
	return shouldRetry, value, found, retErr
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
func (a *ValkeyAdapter[K, V]) ComputeIfAbsent(ctx context.Context, key K, computeFunction func() V) (V, bool, error) {
	ctx, l := logger.From(ctx, log)

	if err := ctx.Err(); err != nil {
		return *new(V), false, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("valkey ComputeIfAbsent exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	for range a.maxComputeRetries {
		didCompute := false
		err := a.client.Dedicated(func(c valkey.DedicatedClient) error {
			computed, txErr := a.computeIfAbsentTransaction(timeoutCtx, c, keyString, computeFunction)
			didCompute = computed
			return txErr
		})

		value, computed, shouldRetry, retryErr := a.handleComputeIfAbsentResult(ctx, key, keyString, err, didCompute)
		if !shouldRetry {
			return value, computed, retryErr
		}
	}

	l.Warn("ComputeIfAbsent max retries exceeded",
		logger.String(logKeyField, keyString),
		logger.Int(logKeyMaxRetries, a.maxComputeRetries))
	return *new(V), false, fmt.Errorf("compute if absent max retries exceeded for key %s", keyString)
}

// handleComputeIfAbsentResult processes the result of a ComputeIfAbsent
// transaction attempt.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which is the cache key being computed.
// Takes keyString (string) which is the string form of the key for logging.
// Takes err (error) which is the error from the transaction attempt.
// Takes didCompute (bool) which indicates whether the value was computed.
//
// Returns value (V) which is the cached value if found after the transaction.
// Returns computed (bool) which indicates whether the value was computed.
// Returns shouldRetry (bool) which is true when a transaction conflict
// occurred.
// Returns error when the operation fails.
func (a *ValkeyAdapter[K, V]) handleComputeIfAbsentResult(ctx context.Context, key K, keyString string, err error, didCompute bool) (value V, computed bool, shouldRetry bool, retryErr error) {
	ctx, l := logger.From(ctx, log)

	var zero V

	if err == nil {
		if finalValue, ok, getErr := a.GetIfPresent(ctx, key); getErr != nil {
			return zero, false, false, getErr
		} else if ok {
			return finalValue, didCompute, false, nil
		}
		return zero, false, false, nil
	}

	if isTransactionConflict(err) {
		l.Trace("ComputeIfAbsent transaction failed, retrying",
			logger.String(logKeyField, keyString))
		return zero, false, true, nil
	}

	l.Warn("ComputeIfAbsent transaction error",
		logger.String(logKeyField, keyString),
		logger.Error(err))
	return zero, false, false, fmt.Errorf("compute if absent transaction error: %w", err)
}

// computeIfAbsentTransaction executes the WATCH/EXISTS/MULTI/SET/EXEC
// sequence for ComputeIfAbsent within a dedicated connection.
//
// Takes c (valkey.DedicatedClient) which provides the dedicated connection
// for the transaction.
// Takes keyString (string) which is the cache key to check and possibly set.
// Takes computeFunction (func() V) which computes the value if the key is absent.
//
// Returns bool which indicates whether the compute function was called.
// Returns error when the transaction fails at any stage.
func (a *ValkeyAdapter[K, V]) computeIfAbsentTransaction(ctx context.Context, c valkey.DedicatedClient, keyString string, computeFunction func() V) (bool, error) {
	if err := c.Do(ctx, c.B().Watch().Key(keyString).Build()).Error(); err != nil {
		return false, fmt.Errorf("valkey WATCH failed for ComputeIfAbsent key %s: %w", keyString, err)
	}

	exists, err := c.Do(ctx, c.B().Exists().Key(keyString).Build()).AsInt64()
	if err != nil {
		return false, fmt.Errorf("valkey exists check failed: %w", err)
	}
	if exists > 0 {
		return false, nil
	}

	newValue := computeFunction()
	valBytes, err := a.encodeValue(newValue)
	if err != nil {
		return true, fmt.Errorf("failed to encode value for ComputeIfAbsent key %s: %w", keyString, err)
	}

	cmds := valkey.Commands{
		c.B().Multi().Build(),
		c.B().Set().Key(keyString).Value(string(valBytes)).Ex(a.ttl).Build(),
		c.B().Exec().Build(),
	}
	results := c.DoMulti(ctx, cmds...)
	return true, results[len(results)-1].Error()
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
//
//nolint:dupl // similar ops, different semantics
func (a *ValkeyAdapter[K, V]) ComputeIfPresent(
	ctx context.Context, key K, computeFunction func(oldValue V) (newValue V, action cache.ComputeAction),
) (V, bool, error) {
	ctx, l := logger.From(ctx, log)

	if err := ctx.Err(); err != nil {
		return *new(V), false, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("valkey ComputeIfPresent exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	for attempt := range a.maxComputeRetries {
		err := a.client.Dedicated(func(c valkey.DedicatedClient) error {
			return a.computeIfPresentTransaction(timeoutCtx, c, keyString, computeFunction)
		})

		value, found, shouldRetry, retryErr := a.handleComputeIfPresentResult(ctx, key, keyString, err, attempt)
		if !shouldRetry {
			return value, found, retryErr
		}
	}

	l.Warn("ComputeIfPresent max retries exceeded",
		logger.String(logKeyField, keyString),
		logger.Int(logKeyMaxRetries, a.maxComputeRetries))
	return *new(V), false, fmt.Errorf("compute if present max retries exceeded for key %s", keyString)
}

// computeIfPresentTransaction executes the WATCH/GET/MULTI/action/EXEC
// sequence for ComputeIfPresent within a dedicated connection.
//
// Takes c (valkey.DedicatedClient) which provides the dedicated connection.
// Takes keyString (string) which is the cache key to operate on.
// Takes computeFunction (func(V) (V, cache.ComputeAction)) which computes the new
// value from the existing one.
//
// Returns error when the watch, fetch, or execute operation fails.
func (a *ValkeyAdapter[K, V]) computeIfPresentTransaction(ctx context.Context, c valkey.DedicatedClient, keyString string, computeFunction func(V) (V, cache.ComputeAction)) error {
	if err := c.Do(ctx, c.B().Watch().Key(keyString).Build()).Error(); err != nil {
		return fmt.Errorf("valkey WATCH failed for ComputeIfPresent key %s: %w", keyString, err)
	}

	oldValue, found, getErr := a.fetchValueInDedicated(ctx, c, keyString)
	if getErr != nil {
		return getErr
	}
	if !found {
		return nil
	}

	newValue, action := computeFunction(oldValue)

	return a.executeComputeInDedicated(ctx, c, keyString, newValue, action, true)
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
// Returns found (bool) which indicates whether a value was found.
// Returns shouldRetry (bool) which indicates whether the operation should be
// retried due to a transaction conflict.
// Returns error when the operation fails.
func (a *ValkeyAdapter[K, V]) handleComputeIfPresentResult(ctx context.Context, key K, keyString string, err error, attempt int) (value V, found bool, shouldRetry bool, retryErr error) {
	return a.handleComputeResult(ctx, key, keyString, "ComputeIfPresent", err, attempt)
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
//
//nolint:dupl // similar ops, different semantics
func (a *ValkeyAdapter[K, V]) ComputeWithTTL(
	ctx context.Context, key K, computeFunction func(oldValue V, found bool) cache.ComputeResult[V],
) (V, bool, error) {
	ctx, l := logger.From(ctx, log)

	if err := ctx.Err(); err != nil {
		return *new(V), false, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("valkey ComputeWithTTL exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	for attempt := range a.maxComputeRetries {
		err := a.client.Dedicated(func(c valkey.DedicatedClient) error {
			return a.computeWithTTLTransaction(timeoutCtx, c, keyString, computeFunction)
		})

		shouldRetry, value, found, retryErr := a.handleComputeRetryResult(ctx, key, keyString, err, attempt)
		if !shouldRetry {
			return value, found, retryErr
		}
	}

	l.Warn("ComputeWithTTL max retries exceeded",
		logger.String(logKeyField, keyString),
		logger.Int(logKeyMaxRetries, a.maxComputeRetries))
	return *new(V), false, fmt.Errorf("compute with TTL max retries exceeded for key %s", keyString)
}

// computeWithTTLTransaction executes the WATCH/GET/MULTI/action/EXEC
// sequence for ComputeWithTTL within a dedicated connection.
//
// Takes c (valkey.DedicatedClient) which provides the dedicated connection.
// Takes keyString (string) which is the cache key to operate on.
// Takes computeFunction (func(V, bool) cache.ComputeResult[V]) which computes the
// new value based on the old value and whether it was found.
//
// Returns error when the watch, fetch, or execute operations fail.
func (a *ValkeyAdapter[K, V]) computeWithTTLTransaction(ctx context.Context, c valkey.DedicatedClient, keyString string, computeFunction func(V, bool) cache.ComputeResult[V]) error {
	if err := c.Do(ctx, c.B().Watch().Key(keyString).Build()).Error(); err != nil {
		return fmt.Errorf("valkey WATCH failed for ComputeWithTTL key %s: %w", keyString, err)
	}

	oldValue, found, getErr := a.fetchValueInDedicated(ctx, c, keyString)
	if getErr != nil {
		return getErr
	}

	result := computeFunction(oldValue, found)

	return a.executeComputeWithTTLInDedicated(ctx, c, keyString, result, found)
}

// executeComputeWithTTLInDedicated builds and executes the MULTI/EXEC
// transaction for a ComputeResult that may carry its own TTL.
//
// Takes c (valkey.DedicatedClient) which provides the dedicated connection for
// the transaction.
// Takes keyString (string) which is the cache key to operate on.
// Takes result (cache.ComputeResult[V]) which contains the action to perform
// and the value to set.
// Takes found (bool) which indicates whether the key existed before compute.
//
// Returns error when the transaction fails or value encoding fails.
func (a *ValkeyAdapter[K, V]) executeComputeWithTTLInDedicated(ctx context.Context, c valkey.DedicatedClient, keyString string, result cache.ComputeResult[V], found bool) error {
	cmds := make(valkey.Commands, 0, maxTransactionCommands)
	cmds = append(cmds, c.B().Multi().Build())

	switch result.Action {
	case cache.ComputeActionSet:
		valBytes, err := a.encodeValue(result.Value)
		if err != nil {
			return fmt.Errorf("failed to encode value for ComputeWithTTL key %s: %w", keyString, err)
		}
		effectiveTTL := a.ttl
		if result.TTL > 0 {
			effectiveTTL = result.TTL
		}
		cmds = append(cmds, c.B().Set().Key(keyString).Value(string(valBytes)).Ex(effectiveTTL).Build())
	case cache.ComputeActionDelete:
		if found {
			cmds = append(cmds, c.B().Del().Key(keyString).Build())
		}
	case cache.ComputeActionNoop:
	}

	cmds = append(cmds, c.B().Exec().Build())
	results := c.DoMulti(ctx, cmds...)
	return results[len(results)-1].Error()
}

// bulkEncodeKeys encodes a slice of keys and returns the encoded string keys
// and a reverse map for decoding. Keys that fail to encode are skipped.
//
// Takes keys ([]K) which contains the keys to encode.
//
// Returns []string which contains the encoded keys.
// Returns map[string]K which maps encoded strings back to their original keys.
func (a *ValkeyAdapter[K, V]) bulkEncodeKeys(ctx context.Context, keys []K) ([]string, map[string]K) {
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
// Takes value (string) which is the raw value from the MGET response.
// Takes keyString (string) which identifies the key for logging purposes.
//
// Returns V which is the unmarshalled value.
// Returns bool which indicates whether processing succeeded.
func (a *ValkeyAdapter[K, V]) processBulkGetResult(ctx context.Context, value string, keyString string) (V, bool) {
	_, l := logger.From(ctx, log)

	var zero V

	encoder, err := a.registry.GetByType(reflect.TypeOf(zero))
	if err != nil {
		l.Warn("Failed to find encoder for type", logger.String(logKeyField, keyString), logger.Error(err))
		return zero, false
	}

	unmarshalled, err := encoder.UnmarshalAny([]byte(value))
	if err != nil {
		l.Warn("Failed to unmarshal value from Valkey MGET", logger.String(logKeyField, keyString), logger.Error(err))
		return zero, false
	}

	result, ok := unmarshalled.(V)
	if !ok {
		l.Warn("Type assertion failed after unmarshal", logger.String(logKeyField, keyString))
		return zero, false
	}

	return result, true
}

// storeLoadedValues stores loaded values to Valkey using DoMulti and updates
// the results map.
//
// Takes loaded (map[K]V) which contains the key-value pairs to store.
// Takes results (map[K]V) which receives successfully stored entries.
func (a *ValkeyAdapter[K, V]) storeLoadedValues(ctx context.Context, loaded map[K]V, results map[K]V) {
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
			l.Warn("Failed to marshal loaded value for Valkey", logger.String(logKeyField, keyString), logger.Error(marshalErr))
			continue
		}
		cmds = append(cmds, a.client.B().Set().Key(keyString).Value(string(valBytes)).Ex(a.ttl).Build())
		results[k] = v
	}

	if len(cmds) > 0 {
		for _, response := range a.client.DoMulti(ctx, cmds...) {
			if err := response.Error(); err != nil {
				l.Warn("Failed to execute SET in bulk load", logger.Error(err))
			}
		}
	}
}

// BulkGet retrieves multiple values from the cache, loading missing ones
// via the bulk loader.
//
// Takes keys ([]K) which specifies the cache keys to retrieve.
// Takes bulkLoader (BulkLoader[K, V]) which loads values for any cache misses.
//
// Returns map[K]V which contains the retrieved and loaded values.
// Returns error when the Valkey MGET operation or bulk loader fails.
func (a *ValkeyAdapter[K, V]) BulkGet(ctx context.Context, keys []K, bulkLoader cache.BulkLoader[K, V]) (map[K]V, error) {
	ctx, l := logger.From(ctx, log)

	if len(keys) == 0 {
		return make(map[K]V), nil
	}

	results := make(map[K]V, len(keys))
	keyStrs, keyMap := a.bulkEncodeKeys(ctx, keys)
	if len(keyStrs) == 0 {
		return results, nil
	}

	mgetResults := a.client.DoMulti(ctx,
		a.client.B().Mget().Key(keyStrs...).Build(),
	)
	if len(mgetResults) == 0 {
		return results, errors.New("valkey MGET returned empty response")
	}

	vals, err := mgetResults[0].ToArray()
	if err != nil {
		l.Warn("Valkey MGET failed", logger.Int("key_count", len(keys)), logger.Error(err))
		return results, fmt.Errorf("valkey MGET failed: %w", err)
	}

	var misses []K
	for i, message := range vals {
		keyString := keyStrs[i]
		originalKey := keyMap[keyString]

		value, err := message.ToString()
		if err != nil {
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
// Takes value (V) which is the value to marshal.
// Takes defaultTTL (time.Duration) which is the fallback TTL if no calculator
// is configured.
//
// Returns string which is the encoded key.
// Returns []byte which is the marshalled value.
// Returns time.Duration which is the TTL for this entry.
// Returns bool which is false when encoding or marshalling fails.
func (a *ValkeyAdapter[K, V]) prepareBulkSetItem(ctx context.Context, key K, value V, defaultTTL time.Duration) (string, []byte, time.Duration, bool) {
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
		l.Warn("Failed to marshal value for Valkey in BulkSet",
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
// Takes tags (...string) which specifies optional tags to associate with the
// keys.
//
// Returns error when the pipeline execution fails.
func (a *ValkeyAdapter[K, V]) BulkSet(ctx context.Context, items map[K]V, tags ...string) error {
	ctx, l := logger.From(ctx, log)

	if len(items) == 0 {
		return nil
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.bulkOperationTimeout, fmt.Errorf("valkey BulkSet exceeded %s timeout", a.bulkOperationTimeout))
	defer cancel()

	if a.needsJSONStorage() {
		return a.bulkSetJSON(timeoutCtx, items, tags...)
	}

	cmds := make(valkey.Commands, 0, len(items)*(1+len(tags)))

	for key, value := range items {
		keyString, valBytes, entryTTL, ok := a.prepareBulkSetItem(ctx, key, value, a.ttl)
		if !ok {
			continue
		}

		cmds = append(cmds, a.client.B().Set().Key(keyString).Value(string(valBytes)).Ex(entryTTL).Build())

		for _, tag := range tags {
			cmds = append(cmds, a.client.B().Sadd().Key(a.namespace+tagPrefix+tag).Member(keyString).Build())
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

// bulkSetJSON stores multiple items using JSON.SET for search-indexed
// namespaces.
//
// Takes items (map[K]V) which contains the key-value pairs to store.
// Takes tags (...string) which specifies optional tags to associate with each
// key.
//
// Returns error when an item cannot be stored.
func (a *ValkeyAdapter[K, V]) bulkSetJSON(ctx context.Context, items map[K]V, tags ...string) error {
	ctx, l := logger.From(ctx, log)

	for key, value := range items {
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
func (a *ValkeyAdapter[K, V]) InvalidateByTags(ctx context.Context, tags ...string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.bulkOperationTimeout, fmt.Errorf("valkey InvalidateByTags exceeded %s timeout", a.bulkOperationTimeout))
	defer cancel()

	count, err := performTagInvalidation(timeoutCtx, a.client, a.namespace, tags)
	if err != nil {
		return 0, fmt.Errorf("failed to invalidate by tags: %w", err)
	}
	return count, nil
}

// flushUnsafe runs FLUSHDB on the Valkey instance, deleting all keys in the
// current database.
//
// Returns error when the FLUSHDB command fails.
func (a *ValkeyAdapter[K, V]) flushUnsafe(ctx context.Context) error {
	ctx, l := logger.From(ctx, log)

	l.Warn("Executing FLUSHDB on Valkey. ALL keys in the current database will be deleted.",
		logger.Bool("unsafe_mode", true))
	if err := a.client.Do(ctx, a.client.B().Flushdb().Build()).Error(); err != nil {
		return fmt.Errorf("valkey FLUSHDB failed: %w", err)
	}
	return nil
}

// deleteBatch deletes a batch of keys and returns the number deleted.
//
// Takes batch ([]string) which contains the keys to delete.
//
// Returns int which is the number of keys successfully deleted, or zero on
// error.
func (a *ValkeyAdapter[K, V]) deleteBatch(ctx context.Context, batch []string) int {
	ctx, l := logger.From(ctx, log)

	deleted, err := a.client.Do(ctx, a.client.B().Del().Key(batch...).Build()).AsInt64()
	if err != nil {
		l.Warn("Failed to delete batch of keys",
			logger.Int("batch_size", len(batch)), logger.Error(err))
		return 0
	}
	return int(deleted)
}

// invalidateByNamespace scans for and deletes all keys that match the
// namespace pattern.
func (a *ValkeyAdapter[K, V]) invalidateByNamespace(ctx context.Context) {
	ctx, l := logger.From(ctx, log)

	scanPattern := a.namespace + "*"
	deletedCount := 0

	l.Internal("Invalidating all keys in namespace",
		logger.String("namespace", a.namespace),
		logger.String("pattern", scanPattern))

	var cursor uint64
	for {
		response, err := a.client.Do(ctx, a.client.B().Scan().Cursor(cursor).Match(scanPattern).Count(scanBatchSize).Build()).AsScanEntry()
		if err != nil {
			l.Error("Error during SCAN iteration in InvalidateAll", logger.Error(err))
			break
		}

		if len(response.Elements) > 0 {
			deletedCount += a.deleteBatch(ctx, response.Elements)
		}

		cursor = response.Cursor
		if cursor == 0 {
			break
		}
	}

	l.Internal("InvalidateAll completed",
		logger.String("namespace", a.namespace),
		logger.Int("keys_deleted", deletedCount))
}

// InvalidateAll removes all cache entries within the set namespace.
// If search is enabled, it also drops the search index.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// When no namespace is set and AllowUnsafeFLUSHDB is false, the operation
// is blocked and an error is returned.
//
// Takes ctx (context.Context) for cancellation and timeout.
//
// Returns error when the operation fails.
func (a *ValkeyAdapter[K, V]) InvalidateAll(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("valkey InvalidateAll context cancelled: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.flushTimeout, fmt.Errorf("valkey InvalidateAll exceeded %s timeout", a.flushTimeout))
	defer cancel()

	a.dropIndex(timeoutCtx)

	if a.namespace == "" && a.allowUnsafeFLUSHDB {
		return a.flushUnsafe(timeoutCtx)
	}

	if a.namespace == "" {
		return errors.New("invalidate all blocked: no namespace configured and AllowUnsafeFLUSHDB is false")
	}

	a.invalidateByNamespace(timeoutCtx)
	return nil
}

// BulkRefresh updates several cache entries in the background using the
// bulk loader.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes keys ([]K) which specifies the cache keys to refresh.
// Takes bulkLoader (BulkLoader) which loads the new values for the keys.
//
// Safe for concurrent use. Spawns a goroutine that loads values and updates
// the cache. Returns immediately; errors are logged but not returned.
func (a *ValkeyAdapter[K, V]) BulkRefresh(ctx context.Context, keys []K, bulkLoader cache.BulkLoader[K, V]) {
	ctx, l := logger.From(ctx, log)

	go func() {
		defer goroutine.RecoverPanic(ctx, "cache.valkeyBulkRefresh")
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
// Returns <-chan cache.LoadResult[V] which receives the load outcome
// once the background goroutine completes.
//
// Safe for concurrent use. Spawns a goroutine that loads the value and updates
// the cache. The channel is closed after the result is sent.
func (a *ValkeyAdapter[K, V]) Refresh(ctx context.Context, key K, loader cache.Loader[K, V]) <-chan cache.LoadResult[V] {
	ctx, l := logger.From(ctx, log)

	resultChan := make(chan cache.LoadResult[V], 1)
	go func() {
		defer close(resultChan)
		defer goroutine.RecoverPanic(ctx, "cache.valkeyRefresh")
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
// Returns iter.Seq2[K, V] which yields each key-value pair found in
// the namespace via a full SCAN.
func (a *ValkeyAdapter[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		scanPattern := "*"
		if a.namespace != "" {
			scanPattern = a.namespace + "*"
		}

		a.scanAndYield(scanPattern, yield)
	}
}

// scanAndYield performs a full SCAN over the given pattern and
// yields each decoded key-value pair. It returns when the scan
// completes or the yield function signals to stop.
//
// Takes pattern (string) which specifies the key pattern to match.
// Takes yield (func(K, V) bool) which receives each key-value pair
// and returns false to stop iteration.
//
// Note: iterator methods (All, Keys, Values) do not accept a
// context from the interface, so a background context is used
// here.
func (a *ValkeyAdapter[K, V]) scanAndYield(pattern string, yield func(K, V) bool) {
	ctx := context.Background()
	_, l := logger.From(ctx, log)

	var cursor uint64

	for {
		response, err := a.client.Do(ctx, a.client.B().Scan().Cursor(cursor).Match(pattern).Count(scanBatchSize).Build()).AsScanEntry()
		if err != nil {
			l.Trace("Error during SCAN iteration", logger.Error(err))
			return
		}

		if !a.yieldScannedElements(ctx, response.Elements, yield) {
			return
		}

		cursor = response.Cursor
		if cursor == 0 {
			return
		}
	}
}

// yieldScannedElements decodes and yields each key-value pair from a SCAN
// batch.
//
// Takes ctx (context.Context) for cancellation and timeout on value retrieval.
// Takes elements ([]string) which contains the raw keys from a SCAN result.
// Takes yield (func(K, V) bool) which receives each decoded key-value pair.
//
// Returns bool which is false if yield signals to stop iteration, true
// otherwise.
func (a *ValkeyAdapter[K, V]) yieldScannedElements(ctx context.Context, elements []string, yield func(K, V) bool) bool {
	ctx, l := logger.From(ctx, log)

	for _, keyString := range elements {
		key, err := a.decodeKey(keyString)
		if err != nil {
			l.Trace("Failed to decode key during iteration",
				logger.String(logKeyField, keyString),
				logger.Error(err))
			continue
		}

		if value, ok, getErr := a.GetIfPresent(ctx, key); getErr != nil {
			l.Trace("Failed to retrieve value during iteration",
				logger.String(logKeyField, keyString),
				logger.Error(getErr))
			continue
		} else if ok {
			if !yield(key, value) {
				return false
			}
		}
	}
	return true
}

// Keys returns an iterator over all keys in the cache namespace.
//
// Returns iter.Seq[K] which yields each key found in the namespace.
func (a *ValkeyAdapter[K, V]) Keys() iter.Seq[K] {
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
func (a *ValkeyAdapter[K, V]) Values() iter.Seq[V] {
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
// Returns cache.Entry[K, V] which contains the entry metadata and
// value.
// Returns bool which indicates whether the key was found.
// Returns error when the operation fails.
func (a *ValkeyAdapter[K, V]) GetEntry(ctx context.Context, key K) (cache.Entry[K, V], bool, error) {
	return a.ProbeEntry(ctx, key)
}

// ProbeEntry retrieves entry metadata without affecting access patterns
// or TTL.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to probe.
//
// Returns cache.Entry[K, V] which contains the entry metadata and
// value.
// Returns bool which indicates whether the key was found.
// Returns error when the operation fails.
func (a *ValkeyAdapter[K, V]) ProbeEntry(ctx context.Context, key K) (cache.Entry[K, V], bool, error) {
	if err := ctx.Err(); err != nil {
		return cache.Entry[K, V]{}, false, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("valkey ProbeEntry exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return cache.Entry[K, V]{}, false, fmt.Errorf(errFmtEncodeKey, err)
	}

	results := a.client.DoMulti(timeoutCtx,
		a.client.B().Get().Key(keyString).Build(),
		a.client.B().Ttl().Key(keyString).Build(),
	)

	valBytes, err := results[0].AsBytes()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return cache.Entry[K, V]{}, false, nil
		}
		return cache.Entry[K, V]{}, false, fmt.Errorf("valkey get failed for key %s: %w", keyString, err)
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
		return cache.Entry[K, V]{}, false, fmt.Errorf("type assertion failed after unmarshal for key %s", keyString)
	}

	ttlSeconds, _ := results[1].AsInt64()

	var expiresAtNano int64
	if ttlSeconds > 0 {
		expiresAtNano = time.Now().Add(time.Duration(ttlSeconds) * time.Second).UnixNano()
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

// isTransactionConflict reports whether the error indicates a WATCH/MULTI/EXEC
// transaction conflict (the watched key was modified by another client).
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true if the error indicates a transaction conflict.
func isTransactionConflict(err error) bool {
	if err == nil {
		return false
	}
	errMessage := err.Error()
	return strings.Contains(errMessage, "EXECABORT") ||
		strings.Contains(errMessage, "nil") ||
		valkey.IsValkeyNil(err)
}
