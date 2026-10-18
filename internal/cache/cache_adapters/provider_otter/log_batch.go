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
	"errors"
	"fmt"
	"sync"
	"time"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/wal/wal_domain"
)

var (
	_ cache_domain.LogBatcher = (*OtterAdapter[string, any])(nil)
)

// logBatch collects the log entries of one transaction until it commits.
type logBatch[K comparable, V any] struct {
	// entries holds the collected entries in the order they were written.
	entries []wal_domain.Entry[K, V]

	// mu guards entries against a stray concurrent writer.
	mu sync.Mutex
}

// add appends entries to the batch.
//
// Takes entries (...wal_domain.Entry[K, V]) which are the entries to collect.
//
// Safe for concurrent use.
func (b *logBatch[K, V]) add(entries ...wal_domain.Entry[K, V]) {
	b.mu.Lock()
	b.entries = append(b.entries, entries...)
	b.mu.Unlock()
}

// BeginLogBatch starts collecting the cache's log entries so a transaction reaches the
// log as one record. It is a no-op when the cache keeps no log.
//
// Concurrency: holds the batch gate until CommitLogBatch or AbortLogBatch, so a second
// batch and any explicit checkpoint or close wait for this one to end.
func (a *OtterAdapter[K, V]) BeginLogBatch() {
	if !a.walEnabled || a.wal == nil {
		return
	}
	a.logBatchGate.Lock()
	a.openBatch.Store(&logBatch[K, V]{})
}

// CommitLogBatch writes the collected entries to the log as one record and ends the
// batch. A single entry is written as itself; a batch too large for one record is written
// entry by entry, which loses atomicity but never the entries.
//
// Returns error when the log cannot be written.
//
// Concurrency: releases the batch gate taken by BeginLogBatch.
func (a *OtterAdapter[K, V]) CommitLogBatch(ctx context.Context) error {
	batch := a.openBatch.Swap(nil)
	if batch == nil {
		return nil
	}
	defer a.logBatchGate.Unlock()

	a.checkpointMu.RLock()
	err := a.appendEntries(ctx, batch.entries)
	a.checkpointMu.RUnlock()
	if err != nil {
		return err
	}
	a.maybeCheckpoint()
	return nil
}

// AbortLogBatch discards the collected entries and ends the batch.
//
// Concurrency: releases the batch gate taken by BeginLogBatch.
func (a *OtterAdapter[K, V]) AbortLogBatch() {
	if a.openBatch.Swap(nil) != nil {
		a.logBatchGate.Unlock()
	}
}

// logEntries records entries in the open batch, or writes them to the log as one record
// when no batch is open.
//
// Takes entries (...wal_domain.Entry[K, V]) which are the entries to record.
//
// Returns error when the log cannot be written.
//
// Must be called with checkpointMu held for reading.
func (a *OtterAdapter[K, V]) logEntries(ctx context.Context, entries ...wal_domain.Entry[K, V]) error {
	if batch := a.openBatch.Load(); batch != nil {
		batch.add(entries...)
		return nil
	}
	return a.appendEntries(ctx, entries)
}

// appendEntries writes one log record when possible, falling back to separate records
// when the batch exceeds the limit.
//
// Takes entries ([]wal_domain.Entry[K, V]) which are the entries to write.
//
// Returns error when the log cannot be written.
//
// Must be called with checkpointMu held for reading.
func (a *OtterAdapter[K, V]) appendEntries(ctx context.Context, entries []wal_domain.Entry[K, V]) error {
	ctx = context.WithoutCancel(ctx)
	switch len(entries) {
	case 0:
		return nil
	case 1:
		return a.wal.Append(ctx, entries[0])
	}

	err := a.wal.Append(ctx, wal_domain.Entry[K, V]{
		Operation: wal_domain.OpBatch,
		Batch:     entries,
		Timestamp: time.Now().UnixNano(),
		Key:       *new(K),
		Value:     *new(V),
		Tags:      nil,
		ExpiresAt: 0,
	})
	if !errors.Is(err, wal_domain.ErrEntryTooLarge) {
		return err
	}

	_, l := logger_domain.From(ctx, log)
	l.Warn("Transaction too large for one log record; writing its entries one by one",
		logger_domain.Int("entries", len(entries)))
	for i, entry := range entries {
		if err := a.wal.Append(ctx, entry); err != nil {
			return fmt.Errorf("writing entry %d of an oversized transaction: %w", i, err)
		}
	}
	return nil
}
