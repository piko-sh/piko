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
	"strings"
	"time"

	"github.com/valkey-io/valkey-go"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
)

const (
	// unwatchTimeout bounds the UNWATCH sent after a transaction that ended without EXEC.
	unwatchTimeout = 5 * time.Second
)

var (
	// errUnwatchTimedOut is the cause given when clearing a WATCH takes longer than
	// unwatchTimeout.
	errUnwatchTimedOut = errors.New("valkey cluster UNWATCH exceeded its timeout")
)

// execTracker wraps a dedicated connection and records whether a transaction called
// DoMulti for its MULTI/EXEC block.
type execTracker struct {
	valkey.DedicatedClient

	// execSent records a DoMulti call, which may fail before EXEC reaches the server.
	execSent bool
}

// DoMulti forwards a transaction's MULTI/EXEC block and records the attempt.
//
// Takes multi (...valkey.Completed) which are the commands of the block.
//
// Returns []valkey.ValkeyResult which holds one result per command.
func (t *execTracker) DoMulti(ctx context.Context, multi ...valkey.Completed) []valkey.ValkeyResult {
	t.execSent = true
	return t.DedicatedClient.DoMulti(ctx, multi...)
}

// runWatchedTransaction runs one transaction on a dedicated connection.
//
// If the attempt ends without calling DoMulti, it clears WATCH so the pooled connection
// cannot abort a later transaction. A failed DoMulti call does not trigger this cleanup.
//
// Takes transaction (func(valkey.DedicatedClient) error) which runs the attempt.
//
// Returns error which is the attempt's error.
func (a *ValkeyClusterAdapter[K, V]) runWatchedTransaction(ctx context.Context, transaction func(c valkey.DedicatedClient) error) error {
	return a.client.Dedicated(func(c valkey.DedicatedClient) error {
		tracker := &execTracker{DedicatedClient: c, execSent: false}
		err := transaction(tracker)
		if !tracker.execSent {
			releaseWatch(ctx, c)
		}
		return err
	})
}

// fetchValueInDedicated fetches and decodes a value inside a dedicated client context.
//
// Takes c (valkey.DedicatedClient) which is the dedicated client for the watch operation.
// Takes keyString (string) which is the key to fetch from Valkey.
//
// Returns V which is the decoded value if found.
// Returns bool which indicates whether the key exists.
// Returns error when the Valkey get operation or value decoding fails.
func (a *ValkeyClusterAdapter[K, V]) fetchValueInDedicated(ctx context.Context, c valkey.DedicatedClient, keyString string) (V, bool, error) {
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

// handleComputeRetryResult processes the result of a compute transaction attempt.
//
// Takes key (K) which is the cache key being computed.
// Takes keyString (string) which is the string form of the key for logging.
// Takes err (error) which is the error from the transaction attempt.
// Takes attempt (int) which is the current retry attempt number.
//
// Returns bool which indicates whether to retry the transaction.
// Returns V which is the final value if computation succeeded.
// Returns bool which indicates whether a valid value was retrieved.
// Returns error when the operation fails.
func (a *ValkeyClusterAdapter[K, V]) handleComputeRetryResult(ctx context.Context, key K, keyString string, err error, attempt int) (bool, V, bool, error) {
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

	if isTransactionConflict(err) {
		l.Trace("Compute transaction failed, retrying",
			logger.String(logKeyField, keyString),
			logger.Int("attempt", attempt+1))
		return true, zero, false, nil
	}

	return false, zero, false, fmt.Errorf("compute transaction error for key %s: %w", keyString, err)
}

// Compute atomically updates a cache entry using a compute function with optimistic
// locking. Computes and writes the new value in one round trip.
//
// A cancelled or expired context returns its error before work begins.
//
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(...)) which receives the old value and found flag,
// returning the new value and action to perform.
//
// Returns V which is the resulting value after the compute operation.
// Returns bool which indicates whether the operation succeeded.
// Returns error when the operation fails.
func (a *ValkeyClusterAdapter[K, V]) Compute(ctx context.Context, key K, computeFunction func(oldValue V, found bool) (newValue V, action cache.ComputeAction)) (V, bool, error) {
	if err := ctx.Err(); err != nil {
		return *new(V), false, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("valkey cluster Compute exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	return a.retryCompute(ctx, key, keyString, func(c valkey.DedicatedClient) error {
		return a.computeTransaction(timeoutCtx, c, keyString, computeFunction)
	})
}

// retryCompute runs transaction on a dedicated client until it commits, retrying when a
// concurrent write aborts it, up to the configured number of attempts.
//
// Takes key (K) which identifies the entry.
// Takes keyString (string) which is the encoded cache key.
// Takes transaction (func(valkey.DedicatedClient) error) which runs one attempt.
//
// Returns V which is the value after the compute.
// Returns bool which reports whether a value is present.
// Returns error when an attempt fails for a reason other than a conflict, or every
// attempt conflicts.
func (a *ValkeyClusterAdapter[K, V]) retryCompute(
	ctx context.Context,
	key K,
	keyString string,
	transaction func(c valkey.DedicatedClient) error,
) (V, bool, error) {
	for attempt := range a.maxComputeRetries {
		err := a.runWatchedTransaction(ctx, transaction)

		shouldRetry, value, found, retryErr := a.handleComputeRetryResult(ctx, key, keyString, err, attempt)
		if retryErr != nil {
			return *new(V), false, retryErr
		}
		if !shouldRetry {
			return value, found, nil
		}
	}

	return *new(V), false, fmt.Errorf("compute max retries exceeded for key %s", keyString)
}

// computeTransaction runs one WATCH/GET/MULTI/EXEC attempt of Compute on a dedicated
// client.
//
// Takes c (valkey.DedicatedClient) which is the dedicated client for the transaction.
// Takes keyString (string) which is the encoded cache key.
// Takes computeFunction (func(V, bool) (V, cache.ComputeAction)) which computes the new
// value and action.
//
// Returns error when any step fails, including a transaction the server aborted because
// the key changed.
func (a *ValkeyClusterAdapter[K, V]) computeTransaction(
	ctx context.Context,
	c valkey.DedicatedClient,
	keyString string,
	computeFunction func(oldValue V, found bool) (newValue V, action cache.ComputeAction),
) error {
	if err := c.Do(ctx, c.B().Watch().Key(keyString).Build()).Error(); err != nil {
		return fmt.Errorf(errFmtWatchFailed, err)
	}

	oldValue, found, err := a.fetchValueInDedicated(ctx, c, keyString)
	if err != nil {
		return err
	}

	newValue, action := computeFunction(oldValue, found)

	cmds, err := a.buildComputeCommands(c, keyString, newValue, action, found, a.writeExpiry(0, a.ttl))
	if err != nil {
		return err
	}
	results := c.DoMulti(ctx, cmds...)
	return results[len(results)-1].Error()
}

// buildComputeCommands builds the MULTI/action/EXEC commands for a compute transaction.
//
// Takes c (valkey.DedicatedClient) which provides the client for building commands.
// Takes keyString (string) which is the cache key to operate on.
// Takes newValue (V) which is the value to set when action is Set.
// Takes action (cache.ComputeAction) which specifies the operation to perform.
// Takes found (bool) which indicates whether the key exists in the cache.
// Takes expiration (time.Duration) which is the writeExpiry result for Set operations.
//
// Returns valkey.Commands which contains the transaction commands to execute.
// Returns error when the new value cannot be encoded. Nothing is sent then.
func (a *ValkeyClusterAdapter[K, V]) buildComputeCommands(
	c valkey.DedicatedClient,
	keyString string,
	newValue V,
	action cache.ComputeAction,
	found bool,
	expiration time.Duration,
) (valkey.Commands, error) {
	cmds := make(valkey.Commands, 0, maxTransactionCommands)
	cmds = append(cmds, c.B().Multi().Build())

	switch action {
	case cache.ComputeActionSet:
		valBytes, err := a.encodeValue(newValue)
		if err != nil {
			return nil, fmt.Errorf("encoding computed value for key %s: %w", keyString, err)
		}
		cmds = append(cmds, buildSet(c.B(), keyString, string(valBytes), expiration))
	case cache.ComputeActionDelete:
		if found {
			cmds = append(cmds, c.B().Del().Key(keyString).Build())
		}
	case cache.ComputeActionNoop:
	}

	cmds = append(cmds, c.B().Exec().Build())
	return cmds, nil
}

// computeIfAbsentTransaction executes the WATCH/EXISTS/MULTI/SET/EXEC transaction for
// ComputeIfAbsent within a dedicated client.
//
// Takes c (valkey.DedicatedClient) which is the dedicated client for the transaction.
// Takes keyString (string) which is the encoded cache key.
// Takes computeFunction (func() V) which computes the value if the key is absent.
//
// Returns bool which indicates whether the compute function was invoked.
// Returns error when any step of the transaction fails.
func (a *ValkeyClusterAdapter[K, V]) computeIfAbsentTransaction(ctx context.Context, c valkey.DedicatedClient, keyString string, computeFunction func() V) (bool, error) {
	if err := c.Do(ctx, c.B().Watch().Key(keyString).Build()).Error(); err != nil {
		return false, fmt.Errorf(errFmtWatchFailed, err)
	}

	exists, err := c.Do(ctx, c.B().Exists().Key(keyString).Build()).AsInt64()
	if err != nil {
		return false, fmt.Errorf("EXISTS check failed: %w", err)
	}
	if exists > 0 {
		return false, nil
	}

	newValue := computeFunction()
	valBytes, err := a.encodeValue(newValue)
	if err != nil {
		return true, err
	}

	cmds := valkey.Commands{
		c.B().Multi().Build(),
		buildSet(c.B(), keyString, string(valBytes), a.writeExpiry(0, a.ttl)),
		c.B().Exec().Build(),
	}
	results := c.DoMulti(ctx, cmds...)
	return true, results[len(results)-1].Error()
}

// ComputeIfAbsent atomically computes and stores a value only if the key is not present.
//
// A cancelled or expired context returns its error before work begins.
//
// Takes key (K) which is the cache key to look up or compute.
// Takes computeFunction (func() V) which computes the value if the key is absent.
//
// Returns V which is the existing or newly computed value.
// Returns bool which is true if the value was computed, false otherwise.
// Returns error when the operation fails.
func (a *ValkeyClusterAdapter[K, V]) ComputeIfAbsent(ctx context.Context, key K, computeFunction func() V) (V, bool, error) {
	if err := ctx.Err(); err != nil {
		return *new(V), false, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("valkey cluster computeIfAbsent exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	for range a.maxComputeRetries {
		didCompute := false
		err := a.runWatchedTransaction(timeoutCtx, func(c valkey.DedicatedClient) error {
			computed, txErr := a.computeIfAbsentTransaction(timeoutCtx, c, keyString, computeFunction)
			didCompute = computed
			return txErr
		})

		value, computed, shouldRetry, retryErr := a.handleComputeIfAbsentResult(ctx, key, keyString, err, didCompute)
		if retryErr != nil {
			return *new(V), false, retryErr
		}
		if !shouldRetry {
			return value, computed, nil
		}
	}

	return *new(V), false, fmt.Errorf("ComputeIfAbsent max retries exceeded for key %s", keyString)
}

// handleComputeIfAbsentResult processes the result of a ComputeIfAbsent transaction
// attempt.
//
// Takes key (K) which is the cache key being computed.
// Takes keyString (string) which is the string form of the key for logging.
// Takes err (error) which is the error from the transaction attempt.
// Takes didCompute (bool) which indicates whether computation occurred.
//
// Returns value (V) which is the final cached value if present.
// Returns computed (bool) which indicates whether the value was computed.
// Returns shouldRetry (bool) which indicates whether the caller should retry the
// transaction due to a conflict.
// Returns error when the operation fails.
func (a *ValkeyClusterAdapter[K, V]) handleComputeIfAbsentResult(ctx context.Context, key K, keyString string, err error, didCompute bool) (value V, computed bool, shouldRetry bool, retryErr error) {
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

	if isTransactionConflict(err) {
		l.Trace("ComputeIfAbsent transaction failed, retrying",
			logger.String(logKeyField, keyString))
		return zero, false, true, nil
	}

	return zero, false, false, fmt.Errorf("ComputeIfAbsent transaction error for key %s: %w", keyString, err)
}

// ComputeIfPresent atomically updates a value only if the key exists in the cache.
//
// When the context is already cancelled or has exceeded its deadline, returns the
// context's error without performing any work.
//
// Takes key (K) which identifies the entry to update.
// Takes computeFunction (func(...)) which computes the new value from the old value.
//
// Returns V which is the computed value, or the zero value if the key was not found or
// the operation failed.
// Returns bool which indicates whether the key existed and was updated.
// Returns error when the operation fails.
//
//nolint:dupl // similar ops, different semantics
func (a *ValkeyClusterAdapter[K, V]) ComputeIfPresent(ctx context.Context, key K, computeFunction func(oldValue V) (newValue V, action cache.ComputeAction)) (V, bool, error) {
	if err := ctx.Err(); err != nil {
		return *new(V), false, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("valkey cluster computeIfPresent exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	for attempt := range a.maxComputeRetries {
		err := a.runWatchedTransaction(timeoutCtx, func(c valkey.DedicatedClient) error {
			return a.computeIfPresentTxn(timeoutCtx, c, keyString, computeFunction)
		})

		value, found, shouldRetry, retryErr := a.handleComputeIfPresentResult(ctx, key, keyString, err, attempt)
		if retryErr != nil {
			return *new(V), false, retryErr
		}
		if !shouldRetry {
			return value, found, nil
		}
	}

	return *new(V), false, fmt.Errorf("ComputeIfPresent max retries exceeded for key %s", keyString)
}

// computeIfPresentTxn executes the WATCH/compute/MULTI/EXEC sequence for ComputeIfPresent
// within a dedicated connection.
//
// Takes c (valkey.DedicatedClient) which is the dedicated client for the watch
// transaction.
// Takes keyString (string) which is the encoded cache key.
// Takes computeFunction (func(...)) which computes the new value from the existing value.
//
// Returns error when any step of the transaction fails.
func (a *ValkeyClusterAdapter[K, V]) computeIfPresentTxn(
	ctx context.Context, c valkey.DedicatedClient, keyString string,
	computeFunction func(oldValue V) (newValue V, action cache.ComputeAction),
) error {
	if err := c.Do(ctx, c.B().Watch().Key(keyString).Build()).Error(); err != nil {
		return fmt.Errorf(errFmtWatchFailed, err)
	}

	oldValue, found, getErr := a.fetchValueInDedicated(ctx, c, keyString)
	if getErr != nil {
		return getErr
	}
	if !found {
		return nil
	}

	newValue, action := computeFunction(oldValue)

	cmds, err := a.buildComputePresentCommands(c, keyString, newValue, action, a.writeExpiry(0, a.ttl))
	if err != nil {
		return err
	}
	results := c.DoMulti(ctx, cmds...)
	return results[len(results)-1].Error()
}

// buildComputePresentCommands builds the MULTI/action/EXEC commands for ComputeIfPresent.
//
// Takes c (valkey.DedicatedClient) which provides the client for building commands.
// Takes keyString (string) which is the cache key to operate on.
// Takes newValue (V) which is the value to set if action is ComputeActionSet.
// Takes action (cache.ComputeAction) which specifies the operation to perform.
// Takes expiration (time.Duration) which is the writeExpiry result for set operations.
//
// Returns valkey.Commands which contains the transaction commands to execute.
// Returns error when the new value cannot be encoded. Nothing is sent then.
func (a *ValkeyClusterAdapter[K, V]) buildComputePresentCommands(
	c valkey.DedicatedClient,
	keyString string,
	newValue V,
	action cache.ComputeAction,
	expiration time.Duration,
) (valkey.Commands, error) {
	cmds := make(valkey.Commands, 0, maxTransactionCommands)
	cmds = append(cmds, c.B().Multi().Build())

	switch action {
	case cache.ComputeActionSet:
		valBytes, err := a.encodeValue(newValue)
		if err != nil {
			return nil, fmt.Errorf("encoding computed value for key %s: %w", keyString, err)
		}
		cmds = append(cmds, buildSet(c.B(), keyString, string(valBytes), expiration))
	case cache.ComputeActionDelete:
		cmds = append(cmds, c.B().Del().Key(keyString).Build())
	case cache.ComputeActionNoop:
	}

	cmds = append(cmds, c.B().Exec().Build())
	return cmds, nil
}

// handleComputeIfPresentResult processes the result of a ComputeIfPresent transaction
// attempt.
//
// Takes key (K) which is the cache key being computed.
// Takes keyString (string) which is the string form of the key for logging.
// Takes err (error) which is the error from the transaction attempt.
// Takes attempt (int) which is the current retry attempt number.
//
// Returns value (V) which is the computed value if present and successful.
// Returns found (bool) which indicates whether the key was present.
// Returns shouldRetry (bool) which indicates whether the operation should be retried due
// to a transaction conflict.
// Returns error when the operation fails.
func (a *ValkeyClusterAdapter[K, V]) handleComputeIfPresentResult(ctx context.Context, key K, keyString string, err error, attempt int) (value V, found bool, shouldRetry bool, retryErr error) {
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

	if isTransactionConflict(err) {
		l.Trace("ComputeIfPresent transaction failed, retrying",
			logger.String(logKeyField, keyString),
			logger.Int("attempt", attempt+1))
		return zero, false, true, nil
	}

	return zero, false, false, fmt.Errorf("ComputeIfPresent transaction error for key %s: %w", keyString, err)
}

// ComputeWithTTL atomically computes a new value with per-call TTL control.
//
// When the context is already cancelled or has exceeded its deadline, returns the
// context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(...)) which receives the old value and found flag,
// returning a ComputeResult containing the new value, action, and optional TTL.
//
// Returns V which is the resulting value after the operation.
// Returns bool which indicates whether a value is now present.
// Returns error when the operation fails.
//
//nolint:dupl // similar ops, different semantics
func (a *ValkeyClusterAdapter[K, V]) ComputeWithTTL(ctx context.Context, key K, computeFunction func(oldValue V, found bool) cache.ComputeResult[V]) (V, bool, error) {
	if err := ctx.Err(); err != nil {
		return *new(V), false, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("valkey cluster computeWithTTL exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	for attempt := range a.maxComputeRetries {
		err := a.runWatchedTransaction(timeoutCtx, func(c valkey.DedicatedClient) error {
			return a.computeWithTTLTxn(timeoutCtx, c, keyString, computeFunction)
		})

		shouldRetry, value, found, retryErr := a.handleComputeRetryResult(ctx, key, keyString, err, attempt)
		if retryErr != nil {
			return *new(V), false, retryErr
		}
		if !shouldRetry {
			return value, found, nil
		}
	}

	return *new(V), false, fmt.Errorf("ComputeWithTTL max retries exceeded for key %s", keyString)
}

// computeWithTTLTxn executes the WATCH/compute/MULTI/EXEC sequence for ComputeWithTTL
// within a dedicated connection.
//
// Takes c (valkey.DedicatedClient) which is the dedicated client for the watch
// transaction.
// Takes keyString (string) which is the encoded cache key.
// Takes computeFunction (func(...)) which computes the new value, action, and optional
// TTL from the existing value.
//
// Returns error when any step of the transaction fails.
func (a *ValkeyClusterAdapter[K, V]) computeWithTTLTxn(
	ctx context.Context, c valkey.DedicatedClient, keyString string,
	computeFunction func(oldValue V, found bool) cache.ComputeResult[V],
) error {
	if err := c.Do(ctx, c.B().Watch().Key(keyString).Build()).Error(); err != nil {
		return fmt.Errorf(errFmtWatchFailed, err)
	}

	oldValue, found, getErr := a.fetchValueInDedicated(ctx, c, keyString)
	if getErr != nil {
		return getErr
	}

	result := computeFunction(oldValue, found)

	cmds, err := a.buildComputeCommands(c, keyString, result.Value, result.Action, found, a.writeExpiry(result.TTL, a.ttl))
	if err != nil {
		return err
	}
	results := c.DoMulti(ctx, cmds...)
	return results[len(results)-1].Error()
}

// isTransactionConflict checks if an error indicates a WATCH/MULTI/EXEC conflict
// (optimistic lock failure). When EXEC runs after a WATCH conflict, Valkey returns a nil
// response which valkey-go surfaces as a ValkeyNil error.
//
// Takes err (error) which is the error to check for transaction conflict.
//
// Returns bool which is true if the error indicates a transaction conflict.
func isTransactionConflict(err error) bool {
	if err == nil {
		return false
	}
	errString := err.Error()
	return strings.Contains(errString, "EXECABORT") ||
		strings.Contains(errString, "nil") ||
		valkey.IsValkeyNil(err)
}

// releaseWatch sends UNWATCH on a dedicated connection. When that fails it closes the
// connection, so the pool never hands out one that may still be watching a key.
//
// The UNWATCH keeps the values of ctx but not its cancellation, because the transaction
// may have ended by running out of time.
//
// Takes c (valkey.DedicatedClient) which is the connection to clear.
func releaseWatch(ctx context.Context, c valkey.DedicatedClient) {
	unwatchCtx, cancel := context.WithTimeoutCause(context.WithoutCancel(ctx), unwatchTimeout, errUnwatchTimedOut)
	defer cancel()
	if err := c.Do(unwatchCtx, c.B().Unwatch().Build()).Error(); err != nil {
		_, l := logger.From(ctx, log)
		l.Warn("Closing a dedicated Valkey connection whose WATCH could not be cleared", logger.Error(err))
		c.Close()
	}
}
