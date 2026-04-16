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

package analytics_domain

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sony/gobreaker/v2"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/retry"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// circuitBreakerBucketPeriod is the measurement bucket duration,
	// matching the standard used across the project.
	circuitBreakerBucketPeriod = 10 * time.Second

	// logFieldBatcher is the logging field name for batcher identification.
	logFieldBatcher = "batcher"
)

// analyticsErrorClassifier determines whether a batch send error is
// retryable. The default classifier handles network failures and
// server errors; the added pattern retries circuit breaker rejections.
var analyticsErrorClassifier = retry.NewErrorClassifier(
	retry.WithRetryablePatterns(
		"circuit breaker open",
	),
)

// BatchSendFunc is called by the Batcher when a batch is ready to be
// sent.
//
// The implementation owns encoding and I/O. The batch slice is only
// valid for the duration of the call; the caller must not retain it.
type BatchSendFunc[T any] func(ctx context.Context, batch []T) error

// BatcherConfig provides the settings for a Batcher.
type BatcherConfig struct {
	// Retry configures exponential backoff for failed batch sends.
	// A nil value or zero MaxRetries disables retry.
	Retry *retry.Config

	// CircuitBreaker configures the circuit breaker that protects
	// the send path. A nil value disables the circuit breaker.
	CircuitBreaker *CircuitBreakerConfig

	// Clock provides time operations for tickers and timers.
	// Defaults to clock.RealClock() when nil, enabling
	// deterministic testing with clock.MockClock.
	Clock clock.Clock

	// Name identifies this batcher in logs and circuit breaker
	// metrics (e.g. "analytics-webhook", "analytics-ga4").
	Name string

	// FlushInterval is the time between automatic timer-based
	// flushes. Must be > 0.
	FlushInterval time.Duration

	// BatchSize is the number of items that triggers an immediate
	// flush signal. Must be > 0.
	BatchSize int
}

// CircuitBreakerConfig holds settings for a batcher's circuit
// breaker.
type CircuitBreakerConfig struct {
	// Timeout is how long the circuit stays open before
	// transitioning to half-open and allowing a probe request.
	Timeout time.Duration

	// Interval is the cyclic period of the closed state for the
	// circuit breaker to clear the internal counts. If 0, the
	// circuit breaker doesn't clear internal counts during the
	// closed state.
	Interval time.Duration

	// MaxConsecutiveFailures is the number of consecutive send
	// failures that trips the circuit breaker to open.
	MaxConsecutiveFailures int
}

// Batcher accumulates items of type T and periodically flushes them
// via a caller-supplied send function. It manages the buffer, flush
// loop, concurrency, graceful shutdown, retry with exponential
// backoff, and circuit breaking so that collectors only need to
// provide conversion and send logic.
//
// Safe for concurrent use from multiple goroutines.
type Batcher[T any] struct {
	// sendFunc is called with the accumulated batch on each flush.
	sendFunc BatchSendFunc[T]

	// circuitBreaker protects the send path. Nil when circuit
	// breaking is disabled.
	circuitBreaker *gobreaker.CircuitBreaker[any]

	// retryConfig holds exponential backoff settings. Nil when retry
	// is disabled.
	retryConfig *retry.Config

	// circuitBreakerConfig is stored during construction and used
	// in Start to create the circuit breaker with the detached ctx.
	circuitBreakerConfig *CircuitBreakerConfig

	// clock provides time operations (tickers, timers). Injected
	// via BatcherConfig for testability; defaults to RealClock.
	clock clock.Clock

	// batchPool recycles batch slices to avoid allocating on every
	// flush cycle under high throughput.
	batchPool sync.Pool

	// stopCh signals the flush goroutine to exit.
	stopCh chan struct{}

	// doneCh is closed when the flush goroutine exits.
	doneCh chan struct{}

	// flushCh is a non-blocking signal that tells the flush
	// goroutine to flush immediately because the buffer reached
	// batchSize.
	flushCh chan struct{}

	// name identifies this batcher in logs and metrics.
	name string

	// buffer accumulates items until the batch is flushed.
	buffer []T

	// flushInterval is the time between automatic timer-based
	// flushes.
	flushInterval time.Duration

	// batchSize is the number of items that triggers an immediate
	// flush signal.
	batchSize int

	// closeOnce ensures Close is idempotent.
	closeOnce sync.Once

	// mu guards buffer access from concurrent Add and flushLoop
	// calls.
	mu sync.Mutex

	// stopped is set during Close before closing stopCh. Checked by
	// Add to prevent silent buffer leaks after shutdown.
	stopped atomic.Bool

	// started tracks whether Start has been called. Close is a
	// no-op if the flush loop was never started.
	started atomic.Bool
}

// NewBatcher creates a Batcher with the given configuration and send
// function. The Batcher is not running until Start is called.
//
// Takes config (BatcherConfig) which provides batch size, flush
// interval, and optional retry and circuit breaker settings.
// Takes sendFunc (BatchSendFunc[T]) which is called with each batch.
//
// Returns *Batcher[T] which must be started via Start.
// Returns error when the configuration is invalid.
func NewBatcher[T any](config BatcherConfig, sendFunc BatchSendFunc[T]) (*Batcher[T], error) {
	if config.BatchSize <= 0 {
		return nil, errors.New("analytics: BatchSize must be > 0")
	}
	if config.FlushInterval <= 0 {
		return nil, errors.New("analytics: FlushInterval must be > 0")
	}

	batchSize := config.BatchSize
	b := &Batcher[T]{
		sendFunc:      sendFunc,
		name:          config.Name,
		batchSize:     batchSize,
		flushInterval: config.FlushInterval,
		retryConfig:   config.Retry,
		batchPool: sync.Pool{
			New: func() any {
				s := make([]T, 0, batchSize)
				return &s
			},
		},
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
		flushCh: make(chan struct{}, 1),
	}
	b.buffer = make([]T, 0, b.batchSize)
	b.circuitBreakerConfig = config.CircuitBreaker
	b.clock = config.Clock
	if b.clock == nil {
		b.clock = clock.RealClock()
	}

	return b, nil
}

// Start launches the background flush loop goroutine.
func (b *Batcher[T]) Start(ctx context.Context) {
	b.started.Store(true)
	backgroundCtx := context.WithoutCancel(ctx)

	if b.circuitBreakerConfig != nil {
		b.circuitBreaker = newBatcherCircuitBreaker(backgroundCtx, b.name, b.circuitBreakerConfig)
	}

	go b.flushLoop(backgroundCtx)
}

// Add appends an item to the buffer.
//
// When the buffer reaches batchSize the flush goroutine is signalled
// to send the batch asynchronously; Add itself never performs I/O.
//
// Takes item (T) which is the item to buffer.
//
// Concurrency: acquires b.mu briefly to append the item.
func (b *Batcher[T]) Add(item T) {
	if b.stopped.Load() {
		return
	}

	b.mu.Lock()
	b.buffer = append(b.buffer, item)
	full := len(b.buffer) >= b.batchSize
	b.mu.Unlock()

	if full {
		select {
		case b.flushCh <- struct{}{}:
		default:
		}
	}
}

// Flush sends any buffered items via the send function. The mutex
// is only held while copying the buffer; HTTP I/O and retries run
// without the lock so that Add is never blocked by network latency.
//
// Returns error when the send function fails.
func (b *Batcher[T]) Flush(ctx context.Context) error {
	batch := b.drainBuffer()
	if batch == nil {
		return nil
	}
	err := b.sendWithResilience(ctx, *batch)
	b.releaseBatch(batch)
	return err
}

// Close stops the flush timer and waits for the flush goroutine to
// exit.
//
// Any remaining buffered items should be flushed via Flush before
// calling Close. Safe to call multiple times.
//
// Returns error which is always nil.
func (b *Batcher[T]) Close() error {
	if !b.started.Load() {
		return nil
	}
	b.stopped.Store(true)
	b.closeOnce.Do(func() {
		close(b.stopCh)
	})
	<-b.doneCh
	return nil
}

// flushLoop runs a periodic timer that flushes buffered items.
//
// It also listens for immediate flush signals when the buffer reaches
// batchSize. The backgroundCtx carries context values (logger, trace
// spans) without cancellation.
func (b *Batcher[T]) flushLoop(backgroundCtx context.Context) {
	defer close(b.doneCh)
	defer goroutine.RecoverPanic(backgroundCtx, "analytics.batcher."+b.name+".flushLoop")
	backgroundCtx, l := logger_domain.From(backgroundCtx, log)
	ticker := b.clock.NewTicker(b.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C():
			if err := b.Flush(backgroundCtx); err != nil {
				l.Warn("Analytics batcher periodic flush failed",
					logger_domain.String(logFieldBatcher, b.name),
					logger_domain.Error(err))
			}
		case <-b.flushCh:
			if err := b.Flush(backgroundCtx); err != nil {
				l.Warn("Analytics batcher signal flush failed",
					logger_domain.String(logFieldBatcher, b.name),
					logger_domain.Error(err))
			}
		case <-b.stopCh:
			return
		}
	}
}

// drainBuffer copies the buffered items into a pooled slice and
// clears the buffer.
//
// Returns *[]T which is the pooled batch, or nil when the buffer
// is empty.
//
// Concurrency: acquires b.mu for the duration of the copy. The lock
// is released before any subsequent I/O.
func (b *Batcher[T]) drainBuffer() *[]T {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.buffer) == 0 {
		return nil
	}

	batch, ok := b.batchPool.Get().(*[]T)
	if !ok {
		fresh := make([]T, 0, b.batchSize)
		batch = &fresh
	}
	*batch = append((*batch)[:0], b.buffer...)
	b.buffer = b.buffer[:0]
	return batch
}

// releaseBatch returns a batch slice to the pool after the send
// completes.
//
// Takes batch (*[]T) which is the pooled slice to return.
func (b *Batcher[T]) releaseBatch(batch *[]T) {
	*batch = (*batch)[:0]
	b.batchPool.Put(batch)
}

// sendWithResilience wraps the send function with circuit breaker
// and retry logic. When neither is configured, it calls sendFunc
// directly.
//
// Takes batch ([]T) which is the batch to send.
//
// Returns error when the send fails after all resilience attempts.
func (b *Batcher[T]) sendWithResilience(ctx context.Context, batch []T) error {
	operation := func() error {
		return b.sendFunc(ctx, batch)
	}

	if b.circuitBreaker != nil {
		original := operation
		operation = func() error {
			_, err := b.circuitBreaker.Execute(func() (any, error) {
				return nil, original()
			})
			return err
		}
	}

	if b.retryConfig != nil && b.retryConfig.MaxRetries > 0 {
		return b.executeWithRetry(ctx, operation)
	}

	return operation()
}

// executeWithRetry runs the operation with exponential backoff. Only
// retryable errors (network failures, 5xx) trigger a retry;
// permanent errors (auth, context cancellation) fail immediately.
//
// Takes operation (func() error) which is the send operation to retry.
//
// Returns error when all retry attempts are exhausted.
func (b *Batcher[T]) executeWithRetry(ctx context.Context, operation func() error) error {
	ctx, l := logger_domain.From(ctx, log)
	var lastError error

	for attempt := 0; b.retryConfig.ShouldRetry(attempt); attempt++ {
		lastError = operation()
		if lastError == nil {
			return nil
		}

		if !analyticsErrorClassifier.IsRetryable(lastError) {
			return lastError
		}

		if attempt < b.retryConfig.MaxRetries {
			batcherRetriesCount.Add(ctx, 1)

			l.Warn("Analytics batch send failed, retrying",
				logger_domain.String(logFieldBatcher, b.name),
				logger_domain.Int("attempt", attempt+1),
				logger_domain.Int("max_retries", b.retryConfig.MaxRetries),
				logger_domain.Error(lastError))

			nextRetry := b.retryConfig.CalculateNextRetry(attempt, time.Now())
			delay := time.Until(nextRetry)

			retryTimer := time.NewTimer(delay)
			select {
			case <-retryTimer.C:
			case <-ctx.Done():
				retryTimer.Stop()
				return fmt.Errorf("retry cancelled: %w", ctx.Err())
			case <-b.stopCh:
				retryTimer.Stop()
				return errors.New("retry aborted: batcher closing")
			}
		}
	}

	return lastError
}

// newBatcherCircuitBreaker creates a gobreaker circuit breaker with
// the standard project conventions: 1 request in half-open,
// consecutive failure threshold, state change logging, and context
// error exclusion.
//
// Takes name (string) which identifies the batcher in logs.
// Takes config (*CircuitBreakerConfig) which provides the settings.
//
// Returns *gobreaker.CircuitBreaker[any] which is the configured
// breaker.
func newBatcherCircuitBreaker(ctx context.Context, name string, config *CircuitBreakerConfig) *gobreaker.CircuitBreaker[any] {
	ctx, l := logger_domain.From(ctx, log)
	settings := gobreaker.Settings{
		Name:         fmt.Sprintf("analytics-batcher-%s", name),
		MaxRequests:  1,
		Interval:     config.Interval,
		Timeout:      config.Timeout,
		BucketPeriod: circuitBreakerBucketPeriod,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= safeconv.IntToUint32(config.MaxConsecutiveFailures)
		},
		OnStateChange: func(breakerName string, from gobreaker.State, to gobreaker.State) {
			l.Warn("Analytics batcher circuit breaker state changed",
				logger_domain.String(logFieldBatcher, breakerName),
				logger_domain.String("from_state", from.String()),
				logger_domain.String("to_state", to.String()))

			if to == gobreaker.StateOpen {
				batcherCircuitOpenCount.Add(ctx, 1)
			}
		},
		IsExcluded: func(err error) bool {
			return errors.Is(err, context.Canceled) ||
				errors.Is(err, context.DeadlineExceeded)
		},
	}
	return gobreaker.NewCircuitBreaker[any](settings)
}
