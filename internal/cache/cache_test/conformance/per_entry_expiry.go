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

package conformance

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
)

const (
	// perEntryNamespace names the caches BuildPerEntryCache builds.
	perEntryNamespace = "per-entry-conformance"

	// perEntryCapacity bounds the caches BuildPerEntryCache builds.
	perEntryCapacity = 100_000

	// perEntryKey is the key most per-entry scenarios write.
	perEntryKey = "key"

	// perEntryValue is the value most per-entry scenarios write.
	perEntryValue = "value"

	// mockTTL is the TTL the scenarios use when the harness moves time itself.
	mockTTL = time.Minute

	// realTTL is the TTL the scenarios use when they wait in real time.
	realTTL = 1500 * time.Millisecond

	// realSubSecondTTL is the sub-second TTL used when waiting in real time.
	realSubSecondTTL = 300 * time.Millisecond

	// realExpirySlack is how much longer than a TTL a real-time wait allows for expiry.
	realExpirySlack = 2 * time.Second

	// realPollInterval is how often a real-time wait checks the cache.
	realPollInterval = 25 * time.Millisecond

	// concurrentWriters is how many goroutines race on each key in the concurrency scenario.
	concurrentWriters = 8

	// concurrentKeys is how many keys the concurrency scenario writes.
	concurrentKeys = 256

	// concurrentKeysInFlight is how many keys the concurrency scenario races at once. Every
	// writer on a key still races the others, but the number of operations in flight stays
	// bounded, so a provider's per-operation timeout measures the operation and not the time
	// it spent queued behind thousands of others.
	concurrentKeysInFlight = 16

	// longerTTLFactor scales the TTL for writes that must outlast the scenario's TTL.
	longerTTLFactor = 10

	// windowSteps is how many increments the fixed-window scenario makes per window.
	windowSteps = 4
)

// PerEntryHarness is one cache under test with PerEntryExpiration and the means to move
// its time.
type PerEntryHarness struct {
	// Cache is an empty cache built with PerEntryExpiration.
	Cache cache_domain.ProviderPort[string, string]

	// AdvanceTime moves the cache's time forward; nil means the scenarios wait in real time
	// with short TTLs.
	AdvanceTime func(duration time.Duration)

	// Clock is the cache's clock when it is not the system clock, so a transaction can
	// measure how long it ran; nil uses the system clock.
	Clock cache_dto.Clock
}

// PerEntryConfig configures RunPerEntryExpirySuite.
type PerEntryConfig struct {
	// NewHarness builds a fresh harness for each scenario.
	NewHarness func(t *testing.T) PerEntryHarness
}

// scenarioT is the part of a test a scenario uses, so the suite's own tests can check
// that a scenario fails.
type scenarioT interface {
	require.TestingT

	Helper()

	Context() context.Context
}

// perEntryRun drives one scenario against one harness.
type perEntryRun struct {
	harness PerEntryHarness

	t scenarioT

	ttl time.Duration

	subSecondTTL time.Duration
}

// requireExpires checks that key is present now and gone once duration has passed.
//
// Takes key (string) which identifies the entry.
// Takes duration (time.Duration) which is the TTL the entry should have left.
func (r perEntryRun) requireExpires(key string, duration time.Duration) {
	r.t.Helper()

	require.True(r.t, r.present(key), "%s should be present before its TTL passes", key)
	if r.harness.AdvanceTime != nil {
		r.harness.AdvanceTime(duration / 2)
		require.True(r.t, r.present(key), "%s should be present halfway through its TTL", key)
		r.harness.AdvanceTime(duration - duration/2)
		require.False(r.t, r.present(key), "%s should be gone once its TTL passes", key)
		return
	}
	halfway := duration / 2
	require.True(r.t, pollWhile(halfway, func() bool { return r.present(key) }),
		"%s should be present halfway through its TTL", key)
	require.True(r.t, pollUntil(duration-halfway+realExpirySlack, func() bool { return !r.present(key) }),
		"%s should be gone once its TTL passes", key)
}

// requireLasts checks that key stays present while duration passes.
//
// Takes key (string) which identifies the entry.
// Takes duration (time.Duration) which is how long the entry must last.
func (r perEntryRun) requireLasts(key string, duration time.Duration) {
	r.t.Helper()

	require.True(r.t, r.present(key), "%s should be present", key)
	if r.harness.AdvanceTime != nil {
		r.harness.AdvanceTime(duration)
		require.True(r.t, r.present(key), "%s should still be present after %s", key, duration)
		return
	}
	require.True(r.t, pollWhile(duration, func() bool { return r.present(key) }),
		"%s should still be present after %s", key, duration)
}

// requireNoExpiry checks that key is present and has no expiry.
//
// Takes key (string) which identifies the entry.
func (r perEntryRun) requireNoExpiry(key string) {
	r.t.Helper()

	entry, found, err := r.harness.Cache.ProbeEntry(r.t.Context(), key)
	require.NoError(r.t, err)
	require.True(r.t, found, "%s should be present", key)
	assert.True(r.t, hasNoExpiry(entry.ExpiresAtNano), "%s should have no expiry, got %d", key, entry.ExpiresAtNano)
}

// requireExpiry checks that key is present and has a finite expiry.
//
// Takes key (string) which identifies the entry.
func (r perEntryRun) requireExpiry(key string) {
	r.t.Helper()

	entry, found, err := r.harness.Cache.ProbeEntry(r.t.Context(), key)
	require.NoError(r.t, err)
	require.True(r.t, found, "%s should be present", key)
	assert.False(r.t, hasNoExpiry(entry.ExpiresAtNano), "%s should have an expiry", key)
}

func (r perEntryRun) present(key string) bool {
	r.t.Helper()

	_, found, err := r.harness.Cache.GetIfPresent(r.t.Context(), key)
	require.NoError(r.t, err)
	return found
}

func (r perEntryRun) set(key string) {
	r.t.Helper()
	require.NoError(r.t, r.harness.Cache.Set(r.t.Context(), key, perEntryValue))
}

func (r perEntryRun) setWithTTL(key string, ttl time.Duration) {
	r.t.Helper()
	require.NoError(r.t, r.harness.Cache.SetWithTTL(r.t.Context(), key, perEntryValue, ttl))
}

func (r perEntryRun) compute(key string, action cache_dto.ComputeAction, ttl time.Duration) {
	r.t.Helper()
	_, _, err := r.harness.Cache.ComputeWithTTL(r.t.Context(), key, func(string, bool) cache_dto.ComputeResult[string] {
		return cache_dto.ComputeResult[string]{Value: perEntryValue, Action: action, TTL: ttl}
	})
	require.NoError(r.t, err)
}

func (r perEntryRun) expireAfter(key string, duration time.Duration) {
	r.t.Helper()
	require.NoError(r.t, r.harness.Cache.SetExpiresAfter(r.t.Context(), key, duration))
}

// perEntryScenario is one named scenario of the per-entry suite.
type perEntryScenario struct {
	run func(r perEntryRun)

	name string
}

// transactionWrite is a write a rollback scenario makes inside the transaction.
type transactionWrite struct {
	write func(ctx context.Context, transaction cache_domain.TransactionCache[string, string], ttl time.Duration) error

	name string
}

var (
	// perEntryBlueprintCount numbers the factory blueprints BuildPerEntryCache registers.
	perEntryBlueprintCount atomic.Uint64
)

// RunPerEntryExpirySuite checks that a provider follows the TTL contract of a cache built
// with PerEntryExpiration: writes without a TTL keep the current expiry, a positive TTL
// sets it, cache_dto.NoExpiry removes it, SetExpiresAfter with zero or less does nothing,
// TTLs survive concurrent writes, and a rolled-back transaction restores the expiry each
// entry had.
//
// Takes t (*testing.T) which is the test context.
// Takes config (PerEntryConfig) which builds the caches under test.
func RunPerEntryExpirySuite(t *testing.T, config PerEntryConfig) {
	t.Helper()

	scenarios := slices.Concat(perEntryWriteScenarios(), perEntryExpiresAfterScenarios(), perEntryBehaviourScenarios())
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			scenario.run(newPerEntryRun(t, config.NewHarness(t)))
		})
	}

	for _, write := range rollbackWrites() {
		t.Run("Rollback_"+write.name+"_RestoresNoExpiry", func(t *testing.T) {
			t.Parallel()
			testRollbackRestoresNoExpiry(newPerEntryRun(t, config.NewHarness(t)), write)
		})
	}

	t.Run("Rollback_RestoresRemainingTTL", func(t *testing.T) {
		t.Parallel()
		testRollbackRestoresRemainingTTL(newPerEntryRun(t, config.NewHarness(t)))
	})
}

// BuildPerEntryCache builds a cache with CacheBuilder.PerEntryExpiration, creating the
// provider through factory the way a domain package builds a typed cache from a factory
// blueprint, and closes it when the test ends.
//
// Takes t (*testing.T) which owns the cache cleanup.
// Takes factory (func(cache_dto.Options[string, V]) (cache_domain.ProviderPort[string,
// V], error)) which creates the provider from the builder's options.
// Takes configure (...func(*cache_domain.CacheBuilder[string, V])) which adjusts the
// builder, for example to set a clock or persistence.
//
// Returns cache_domain.ProviderPort[string, V] which is the cache the builder built.
func BuildPerEntryCache[V any](
	t *testing.T,
	factory func(options cache_dto.Options[string, V]) (cache_domain.ProviderPort[string, V], error),
	configure ...func(builder *cache_domain.CacheBuilder[string, V]),
) cache_domain.ProviderPort[string, V] {
	t.Helper()

	name := fmt.Sprintf("conformance-per-entry-%d", perEntryBlueprintCount.Add(1))
	cache_domain.RegisterProviderFactory(name, func(_ cache_domain.Service, _ string, options any) (any, error) {
		typed, ok := options.(cache_dto.Options[string, V])
		if !ok {
			return nil, fmt.Errorf("blueprint %s received options of type %T", name, options)
		}
		return factory(typed)
	})
	defer cache_domain.UnregisterProviderFactory(name)

	builder := cache_domain.NewCacheBuilder[string, V](nil).
		FactoryBlueprint(name).
		Namespace(perEntryNamespace).
		MaximumEntries(perEntryCapacity).
		PerEntryExpiration()
	for _, apply := range configure {
		apply(builder)
	}

	built, err := builder.Build(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, built.Close(context.Background())) })

	return built
}

// perEntryWriteScenarios returns the scenarios for Set, SetWithTTL and ComputeWithTTL.
//
// Returns []perEntryScenario which lists the scenarios.
func perEntryWriteScenarios() []perEntryScenario {
	return []perEntryScenario{
		{name: "Set_CreatesEntryWithoutExpiry", run: func(r perEntryRun) {
			r.set(perEntryKey)
			r.requireNoExpiry(perEntryKey)
			r.requireLasts(perEntryKey, 2*r.ttl)
		}},
		{name: "SetWithTTL_SetsExpiry", run: func(r perEntryRun) {
			r.setWithTTL(perEntryKey, r.ttl)
			r.requireExpiry(perEntryKey)
			r.requireExpires(perEntryKey, r.ttl)
		}},
		{name: "Set_KeepsExpiry", run: func(r perEntryRun) {
			r.setWithTTL(perEntryKey, r.ttl)
			r.set(perEntryKey)
			r.requireExpiry(perEntryKey)
			r.requireExpires(perEntryKey, r.ttl)
		}},
		{name: "SetWithTTLZero_KeepsExpiry", run: func(r perEntryRun) {
			r.setWithTTL(perEntryKey, r.ttl)
			r.setWithTTL(perEntryKey, 0)
			r.requireExpires(perEntryKey, r.ttl)
		}},
		{name: "ComputeWithTTLZero_KeepsExpiry", run: func(r perEntryRun) {
			r.setWithTTL(perEntryKey, r.ttl)
			r.compute(perEntryKey, cache_dto.ComputeActionSet, 0)
			r.requireExpires(perEntryKey, r.ttl)
		}},
		{name: "ComputeWithTTL_ReplacesExpiry", run: func(r perEntryRun) {
			r.setWithTTL(perEntryKey, longerTTLFactor*r.ttl)
			r.compute(perEntryKey, cache_dto.ComputeActionSet, r.ttl)
			r.requireExpires(perEntryKey, r.ttl)
		}},
		{name: "SetWithTTLNoExpiry_RemovesExpiry", run: func(r perEntryRun) {
			r.setWithTTL(perEntryKey, r.ttl)
			r.setWithTTL(perEntryKey, cache_dto.NoExpiry)
			r.requireNoExpiry(perEntryKey)
			r.requireLasts(perEntryKey, 2*r.ttl)
		}},
		{name: "ComputeWithTTLNoExpiry_RemovesExpiry", run: func(r perEntryRun) {
			r.setWithTTL(perEntryKey, r.ttl)
			r.compute(perEntryKey, cache_dto.ComputeActionSet, cache_dto.NoExpiry)
			r.requireNoExpiry(perEntryKey)
			r.requireLasts(perEntryKey, 2*r.ttl)
		}},
		{name: "SubSecondTTL_Expires", run: func(r perEntryRun) {
			r.setWithTTL(perEntryKey, r.subSecondTTL)
			r.requireExpires(perEntryKey, r.subSecondTTL)
		}},
	}
}

// perEntryExpiresAfterScenarios returns the scenarios for SetExpiresAfter.
//
// Returns []perEntryScenario which lists the scenarios.
func perEntryExpiresAfterScenarios() []perEntryScenario {
	return []perEntryScenario{
		{name: "SetExpiresAfter_SetsExpiry", run: func(r perEntryRun) {
			r.set(perEntryKey)
			r.expireAfter(perEntryKey, r.ttl)
			r.requireExpires(perEntryKey, r.ttl)
		}},
		{name: "SetExpiresAfter_NeverCreates", run: func(r perEntryRun) {
			r.expireAfter(perEntryKey, r.ttl)
			assert.False(r.t, r.present(perEntryKey))
		}},
		{name: "SetExpiresAfterNonPositive_DoesNothing", run: func(r perEntryRun) {
			r.set(perEntryKey)
			r.expireAfter(perEntryKey, 0)
			r.expireAfter(perEntryKey, -time.Second)
			r.requireNoExpiry(perEntryKey)
			r.setWithTTL("expiring", r.ttl)
			r.expireAfter("expiring", 0)
			r.expireAfter("expiring", -time.Second)
			r.requireExpiry("expiring")
			r.requireExpires("expiring", r.ttl)
			assert.True(r.t, r.present(perEntryKey), "a non-positive SetExpiresAfter must not delete")
		}},
		{name: "SetExpiresAfterNoExpiry_RemovesExpiry", run: func(r perEntryRun) {
			r.setWithTTL(perEntryKey, r.ttl)
			r.expireAfter(perEntryKey, cache_dto.NoExpiry)
			r.requireNoExpiry(perEntryKey)
			r.requireLasts(perEntryKey, 2*r.ttl)
		}},
	}
}

// perEntryBehaviourScenarios returns the scenarios for noop computes, fixed windows and
// concurrent writes.
//
// Returns []perEntryScenario which lists the scenarios.
func perEntryBehaviourScenarios() []perEntryScenario {
	return []perEntryScenario{
		{name: "NoopCompute_DoesNotSetExpiry", run: func(r perEntryRun) {
			r.set(perEntryKey)
			r.compute(perEntryKey, cache_dto.ComputeActionNoop, r.ttl)
			r.requireNoExpiry(perEntryKey)
		}},
		{name: "NoopCompute_KeepsExpiry", run: func(r perEntryRun) {
			r.setWithTTL(perEntryKey, r.ttl)
			r.compute(perEntryKey, cache_dto.ComputeActionNoop, longerTTLFactor*r.ttl)
			r.requireExpires(perEntryKey, r.ttl)
		}},
		{name: "FixedWindowCounter_ResetsAtWindowEnd", run: testFixedWindowCounter},
		{name: "ConcurrentFirstIncrements_KeepTheirExpiry", run: testConcurrentFirstIncrements},
	}
}

// rollbackWrites returns the writes the rollback scenarios make inside a transaction.
//
// Returns []transactionWrite which lists the writes.
func rollbackWrites() []transactionWrite {
	return []transactionWrite{
		{name: "SetWithTTL", write: func(ctx context.Context, transaction cache_domain.TransactionCache[string, string], ttl time.Duration) error {
			return transaction.SetWithTTL(ctx, perEntryKey, "changed", ttl)
		}},
		{name: "ComputeWithTTL", write: func(ctx context.Context, transaction cache_domain.TransactionCache[string, string], ttl time.Duration) error {
			_, _, err := transaction.ComputeWithTTL(ctx, perEntryKey, func(string, bool) cache_dto.ComputeResult[string] {
				return cache_dto.ComputeResult[string]{Value: "changed", Action: cache_dto.ComputeActionSet, TTL: ttl}
			})
			return err
		}},
		{name: "SetExpiresAfter", write: func(ctx context.Context, transaction cache_domain.TransactionCache[string, string], ttl time.Duration) error {
			return transaction.SetExpiresAfter(ctx, perEntryKey, ttl)
		}},
	}
}

// testFixedWindowCounter checks that a counter that sets its TTL only on the first
// increment resets when the window ends, even under steady traffic.
//
// Takes r (perEntryRun) which drives the scenario.
func testFixedWindowCounter(r perEntryRun) {
	r.t.Helper()

	window := r.ttl
	require.Equal(r.t, int64(1), incrementWithWindow(r, perEntryKey, window))

	if r.harness.AdvanceTime != nil {
		for step := 1; step < windowSteps; step++ {
			r.harness.AdvanceTime(window / windowSteps)
			require.Equal(r.t, int64(step+1), incrementWithWindow(r, perEntryKey, window))
		}
		r.harness.AdvanceTime(window - (windowSteps-1)*(window/windowSteps))
		assert.Equal(r.t, int64(1), incrementWithWindow(r, perEntryKey, window),
			"the counter resets when the window that began with the first increment ends")
		return
	}

	halfway := window / 2
	require.True(r.t, pollWhile(halfway, func() bool {
		return incrementWithWindow(r, perEntryKey, window) > 1
	}), "the counter must keep counting for the first half of its window")
	require.True(r.t, pollUntil(window-halfway+realExpirySlack, func() bool {
		return incrementWithWindow(r, perEntryKey, window) == 1
	}), "under steady traffic the counter must still reset when its window ends")
}

// testConcurrentFirstIncrements races several first increments on each key and checks
// that every key keeps the TTL the creating increment set.
//
// Takes r (perEntryRun) which drives the scenario.
func testConcurrentFirstIncrements(r perEntryRun) {
	r.t.Helper()

	for first := 0; first < concurrentKeys; first += concurrentKeysInFlight {
		raceFirstIncrements(r, first, min(first+concurrentKeysInFlight, concurrentKeys))
	}

	lost := 0
	for index := range concurrentKeys {
		entry, found, err := r.harness.Cache.ProbeEntry(r.t.Context(), fmt.Sprintf("counter-%d", index))
		require.NoError(r.t, err)
		require.True(r.t, found)
		if hasNoExpiry(entry.ExpiresAtNano) {
			lost++
		}
	}
	assert.Zero(r.t, lost, "%d of %d keys lost the TTL their first increment set", lost, concurrentKeys)
}

// raceFirstIncrements races concurrentWriters first increments on each key from first up
// to but not including last, and fails the test if any increment returns an error.
//
// Takes r (perEntryRun) which drives the scenario.
// Takes first (int) which is the index of the first key raced.
// Takes last (int) which is one past the index of the last key raced.
func raceFirstIncrements(r perEntryRun, first, last int) {
	r.t.Helper()

	start := make(chan struct{})
	var group sync.WaitGroup
	errs := make(chan error, concurrentWriters*(last-first))
	for index := first; index < last; index++ {
		key := fmt.Sprintf("counter-%d", index)
		for range concurrentWriters {
			group.Go(func() {
				<-start
				_, _, err := r.harness.Cache.ComputeWithTTL(r.t.Context(), key, incrementFunction(longerTTLFactor*r.ttl))
				errs <- err
			})
		}
	}
	close(start)
	group.Wait()
	close(errs)
	for err := range errs {
		require.NoError(r.t, err)
	}
}

// testRollbackRestoresNoExpiry checks that rolling back a write that gave an entry a TTL
// leaves the entry without one.
//
// Takes r (perEntryRun) which drives the scenario.
// Takes write (transactionWrite) which is the write rolled back.
func testRollbackRestoresNoExpiry(r perEntryRun, write transactionWrite) {
	r.t.Helper()

	ctx := r.t.Context()
	r.set(perEntryKey)
	transaction := cache_domain.BeginTransaction(ctx, r.harness.Cache, cache_domain.WithTransactionClock(r.harness.Clock))
	require.NoError(r.t, write.write(ctx, transaction, r.ttl))
	require.NoError(r.t, transaction.Rollback(ctx))

	value, found, err := r.harness.Cache.GetIfPresent(ctx, perEntryKey)
	require.NoError(r.t, err)
	require.True(r.t, found)
	assert.Equal(r.t, perEntryValue, value)
	r.requireNoExpiry(perEntryKey)
	r.requireLasts(perEntryKey, 2*r.ttl)
}

// testRollbackRestoresRemainingTTL checks that rolling back a write to an entry with a
// TTL restores the TTL it had left.
//
// Takes r (perEntryRun) which drives the scenario.
func testRollbackRestoresRemainingTTL(r perEntryRun) {
	r.t.Helper()

	ctx := r.t.Context()
	r.setWithTTL(perEntryKey, r.ttl)
	transaction := cache_domain.BeginTransaction(ctx, r.harness.Cache, cache_domain.WithTransactionClock(r.harness.Clock))
	require.NoError(r.t, transaction.SetWithTTL(ctx, perEntryKey, "changed", cache_dto.NoExpiry))
	require.NoError(r.t, transaction.Rollback(ctx))

	r.requireExpiry(perEntryKey)
	r.requireExpires(perEntryKey, r.ttl)
}

// newPerEntryRun prepares a scenario run with TTLs suited to how the harness moves time.
//
// Takes harness (PerEntryHarness) which is the cache under test.
//
// Returns perEntryRun which drives the scenario.
func newPerEntryRun(t *testing.T, harness PerEntryHarness) perEntryRun {
	t.Helper()

	if harness.AdvanceTime != nil {
		return perEntryRun{harness: harness, t: t, ttl: mockTTL, subSecondTTL: 500 * time.Millisecond}
	}
	return perEntryRun{harness: harness, t: t, ttl: realTTL, subSecondTTL: realSubSecondTTL}
}

// incrementWithWindow increments the counter under key, giving it a TTL of window only
// when the increment creates it.
//
// Takes key (string) which identifies the counter.
// Takes window (time.Duration) which is the counter's window.
//
// Returns int64 which is the counter's value after the increment.
func incrementWithWindow(r perEntryRun, key string, window time.Duration) int64 {
	r.t.Helper()

	value, _, err := r.harness.Cache.ComputeWithTTL(r.t.Context(), key, incrementFunction(window))
	require.NoError(r.t, err)
	count, err := strconv.ParseInt(value, 10, 64)
	require.NoError(r.t, err)
	return count
}

// incrementFunction returns a compute function that increments a decimal counter and sets
// window as its TTL only when it creates the counter.
//
// Takes window (time.Duration) which is the TTL of a new counter.
//
// Returns func(string, bool) cache_dto.ComputeResult[string] which performs the
// increment.
func incrementFunction(window time.Duration) func(string, bool) cache_dto.ComputeResult[string] {
	return func(oldValue string, found bool) cache_dto.ComputeResult[string] {
		if !found {
			return cache_dto.ComputeResult[string]{Value: "1", Action: cache_dto.ComputeActionSet, TTL: window}
		}
		count, err := strconv.ParseInt(oldValue, 10, 64)
		if err != nil {
			count = 0
		}
		return cache_dto.ComputeResult[string]{Value: strconv.FormatInt(count+1, 10), Action: cache_dto.ComputeActionSet, TTL: 0}
	}
}

// hasNoExpiry reports whether an entry's expiry timestamp means it never expires.
// Providers report this as zero or as the largest representable time.
//
// Takes expiresAtNano (int64) which is the entry's expiry in Unix nanoseconds.
//
// Returns bool which is true when the entry has no expiry.
func hasNoExpiry(expiresAtNano int64) bool {
	return expiresAtNano == 0 || expiresAtNano == math.MaxInt64
}

// pollUntil checks condition in the calling goroutine on every poll tick until it holds
// or timeout passes. Polling in the test's own goroutine means no check can outlive the
// test.
//
// Takes timeout (time.Duration) which is how long to keep checking.
// Takes condition (func() bool) which is the check.
//
// Returns bool which is true once condition holds, or false when timeout passes first.
func pollUntil(timeout time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(realPollInterval)
	defer ticker.Stop()
	for {
		if condition() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		<-ticker.C
	}
}

// pollWhile checks condition in the calling goroutine on every poll tick for duration.
//
// Takes duration (time.Duration) which is how long condition must keep holding.
// Takes condition (func() bool) which is the check.
//
// Returns bool which is false as soon as condition fails, or true when it held
// throughout.
func pollWhile(duration time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(duration)
	ticker := time.NewTicker(realPollInterval)
	defer ticker.Stop()
	for time.Now().Before(deadline) {
		if !condition() {
			return false
		}
		<-ticker.C
	}
	return condition()
}
