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

package cache_provider_redis_cluster

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
)

// fetchValueInWatch fetches and decodes a value inside a Redis watch context.
//
// Takes tx (*redis.Tx) which is the Redis transaction for the watch operation.
// Takes keyString (string) which is the key to fetch from Redis.
//
// Returns V which is the decoded value if found.
// Returns bool which indicates whether the key exists.
// Returns error when the Redis get operation or value decoding fails.
func (a *RedisClusterAdapter[K, V]) fetchValueInWatch(ctx context.Context, tx *redis.Tx, keyString string) (V, bool, error) {
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

// executeComputeAction executes the compute action within a Redis pipeline.
//
// Takes pipe (redis.Pipeliner) which is the pipeline to add commands to.
// Takes keyString (string) which is the cache key to operate on.
// Takes newValue (V) which is the value to set when action is Set.
// Takes action (cache.ComputeAction) which determines the operation to run.
// Takes found (bool) which indicates whether the key exists in cache.
//
// Returns error when value encoding fails for a Set action.
func (a *RedisClusterAdapter[K, V]) executeComputeAction(ctx context.Context, pipe redis.Pipeliner, keyString string, newValue V, action cache.ComputeAction, found bool) error {
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

// executeComputeActionPresent runs the compute action for ComputeIfPresent.
//
// Takes pipe (redis.Pipeliner) which queues the Redis commands.
// Takes keyString (string) which is the cache key to operate on.
// Takes newValue (V) which is the value to set when action is Set.
// Takes action (cache.ComputeAction) which specifies what operation to do.
//
// Returns error when the value cannot be encoded.
func (a *RedisClusterAdapter[K, V]) executeComputeActionPresent(ctx context.Context, pipe redis.Pipeliner, keyString string, newValue V, action cache.ComputeAction) error {
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

// handleComputeResult is the shared implementation for processing the result
// of a compute transaction attempt.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which is the cache key being computed.
// Takes keyString (string) which is the string form of the key for logging.
// Takes opName (string) which is the operation name for log messages.
// Takes err (error) which is the error from the transaction attempt.
// Takes attempt (int) which is the current retry attempt number.
//
// Returns value (V) which is the final value if computation succeeded.
// Returns found (bool) which indicates whether a valid value was retrieved.
// Returns shouldRetry (bool) which indicates whether to retry the transaction.
// Returns retErr (error) which is any non-retryable error encountered.
func (a *RedisClusterAdapter[K, V]) handleComputeResult(ctx context.Context, key K, keyString, opName string, err error, attempt int) (value V, found bool, shouldRetry bool, retErr error) {
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

	if errors.Is(err, redis.TxFailedErr) {
		l.Trace(opName+" transaction failed, retrying",
			logger.String(logKeyField, keyString),
			logger.Int("attempt", attempt+1))
		return zero, false, true, nil
	}

	return zero, false, false, fmt.Errorf("%s transaction error: %w", opName, err)
}

// handleComputeRetryResult processes the result of a Compute transaction
// attempt. Delegates to handleComputeResult with reordered return values.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which is the cache key being computed.
// Takes keyString (string) which is the string form of the key for logging.
// Takes err (error) which is the error from the transaction attempt.
// Takes attempt (int) which is the current retry attempt number.
//
// Returns shouldRetry (bool) which indicates whether to retry the transaction.
// Returns V which is the final value if computation succeeded.
// Returns found (bool) which indicates whether a valid value was retrieved.
// Returns retErr (error) which is any non-retryable error encountered.
func (a *RedisClusterAdapter[K, V]) handleComputeRetryResult(ctx context.Context, key K, keyString string, err error, attempt int) (shouldRetry bool, value V, found bool, retErr error) {
	v, f, s, e := a.handleComputeResult(ctx, key, keyString, "Compute", err, attempt)
	return s, v, f, e
}

// Compute atomically updates a cache entry using a compute function with
// optimistic locking.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(...)) which receives the old value and found flag,
// returning the new value and action to perform.
//
// Returns V which is the resulting value after the compute operation.
// Returns bool which indicates whether the operation succeeded.
// Returns error when the operation fails.
func (a *RedisClusterAdapter[K, V]) Compute(ctx context.Context, key K, computeFunction func(oldValue V, found bool) (newValue V, action cache.ComputeAction)) (V, bool, error) {
	if err := ctx.Err(); err != nil {
		return *new(V), false, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("redis cluster Compute exceeded %s timeout", a.atomicOperationTimeout))
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

		shouldRetry, value, found, retErr := a.handleComputeRetryResult(timeoutCtx, key, keyString, err, attempt)
		if retErr != nil {
			return *new(V), false, retErr
		}
		if !shouldRetry {
			return value, found, nil
		}
	}

	return *new(V), false, fmt.Errorf("compute max retries exceeded for key %q", keyString)
}

// computeIfAbsentWatchFunction executes the WATCH callback for ComputeIfAbsent.
//
// Takes tx (*redis.Tx) which is the Redis transaction for atomic operations.
// Takes keyString (string) which is the cache key to check and potentially set.
// Takes computeFunction (func() V) which computes the value if the key is absent.
// Takes didCompute (*bool) which is set to true if the value was computed.
//
// Returns error when the existence check fails or the transaction cannot
// complete.
func (a *RedisClusterAdapter[K, V]) computeIfAbsentWatchFunction(ctx context.Context, tx *redis.Tx, keyString string, computeFunction func() V, didCompute *bool) error {
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
// Takes key (K) which identifies the cache entry.
// Takes keyString (string) which is the string representation for logging.
// Takes err (error) which is the error from the transaction attempt.
// Takes didCompute (bool) which tracks whether the compute function was called.
//
// Returns value (V) which is the cached value if found.
// Returns computed (bool) which indicates if the value was computed on this
// attempt.
// Returns shouldRetry (bool) which indicates if the transaction should be
// retried.
// Returns retErr (error) which is any non-retryable error encountered.
func (a *RedisClusterAdapter[K, V]) handleComputeIfAbsentResult(ctx context.Context, key K, keyString string, err error, didCompute bool) (value V, computed bool, shouldRetry bool, retErr error) {
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

	if errors.Is(err, redis.TxFailedErr) {
		l.Trace("ComputeIfAbsent transaction failed, retrying",
			logger.String(logKeyField, keyString))
		return zero, false, true, nil
	}

	return zero, false, false, fmt.Errorf("ComputeIfAbsent transaction error: %w", err)
}

// ComputeIfAbsent atomically computes and stores a value only if the key is
// not present.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which is the cache key to look up or compute.
// Takes computeFunction (func() V) which computes the value if the key is absent.
//
// Returns V which is the existing or newly computed value.
// Returns bool which is true if the value was computed, false otherwise.
// Returns error when the operation fails.
func (a *RedisClusterAdapter[K, V]) ComputeIfAbsent(ctx context.Context, key K, computeFunction func() V) (V, bool, error) {
	if err := ctx.Err(); err != nil {
		return *new(V), false, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("redis cluster computeIfAbsent exceeded %s timeout", a.atomicOperationTimeout))
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

		value, computed, shouldRetry, retErr := a.handleComputeIfAbsentResult(timeoutCtx, key, keyString, err, didCompute)
		if retErr != nil {
			return *new(V), false, retErr
		}
		if !shouldRetry {
			return value, computed, nil
		}
	}

	return *new(V), false, fmt.Errorf("ComputeIfAbsent max retries exceeded for key %q", keyString)
}

// computeIfPresentWatchFunction runs the WATCH callback for ComputeIfPresent.
//
// Takes tx (*redis.Tx) which is the transaction for the watch operation.
// Takes keyString (string) which is the cache key to compute.
// Takes computeFunction (func(...)) which computes the new value from the existing
// one.
//
// Returns error when the fetch or transaction fails.
func (a *RedisClusterAdapter[K, V]) computeIfPresentWatchFunction(ctx context.Context, tx *redis.Tx, keyString string, computeFunction func(oldValue V) (V, cache.ComputeAction)) error {
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
// Takes keyString (string) which is the string representation of the key for
// logging.
// Takes err (error) which is the error from the transaction attempt, if any.
// Takes attempt (int) which is the current retry attempt number.
//
// Returns value (V) which is the computed value if the operation succeeded.
// Returns found (bool) which indicates whether the key was present and
// computed.
// Returns shouldRetry (bool) which indicates whether the caller should retry
// the transaction.
// Returns retErr (error) which is any non-retryable error encountered.
func (a *RedisClusterAdapter[K, V]) handleComputeIfPresentResult(ctx context.Context, key K, keyString string, err error, attempt int) (value V, found bool, shouldRetry bool, retErr error) {
	return a.handleComputeResult(ctx, key, keyString, "ComputeIfPresent", err, attempt)
}

// ComputeIfPresent atomically updates a value only if the key exists in the
// cache.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the entry to update.
// Takes computeFunction (func(...)) which computes the new value
// from the old value.
//
// Returns V which is the computed value, or the zero value if the key was not
// found or the operation failed.
// Returns bool which indicates whether the key existed and was updated.
// Returns error when the operation fails.
func (a *RedisClusterAdapter[K, V]) ComputeIfPresent(ctx context.Context, key K, computeFunction func(oldValue V) (newValue V, action cache.ComputeAction)) (V, bool, error) {
	if err := ctx.Err(); err != nil {
		return *new(V), false, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("redis cluster computeIfPresent exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	for attempt := range a.maxComputeRetries {
		err := a.client.Watch(timeoutCtx, func(tx *redis.Tx) error {
			return a.computeIfPresentWatchFunction(timeoutCtx, tx, keyString, computeFunction)
		}, keyString)

		value, found, shouldRetry, retErr := a.handleComputeIfPresentResult(timeoutCtx, key, keyString, err, attempt)
		if retErr != nil {
			return *new(V), false, retErr
		}
		if !shouldRetry {
			return value, found, nil
		}
	}

	return *new(V), false, fmt.Errorf("ComputeIfPresent max retries exceeded for key %q", keyString)
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
func (a *RedisClusterAdapter[K, V]) executeComputeActionWithTTL(
	ctx context.Context,
	pipe redis.Pipeliner,
	keyString string,
	newValue V,
	action cache.ComputeAction,
	found bool,
	ttl time.Duration,
) error {
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
func (a *RedisClusterAdapter[K, V]) ComputeWithTTL(ctx context.Context, key K, computeFunction func(oldValue V, found bool) cache.ComputeResult[V]) (V, bool, error) {
	if err := ctx.Err(); err != nil {
		return *new(V), false, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("redis cluster computeWithTTL exceeded %s timeout", a.atomicOperationTimeout))
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

		shouldRetry, value, found, retErr := a.handleComputeRetryResult(timeoutCtx, key, keyString, err, attempt)
		if retErr != nil {
			return *new(V), false, retErr
		}
		if !shouldRetry {
			return value, found, nil
		}
	}

	return *new(V), false, fmt.Errorf("ComputeWithTTL max retries exceeded for key %q", keyString)
}
