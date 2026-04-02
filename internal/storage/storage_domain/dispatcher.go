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

package storage_domain

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/wdk/clock"
)

const (
	// defaultBatchSize is the default number of items to process in each batch.
	defaultBatchSize = 10

	// defaultFlushInterval is the time between automatic batch flushes.
	defaultFlushInterval = 30 * time.Second

	// defaultQueueSize is the default number of items the dispatcher queue can
	// hold.
	defaultQueueSize = 1000

	// defaultMaxRetries is the most times to retry a failed operation.
	defaultMaxRetries = 3

	// logFieldProvider is the log field key for the storage provider name.
	logFieldProvider = "provider"

	// logFieldBatchSize is the log field key for the batch size value.
	logFieldBatchSize = "batch_size"

	// logFieldFlushInterval is the log field key for the flush interval duration.
	logFieldFlushInterval = "flush_interval"

	// logFieldRepository is the log field key for the storage repository name.
	logFieldRepository = "repository"

	// logFieldKey is the logging field name for storage keys.
	logFieldKey = "key"

	// logFieldMaxAttempts is the log field key for the maximum retry attempts.
	logFieldMaxAttempts = "max_attempts"

	// logOpPut is the operation name for Put operations in logs and failed
	// entries.
	logOpPut = "Put"

	// logOpRemove is the operation name for storage remove operations.
	logOpRemove = "Remove"
)

// StorageDispatcher handles batched storage operations with automatic flushing.
// It implements StorageDispatcherPort and DispatcherPort, queuing operations
// for batch processing with a retry mechanism for transient errors.
type StorageDispatcher struct {
	// provider handles storage operations for Put and Remove requests.
	provider StorageProviderPort

	// clock provides time operations for recording timestamps and creating
	// tickers.
	clock clock.Clock

	// removeQueue buffers pending remove operations for batch processing.
	removeQueue chan *queuedRemove

	// putQueue buffers put operations for batch processing.
	putQueue chan *queuedPut

	// flushChan signals processing loops to flush pending operations at once.
	flushChan chan struct{}

	// shutdownChan signals when to stop processing loops and drain queues.
	shutdownChan chan struct{}

	// providerName identifies the storage provider for logging and shutdown
	// registration.
	providerName string

	// wg waits for processing loops to finish during shutdown.
	wg sync.WaitGroup

	// totalQueued is the total number of operations added to the queue.
	totalQueued int64

	// totalProcessed is the count of operations completed; accessed atomically.
	totalProcessed int64

	// totalFailed counts operations that failed permanently and were sent to the
	// DLQ.
	totalFailed int64

	// batchSize is the number of put operations to collect before processing.
	batchSize int

	// maxRetries is the maximum number of times a failed operation will be
	// retried.
	maxRetries int

	// flushInterval is the time between batch flushes; used by processing loops.
	flushInterval time.Duration

	// mu guards access to isRunning and shutdownChan.
	mu sync.RWMutex

	// isRunning indicates whether the dispatcher is actively processing.
	isRunning bool
}

var _ StorageDispatcherPort = (*StorageDispatcher)(nil)

// queuedOperation is an interface for queued operations to enable shared
// failure handling.
type queuedOperation interface {
	// getKey returns the unique identifier for this element.
	getKey() string

	// getAttempts returns the number of retry attempts made.
	//
	// Returns int which is the count of attempts.
	getAttempts() int

	// incrementAttempts adds one to the retry attempt counter.
	incrementAttempts()
}

// queuedPut represents a Put operation waiting in the queue.
// It implements queuedOperation for retry handling.
type queuedPut struct {
	// params holds the storage parameters for the put operation.
	params *storage_dto.PutParams

	// firstAttempt is when the first storage attempt was made.
	firstAttempt time.Time

	// attempts is the number of times this operation has been tried.
	attempts int
}

// getKey returns the key for this queued put operation.
//
// Returns string which is the key from the put parameters.
func (q *queuedPut) getKey() string { return q.params.Key }

// getAttempts returns the number of retry attempts made for this put.
//
// Returns int which is the current attempt count.
func (q *queuedPut) getAttempts() int { return q.attempts }

// incrementAttempts adds one to the retry counter for this queued put.
func (q *queuedPut) incrementAttempts() { q.attempts++ }

// queuedRemove represents a remove operation waiting in the queue.
// It implements the queuedOperation interface.
type queuedRemove struct {
	// firstAttempt records when the remove was first tried.
	firstAttempt time.Time

	// params holds the storage parameters for the remove operation.
	params storage_dto.GetParams

	// attempts is the number of times this removal has been tried.
	attempts int
}

// getKey returns the key of the item to be removed from the queue.
//
// Returns string which is the key identifying the item.
func (q *queuedRemove) getKey() string { return q.params.Key }

// getAttempts returns the number of removal attempts made for this item.
//
// Returns int which is the current attempt count.
func (q *queuedRemove) getAttempts() int { return q.attempts }

// incrementAttempts increases the retry counter by one.
func (q *queuedRemove) incrementAttempts() { q.attempts++ }

// DispatcherConfig holds configuration for the dispatcher.
type DispatcherConfig struct {
	// Clock provides time operations. If nil, the real system clock is used.
	Clock clock.Clock

	// FlushInterval is how often to flush pending operations; 0 uses a default.
	FlushInterval time.Duration

	// BatchSize is the number of operations to batch together.
	BatchSize int

	// QueueSize is the maximum number of pending operations; defaults to a
	// system value if zero or negative.
	QueueSize int

	// MaxRetries is the maximum number of retry attempts for a failed operation
	// before sending to the dead letter queue; negative values use the default.
	MaxRetries int
}

// NewStorageDispatcher creates a new storage dispatcher.
//
// Takes provider (StorageProviderPort) which handles the storage operations.
// Takes providerName (string) which identifies this dispatcher in logs.
// Takes config (DispatcherConfig) which specifies batch size, flush interval,
// queue size, max retries, and clock. Zero or nil values use defaults.
//
// Returns *StorageDispatcher which is ready to process storage operations.
func NewStorageDispatcher(provider StorageProviderPort, providerName string, config DispatcherConfig) *StorageDispatcher {
	if config.BatchSize <= 0 {
		config.BatchSize = defaultBatchSize
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = defaultFlushInterval
	}
	if config.QueueSize <= 0 {
		config.QueueSize = defaultQueueSize
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = defaultMaxRetries
	}
	if config.Clock == nil {
		config.Clock = clock.RealClock()
	}

	return &StorageDispatcher{
		provider:       provider,
		clock:          config.Clock,
		providerName:   providerName,
		putQueue:       make(chan *queuedPut, config.QueueSize),
		removeQueue:    make(chan *queuedRemove, config.QueueSize),
		flushChan:      make(chan struct{}, 1),
		shutdownChan:   make(chan struct{}),
		batchSize:      config.BatchSize,
		maxRetries:     config.MaxRetries,
		flushInterval:  config.FlushInterval,
		wg:             sync.WaitGroup{},
		totalQueued:    0,
		totalProcessed: 0,
		totalFailed:    0,
		mu:             sync.RWMutex{},
		isRunning:      false,
	}
}

// Start begins the dispatcher's generic processing loops for puts and removes.
//
// Returns error when the dispatcher is already running.
//
// Safe for concurrent use. Spawns two goroutines for put and remove processing
// that run until the context is cancelled.
func (d *StorageDispatcher) Start(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.isRunning {
		return fmt.Errorf("dispatcher for provider '%s' is already running", d.providerName)
	}

	d.isRunning = true
	d.wg.Add(2)

	shutdownName := fmt.Sprintf("storage-dispatcher-%s", d.providerName)
	shutdown.Register(ctx, shutdownName, d.shutdownCleanup)

	go d.runPutProcessingLoop(ctx)
	go d.runRemoveProcessingLoop(ctx)

	l.Internal("Storage dispatcher started",
		logger_domain.String(logFieldProvider, d.providerName),
		logger_domain.Int(logFieldBatchSize, d.batchSize),
		logger_domain.Duration(logFieldFlushInterval, d.flushInterval))

	return nil
}

// Stop gracefully shuts down the dispatcher, ensuring all queued items
// are processed.
//
// Returns error when shutdown fails.
//
// Safe for concurrent use. Blocks until all pending work completes.
func (d *StorageDispatcher) Stop(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	_ = ctx
	d.mu.Lock()
	if !d.isRunning {
		d.mu.Unlock()
		return nil
	}
	select {
	case <-d.shutdownChan:
	default:
		close(d.shutdownChan)
	}
	d.mu.Unlock()

	l.Internal("Stopping storage dispatcher", logger_domain.String(logFieldProvider, d.providerName))
	d.wg.Wait()

	d.mu.Lock()
	d.isRunning = false
	d.mu.Unlock()

	l.Internal("Storage dispatcher stopped", logger_domain.String(logFieldProvider, d.providerName))
	return nil
}

// QueuePut adds a Put operation to the queue.
//
// Takes params (*storage_dto.PutParams) which specifies the data to store.
// The pointer is accepted for efficiency.
//
// Returns error when the context is cancelled or the queue is full.
func (d *StorageDispatcher) QueuePut(ctx context.Context, params *storage_dto.PutParams) error {
	atomic.AddInt64(&d.totalQueued, 1)
	qp := &queuedPut{params: params, firstAttempt: d.clock.Now(), attempts: 1}

	select {
	case d.putQueue <- qp:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("queueing put operation: %w", ctx.Err())
	default:
		return fmt.Errorf("put queue is full (capacity: %d), cannot accept more operations", cap(d.putQueue))
	}
}

// QueueRemove adds a Remove operation to the queue.
//
// Takes params (storage_dto.GetParams) which identifies the item to remove.
//
// Returns error when the context is cancelled or the queue is full.
func (d *StorageDispatcher) QueueRemove(ctx context.Context, params storage_dto.GetParams) error {
	atomic.AddInt64(&d.totalQueued, 1)
	qr := &queuedRemove{params: params, firstAttempt: d.clock.Now(), attempts: 1}

	select {
	case d.removeQueue <- qr:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("queueing remove operation: %w", ctx.Err())
	default:
		return fmt.Errorf("remove queue is full (capacity: %d), cannot accept more operations", cap(d.removeQueue))
	}
}

// Flush forces an immediate flush of all queued operations.
//
// Returns error when the context is cancelled before the flush signal is sent.
func (d *StorageDispatcher) Flush(ctx context.Context) error {
	select {
	case d.flushChan <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("flushing dispatcher: %w", ctx.Err())
	default:
		return nil
	}
}

// GetStats returns a snapshot of the dispatcher's current statistics.
//
// Returns DispatcherStats which contains queue depth and operation counts.
func (d *StorageDispatcher) GetStats() DispatcherStats {
	return DispatcherStats{
		QueuedOperations: int64(len(d.putQueue) + len(d.removeQueue)),
		TotalQueued:      atomic.LoadInt64(&d.totalQueued),
		TotalProcessed:   atomic.LoadInt64(&d.totalProcessed),
		TotalFailed:      atomic.LoadInt64(&d.totalFailed),
	}
}

// SetBatchSize dynamically updates the batch size for storage operations.
//
// Takes size (int) which specifies the new batch size; values less than one
// are ignored.
//
// Safe for concurrent use.
func (d *StorageDispatcher) SetBatchSize(size int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if size > 0 {
		d.batchSize = size
	}
}

// SetFlushInterval dynamically updates the flush interval for pending
// operations.
//
// Takes interval (time.Duration) which specifies the new flush interval.
// Intervals of zero or less are ignored.
//
// Safe for concurrent use.
func (d *StorageDispatcher) SetFlushInterval(interval time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if interval > 0 {
		d.flushInterval = interval
	}
}

// DispatcherStats holds dispatcher statistics for monitoring.
type DispatcherStats struct {
	// QueuedOperations is the number of operations waiting in the queue.
	QueuedOperations int64

	// TotalQueued is the total number of operations queued since start.
	TotalQueued int64

	// TotalProcessed is the total number of operations processed, including
	// retries.
	TotalProcessed int64

	// TotalFailed is the count of operations that failed permanently and went to
	// DLQ.
	TotalFailed int64
}

// runProcessingLoop is the shared batch processing loop used by
// runPutProcessingLoop and runRemoveProcessingLoop. It reads from queue,
// accumulates items into batches up to batchSize, and flushes on tick, explicit
// flush signal, shutdown, or context cancellation.
//
// Takes queue (chan *T) which supplies incoming operations.
// Takes processBatch (func(context.Context, []*T)) which processes a full batch.
// Takes drain (func(context.Context)) which drains remaining queue items on exit.
// Takes panicLabel (string) which identifies this loop in panic recovery logs.
func runProcessingLoop[T any](
	ctx context.Context,
	d *StorageDispatcher,
	queue chan *T,
	processBatch func(context.Context, []*T),
	drain func(context.Context),
	panicLabel string,
) {
	ctx, _ = logger_domain.From(ctx, log)
	defer d.wg.Done()
	defer goroutine.RecoverPanic(ctx, panicLabel)
	ticker := d.clock.NewTicker(d.flushInterval)
	defer ticker.Stop()

	batch := make([]*T, 0, d.batchSize)

	processAndReset := func() {
		if len(batch) > 0 {
			processBatch(ctx, batch)
			batch = make([]*T, 0, d.batchSize)
		}
	}

	for {
		select {
		case operation := <-queue:
			batch = append(batch, operation)
			if len(batch) >= d.batchSize {
				processAndReset()
			}
		case <-ticker.C():
			processAndReset()
		case <-d.flushChan:
			processAndReset()
		case <-d.shutdownChan:
			processAndReset()
			drain(ctx)
			return
		case <-ctx.Done():
			processAndReset()
			drain(ctx)
			return
		}
	}
}

// runPutProcessingLoop runs the main loop for Put operations, batching them
// for processing at regular intervals or when the batch is full.
func (d *StorageDispatcher) runPutProcessingLoop(ctx context.Context) {
	runProcessingLoop(ctx, d, d.putQueue, d.processPutBatch, d.drainPutQueue, "storage.runPutProcessingLoop")
}

// runRemoveProcessingLoop handles the processing loop for remove operations.
func (d *StorageDispatcher) runRemoveProcessingLoop(ctx context.Context) {
	runProcessingLoop(ctx, d, d.removeQueue, d.processRemoveBatch, d.drainRemoveQueue, "storage.runRemoveProcessingLoop")
}

// drainPutQueue processes all remaining Put items in the queue.
func (d *StorageDispatcher) drainPutQueue(ctx context.Context) {
	batch := make([]*queuedPut, 0, d.batchSize)
	for {
		select {
		case operation := <-d.putQueue:
			batch = append(batch, operation)
			if len(batch) >= d.batchSize {
				d.processPutBatch(ctx, batch)
				batch = make([]*queuedPut, 0, d.batchSize)
			}
		default:
			if len(batch) > 0 {
				d.processPutBatch(ctx, batch)
			}
			return
		}
	}
}

// drainRemoveQueue processes all remaining remove items in the queue.
func (d *StorageDispatcher) drainRemoveQueue(ctx context.Context) {
	batch := make([]*queuedRemove, 0, d.batchSize)
	for {
		select {
		case operation := <-d.removeQueue:
			batch = append(batch, operation)
			if len(batch) >= d.batchSize {
				d.processRemoveBatch(ctx, batch)
				batch = make([]*queuedRemove, 0, d.batchSize)
			}
		default:
			if len(batch) > 0 {
				d.processRemoveBatch(ctx, batch)
			}
			return
		}
	}
}

// processPutBatch processes a batch of Put operations with retry logic.
//
// Takes batch ([]*queuedPut) which contains the Put operations to process.
func (d *StorageDispatcher) processPutBatch(ctx context.Context, batch []*queuedPut) {
	for _, operation := range batch {
		if ctx.Err() != nil {
			return
		}
		err := d.provider.Put(ctx, operation.params)
		atomic.AddInt64(&d.totalProcessed, 1)

		if err == nil {
			continue
		}

		d.handlePutFailure(ctx, operation, err)
	}
}

// handlePutFailure handles a failed Put operation by retrying or sending to
// the dead letter queue.
//
// Takes operation (*queuedPut) which is the failed Put operation to handle.
// Takes err (error) which is the error that caused the failure.
func (d *StorageDispatcher) handlePutFailure(ctx context.Context, operation *queuedPut, err error) {
	d.handleOperationFailure(ctx, operation, logOpPut, err,
		func() bool { return d.tryRequeuePut(ctx, operation, err) },
		func() { d.logPutFailure(ctx, operation, err) })
}

// processRemoveBatch processes a batch of remove operations, with internal
// retry logic.
//
// Takes batch ([]*queuedRemove) which contains the remove operations to run.
func (d *StorageDispatcher) processRemoveBatch(ctx context.Context, batch []*queuedRemove) {
	for _, operation := range batch {
		if ctx.Err() != nil {
			return
		}
		err := d.provider.Remove(ctx, operation.params)
		atomic.AddInt64(&d.totalProcessed, 1)

		if err == nil {
			continue
		}

		d.handleRemoveFailure(ctx, operation, err)
	}
}

// handleRemoveFailure handles a failed remove operation by retrying or sending
// to the dead letter queue.
//
// Takes operation (*queuedRemove) which is the failed remove operation.
// Takes err (error) which is the error that caused the failure.
func (d *StorageDispatcher) handleRemoveFailure(ctx context.Context, operation *queuedRemove, err error) {
	d.handleOperationFailure(ctx, operation, logOpRemove, err,
		func() bool { return d.tryRequeueRemove(ctx, operation, err) },
		func() { d.logRemoveFailure(ctx, operation, err) })
}

// tryRequeue is the shared requeue implementation used by tryRequeuePut and
// tryRequeueRemove. It attempts a non-blocking send to queue, and logs failure
// on shutdown or context cancellation.
//
// Takes queue (chan *T) which is the destination channel.
// Takes key (string) which identifies the operation for logging.
// Takes shutdownChan (chan struct{}) which signals dispatcher shutdown.
// Takes logFailure (func()) which logs permanent failure when requeue is impossible.
//
// Returns bool which is true if the operation was successfully requeued.
func tryRequeue[T any](
	ctx context.Context,
	queue chan *T,
	operation *T,
	key string,
	shutdownChan chan struct{},
	logFailure func(),
) bool {
	_, l := logger_domain.From(ctx, log)
	select {
	case queue <- operation:
		return true
	case <-shutdownChan:
		l.Warn("Dispatcher shutting down, cannot re-queue failed operation. ", logger_domain.String(logFieldKey, key))
		logFailure()
		return false
	case <-ctx.Done():
		l.Warn("Context cancelled, cannot re-queue failed operation. ", logger_domain.String(logFieldKey, key))
		logFailure()
		return false
	}
}

// tryRequeuePut attempts to re-queue a failed Put operation.
//
// Takes operation (*queuedPut) which is the operation to re-queue.
// Takes err (error) which is the original error that caused the failure.
//
// Returns bool which is true if the operation was re-queued successfully.
func (d *StorageDispatcher) tryRequeuePut(ctx context.Context, operation *queuedPut, err error) bool {
	return tryRequeue(ctx, d.putQueue, operation, operation.params.Key, d.shutdownChan,
		func() { d.logPutFailure(ctx, operation, err) })
}

// tryRequeueRemove attempts to re-queue a failed Remove operation.
//
// Takes operation (*queuedRemove) which is the failed operation to re-queue.
// Takes err (error) which is the original error that caused the failure.
//
// Returns bool which is true if requeued successfully, or false if the
// dispatcher is shutting down or context is cancelled.
func (d *StorageDispatcher) tryRequeueRemove(ctx context.Context, operation *queuedRemove, err error) bool {
	return tryRequeue(ctx, d.removeQueue, operation, operation.params.Key, d.shutdownChan,
		func() { d.logRemoveFailure(ctx, operation, err) })
}

// logPutFailure logs a Put operation that has failed after all retries.
//
// Takes operation (*queuedPut) which holds the failed Put operation details.
// Takes err (error) which is the error that caused the failure.
func (d *StorageDispatcher) logPutFailure(ctx context.Context, operation *queuedPut, err error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Error("Put operation failed permanently after all retries",
		logger_domain.String(logFieldProvider, d.providerName),
		logger_domain.String(logFieldRepository, operation.params.Repository),
		logger_domain.String(logFieldKey, operation.params.Key),
		logger_domain.String("content_type", operation.params.ContentType),
		logger_domain.Int("size", int(operation.params.Size)),
		logger_domain.Int(logFieldAttempt, operation.attempts),
		logger_domain.Error(err))
}

// logRemoveFailure logs a remove operation that has failed permanently.
//
// Takes operation (*queuedRemove) which contains the failed remove operation.
// Takes err (error) which is the error that caused the failure.
func (d *StorageDispatcher) logRemoveFailure(ctx context.Context, operation *queuedRemove, err error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Error("Remove operation failed permanently after all retries",
		logger_domain.String(logFieldProvider, d.providerName),
		logger_domain.String(logFieldRepository, operation.params.Repository),
		logger_domain.String(logFieldKey, operation.params.Key),
		logger_domain.Int(logFieldAttempt, operation.attempts),
		logger_domain.Error(err))
}

// shutdownCleanup is called by the global shutdown handler to stop the
// storage dispatcher.
//
// Returns error when the storage dispatcher fails to stop.
func (d *StorageDispatcher) shutdownCleanup(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Executing storage dispatcher shutdown cleanup", logger_domain.String(logFieldProvider, d.providerName))

	shutdownCtx, cancel := context.WithTimeoutCause(context.Background(), 30*time.Second,
		errors.New("storage dispatcher shutdown exceeded 30s timeout"))
	defer cancel()

	if err := d.Stop(shutdownCtx); err != nil {
		return fmt.Errorf("stopping dispatcher during shutdown cleanup for provider %q: %w", d.providerName, err)
	}
	return nil
}

// handleOperationFailure handles a failed operation by retrying or logging.
//
// Takes operation (queuedOperation) which is the failed operation to handle.
// Takes operationName (string) which names the operation type for logging.
// Takes err (error) which is the failure cause to check for retry.
// Takes requeue (func(...)) which puts the operation back in the queue.
// Takes logFailure (func(...)) which logs when all retries are used.
func (d *StorageDispatcher) handleOperationFailure(
	ctx context.Context, operation queuedOperation,
	operationName string, err error, requeue func() bool, logFailure func(),
) {
	ctx, l := logger_domain.From(ctx, log)
	if operation.getAttempts() < d.maxRetries && IsRetryableError(err) {
		operation.incrementAttempts()
		l.Warn("Dispatcher "+operationName+" operation failed, will retry",
			logger_domain.String(logFieldProvider, d.providerName),
			logger_domain.String(logFieldKey, operation.getKey()),
			logger_domain.Int(logFieldAttempt, operation.getAttempts()),
			logger_domain.Int(logFieldMaxAttempts, d.maxRetries),
			logger_domain.Error(err))

		if !requeue() {
			return
		}
		return
	}

	atomic.AddInt64(&d.totalFailed, 1)
	l.Error("Dispatcher "+operationName+" operation failed permanently after all retries",
		logger_domain.String(logFieldProvider, d.providerName),
		logger_domain.String(logFieldKey, operation.getKey()),
		logger_domain.Int(logFieldAttempt, operation.getAttempts()),
		logger_domain.Error(err))
	logFailure()
}

// DefaultDispatcherConfig returns sensible defaults for dispatcher settings.
//
// Returns DispatcherConfig which contains default values for batch size, flush
// interval, queue size, and maximum retries.
func DefaultDispatcherConfig() DispatcherConfig {
	return DispatcherConfig{
		BatchSize:     defaultBatchSize,
		FlushInterval: defaultFlushInterval,
		QueueSize:     defaultQueueSize,
		MaxRetries:    defaultMaxRetries,
	}
}
