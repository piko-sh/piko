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
	"context"
	"time"

	"github.com/maypok86/otter/v2"
	"github.com/maypok86/otter/v2/stats"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/wdk/safeconv"
)

// expiryCalculatorAdapter wraps a cache_dto.ExpiryCalculator to implement
// otter.ExpiryCalculator.
type expiryCalculatorAdapter[K comparable, V any] struct {
	// calculator works out when cache entries should expire.
	calculator cache_dto.ExpiryCalculator[K, V]
}

// ExpireAfterCreate returns the expiration duration after an entry is created.
//
// Takes entry (otter.Entry[K, V]) which is the cache entry being created.
//
// Returns time.Duration which is how long until the entry expires.
func (a *expiryCalculatorAdapter[K, V]) ExpireAfterCreate(entry otter.Entry[K, V]) time.Duration {
	return a.calculator.ExpireAfterCreate(convertEntryToDTO(entry))
}

// ExpireAfterUpdate returns the expiration duration after an entry is updated.
//
// Takes entry (otter.Entry[K, V]) which is the cache entry being updated.
// Takes oldValue (V) which is the previous value before the update.
//
// Returns time.Duration which is how long until the entry expires.
func (a *expiryCalculatorAdapter[K, V]) ExpireAfterUpdate(entry otter.Entry[K, V], oldValue V) time.Duration {
	return a.calculator.ExpireAfterUpdate(convertEntryToDTO(entry), oldValue)
}

// ExpireAfterRead returns the expiration duration after an entry is read.
//
// Takes entry (otter.Entry[K, V]) which is the cache entry that was read.
//
// Returns time.Duration which is how long the entry should remain valid.
func (a *expiryCalculatorAdapter[K, V]) ExpireAfterRead(entry otter.Entry[K, V]) time.Duration {
	return a.calculator.ExpireAfterRead(convertEntryToDTO(entry))
}

// refreshCalculatorAdapter wraps a cache_dto.RefreshCalculator to implement
// the otter.RefreshCalculator interface.
type refreshCalculatorAdapter[K comparable, V any] struct {
	// calculator decides when to refresh the cache and which rules to use.
	calculator cache_dto.RefreshCalculator[K, V]
}

// RefreshAfterCreate returns the refresh duration after an entry is created.
//
// Takes entry (otter.Entry[K, V]) which is the cache entry that was created.
//
// Returns time.Duration which specifies how long to wait before refreshing.
func (a *refreshCalculatorAdapter[K, V]) RefreshAfterCreate(entry otter.Entry[K, V]) time.Duration {
	return a.calculator.RefreshAfterCreate(convertEntryToDTO(entry))
}

// RefreshAfterUpdate returns the refresh duration after an entry is updated.
//
// Takes entry (otter.Entry[K, V]) which is the cache entry that was updated.
// Takes oldValue (V) which is the previous value before the update.
//
// Returns time.Duration which is the duration until the entry should refresh.
func (a *refreshCalculatorAdapter[K, V]) RefreshAfterUpdate(entry otter.Entry[K, V], oldValue V) time.Duration {
	return a.calculator.RefreshAfterUpdate(convertEntryToDTO(entry), oldValue)
}

// RefreshAfterReload returns the duration to wait before the next refresh
// after an entry is reloaded.
//
// Takes entry (otter.Entry[K, V]) which is the cache entry that was reloaded.
// Takes oldValue (V) which is the previous value before the reload.
//
// Returns time.Duration which is how long to wait before the next refresh.
func (a *refreshCalculatorAdapter[K, V]) RefreshAfterReload(entry otter.Entry[K, V], oldValue V) time.Duration {
	return a.calculator.RefreshAfterReload(convertEntryToDTO(entry), oldValue)
}

// RefreshAfterReloadFailure returns the refresh duration after a reload
// failure.
//
// Takes entry (otter.Entry) which is the cache entry that failed to reload.
// Takes err (error) which is the error that caused the reload failure.
//
// Returns time.Duration which is the delay before the next refresh attempt.
func (a *refreshCalculatorAdapter[K, V]) RefreshAfterReloadFailure(entry otter.Entry[K, V], err error) time.Duration {
	return a.calculator.RefreshAfterReloadFailure(convertEntryToDTO(entry), err)
}

// statsRecorderAdapter wraps a StatsRecorder to provide the otter
// stats.Recorder interface.
type statsRecorderAdapter struct {
	// recorder tracks cache statistics such as hits, misses, and evictions.
	recorder cache_dto.StatsRecorder
}

// RecordHits records the given number of cache hits.
//
// Takes count (int) which specifies how many hits to record.
func (a *statsRecorderAdapter) RecordHits(count int) {
	if count > 0 {
		a.recorder.RecordHits(safeconv.IntToUint64(count))
	}
}

// RecordMisses records cache miss events.
//
// Takes count (int) which specifies the number of misses to record.
func (a *statsRecorderAdapter) RecordMisses(count int) {
	if count > 0 {
		a.recorder.RecordMisses(safeconv.IntToUint64(count))
	}
}

// RecordEviction records a cache eviction event.
func (a *statsRecorderAdapter) RecordEviction(_ uint32) {
	a.recorder.RecordEviction()
}

// RecordLoadSuccess records a successful load event.
//
// Takes loadTime (time.Duration) which specifies how long the load took.
func (a *statsRecorderAdapter) RecordLoadSuccess(loadTime time.Duration) {
	a.recorder.RecordLoadSuccess(loadTime)
}

// RecordLoadFailure records a failed load event.
//
// Takes loadTime (time.Duration) which is the time spent before the failure.
func (a *statsRecorderAdapter) RecordLoadFailure(loadTime time.Duration) {
	a.recorder.RecordLoadFailure(loadTime)
}

// clockAdapter wraps a cache_dto.Clock to implement the otter.Clock interface.
type clockAdapter struct {
	// clock provides the current time for cache operations.
	clock cache_dto.Clock
}

// NowNano returns the current time in nanoseconds.
//
// Returns int64 which is the current time as nanoseconds since the Unix epoch.
func (a *clockAdapter) NowNano() int64 {
	return a.clock.Now().UnixNano()
}

// Tick returns a channel that sends the current time at regular intervals.
//
// Takes duration (time.Duration) which sets the time between ticks.
//
// Returns <-chan time.Time which yields the current time at each tick.
func (*clockAdapter) Tick(duration time.Duration) <-chan time.Time {
	return time.Tick(duration)
}

// loggerAdapter wraps a cache_dto.Logger to satisfy the otter.Logger interface.
type loggerAdapter struct {
	// logger is the wrapped logger that receives all log output.
	logger cache_dto.Logger
}

// Warn logs a warning message with optional error details.
//
// Takes message (string) which is the warning message to log.
// Takes err (error) which is an optional error to include, or nil if none.
func (a *loggerAdapter) Warn(_ context.Context, message string, err error) {
	if err != nil {
		a.logger.Warn(message, "error", err)
	} else {
		a.logger.Warn(message)
	}
}

// Error logs an error message with optional error details.
//
// Takes message (string) which is the error message to log.
// Takes err (error) which provides optional error details; if not nil, it is
// added to the log output.
func (a *loggerAdapter) Error(_ context.Context, message string, err error) {
	if err != nil {
		a.logger.Error(message, "error", err)
	} else {
		a.logger.Error(message)
	}
}

// convertEntryToDTO converts an otter.Entry to cache_dto.Entry.
//
// Takes otterEntry (otter.Entry[K, V]) which is the otter cache
// entry to convert.
//
// Returns cache_dto.Entry[K, V] which is the converted DTO entry.
func convertEntryToDTO[K comparable, V any](otterEntry otter.Entry[K, V]) cache_dto.Entry[K, V] {
	return cache_dto.Entry[K, V]{
		Key:               otterEntry.Key,
		Value:             otterEntry.Value,
		Weight:            otterEntry.Weight,
		ExpiresAtNano:     otterEntry.ExpiresAtNano,
		RefreshableAtNano: otterEntry.RefreshableAtNano,
		SnapshotAtNano:    otterEntry.SnapshotAtNano,
	}
}

// convertDeletionEventToDTO converts an otter.DeletionEvent to
// cache_dto.DeletionEvent.
//
// Takes otterEvent (otter.DeletionEvent[K, V]) which is the otter
// deletion event to convert.
//
// Returns cache_dto.DeletionEvent[K, V] which is the converted
// DTO deletion event.
func convertDeletionEventToDTO[K comparable, V any](otterEvent otter.DeletionEvent[K, V]) cache_dto.DeletionEvent[K, V] {
	return cache_dto.DeletionEvent[K, V]{
		Key:   otterEvent.Key,
		Value: otterEvent.Value,
		Cause: cache_dto.DeletionCause(otterEvent.Cause),
	}
}

// wrapExpiryCalculator wraps a cache_dto.ExpiryCalculator to work with otter.
//
// Takes calculator (cache_dto.ExpiryCalculator[K, V]) which is
// the calculator to wrap.
//
// Returns otter.ExpiryCalculator[K, V] which is the wrapped
// calculator, or nil if the input is nil.
func wrapExpiryCalculator[K comparable, V any](calculator cache_dto.ExpiryCalculator[K, V]) otter.ExpiryCalculator[K, V] {
	if calculator == nil {
		return nil
	}
	return &expiryCalculatorAdapter[K, V]{calculator: calculator}
}

// wrapRefreshCalculator wraps a cache_dto.RefreshCalculator to work with otter.
//
// Takes calculator (cache_dto.RefreshCalculator[K, V]) which is
// the calculator to wrap.
//
// Returns otter.RefreshCalculator[K, V] which is the wrapped
// calculator, or nil if the input is nil.
func wrapRefreshCalculator[K comparable, V any](calculator cache_dto.RefreshCalculator[K, V]) otter.RefreshCalculator[K, V] {
	if calculator == nil {
		return nil
	}
	return &refreshCalculatorAdapter[K, V]{calculator: calculator}
}

// wrapOnDeletion wraps a cache_dto OnDeletion callback to work with otter.
//
// Takes callback (func(e cache_dto.DeletionEvent[K, V])) which is the deletion
// event handler to wrap.
//
// Returns func(e otter.DeletionEvent[K, V]) which is the wrapped callback for
// otter. Returns nil if callback is nil.
func wrapOnDeletion[K comparable, V any](callback func(e cache_dto.DeletionEvent[K, V])) func(e otter.DeletionEvent[K, V]) {
	if callback == nil {
		return nil
	}
	return func(otterEvent otter.DeletionEvent[K, V]) {
		callback(convertDeletionEventToDTO(otterEvent))
	}
}

// wrapOnAtomicDeletion wraps a cache_dto OnAtomicDeletion callback to work
// with otter.
//
// Takes callback (func(e cache_dto.DeletionEvent[K, V])) which is the
// cache_dto deletion event handler to wrap.
//
// Returns func(e otter.DeletionEvent[K, V]) which is the wrapped callback
// that works with otter. Returns nil if callback is nil.
func wrapOnAtomicDeletion[K comparable, V any](callback func(e cache_dto.DeletionEvent[K, V])) func(e otter.DeletionEvent[K, V]) {
	if callback == nil {
		return nil
	}
	return func(otterEvent otter.DeletionEvent[K, V]) {
		callback(convertDeletionEventToDTO(otterEvent))
	}
}

// wrapStatsRecorder wraps a StatsRecorder to work with otter's stats system.
//
// Takes recorder (cache_dto.StatsRecorder) which provides the stats recording
// interface to adapt.
//
// Returns stats.Recorder which is the adapted recorder, or nil if the input
// is nil.
func wrapStatsRecorder(recorder cache_dto.StatsRecorder) stats.Recorder {
	if recorder == nil {
		return nil
	}
	return &statsRecorderAdapter{recorder: recorder}
}

// wrapClock wraps a cache_dto.Clock to work with otter.
//
// Takes clock (cache_dto.Clock) which provides time functions.
//
// Returns otter.Clock which is the wrapped clock, or nil if clock is nil.
func wrapClock(clock cache_dto.Clock) otter.Clock {
	if clock == nil {
		return nil
	}
	return &clockAdapter{clock: clock}
}

// wrapLogger wraps a Logger to work with otter's logging interface.
//
// Takes logger (cache_dto.Logger) which is the logger to wrap.
//
// Returns otter.Logger which is the wrapped logger, or nil if logger is nil.
func wrapLogger(logger cache_dto.Logger) otter.Logger {
	if logger == nil {
		return nil
	}
	return &loggerAdapter{logger: logger}
}
