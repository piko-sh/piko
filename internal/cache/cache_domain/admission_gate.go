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

package cache_domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"piko.sh/piko/internal/cache/cache_dto"
)

// admissionGate refuses values heavier than the cache's per-entry ceiling.
type admissionGate[K comparable, V any] struct {
	Cache[K, V]

	// weigher measures a value against the ceiling.
	weigher func(K, V) uint32

	// onRejection is notified of a refused value, so an operator who ignores the returned
	// error still sees it. It may be nil.
	onRejection func(cache_dto.DeletionEvent[K, V])

	// maxEntryWeight is the largest weight a single entry may have.
	maxEntryWeight uint32
}

// newAdmissionGate wraps a cache so values above the ceiling are refused.
//
// Takes inner (Cache[K, V]) which is the cache being decorated.
// Takes options (cache_dto.Options[K, V]) which supply the ceiling, weigher and callback.
//
// Returns Cache[K, V] which is the gate, or inner unchanged when no ceiling is set.
func newAdmissionGate[K comparable, V any](inner Cache[K, V], options cache_dto.Options[K, V]) Cache[K, V] {
	if options.MaxEntryWeight == 0 || options.Weigher == nil {
		return inner
	}

	return &admissionGate[K, V]{
		Cache:          inner,
		weigher:        options.Weigher,
		onRejection:    options.OnDeletion,
		maxEntryWeight: options.MaxEntryWeight,
	}
}

// Checkpoint forwards to the decorated cache so a gated cache still answers the optional
// checkpoint probe.
//
// Returns error when the decorated cache supports checkpointing and the checkpoint fails.
func (g *admissionGate[K, V]) Checkpoint(ctx context.Context) error {
	checkpointer, ok := g.Cache.(interface{ Checkpoint(context.Context) error })
	if !ok {
		return nil
	}

	return checkpointer.Checkpoint(ctx)
}

// BeginTransaction forwards to the decorated cache so a gated cache still satisfies
// Transactional, which embedding alone would hide.
//
// Returns TransactionCache[K, V] which is the transactional view of this cache.
func (g *admissionGate[K, V]) BeginTransaction(ctx context.Context) TransactionCache[K, V] {
	if transactional, ok := g.Cache.(Transactional[K, V]); ok {
		return transactional.BeginTransaction(ctx)
	}

	return newTransactionJournal[K, V](g)
}

// Set stores a value unless it exceeds the ceiling.
//
// Takes key (K) which identifies the entry.
// Takes value (V) which is the data to store.
// Takes tags (...string) which are optional labels for grouping entries.
//
// Returns error which is ErrEntryTooLarge when the value is refused.
func (g *admissionGate[K, V]) Set(ctx context.Context, key K, value V, tags ...string) error {
	if err := g.admit(key, value); err != nil {
		return err
	}

	return g.Cache.Set(ctx, key, value, tags...)
}

// SetWithTTL stores a value with an expiry unless it exceeds the ceiling.
//
// Takes key (K) which identifies the entry.
// Takes value (V) which is the data to store.
// Takes ttl (time.Duration) which is how long the entry remains valid.
// Takes tags (...string) which are optional labels for grouping entries.
//
// Returns error which is ErrEntryTooLarge when the value is refused.
func (g *admissionGate[K, V]) SetWithTTL(ctx context.Context, key K, value V, ttl time.Duration, tags ...string) error {
	if err := g.admit(key, value); err != nil {
		return err
	}

	return g.Cache.SetWithTTL(ctx, key, value, ttl, tags...)
}

// BulkSet stores every admissible item and reports the rest.
//
// Takes items (map[K]V) which contains the key-value pairs to store.
// Takes tags (...string) which are optional labels for grouping entries.
//
// Returns error which joins one ErrEntryTooLarge per refused key, or the write's own
// failure.
func (g *admissionGate[K, V]) BulkSet(ctx context.Context, items map[K]V, tags ...string) error {
	admissible := make(map[K]V, len(items))
	var rejections []error

	for key, value := range items {
		if err := g.admit(key, value); err != nil {
			rejections = append(rejections, err)

			continue
		}
		admissible[key] = value
	}

	if len(admissible) > 0 {
		if err := g.Cache.BulkSet(ctx, admissible, tags...); err != nil {
			rejections = append(rejections, err)
		}
	}

	return errors.Join(rejections...)
}

// Compute stores a computed value unless it exceeds the ceiling.
//
// Takes key (K) which identifies the entry.
// Takes computeFunction (func) which produces the new value.
//
// Returns V which is the computed value.
// Returns bool which reports whether a value is now present.
// Returns error which is ErrEntryTooLarge when the computed value is refused.
func (g *admissionGate[K, V]) Compute(
	ctx context.Context,
	key K,
	computeFunction func(oldValue V, found bool) (newValue V, action cache_dto.ComputeAction),
) (V, bool, error) {
	return g.Cache.Compute(ctx, key, g.gateCompute(key, computeFunction))
}

// ComputeIfAbsent stores a computed value unless it exceeds the ceiling.
//
// Takes key (K) which identifies the entry.
// Takes computeFunction (func) which produces the new value.
//
// Returns V which is the computed value.
// Returns bool which reports whether a value is now present.
// Returns error which is ErrEntryTooLarge when the computed value is refused.
func (g *admissionGate[K, V]) ComputeIfAbsent(
	ctx context.Context,
	key K,
	computeFunction func() V,
) (V, bool, error) {
	var refused bool

	value, found, err := g.Cache.ComputeIfAbsent(ctx, key, func() V {
		computed := computeFunction()
		if g.admit(key, computed) != nil {
			refused = true
		}

		return computed
	})
	if refused {
		return g.refuse(ctx, key, value)
	}

	return value, found, err
}

// ComputeIfPresent stores a computed value unless it exceeds the ceiling.
//
// Takes key (K) which identifies the entry.
// Takes computeFunction (func) which produces the new value.
//
// Returns V which is the computed value.
// Returns bool which reports whether a value is now present.
// Returns error which is ErrEntryTooLarge when the computed value is refused.
func (g *admissionGate[K, V]) ComputeIfPresent(
	ctx context.Context,
	key K,
	computeFunction func(oldValue V) (newValue V, action cache_dto.ComputeAction),
) (V, bool, error) {
	return g.Cache.ComputeIfPresent(ctx, key, func(oldValue V) (V, cache_dto.ComputeAction) {
		newValue, action := computeFunction(oldValue)
		if action == cache_dto.ComputeActionSet && g.admit(key, newValue) != nil {
			return newValue, cache_dto.ComputeActionNoop
		}

		return newValue, action
	})
}

// ComputeWithTTL stores a computed value with an expiry unless it exceeds the ceiling.
//
// Takes key (K) which identifies the entry.
// Takes computeFunction (func) which produces the new value.
//
// Returns V which is the computed value.
// Returns bool which reports whether a value is now present.
// Returns error which is ErrEntryTooLarge when the computed value is refused.
func (g *admissionGate[K, V]) ComputeWithTTL(
	ctx context.Context,
	key K,
	computeFunction func(oldValue V, found bool) cache_dto.ComputeResult[V],
) (V, bool, error) {
	return g.Cache.ComputeWithTTL(ctx, key, func(oldValue V, found bool) cache_dto.ComputeResult[V] {
		result := computeFunction(oldValue, found)
		if result.Action == cache_dto.ComputeActionSet && g.admit(key, result.Value) != nil {
			result.Action = cache_dto.ComputeActionNoop
		}

		return result
	})
}

// gateCompute wraps a compute function so a value above the ceiling is turned into a
// no-op rather than being stored.
//
// Takes key (K) which identifies the entry.
// Takes computeFunction (func) which produces the new value.
//
// Returns func(oldValue V, found bool) (V, cache_dto.ComputeAction) which is the wrapped
// compute function.
func (g *admissionGate[K, V]) gateCompute(
	key K,
	computeFunction func(oldValue V, found bool) (newValue V, action cache_dto.ComputeAction),
) func(oldValue V, found bool) (V, cache_dto.ComputeAction) {
	return func(oldValue V, found bool) (V, cache_dto.ComputeAction) {
		newValue, action := computeFunction(oldValue, found)
		if action == cache_dto.ComputeActionSet && g.admit(key, newValue) != nil {
			return newValue, cache_dto.ComputeActionNoop
		}

		return newValue, action
	}
}

// refuse reports an over-ceiling computed value to the caller.
//
// Takes key (K) which identifies the entry.
// Takes value (V) which is the value that was refused.
//
// Returns V which is the value the caller computed.
// Returns bool which is false because nothing was stored.
// Returns error which is ErrEntryTooLarge.
func (g *admissionGate[K, V]) refuse(_ context.Context, key K, value V) (V, bool, error) {
	return value, false, fmt.Errorf("%w: key weighs %d, ceiling is %d",
		ErrEntryTooLarge, g.weigher(key, value), g.maxEntryWeight)
}

// admit reports whether a value fits under the ceiling, notifying the deletion callback
// when it does not.
//
// Takes key (K) which identifies the entry.
// Takes value (V) which is the data being written.
//
// Returns error which is ErrEntryTooLarge when the value is refused.
func (g *admissionGate[K, V]) admit(key K, value V) error {
	weight := g.weigher(key, value)
	if weight <= g.maxEntryWeight {
		return nil
	}

	if g.onRejection != nil {
		g.onRejection(cache_dto.DeletionEvent[K, V]{Key: key, Value: value, Cause: cache_dto.CauseRejected})
	}

	return fmt.Errorf("%w: key weighs %d, ceiling is %d", ErrEntryTooLarge, weight, g.maxEntryWeight)
}
