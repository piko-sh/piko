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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_dto"
)

func gateOptions(ceiling uint32, onDeletion func(cache_dto.DeletionEvent[string, string])) cache_dto.Options[string, string] {
	return cache_dto.Options[string, string]{
		MaximumWeight:  1 << 20,
		MaxEntryWeight: ceiling,
		Weigher:        func(_ string, value string) uint32 { return uint32(len(value)) },
		OnDeletion:     onDeletion,
	}
}

type computingCache struct {
	*mockCache[string, string]
}

func newComputingCache(initial map[string]string) *computingCache {
	return &computingCache{mockCache: &mockCache[string, string]{data: initial}}
}

func (c *computingCache) Compute(
	_ context.Context,
	key string,
	computeFunction func(oldValue string, found bool) (string, cache_dto.ComputeAction),
) (string, bool, error) {
	old, found := c.data[key]
	value, action := computeFunction(old, found)

	return c.apply(key, value, action)
}

func (c *computingCache) ComputeIfAbsent(
	_ context.Context,
	key string,
	computeFunction func() string,
) (string, bool, error) {
	if existing, found := c.data[key]; found {
		return existing, true, nil
	}
	value := computeFunction()
	c.data[key] = value

	return value, true, nil
}

func (c *computingCache) ComputeIfPresent(
	_ context.Context,
	key string,
	computeFunction func(oldValue string) (string, cache_dto.ComputeAction),
) (string, bool, error) {
	old, found := c.data[key]
	if !found {
		return "", false, nil
	}
	value, action := computeFunction(old)

	return c.apply(key, value, action)
}

func (c *computingCache) ComputeWithTTL(
	_ context.Context,
	key string,
	computeFunction func(oldValue string, found bool) cache_dto.ComputeResult[string],
) (string, bool, error) {
	old, found := c.data[key]
	result := computeFunction(old, found)

	return c.apply(key, result.Value, result.Action)
}

func (c *computingCache) apply(key, value string, action cache_dto.ComputeAction) (string, bool, error) {
	switch action {
	case cache_dto.ComputeActionSet:
		c.data[key] = value

		return value, true, nil
	case cache_dto.ComputeActionDelete:
		delete(c.data, key)

		return value, false, nil
	case cache_dto.ComputeActionNoop:
		existing, found := c.data[key]

		return existing, found, nil
	default:
		return value, false, nil
	}
}

func TestAdmissionGate_SetRefusesAnOversizedValue(t *testing.T) {
	t.Parallel()

	inner := &mockCache[string, string]{data: map[string]string{}}
	gate := newAdmissionGate[string, string](inner, gateOptions(8, nil))

	err := gate.Set(context.Background(), "key", "far-too-long-a-value")

	require.ErrorIs(t, err, ErrEntryTooLarge)
	assert.NotContains(t, inner.data, "key", "a refused value must not reach the cache")
}

func TestAdmissionGate_SetAdmitsAValueAtTheCeiling(t *testing.T) {
	t.Parallel()

	inner := &mockCache[string, string]{data: map[string]string{}}
	gate := newAdmissionGate[string, string](inner, gateOptions(8, nil))

	require.NoError(t, gate.Set(context.Background(), "key", "12345678"))
	assert.Contains(t, inner.data, "key", "the ceiling is inclusive")
}

func TestAdmissionGate_ReportsARejectionThroughTheDeletionCallback(t *testing.T) {
	t.Parallel()

	var events []cache_dto.DeletionEvent[string, string]
	inner := &mockCache[string, string]{data: map[string]string{}}
	gate := newAdmissionGate[string, string](inner, gateOptions(4, func(e cache_dto.DeletionEvent[string, string]) {
		events = append(events, e)
	}))

	_ = gate.Set(context.Background(), "key", "too-long")

	require.Len(t, events, 1, "an operator who ignores the error must still see the rejection")
	assert.Equal(t, cache_dto.CauseRejected, events[0].Cause)
	assert.Equal(t, "key", events[0].Key)
	assert.False(t, events[0].WasEvicted(), "a value that was never stored was not evicted")
}

func TestAdmissionGate_BulkSetAdmitsTheAdmissibleAndReportsTheRest(t *testing.T) {
	t.Parallel()

	inner := &mockCache[string, string]{data: map[string]string{}}
	gate := newAdmissionGate[string, string](inner, gateOptions(4, nil))

	err := gate.BulkSet(context.Background(), map[string]string{
		"small-a": "ok",
		"small-b": "fine",
		"large":   "far-too-long",
	})

	require.ErrorIs(t, err, ErrEntryTooLarge)
	assert.Contains(t, inner.data, "small-a", "one oversized item must not punish the rest of the batch")
	assert.Contains(t, inner.data, "small-b")
	assert.NotContains(t, inner.data, "large")
}

func TestAdmissionGate_SetWithTTLRefusesAnOversizedValue(t *testing.T) {
	t.Parallel()

	inner := &mockCache[string, string]{data: map[string]string{}}
	gate := newAdmissionGate[string, string](inner, gateOptions(4, nil))

	err := gate.SetWithTTL(context.Background(), "key", "far-too-long", time.Minute)

	require.ErrorIs(t, err, ErrEntryTooLarge)
	assert.NotContains(t, inner.data, "key")
}

func TestAdmissionGate_IsNotInstalledWithoutACeiling(t *testing.T) {
	t.Parallel()

	inner := &mockCache[string, string]{data: map[string]string{}}

	gate := newAdmissionGate[string, string](inner, cache_dto.Options[string, string]{})

	assert.Same(t, Cache[string, string](inner), gate,
		"a cache with no ceiling must not pay for a decorator it does not use")
}

func TestAdmissionGate_BeginTransactionNeverReturnsNil(t *testing.T) {
	t.Parallel()

	inner := &mockCache[string, string]{data: map[string]string{}}
	gate := newAdmissionGate[string, string](inner, gateOptions(8, nil))

	transactional, ok := gate.(Transactional[string, string])
	require.True(t, ok, "embedding must not hide an optional interface the caller probes for")

	transaction := transactional.BeginTransaction(context.Background())
	require.NotNil(t, transaction,
		"a nil transaction satisfies the assertion and then panics on first use")

	require.NoError(t, transaction.Set(context.Background(), "key", "value"))
	require.NoError(t, transaction.Commit(context.Background()))
	assert.Equal(t, "value", inner.data["key"])
}

func TestAdmissionGate_TransactionalWritesStillHonourTheCeiling(t *testing.T) {
	t.Parallel()

	inner := &mockCache[string, string]{data: map[string]string{}}
	gate := newAdmissionGate[string, string](inner, gateOptions(4, nil))

	transactional, ok := gate.(Transactional[string, string])
	require.True(t, ok)

	transaction := transactional.BeginTransaction(context.Background())
	require.NotNil(t, transaction)

	err := transaction.Set(context.Background(), "key", "far-too-long")
	require.ErrorIs(t, err, ErrEntryTooLarge,
		"a transaction must not be a way around the ceiling")

	require.NoError(t, transaction.Rollback(context.Background()))
}

func TestAdmissionGate_ComputeRefusesAnOversizedValue(t *testing.T) {
	t.Parallel()

	inner := newComputingCache(map[string]string{})
	gate := newAdmissionGate[string, string](inner, gateOptions(4, nil))

	_, _, err := gate.Compute(context.Background(), "key",
		func(_ string, _ bool) (string, cache_dto.ComputeAction) {
			return "far-too-long", cache_dto.ComputeActionSet
		})
	require.NoError(t, err)

	assert.NotContains(t, inner.data, "key",
		"Compute stores a caller-supplied value and must not bypass the ceiling")
}

func TestAdmissionGate_ComputeIfPresentRefusesAnOversizedValue(t *testing.T) {
	t.Parallel()

	inner := newComputingCache(map[string]string{"key": "ok"})
	gate := newAdmissionGate[string, string](inner, gateOptions(4, nil))

	_, _, err := gate.ComputeIfPresent(context.Background(), "key",
		func(string) (string, cache_dto.ComputeAction) {
			return "far-too-long", cache_dto.ComputeActionSet
		})
	require.NoError(t, err)

	assert.Equal(t, "ok", inner.data["key"], "the superseding value was over the ceiling")
}

func TestAdmissionGate_ComputeWithTTLRefusesAnOversizedValue(t *testing.T) {
	t.Parallel()

	inner := newComputingCache(map[string]string{})
	gate := newAdmissionGate[string, string](inner, gateOptions(4, nil))

	_, _, err := gate.ComputeWithTTL(context.Background(), "key",
		func(string, bool) cache_dto.ComputeResult[string] {
			return cache_dto.ComputeResult[string]{Value: "far-too-long", Action: cache_dto.ComputeActionSet}
		})
	require.NoError(t, err)

	assert.NotContains(t, inner.data, "key")
}

func TestAdmissionGate_ComputeIfAbsentReportsAnOversizedValue(t *testing.T) {
	t.Parallel()

	inner := newComputingCache(map[string]string{})
	gate := newAdmissionGate[string, string](inner, gateOptions(4, nil))

	_, found, err := gate.ComputeIfAbsent(context.Background(), "key", func() string {
		return "far-too-long"
	})

	require.ErrorIs(t, err, ErrEntryTooLarge)
	assert.False(t, found)
}

type checkpointingCache struct {
	*mockCache[string, string]
	checkpointed bool
}

func (c *checkpointingCache) Checkpoint(context.Context) error {
	c.checkpointed = true

	return nil
}

func TestAdmissionGate_ForwardsTheCheckpointProbe(t *testing.T) {
	t.Parallel()

	inner := &checkpointingCache{mockCache: &mockCache[string, string]{data: map[string]string{}}}
	gate := newAdmissionGate[string, string](inner, gateOptions(8, nil))

	checkpointer, ok := gate.(interface{ Checkpoint(context.Context) error })
	require.True(t, ok, "persistence probes a cache for this method and the gate must not hide it")
	require.NoError(t, checkpointer.Checkpoint(context.Background()))

	assert.True(t, inner.checkpointed, "the probe must reach the cache that can actually checkpoint")
}
