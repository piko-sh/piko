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

package driven_disk

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/wal/wal_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// alignment is the 4KB sector size for SSD-optimised writes. Every physical
	// write is padded to a multiple of this size when aligned writes are enabled.
	alignment = 4096

	// readBufferSize is the buffer size for reading during recovery (64KB).
	// Recovery reads through the file in order, and larger reads are more
	// efficient for sequential access.
	readBufferSize = 64 * 1024

	// recoverLogKey is the logger key for file position fields during recovery.
	recoverLogKey = "position"

	// defaultMaxBatchSize is the largest number of pending writes to collect
	// before writing to disk.
	defaultMaxBatchSize = 256

	// defaultMaxBatchWait is the longest time to wait for a batch to fill.
	defaultMaxBatchWait = 50 * time.Microsecond

	// defaultChannelSize is the buffer size for the pending write channel.
	defaultChannelSize = 4096
)

// resultChanPool is a pool for result channels to reduce allocations.
// Each channel has capacity 1 for a single error result.
var resultChanPool = sync.Pool{
	New: func() any {
		return make(chan error, 1)
	},
}

// pendingWrite represents a write operation waiting to be committed.
type pendingWrite struct {
	// result is the channel used to send the write result back to the caller.
	result chan error

	// data holds the serialised entry with its length prefix.
	data []byte

	// enc holds the pooled buffer result to be released after writing.
	enc EncodeResult
}

// DiskWAL implements the WAL interface using disk-based storage.
//
// The file format is a sequence of entries, each with:
// [Length:4][CRC32:4][Payload:var]
// Where Length is the size of CRC32+Payload (i.e., everything after Length).
//
// DiskWAL uses a group commit pattern for high-concurrency performance.
// Writers submit entries to a channel, and a single commit goroutine batches
// writes and syncs, reducing lock contention and fsync overhead.
//
// All file operations go through a safedisk sandbox for security.
type DiskWAL[K comparable, V any] struct {
	// lastSync records when the write-ahead log was last synced to disk.
	lastSync time.Time

	// file is the open WAL file handle.
	file safedisk.FileHandle

	// closeCtx is a non-cancellable context derived from the constructor
	// context. It preserves context values (trace IDs, logger fields) for
	// use in Close and other cleanup paths that lack a caller-provided
	// context.
	closeCtx context.Context

	// clock provides time functions; replaced during testing.
	clock clock.Clock

	// sandbox provides safe file operations within the WAL directory.
	sandbox safedisk.Sandbox

	// pendingChan coordinates group commit operations.
	pendingChan chan pendingWrite

	// codec converts WAL entries to and from binary format.
	codec *BinaryCodec[K, V]

	// commitDone signals when the commit goroutine has stopped.
	commitDone chan struct{}

	// stopChan signals the background goroutines to stop.
	stopChan chan struct{}

	// fatalErr holds a permanent I/O error; once set, all writes are rejected.
	fatalErr atomic.Pointer[error]

	// tailBuf holds the partial 4KB sector from the previous
	// flush.
	//
	// Reused between flushes to avoid allocation. Only used
	// when alignWrites is true.
	tailBuf []byte

	// alignedBuf is a scratch buffer for building write batches. Reused
	// between flushes to avoid allocation.
	alignedBuf []byte

	// config holds the WAL configuration settings. Placed last among
	// pointer-containing fields because its non-pointer tail (ints, bools)
	// would extend the GC scan range if pointer fields followed it.
	config wal_domain.Config

	// entryCount tracks the number of entries written.
	entryCount atomic.Int64

	// fileSize tracks the current size of the WAL file in bytes.
	fileSize atomic.Int64

	// nextOffset is the logical end of WAL data (total bytes of actual entries
	// written). Used as the write position for WriteAt calls.
	nextOffset int64

	// pendingSyncs tracks the number of writes since the last sync in batched mode.
	pendingSyncs int

	// mu guards file writes; only held by the commit goroutine.
	mu sync.Mutex

	// alignWrites controls whether writes are padded to 4KB sector boundaries.
	alignWrites bool

	// closed indicates whether the WAL has been closed.
	closed atomic.Bool
}

// WALOption configures a DiskWAL instance.
type WALOption[K comparable, V any] func(*DiskWAL[K, V])

// initialiseFileState reads the file size, initialises offset tracking and aligned
// write buffers. Must be called before the commit loop starts.
//
// Takes config (wal_domain.Config) which provides the alignment setting.
//
// Returns error when file stat or tail sector reading fails.
func (w *DiskWAL[K, V]) initialiseFileState(config wal_domain.Config) error {
	info, err := w.file.Stat()
	if err != nil {
		return fmt.Errorf("getting WAL file info: %w", err)
	}

	initialSize := info.Size()
	w.alignWrites = !config.DisableAlignedWrites
	w.nextOffset = initialSize
	w.tailBuf = make([]byte, 0, alignment)
	w.alignedBuf = make([]byte, 0, alignment*2)

	if w.alignWrites {
		rem := initialSize % int64(alignment)
		if rem > 0 {
			if err := w.readTailSector(initialSize, rem); err != nil {
				return fmt.Errorf("reading tail sector: %w", err)
			}
		}
	}

	w.lastSync = w.clock.Now()
	w.fileSize.Store(initialSize)
	return nil
}

// closeOnInitFailure closes the file and sandbox during constructor failure,
// logging warnings for any close errors.
//
// Takes ctx (context.Context) which provides logging context.
func (w *DiskWAL[K, V]) closeOnInitFailure(ctx context.Context) {
	_, l := logger_domain.From(context.WithoutCancel(ctx), log)
	if closeErr := w.file.Close(); closeErr != nil {
		l.Warn("closing WAL file during init failure", logger_domain.Error(closeErr))
	}
	if closeErr := w.sandbox.Close(); closeErr != nil {
		l.Warn("closing WAL sandbox during init failure", logger_domain.Error(closeErr))
	}
}

// commitLoop batches writes and syncs to reduce lock contention and fsync
// overhead when many writes happen at the same time. Exits when the WAL is
// closed or a fatal I/O error occurs.
//
// Takes ctx (context.Context) which provides logging and metrics context for
// background operations.
func (w *DiskWAL[K, V]) commitLoop(ctx context.Context) {
	batch := make([]pendingWrite, 0, defaultMaxBatchSize)
	timer := w.clock.NewTimer(defaultMaxBatchWait)
	timer.Stop()

	defer close(w.commitDone)
	defer goroutine.RecoverPanic(context.WithoutCancel(ctx), "wal.commitLoop")

	for {
		if w.fatalErr.Load() != nil {
			w.drainAndFailAll(&batch)
			return
		}

		select {
		case <-w.stopChan:
			w.drainPending(&batch)
			w.flushAndClearBatch(ctx, &batch, timer)
			return

		case pw := <-w.pendingChan:
			batch = append(batch, pw)
			w.drainPending(&batch)
			w.processCollectedBatch(ctx, &batch, timer)

			if w.fatalErr.Load() != nil {
				w.drainAndFailAll(&batch)
				return
			}

		case <-timer.C():
			w.flushAndClearBatch(ctx, &batch, timer)

			if w.fatalErr.Load() != nil {
				w.drainAndFailAll(&batch)
				return
			}
		}
	}
}

// processCollectedBatch decides whether to flush writes now or wait for more.
//
// Takes ctx (context.Context) which provides logging and metrics context.
// Takes batch (*[]pendingWrite) which holds the pending writes to process.
// Takes timer (clock.ChannelTimer) which controls the batch collection delay.
func (w *DiskWAL[K, V]) processCollectedBatch(ctx context.Context, batch *[]pendingWrite, timer clock.ChannelTimer) {
	if len(*batch) >= defaultMaxBatchSize || len(w.pendingChan) == 0 {
		timer.Stop()
		w.flushBatch(ctx, *batch)
		*batch = (*batch)[:0]
		return
	}

	if len(*batch) == 1 {
		timer.Reset(defaultMaxBatchWait)
	}
}

// flushAndClearBatch writes all pending items to disk and resets the batch.
//
// Takes ctx (context.Context) which provides logging and metrics context.
// Takes batch (*[]pendingWrite) which holds the pending writes to flush.
// Takes timer (clock.ChannelTimer) which is stopped before flushing.
func (w *DiskWAL[K, V]) flushAndClearBatch(ctx context.Context, batch *[]pendingWrite, timer clock.ChannelTimer) {
	if len(*batch) == 0 {
		return
	}
	timer.Stop()
	w.flushBatch(ctx, *batch)
	*batch = (*batch)[:0]
}

// drainPending moves all remaining writes from the pending channel into batch.
//
// Takes batch (*[]pendingWrite) which receives the drained writes.
func (w *DiskWAL[K, V]) drainPending(batch *[]pendingWrite) {
	for {
		select {
		case pw := <-w.pendingChan:
			*batch = append(*batch, pw)
		default:
			return
		}
	}
}

// flushBatch writes all entries in the batch and performs a single sync.
//
// Takes ctx (context.Context) which provides logging and metrics context.
// Takes batch ([]pendingWrite) which contains the entries to write.
//
// Safe for concurrent use. Holds the write lock for the full
// flush operation. On I/O failure, sets a permanent fatal error.
func (w *DiskWAL[K, V]) flushBatch(ctx context.Context, batch []pendingWrite) {
	if len(batch) == 0 {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	startTime := w.clock.Now()

	var totalBytes int64
	for _, pw := range batch {
		totalBytes += int64(len(pw.data))
	}

	var writeErr error
	if w.alignWrites {
		writeErr = w.flushAlignedLocked(batch, totalBytes)
	} else {
		writeErr = w.flushUnalignedLocked(batch, totalBytes)
	}

	if writeErr == nil {
		writeErr = w.handleSyncLocked(ctx)
	}

	if writeErr != nil {
		w.setFatalErr(writeErr)
	}

	w.updateBatchStats(ctx, startTime, len(batch), totalBytes)
	notifyBatchWaiters(batch, writeErr)
}

// flushAlignedLocked writes batch data with 4KB sector
// alignment for SSD optimisation.
//
// It prepends the partial tail sector from the previous flush,
// pads to the next 4KB boundary, and uses WriteAt at an aligned
// offset. Must be called with w.mu held.
//
// Takes batch ([]pendingWrite) which contains the entries to write.
// Takes totalBytes (int64) which is the sum of all entry data sizes.
//
// Returns error when the WriteAt operation fails.
func (w *DiskWAL[K, V]) flushAlignedLocked(batch []pendingWrite, totalBytes int64) error {
	w.alignedBuf = w.alignedBuf[:0]
	w.alignedBuf = append(w.alignedBuf, w.tailBuf...)
	for _, pw := range batch {
		w.alignedBuf = append(w.alignedBuf, pw.data...)
	}

	logicalEnd := len(w.alignedBuf)
	pad := (alignment - (logicalEnd % alignment)) % alignment
	if pad > 0 {
		var zeros [alignment]byte
		w.alignedBuf = append(w.alignedBuf, zeros[:pad]...)
	}

	writeOffset := w.nextOffset - int64(len(w.tailBuf))

	if _, err := w.file.WriteAt(w.alignedBuf, writeOffset); err != nil {
		return fmt.Errorf("writing aligned batch: %w", err)
	}

	rem := logicalEnd % alignment
	w.tailBuf = w.tailBuf[:0]
	if rem > 0 {
		w.tailBuf = append(w.tailBuf, w.alignedBuf[logicalEnd-rem:logicalEnd]...)
	}

	w.nextOffset += totalBytes
	return nil
}

// flushUnalignedLocked writes batch data directly without alignment padding.
// Must be called with w.mu held.
//
// Takes batch ([]pendingWrite) which contains the entries to write.
// Takes totalBytes (int64) which is the sum of all entry data sizes.
//
// Returns error when the WriteAt operation fails.
func (w *DiskWAL[K, V]) flushUnalignedLocked(batch []pendingWrite, totalBytes int64) error {
	w.alignedBuf = w.alignedBuf[:0]
	for _, pw := range batch {
		w.alignedBuf = append(w.alignedBuf, pw.data...)
	}

	if _, err := w.file.WriteAt(w.alignedBuf, w.nextOffset); err != nil {
		return fmt.Errorf("writing batch: %w", err)
	}

	w.nextOffset += totalBytes
	return nil
}

// updateBatchStats updates counters and records metrics after a batch flush.
//
// Takes ctx (context.Context) which provides metrics context.
// Takes startTime (time.Time) which marks when the batch operation began.
// Takes count (int) which is the number of entries flushed.
// Takes totalBytes (int64) which is the total size of the flushed data.
func (w *DiskWAL[K, V]) updateBatchStats(ctx context.Context, startTime time.Time, count int, totalBytes int64) {
	w.fileSize.Add(totalBytes)
	w.entryCount.Add(int64(count))
	w.pendingSyncs += count

	metricsCtx := context.WithoutCancel(ctx)
	wal_domain.AppendTotal.Add(metricsCtx, int64(count))
	wal_domain.AppendBytesTotal.Add(metricsCtx, totalBytes)
	wal_domain.AppendDuration.Record(metricsCtx, float64(w.clock.Now().Sub(startTime).Milliseconds()))
}

// handleSyncLocked performs a sync based on the configured sync mode.
// Must be called with w.mu held.
//
// Takes ctx (context.Context) which provides metrics context.
//
// Returns error when the sync operation fails.
func (w *DiskWAL[K, V]) handleSyncLocked(ctx context.Context) error {
	switch w.config.SyncMode {
	case wal_domain.SyncModeEveryWrite:
		return w.syncLocked(ctx)

	case wal_domain.SyncModeBatched:
		if w.pendingSyncs >= w.config.BatchSyncCount {
			return w.syncLocked(ctx)
		}

	case wal_domain.SyncModeNone:
	}

	return nil
}

// syncLocked performs fsync on the WAL file.
// Must be called with w.mu held.
//
// Takes ctx (context.Context) which provides metrics context.
//
// Returns error when the fsync operation fails.
func (w *DiskWAL[K, V]) syncLocked(ctx context.Context) error {
	startTime := w.clock.Now()

	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("fsync failed: %w", err)
	}

	w.pendingSyncs = 0
	w.lastSync = w.clock.Now()

	metricsCtx := context.WithoutCancel(ctx)
	wal_domain.SyncTotal.Add(metricsCtx, 1)
	wal_domain.SyncDuration.Record(metricsCtx, float64(w.clock.Now().Sub(startTime).Milliseconds()))

	return nil
}

// Append writes an entry to the log using group commit.
// Multiple concurrent appenders submit to a channel, and the commit goroutine
// batches writes and syncs for improved throughput.
//
// Takes entry (wal_domain.Entry) which is the log entry to append.
//
// Returns error when the WAL is closed, encoding fails, or context is
// cancelled.
func (w *DiskWAL[K, V]) Append(ctx context.Context, entry wal_domain.Entry[K, V]) error {
	if w.closed.Load() {
		return wal_domain.ErrWALClosed
	}

	if err := w.checkFatalErr(); err != nil {
		return err
	}

	result, err := w.codec.EncodeWithLengthAndCRC(entry)
	if err != nil {
		return fmt.Errorf("encoding entry: %w", err)
	}

	select {
	case <-ctx.Done():
		result.Release()
		return ctx.Err()
	default:
	}

	resultChan := getResultChan()
	pw := pendingWrite{
		data:   result.Data,
		result: resultChan,
		enc:    result,
	}

	select {
	case <-ctx.Done():
		result.Release()
		putResultChan(resultChan)
		return ctx.Err()
	case <-w.stopChan:
		result.Release()
		putResultChan(resultChan)
		return wal_domain.ErrWALClosed
	case w.pendingChan <- pw:
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-w.stopChan:
		select {
		case err := <-resultChan:
			putResultChan(resultChan)
			if err != nil {
				return fmt.Errorf("appending WAL entry: %w", err)
			}
			return nil
		default:
			return wal_domain.ErrWALClosed
		}
	case err := <-resultChan:
		putResultChan(resultChan)
		if err != nil {
			return fmt.Errorf("appending WAL entry: %w", err)
		}
		return nil
	}
}

// Recover returns an iterator over valid entries without loading
// all into memory.
//
// The iterator yields entries one at a time and handles corruption
// by stopping. When corruption is detected, the WAL is truncated
// at that point after iteration completes.
//
// Returns iter.Seq2[wal_domain.Entry[K, V], error] which yields
// each recovered entry and any error encountered during iteration.
//
// Safe for concurrent use. Access is serialised by an internal
// mutex held for the duration of iteration.
//
// IMPORTANT: The iterator holds the WAL lock during iteration.
// Callers should consume all entries promptly or break out of the
// loop to release the lock.
func (w *DiskWAL[K, V]) Recover(ctx context.Context) iter.Seq2[wal_domain.Entry[K, V], error] {
	return func(yield func(wal_domain.Entry[K, V], error) bool) {
		if w.closed.Load() {
			yield(wal_domain.Entry[K, V]{}, wal_domain.ErrWALClosed)
			return
		}

		startTime := w.clock.Now()
		w.mu.Lock()
		defer w.mu.Unlock()

		reader, err := w.prepareForRecovery()
		if err != nil {
			yield(wal_domain.Entry[K, V]{}, err)
			return
		}

		truncatePosition, entryCount := w.yieldRecoveryEntries(ctx, reader, yield)
		w.handleRecoveryCleanup(ctx, startTime, truncatePosition, entryCount)
	}
}

// prepareForRecovery seeks to the start of the WAL file for reading.
// Must be called with w.mu held.
//
// Returns *bufio.Reader which wraps the file for buffered reading.
// Returns error when seeking fails.
func (w *DiskWAL[K, V]) prepareForRecovery() (*bufio.Reader, error) {
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seeking to start: %w", err)
	}

	return bufio.NewReaderSize(w.file, readBufferSize), nil
}

// yieldRecoveryEntries streams WAL entries to the yield function during
// recovery. Must be called with w.mu held.
//
// Takes reader (*bufio.Reader) which provides the buffered input stream.
// Takes yield (func(...)) which receives each entry or error and returns
// whether to continue.
//
// Returns truncatePosition (int64) which is the position to truncate from, or -1
// if no truncation is needed.
// Returns entryCount (int) which is the number of entries read successfully.
func (w *DiskWAL[K, V]) yieldRecoveryEntries(
	ctx context.Context,
	reader *bufio.Reader,
	yield func(wal_domain.Entry[K, V], error) bool,
) (truncatePosition int64, entryCount int) {
	truncatePosition = -1
	var currentPos int64

	for {
		if ctx.Err() != nil {
			yield(wal_domain.Entry[K, V]{}, ctx.Err())
			return truncatePosition, entryCount
		}

		entry, bytesRead, shouldTruncate, err := w.readNextEntry(ctx, reader, currentPos)
		if err != nil {
			yield(wal_domain.Entry[K, V]{}, err)
			return -1, entryCount
		}

		if shouldTruncate {
			return currentPos, entryCount
		}
		if entry == nil {
			return truncatePosition, entryCount
		}

		currentPos += bytesRead
		entryCount++

		if !yield(*entry, nil) {
			return truncatePosition, entryCount
		}
	}
}

// handleRecoveryCleanup performs cleanup after recovery iteration completes.
// Must be called with w.mu held.
//
// Takes startTime (time.Time) which records when recovery began for metrics.
// Takes truncatePosition (int64) which specifies the position to truncate if
// corruption was found.
// Takes entryCount (int) which is the number of valid entries recovered.
func (w *DiskWAL[K, V]) handleRecoveryCleanup(ctx context.Context, startTime time.Time, truncatePosition int64, entryCount int) {
	w.handleTruncation(ctx, truncatePosition)

	w.entryCount.Store(int64(entryCount))
	w.nextOffset = w.fileSize.Load()
	w.rebuildTailBuf(ctx)

	w.recordRecoveryMetrics(ctx, startTime, entryCount, truncatePosition)
}

// readNextEntry reads a single entry from the reader.
//
// Takes reader (io.Reader) which provides the input data stream.
// Takes entryStart (int64) which is the byte offset of this entry.
//
// Returns *wal_domain.Entry[K, V] which is the decoded entry, or nil on
// EOF.
// Returns int64 which is the number of bytes consumed.
// Returns bool which is true when the WAL should be truncated at this
// position.
// Returns error when a non-recoverable read or decode error occurs.
func (w *DiskWAL[K, V]) readNextEntry(ctx context.Context, reader io.Reader, entryStart int64) (*wal_domain.Entry[K, V], int64, bool, error) {
	var lenBuf [lengthSize]byte
	n, err := io.ReadFull(reader, lenBuf[:])
	if err != nil {
		return w.handleLengthReadError(ctx, err, n, entryStart)
	}

	length := binary.BigEndian.Uint32(lenBuf[:])

	if length == 0 {
		_, l := logger_domain.From(ctx, log)
		l.Trace("Zero-length entry detected (alignment padding), truncating",
			logger_domain.Int64(recoverLogKey, entryStart))
		return nil, 0, true, nil
	}

	if length > maxEntrySize || length < crcSize+minPayloadSize {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Invalid entry length detected, truncating",
			logger_domain.Int64(recoverLogKey, entryStart),
			logger_domain.Int64("length", int64(length)))
		return nil, 0, true, nil
	}

	data := make([]byte, length)
	n, err = io.ReadFull(reader, data)
	if err != nil {
		return w.handleDataReadError(ctx, err, entryStart)
	}

	result, err := w.codec.DecodeWithCRC(data)
	if err != nil {
		if errors.Is(err, wal_domain.ErrCorrupted) {
			_, l := logger_domain.From(ctx, log)
			l.Warn("Corrupted entry detected, truncating",
				logger_domain.Int64(recoverLogKey, entryStart),
				logger_domain.Error(err))
			return nil, 0, true, nil
		}
		return nil, 0, false, fmt.Errorf("decoding entry: %w", err)
	}

	return &result.Entry, int64(lengthSize + n), false, nil
}

// handleLengthReadError handles errors when reading the length prefix.
//
// Takes err (error) which is the read error to handle.
// Takes bytesRead (int) which is the number of bytes read before the
// error.
// Takes entryStart (int64) which is the byte offset where the entry
// starts.
//
// Returns *wal_domain.Entry[K, V] which is always nil.
// Returns int64 which is always zero.
// Returns bool which is true when truncation is needed.
// Returns error when the read error is not EOF-related.
func (*DiskWAL[K, V]) handleLengthReadError(ctx context.Context, err error, bytesRead int, entryStart int64) (*wal_domain.Entry[K, V], int64, bool, error) {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		if bytesRead > 0 {
			_, l := logger_domain.From(ctx, log)
			l.Warn("Partial length prefix detected, truncating",
				logger_domain.Int64(recoverLogKey, entryStart))
			return nil, 0, true, nil
		}
		return nil, 0, false, nil
	}
	return nil, 0, false, fmt.Errorf("reading length prefix: %w", err)
}

// handleDataReadError handles errors when reading entry data.
//
// Takes err (error) which is the read error to handle.
// Takes entryStart (int64) which is the byte offset where the entry
// starts.
//
// Returns *wal_domain.Entry[K, V] which is always nil.
// Returns int64 which is always zero.
// Returns bool which is always true, indicating truncation is needed.
// Returns error which is always nil as truncation handles recovery.
func (*DiskWAL[K, V]) handleDataReadError(ctx context.Context, err error, entryStart int64) (*wal_domain.Entry[K, V], int64, bool, error) {
	_, l := logger_domain.From(ctx, log)
	if errors.Is(err, io.ErrUnexpectedEOF) {
		l.Warn("Partial entry detected, truncating",
			logger_domain.Int64(recoverLogKey, entryStart))
	} else {
		l.Warn("Error reading entry, truncating",
			logger_domain.Int64(recoverLogKey, entryStart),
			logger_domain.Error(err))
	}
	return nil, 0, true, nil
}

// handleTruncation performs truncation if corruption was detected.
//
// Takes truncatePosition (int64) which specifies the position to truncate at, or a
// negative value to skip truncation.
func (w *DiskWAL[K, V]) handleTruncation(ctx context.Context, truncatePosition int64) {
	if truncatePosition < 0 {
		return
	}

	ctx, l := logger_domain.From(ctx, log)
	truncatedBytes := w.fileSize.Load() - truncatePosition
	if err := w.truncateAtLocked(truncatePosition); err != nil {
		l.Error("Failed to truncate WAL after corruption",
			logger_domain.Int64(recoverLogKey, truncatePosition),
			logger_domain.Error(err))
		return
	}

	wal_domain.TruncationsTotal.Add(ctx, 1)
	wal_domain.TruncatedBytesTotal.Add(ctx, truncatedBytes)
}

// recordRecoveryMetrics records metrics after WAL recovery completes.
//
// Takes startTime (time.Time) which is the time when recovery began.
// Takes entryCount (int) which is the number of entries that were recovered.
// Takes truncatePosition (int64) which is the final position for truncation.
func (w *DiskWAL[K, V]) recordRecoveryMetrics(ctx context.Context, startTime time.Time, entryCount int, truncatePosition int64) {
	ctx, l := logger_domain.From(ctx, log)
	wal_domain.RecoveryDuration.Record(ctx, float64(w.clock.Now().Sub(startTime).Milliseconds()))
	wal_domain.RecoveryEntriesTotal.Add(ctx, int64(entryCount))

	l.Internal("WAL recovery complete",
		logger_domain.Int("entries_recovered", entryCount),
		logger_domain.Int64("truncate_position", truncatePosition))
}

// truncateAtLocked truncates the WAL file at the given position.
// Must be called with w.mu held.
//
// Takes position (int64) which specifies the byte position to truncate to.
//
// Returns error when truncation or syncing fails.
func (w *DiskWAL[K, V]) truncateAtLocked(position int64) error {
	if err := w.file.Truncate(position); err != nil {
		return fmt.Errorf("truncating file: %w", err)
	}

	w.fileSize.Store(position)
	w.nextOffset = position

	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("syncing after truncate: %w", err)
	}

	return nil
}

// Truncate removes all entries from the log.
//
// Returns error when the WAL is closed or truncation fails.
//
// Safe for concurrent use.
func (w *DiskWAL[K, V]) Truncate(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)
	if w.closed.Load() {
		return wal_domain.ErrWALClosed
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.truncateAtLocked(0); err != nil {
		return fmt.Errorf("truncating WAL: %w", err)
	}

	w.tailBuf = w.tailBuf[:0]
	w.entryCount.Store(0)
	w.pendingSyncs = 0

	l.Internal("WAL truncated")
	return nil
}

// Sync forces all data to stable storage.
//
// Returns error when the WAL is closed or the sync fails.
//
// Safe for concurrent use.
func (w *DiskWAL[K, V]) Sync(ctx context.Context) error {
	if w.closed.Load() {
		return wal_domain.ErrWALClosed
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	return w.syncLocked(ctx)
}

// Close releases resources held by the WAL.
//
// Returns error when the WAL file or sandbox cannot be closed.
//
// Safe for concurrent use. Signals the commit goroutine to stop and waits
// for it to finish before releasing resources.
func (w *DiskWAL[K, V]) Close() error {
	_, l := logger_domain.From(w.closeCtx, log)
	if w.closed.Swap(true) {
		return wal_domain.ErrWALClosed
	}

	close(w.stopChan)
	<-w.commitDone

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.pendingSyncs > 0 {
		if err := w.syncLocked(w.closeCtx); err != nil {
			l.Warn("Final sync failed on close", logger_domain.Error(err))
		}
	}

	if w.alignWrites && w.fatalErr.Load() == nil {
		if err := w.file.Truncate(w.nextOffset); err != nil {
			l.Warn("Failed to truncate padding on close", logger_domain.Error(err))
		} else if err := w.file.Sync(); err != nil {
			l.Warn("Failed to sync after truncating padding on close", logger_domain.Error(err))
		}
	}

	if err := w.file.Close(); err != nil {
		return fmt.Errorf("closing WAL file: %w", err)
	}

	if err := w.sandbox.Close(); err != nil {
		return fmt.Errorf("closing WAL sandbox: %w", err)
	}

	l.Internal("WAL closed")
	return nil
}

// EntryCount returns the number of entries currently in the WAL.
//
// Returns int which is the current entry count.
func (w *DiskWAL[K, V]) EntryCount() int {
	return int(w.entryCount.Load())
}

// Size returns the current size of the WAL file in bytes.
//
// Returns int64 which is the file size in bytes.
func (w *DiskWAL[K, V]) Size() int64 {
	return w.fileSize.Load()
}

// Path returns the path to the WAL file.
//
// Returns string which is the full file path to the write-ahead log.
func (w *DiskWAL[K, V]) Path() string {
	return filepath.Join(w.config.Dir, w.config.WALFileName)
}

// readTailSector reads the partial 4KB sector at the end of the file into
// tailBuf. Called during initialisation when the file size is not 4KB-aligned.
//
// Takes fileSize (int64) which is the total file size.
// Takes rem (int64) which is the number of bytes past the last 4KB boundary.
//
// Returns error when the read fails.
func (w *DiskWAL[K, V]) readTailSector(fileSize, rem int64) error {
	w.tailBuf = w.tailBuf[:rem]
	physicalStart := fileSize - rem
	n, err := w.file.ReadAt(w.tailBuf, physicalStart)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("reading tail at offset %d: %w", physicalStart, err)
	}
	w.tailBuf = w.tailBuf[:n]
	return nil
}

// rebuildTailBuf re-reads the partial 4KB sector from disk
// after recovery or truncation.
//
// Only applicable when alignWrites is enabled. Must be called
// with w.mu held.
func (w *DiskWAL[K, V]) rebuildTailBuf(ctx context.Context) {
	if !w.alignWrites {
		return
	}
	rem := w.nextOffset % int64(alignment)
	w.tailBuf = w.tailBuf[:0]
	if rem > 0 {
		w.tailBuf = w.tailBuf[:rem]
		physicalStart := w.nextOffset - rem
		n, err := w.file.ReadAt(w.tailBuf, physicalStart)
		if err != nil && !errors.Is(err, io.EOF) {
			_, l := logger_domain.From(ctx, log)
			l.Warn("Failed to read tail sector after recovery", logger_domain.Error(err))
			w.tailBuf = w.tailBuf[:0]
		} else {
			w.tailBuf = w.tailBuf[:n]
		}
	}
}

// setFatalErr stores a permanent I/O error.
//
// Once set, all subsequent writes are rejected with
// ErrWriterInBadState. Only the first error is stored.
//
// Takes err (error) which is the I/O error that caused the fatal state.
func (w *DiskWAL[K, V]) setFatalErr(err error) {
	w.fatalErr.CompareAndSwap(nil, &err)
}

// checkFatalErr returns ErrWriterInBadState if a fatal I/O error has been
// recorded, or nil otherwise.
//
// Returns error which is ErrWriterInBadState when the WAL is in a fatal
// state, or nil when healthy.
func (w *DiskWAL[K, V]) checkFatalErr() error {
	if w.fatalErr.Load() != nil {
		return wal_domain.ErrWriterInBadState
	}
	return nil
}

// drainAndFailAll drains the pending channel and fails all writes with
// ErrWriterInBadState. Called when the commit loop detects a fatal error.
//
// Takes batch (*[]pendingWrite) which may contain already-collected writes.
func (w *DiskWAL[K, V]) drainAndFailAll(batch *[]pendingWrite) {
	w.drainPending(batch)
	for _, pw := range *batch {
		sendResult(pw, wal_domain.ErrWriterInBadState)
		pw.enc.Release()
	}
	*batch = (*batch)[:0]
}

// Compile-time check: DiskWAL satisfies WAL.
var _ wal_domain.WAL[string, any] = (*DiskWAL[string, any])(nil)

// WithWALClock sets the clock for the WAL (for testing).
//
// Takes clk (clock.Clock) which provides the time source.
//
// Returns WALOption[K, V] which configures the WAL clock.
func WithWALClock[K comparable, V any](clk clock.Clock) WALOption[K, V] {
	return func(w *DiskWAL[K, V]) {
		w.clock = clk
	}
}

// WithSandbox sets a custom sandbox for the WAL (for testing), causing
// NewDiskWAL to use the provided sandbox and file instead of creating its
// own and enabling error injection testing.
//
// Takes sandbox (safedisk.Sandbox) which provides filesystem operations.
// Takes file (safedisk.FileHandle) which is the opened WAL file.
//
// Returns WALOption[K, V] which configures the WAL to use the provided
// sandbox and file.
//
// Note: The caller is responsible for ensuring the sandbox and file are
// properly initialised and compatible with the WAL's requirements.
func WithSandbox[K comparable, V any](sandbox safedisk.Sandbox, file safedisk.FileHandle) WALOption[K, V] {
	return func(w *DiskWAL[K, V]) {
		w.sandbox = sandbox
		w.file = file
	}
}

// NewDiskWAL creates a new disk-based WAL.
//
// The WAL file is created in config.Dir with the name specified in
// config.WALFileName (default: "data.wal").
//
// All file operations are sandboxed to the WAL directory for security.
// For testing, use WithSandbox to provide a mock sandbox.
//
// Takes ctx (context.Context) which is threaded to background goroutines
// for logging and metrics.
// Takes config (wal_domain.Config) which specifies the WAL directory and
// file settings.
// Takes codec (*BinaryCodec[K, V]) which encodes and decodes entries.
// Takes opts (...WALOption[K, V]) which provide optional configuration.
//
// Returns *DiskWAL[K, V] which is the initialised WAL ready for use.
// Returns error when the codec is nil, configuration is invalid, or
// the file cannot be opened.
//
// Safe for concurrent use. The spawned commit goroutine runs until
// Close is called.
func NewDiskWAL[K comparable, V any](
	ctx context.Context,
	config wal_domain.Config,
	codec *BinaryCodec[K, V],
	opts ...WALOption[K, V],
) (*DiskWAL[K, V], error) {
	if codec == nil {
		return nil, wal_domain.ErrCodecRequired
	}

	config = config.WithDefaults()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validating WAL config: %w", err)
	}

	wal := &DiskWAL[K, V]{
		closeCtx: context.WithoutCancel(ctx),
		codec:    codec,
		config:   config,

		clock:       clock.RealClock(),
		pendingChan: make(chan pendingWrite, defaultChannelSize),
		commitDone:  make(chan struct{}),
		stopChan:    make(chan struct{}),
	}

	for _, opt := range opts {
		opt(wal)
	}

	if wal.sandbox == nil {
		sandbox, file, err := openWALFile(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("opening WAL file: %w", err)
		}
		wal.sandbox = sandbox
		wal.file = file
	}

	if err := wal.initialiseFileState(config); err != nil {
		wal.closeOnInitFailure(ctx)
		return nil, err
	}

	go wal.commitLoop(ctx)

	_, l := logger_domain.From(context.WithoutCancel(ctx), log)
	l.Internal("WAL initialised",
		logger_domain.String("sync_mode", config.SyncMode.String()),
		logger_domain.Bool("align_writes", wal.alignWrites),
		logger_domain.Int64("initial_size", wal.fileSize.Load()))

	return wal, nil
}

// getResultChan returns a result channel from the pool.
//
// Returns chan error which yields operation results from the pool.
func getResultChan() chan error {
	resultChannel, ok := resultChanPool.Get().(chan error)
	if !ok {
		resultChannel = make(chan error, 1)
	}
	return resultChannel
}

// putResultChan returns a result channel to the pool after draining it.
// Safe to call even if the channel still has a value (it will be drained).
//
// Takes resultChannel (chan error) which is the result channel to
// return to the pool.
func putResultChan(resultChannel chan error) {
	select {
	case <-resultChannel:
	default:
	}
	resultChanPool.Put(resultChannel)
}

// openWALFile creates the sandbox and opens the WAL file.
//
// Takes ctx (context.Context) which provides the context for logging.
// Takes config (wal_domain.Config) which specifies the WAL directory and file
// name settings.
//
// Returns safedisk.Sandbox which is the created sandbox for safe file access.
// Returns safedisk.FileHandle which is the opened WAL file ready for use.
// Returns error when the sandbox cannot be created or the file cannot be
// opened.
func openWALFile(ctx context.Context, config wal_domain.Config) (safedisk.Sandbox, safedisk.FileHandle, error) {
	sandbox, err := createWALSandbox(config.Dir)
	if err != nil {
		return nil, nil, fmt.Errorf("creating WAL sandbox: %w", err)
	}

	file, err := sandbox.OpenFile(config.WALFileName, os.O_RDWR|os.O_CREATE, filePermissions)
	if err != nil {
		if closeErr := sandbox.Close(); closeErr != nil {
			_, l := logger_domain.From(context.WithoutCancel(ctx), log)
			l.Warn("closing sandbox after WAL file open failure", logger_domain.Error(closeErr))
		}
		return nil, nil, fmt.Errorf("opening WAL file: %w", err)
	}

	return sandbox, file, nil
}

// createWALSandbox creates a sandboxed filesystem for the WAL directory.
// The directory is created by safedisk when using ModeReadWrite.
//
// Takes directory (string) which specifies the path to the WAL directory.
//
// Returns safedisk.Sandbox which provides restricted filesystem access.
// Returns error when the sandbox factory or sandbox cannot be created.
func createWALSandbox(directory string) (safedisk.Sandbox, error) {
	factory, err := safedisk.NewFactory(safedisk.FactoryConfig{
		Enabled:      true,
		AllowedPaths: []string{directory},
	})
	if err != nil {
		return nil, fmt.Errorf("creating sandbox factory: %w", err)
	}

	sandbox, err := factory.Create("wal", directory, safedisk.ModeReadWrite)
	if err != nil {
		return nil, fmt.Errorf("creating sandbox: %w", err)
	}

	return sandbox, nil
}

// notifyBatchWaiters sends the result to all waiters in a batch.
// The channels are not closed because they may be pooled and reused.
//
// Takes batch ([]pendingWrite) which contains the entries to write.
// Takes err (error) which is the result to send to each waiter.
func notifyBatchWaiters(batch []pendingWrite, err error) {
	for _, pw := range batch {
		sendResult(pw, err)
		pw.enc.Release()
	}
}

// sendResult sends an error to a pending write's result channel.
//
// The send is non-blocking; if the channel is full, the result is dropped.
//
// Takes pw (pendingWrite) which holds the result channel to send to.
// Takes err (error) which is the error to send, or nil on success.
func sendResult(pw pendingWrite, err error) {
	select {
	case pw.result <- err:
	default:
	}
}
