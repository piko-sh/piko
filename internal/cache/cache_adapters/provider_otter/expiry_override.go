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

package provider_otter

import (
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/maypok86/otter/v2"
	"piko.sh/piko/internal/cache/cache_dto"
)

// expiryOverride is one pending TTL for the next write of a key. Each write stores its
// own pointer, so a write can remove its override without removing another write's.
type expiryOverride struct {
	// duration is the TTL to apply, or cache_dto.NoExpiry to remove any expiry.
	duration time.Duration
}

// expiryOverrides passes a TTL from a write to the expiry calculator that runs inside
// that write.
//
// Otter works out an entry's expiry under the same bucket lock as the write itself, so a
// TTL stored inside a compute callback is read by the very write it belongs to. Setting
// the TTL after the write instead would leave a gap in which a concurrent write could
// replace the entry and lose it.
type expiryOverrides[K comparable] struct {
	// pending maps a key to the *expiryOverride its current write stored.
	pending sync.Map

	// count is how many overrides are pending, so writes skip the map when none are.
	count atomic.Int64
}

// store records a TTL for the next write of key.
//
// Must be called inside the otter compute callback of the write it belongs to.
//
// Takes key (K) which identifies the entry being written.
// Takes duration (time.Duration) which is the TTL to apply.
//
// Returns *expiryOverride which identifies this override for discard.
func (o *expiryOverrides[K]) store(key K, duration time.Duration) *expiryOverride {
	override := &expiryOverride{duration: duration}
	o.count.Add(1)
	if _, replaced := o.pending.Swap(key, override); replaced {
		o.count.Add(-1)
	}
	return override
}

// take removes and returns the pending TTL for key.
//
// Takes key (K) which identifies the entry being written.
//
// Returns time.Duration which is the pending TTL.
// Returns bool which is false when no TTL is pending.
func (o *expiryOverrides[K]) take(key K) (time.Duration, bool) {
	if o.count.Load() == 0 {
		return 0, false
	}
	value, ok := o.pending.LoadAndDelete(key)
	if !ok {
		return 0, false
	}
	o.count.Add(-1)
	override, ok := value.(*expiryOverride)
	if !ok {
		return 0, false
	}
	return override.duration, true
}

// discard removes override if the write it belongs to did not use it, which happens only
// when otter skipped the expiry calculation.
//
// Takes key (K) which identifies the entry that was written.
// Takes override (*expiryOverride) which is the override store returned.
func (o *expiryOverrides[K]) discard(key K, override *expiryOverride) {
	if o.pending.CompareAndDelete(key, override) {
		o.count.Add(-1)
	}
}

// overridingExpiryCalculator applies a pending TTL from expiryOverrides when an entry is
// created or updated, and otherwise defers to the cache's own calculator.
type overridingExpiryCalculator[K comparable, V any] struct {
	// inner is the cache's configured expiry calculator.
	inner otter.ExpiryCalculator[K, V]

	// overrides holds the TTLs pending for writes in progress.
	overrides *expiryOverrides[K]
}

// ExpireAfterCreate returns the pending TTL for the entry, or the inner calculator's
// answer when none is pending.
//
// Takes entry (otter.Entry[K, V]) which is the entry being created.
//
// Returns time.Duration which is how long until the entry expires.
func (c *overridingExpiryCalculator[K, V]) ExpireAfterCreate(entry otter.Entry[K, V]) time.Duration {
	if duration, ok := c.overrides.take(entry.Key); ok {
		return overrideDuration(duration, entry.SnapshotAtNano)
	}
	return c.inner.ExpireAfterCreate(entry)
}

// ExpireAfterUpdate returns the pending TTL for the entry, or the inner calculator's
// answer when none is pending.
//
// Takes entry (otter.Entry[K, V]) which is the entry being updated.
// Takes oldValue (V) which is the value being replaced.
//
// Returns time.Duration which is how long until the entry expires.
func (c *overridingExpiryCalculator[K, V]) ExpireAfterUpdate(entry otter.Entry[K, V], oldValue V) time.Duration {
	if duration, ok := c.overrides.take(entry.Key); ok {
		return overrideDuration(duration, entry.SnapshotAtNano)
	}
	return c.inner.ExpireAfterUpdate(entry, oldValue)
}

// ExpireAfterRead returns the inner calculator's answer. A read never carries a TTL of
// its own.
//
// Takes entry (otter.Entry[K, V]) which is the entry being read.
//
// Returns time.Duration which is how long until the entry expires.
func (c *overridingExpiryCalculator[K, V]) ExpireAfterRead(entry otter.Entry[K, V]) time.Duration {
	return c.inner.ExpireAfterRead(entry)
}

// newOverridingExpiryCalculator wraps inner so pending TTLs in overrides take precedence
// on create and update.
//
// Takes inner (otter.ExpiryCalculator[K, V]) which is the cache's own calculator.
// Takes overrides (*expiryOverrides[K]) which holds the pending TTLs.
//
// Returns otter.ExpiryCalculator[K, V] which applies the overrides.
func newOverridingExpiryCalculator[K comparable, V any](
	inner otter.ExpiryCalculator[K, V],
	overrides *expiryOverrides[K],
) otter.ExpiryCalculator[K, V] {
	return &overridingExpiryCalculator[K, V]{inner: inner, overrides: overrides}
}

// overrideDuration turns a pending TTL into the duration otter expects.
//
// Otter adds the duration to the time of the write, so cache_dto.NoExpiry becomes the
// time left until the latest representable instant. Otter ignores a duration of zero or
// less and keeps the old expiry, so any other TTL is raised to at least one nanosecond,
// and it is capped against snapshotAtNano so the sum cannot overflow. Otter reads its
// clock after the write works out its TTL, so this is the only cap that holds for the
// time otter adds the TTL to; it covers live writes and restored entries alike.
//
// Takes duration (time.Duration) which is the pending TTL.
// Takes snapshotAtNano (int64) which is the time of the write in nanoseconds.
//
// Returns time.Duration which is the duration to hand to otter.
func overrideDuration(duration time.Duration, snapshotAtNano int64) time.Duration {
	if duration != cache_dto.NoExpiry {
		return capTTL(max(duration, time.Nanosecond), snapshotAtNano)
	}
	if snapshotAtNano <= 0 {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(math.MaxInt64 - snapshotAtNano)
}
