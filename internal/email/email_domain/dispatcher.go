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

package email_domain

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sony/gobreaker/v2"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/retry"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// defaultBatchSize is the number of emails to handle in one batch.
	defaultBatchSize = 10

	// defaultFlushInterval is the default time between queue flushes.
	defaultFlushInterval = 30 * time.Second

	// defaultQueueSize is the default number of emails that can be queued.
	defaultQueueSize = 1000

	// defaultRetryQueueSize is the default size of the retry queue.
	defaultRetryQueueSize = 500

	// defaultMaxRetries is the default number of retry attempts for failed emails.
	defaultMaxRetries = 3

	// defaultInitialDelay is the starting wait time before retrying a failed
	// email.
	defaultInitialDelay = 5 * time.Second

	// defaultMaxDelay is the maximum wait time between retry attempts.
	defaultMaxDelay = 5 * time.Minute

	// defaultBackoffFactor is the multiplier for retry delays in exponential
	// backoff.
	defaultBackoffFactor = 2.0

	// defaultMaxRetryHeapSize is the largest number of items the retry heap can
	// hold.
	defaultMaxRetryHeapSize = 50000

	// emptyQueueSleep is the duration to sleep when the retry queue is empty.
	emptyQueueSleep = 10 * time.Minute

	// circuitBreakerBucketPeriod is the duration of each measurement bucket
	// for tracking failure counts.
	circuitBreakerBucketPeriod = 10 * time.Second

	// fmtListRecipients is the format string for logging email recipients.
	fmtListRecipients = "%v"

	// logFieldBatchSize is the log field key for batch size values.
	logFieldBatchSize = "batch_size"

	// logFieldFlushInterval is the log field key for flush interval.
	logFieldFlushInterval = "flush_interval"

	// logFieldMaxRetries is the log field key for maximum retry attempts.
	logFieldMaxRetries = "max_retries"

	// logFieldInitialDelay is the log field key for retry initial delay.
	logFieldInitialDelay = "initial_delay"

	// logFieldDeadLetterQueue is the log field key for the dead letter queue
	// setting.
	logFieldDeadLetterQueue = "dead_letter_queue"

	// logFieldCircuitBreaker is the log field key for circuit breaker status.
	logFieldCircuitBreaker = "circuit_breaker_enabled"

	// logFieldRecipients is the log field key for email recipients.
	logFieldRecipients = "recipients"

	// logFieldSubject is the log field key for the email subject.
	logFieldSubject = "subject"

	// logFieldAttempt is the log field key for the retry attempt number.
	logFieldAttempt = "attempt"

	// logFieldNextRetry is the log field key for when the next retry will happen.
	logFieldNextRetry = "next_retry"

	// logFieldError is the log field key for error messages.
	logFieldError = "error"

	// logFieldTotalAttempts is the log field key for the total number of attempts.
	logFieldTotalAttempts = "total_attempts"

	// logFieldItems is the log field key for the number of items.
	logFieldItems = "items"

	// logFieldRetryWorkerID is the log field key for the retry worker identifier.
	logFieldRetryWorkerID = "retry_worker_id"

	// logFieldRetryWorkerCount is the log field key for the number of retry
	// workers.
	logFieldRetryWorkerCount = "retry_worker_count"
)

// retryAction represents the result of waiting for a retry event.
type retryAction int

const (
	// retryActionProcess signals that items ready to send should be processed.
	retryActionProcess retryAction = iota

	// retryActionContinue tells the retry loop to keep processing.
	retryActionContinue

	// retryActionStop signals that the retry loop should exit.
	retryActionStop
)

// deadLetterContext holds details about an email that has failed for good and
// cannot be sent.
type deadLetterContext struct {
	// firstAttempt is the time of the first delivery attempt.
	firstAttempt time.Time

	// lastAttempt is when the last delivery attempt was made.
	lastAttempt time.Time

	// originalError is the error that caused the email to fail permanently.
	originalError error

	// email holds the original email that failed to send.
	email email_dto.SendParams

	// totalAttempts is the number of delivery attempts made for this email.
	totalAttempts int
}

// failureMeta holds data about a failed email send attempt for retry handling.
type failureMeta struct {
	// firstAttempt is the timestamp of the first send attempt for this email;
	// zero value means this is the first attempt.
	firstAttempt time.Time

	// sendErr is the error returned when the send attempt failed.
	sendErr error

	// email holds the email data for the failed send attempt.
	email *email_dto.SendParams

	// attempt is the current retry attempt number for this email.
	attempt int
}

// EmailDispatcher implements EmailDispatcherPort and provides batched email
// sending with retry, dead-letter queue, and circuit breaker capabilities.
type EmailDispatcher struct {
	// clock provides time operations for scheduling and timestamps.
	clock clock.Clock

	// startTime records when the dispatcher was started; zero value means not
	// running.
	startTime time.Time

	// deadLetterQueue stores emails that failed after all retry attempts.
	deadLetterQueue DeadLetterPort

	// provider is the email service used to send messages.
	provider EmailProviderPort

	// circuitBreaker provides fault tolerance by stopping calls to the email
	// provider when repeated failures occur; nil disables circuit breaking.
	circuitBreaker *gobreaker.CircuitBreaker[any]

	// queue holds emails waiting to be batched and sent by the processing loop.
	queue chan *email_dto.SendParams

	// retryHeap stores items that are waiting to be retried, sorted by retry time.
	retryHeap *retry.Heap[*retryItem]

	// retryJobsChan carries items that are ready to be retried.
	retryJobsChan chan *retryItem

	// flushChan signals a request to drain the queue straight away.
	flushChan chan struct{}

	// retrySignal tells the producer loop to check its sleep timer again.
	retrySignal chan struct{}

	// shutdownChan signals all goroutines to stop processing.
	shutdownChan chan struct{}

	// shutdownName is the name used to register the graceful shutdown handler.
	shutdownName string

	// retryConfig holds settings for retry behaviour including maximum attempts,
	// initial delay between retries, and dead letter queue handling.
	retryConfig RetryConfig

	// wg tracks goroutines to ensure clean shutdown.
	wg sync.WaitGroup

	// flushInterval is the time between automatic batch flushes; must be positive.
	flushInterval time.Duration

	// batchSize is the maximum number of emails to collect before sending.
	batchSize int

	// retryWorkerCount is the number of parallel goroutines that process
	// retries.
	retryWorkerCount int

	// maxRetryHeapSize is the maximum number of emails allowed in the retry heap.
	maxRetryHeapSize int

	// totalProcessed counts all emails handled, both successful and failed.
	totalProcessed int64

	// totalSuccessful counts emails sent without error; updated atomically.
	totalSuccessful int64

	// totalFailed counts emails that failed permanently; updated atomically.
	totalFailed int64

	// totalRetries is the total number of retry attempts; updated atomically.
	totalRetries int64

	// mu guards access to isRunning and other mutable dispatcher state.
	mu sync.RWMutex

	// retryMutex guards access to retryHeap.
	retryMutex sync.Mutex

	// isRunning indicates whether the dispatcher is currently active.
	isRunning bool
}

// NewEmailDispatcher creates a new EmailDispatcher with the given provider,
// dead letter adapter, and configuration.
//
// Takes provider (EmailProviderPort) which handles sending emails.
// Takes deadLetterPort (DeadLetterPort) which stores failed messages.
// Takes config (*email_dto.DispatcherConfig) which sets retry behaviour and
// queue sizes.
//
// Returns *EmailDispatcher which is ready to use after calling Start.
func NewEmailDispatcher(provider EmailProviderPort, deadLetterPort DeadLetterPort, config *email_dto.DispatcherConfig) *EmailDispatcher {
	applyDispatcherConfigDefaults(config)

	clk := config.Clock
	if clk == nil {
		clk = clock.RealClock()
	}

	priorityQueue := retry.NewHeap(func(item *retryItem) time.Time { return item.priority })

	var circuitBreaker *gobreaker.CircuitBreaker[any]
	if config.MaxConsecutiveFailures > 0 {
		cbSettings := gobreaker.Settings{
			Name:         fmt.Sprintf("email-provider-%p", provider),
			MaxRequests:  1,
			Interval:     config.CircuitBreakerInterval,
			Timeout:      config.CircuitBreakerTimeout,
			BucketPeriod: circuitBreakerBucketPeriod,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= safeconv.IntToUint32(config.MaxConsecutiveFailures)
			},
			IsExcluded: func(err error) bool {
				return errors.Is(err, context.Canceled) ||
					errors.Is(err, context.DeadlineExceeded)
			},
		}
		circuitBreaker = gobreaker.NewCircuitBreaker[any](cbSettings)
	}

	return &EmailDispatcher{
		clock:            clk,
		provider:         provider,
		queue:            make(chan *email_dto.SendParams, config.QueueSize),
		retryHeap:        priorityQueue,
		retryJobsChan:    make(chan *retryItem, config.RetryQueueSize),
		retrySignal:      make(chan struct{}, 1),
		deadLetterQueue:  deadLetterPort,
		batchSize:        config.BatchSize,
		flushInterval:    config.FlushInterval,
		retryWorkerCount: config.RetryWorkerCount,
		maxRetryHeapSize: config.MaxRetryHeapSize,
		retryConfig: RetryConfig{
			Config: retry.Config{
				JitterFunc:    config.JitterFunc,
				MaxRetries:    config.MaxRetries,
				InitialDelay:  config.InitialDelay,
				MaxDelay:      config.MaxDelay,
				BackoffFactor: config.BackoffFactor,
			},
			DeadLetterQueue: config.DeadLetterQueue || deadLetterPort != nil,
		},
		circuitBreaker: circuitBreaker,
		shutdownChan:   make(chan struct{}),
		flushChan:      make(chan struct{}, 1),
		shutdownName:   fmt.Sprintf("email-dispatcher-%p", provider),
	}
}

// Start begins the dispatcher processing loops and registers for shutdown.
//
// Returns error when the dispatcher is already running.
//
// Spawns one goroutine for main processing, one for retry production, and N
// goroutines for retry workers. All goroutines run until Stop is called.
func (d *EmailDispatcher) Start(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	return l.RunInSpan(ctx, "EmailDispatcher.Start", func(spanCtx context.Context, spanLog logger_domain.Logger) error {
		d.mu.Lock()
		defer d.mu.Unlock()

		if d.isRunning {
			return errDispatcherRunning
		}

		d.isRunning = true
		d.startTime = d.clock.Now()
		d.wg.Add(2 + d.retryWorkerCount)

		shutdown.Register(spanCtx, d.shutdownName, d.shutdownCleanup)

		go d.processingLoop(spanCtx)
		go d.retryProducerLoop(spanCtx)
		for i := range d.retryWorkerCount {
			go d.retryWorker(spanCtx, i+1)
		}

		dispatcherStartCount.Add(spanCtx, 1)

		logFields := []slog.Attr{
			logger_domain.Int(logFieldBatchSize, d.batchSize),
			logger_domain.Duration(logFieldFlushInterval, d.flushInterval),
			logger_domain.Int(logFieldMaxRetries, d.retryConfig.MaxRetries),
			logger_domain.Duration(logFieldInitialDelay, d.retryConfig.InitialDelay),
			logger_domain.Bool(logFieldDeadLetterQueue, d.retryConfig.DeadLetterQueue),
			logger_domain.Int(logFieldRetryWorkerCount, d.retryWorkerCount),
		}
		if d.circuitBreaker != nil {
			logFields = append(logFields, logger_domain.Bool(logFieldCircuitBreaker, true))
			spanLog.Internal("Email dispatcher started with retry, dead letter queue, and circuit breaker", logFields...)
		} else {
			spanLog.Internal("Email dispatcher started with retry and dead letter queue", logFields...)
		}

		return nil
	},
		logger_domain.Int(logFieldBatchSize, d.batchSize),
		logger_domain.Duration(logFieldFlushInterval, d.flushInterval),
		logger_domain.Int(logFieldMaxRetries, d.retryConfig.MaxRetries),
		logger_domain.Duration(logFieldInitialDelay, d.retryConfig.InitialDelay),
		logger_domain.Bool(logFieldDeadLetterQueue, d.retryConfig.DeadLetterQueue),
		logger_domain.Int(logFieldRetryWorkerCount, d.retryWorkerCount),
	)
}

// Stop gracefully shuts down the dispatcher.
//
// Returns error when the shutdown span fails to complete.
//
// Safe for concurrent use. Waits for all pending work to finish and persists
// the retry queue before returning.
func (d *EmailDispatcher) Stop(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	return l.RunInSpan(ctx, "EmailDispatcher.Stop", func(spanCtx context.Context, spanLog logger_domain.Logger) error {
		d.mu.Lock()
		defer d.mu.Unlock()

		if !d.isRunning {
			return nil
		}

		spanLog.Internal("Stopping email dispatcher")
		close(d.shutdownChan)
		d.wg.Wait()
		d.persistRetryQueueOnShutdown(spanCtx)
		d.isRunning = false
		spanLog.Internal("Email dispatcher stopped")

		dispatcherStopCount.Add(spanCtx, 1)
		return nil
	})
}

// Queue adds an email to the queue for batched sending.
//
// It respects the context for cancellation or deadlines, returning an error
// if the context is done before the item can be queued.
//
// Takes params (*email_dto.SendParams) which specifies the email to send.
//
// Returns error when the context is cancelled or the deadline is exceeded.
func (d *EmailDispatcher) Queue(ctx context.Context, params *email_dto.SendParams) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("failed to queue email: %w", err)
	}
	select {
	case d.queue <- params:
		emailQueuedCount.Add(ctx, 1)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("failed to queue email: %w", ctx.Err())
	}
}

// Flush forces an immediate flush of all queued emails.
//
// Returns error when the context is cancelled before the flush is triggered.
func (d *EmailDispatcher) Flush(ctx context.Context) error {
	select {
	case d.flushChan <- struct{}{}:
		flushCount.Add(ctx, 1)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// SetBatchSize updates the batch size for email sending.
//
// Takes size (int) which specifies the new batch size; values less than one
// are ignored.
//
// Safe for concurrent use.
func (d *EmailDispatcher) SetBatchSize(size int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if size > 0 {
		d.batchSize = size
	}
}

// SetFlushInterval updates the flush interval for pending emails.
//
// Takes interval (time.Duration) which specifies the new flush interval.
// Values of zero or less are ignored.
//
// Safe for concurrent use.
func (d *EmailDispatcher) SetFlushInterval(interval time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if interval > 0 {
		d.flushInterval = interval
	}
}

// SetRetryConfig updates the retry configuration.
//
// Takes config (RetryConfig) which specifies the new retry settings.
//
// Safe for concurrent use.
func (d *EmailDispatcher) SetRetryConfig(config RetryConfig) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.retryConfig = config
}

// GetRetryConfig returns the current retry configuration.
//
// Returns RetryConfig which contains the current retry settings.
//
// Safe for concurrent use.
func (d *EmailDispatcher) GetRetryConfig() RetryConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.retryConfig
}

// GetDeadLetterQueue returns the dead letter queue instance.
//
// Returns DeadLetterPort which provides access to the dead letter queue.
func (d *EmailDispatcher) GetDeadLetterQueue() DeadLetterPort {
	return d.deadLetterQueue
}

// GetDeadLetterCount returns the number of emails in the dead letter queue.
//
// Returns int which is the count of emails awaiting retry or manual review.
// Returns error when the queue count operation fails.
func (d *EmailDispatcher) GetDeadLetterCount(ctx context.Context) (int, error) {
	if d.deadLetterQueue == nil {
		return 0, nil
	}
	return d.deadLetterQueue.Count(ctx)
}

// ClearDeadLetterQueue removes all emails from the dead letter queue.
//
// Returns error when no dead letter queue is configured or clearing fails.
func (d *EmailDispatcher) ClearDeadLetterQueue(ctx context.Context) error {
	if d.deadLetterQueue == nil {
		return errNoDLQ
	}
	return d.deadLetterQueue.Clear(ctx)
}

// GetRetryQueueSize returns the number of emails currently in the retry queue.
//
// Returns int which is the count of emails awaiting retry.
// Returns error which is always nil.
//
// Safe for concurrent use.
func (d *EmailDispatcher) GetRetryQueueSize(_ context.Context) (int, error) {
	d.retryMutex.Lock()
	defer d.retryMutex.Unlock()
	return d.retryHeap.Len(), nil
}

// GetProcessingStats returns statistics about the dispatcher.
//
// retrieval.
//
// Returns DispatcherStats which contains queue sizes, counters, and uptime.
// Returns error when stats retrieval fails.
//
// Safe for concurrent use. Uses mutex locking for retry queue access and atomic
// operations for counters.
func (d *EmailDispatcher) GetProcessingStats(ctx context.Context) (DispatcherStats, error) {
	deadLetterCount := 0
	if d.deadLetterQueue != nil {
		if count, err := d.deadLetterQueue.Count(ctx); err == nil {
			deadLetterCount = count
		}
	}

	var uptime time.Duration
	if !d.startTime.IsZero() {
		uptime = d.clock.Now().Sub(d.startTime)
	}

	d.retryMutex.Lock()
	retryQueueSize := d.retryHeap.Len()
	d.retryMutex.Unlock()

	return DispatcherStats{
		QueuedEmails:    len(d.queue),
		RetryQueueSize:  retryQueueSize,
		DeadLetterCount: deadLetterCount,
		TotalProcessed:  atomic.LoadInt64(&d.totalProcessed),
		TotalSuccessful: atomic.LoadInt64(&d.totalSuccessful),
		TotalFailed:     atomic.LoadInt64(&d.totalFailed),
		TotalRetries:    atomic.LoadInt64(&d.totalRetries),
		Uptime:          uptime,
	}, nil
}

// retryProducerLoop manages the retry priority queue as a producer.
// It wakes up when the next item is due, collects all ready items, and pushes
// them onto a channel for the retry workers to consume.
func (d *EmailDispatcher) retryProducerLoop(ctx context.Context) {
	ctx, _ = logger_domain.From(ctx, log)
	defer d.wg.Done()
	defer close(d.retryJobsChan)
	defer goroutine.RecoverPanic(ctx, "email.retryProducerLoop")

	timer := d.clock.NewTimer(0)
	defer timer.Stop()
	if !timer.Stop() {
		<-timer.C()
	}

	for {
		if d.dispatchReadyItems() {
			return
		}

		sleepDuration := d.calculateRetrySleep()
		if sleepDuration <= 0 {
			continue
		}
		timer.Reset(sleepDuration)

		action := d.waitForRetryEvent(ctx, timer)
		switch action {
		case retryActionContinue:
			continue
		case retryActionStop:
			return
		case retryActionProcess:
			if d.dispatchReadyItems() {
				return
			}
		}
	}
}

// waitForRetryEvent waits for timer, signal, or shutdown events.
//
// Takes timer (clock.ChannelTimer) which triggers retry processing.
//
// Returns retryAction which indicates how the caller should proceed.
func (d *EmailDispatcher) waitForRetryEvent(ctx context.Context, timer clock.ChannelTimer) retryAction {
	ctx, l := logger_domain.From(ctx, log)
	select {
	case <-timer.C():
		return retryActionProcess
	case <-d.retrySignal:
		drainTimer(timer)
		return retryActionContinue
	case <-d.shutdownChan:
		return retryActionStop
	case <-ctx.Done():
		l.Internal("Context cancelled, shutting down retry producer loop")
		return retryActionStop
	}
}

// dispatchReadyItems dispatches all ready items to workers.
//
// Returns bool which is true if shutdown occurred during dispatch.
//
// Safe for concurrent use. Items that cannot be dispatched due to shutdown
// are returned to the retry heap.
func (d *EmailDispatcher) dispatchReadyItems() bool {
	for _, item := range d.collectReadyRetryItems() {
		select {
		case d.retryJobsChan <- item:
		case <-d.shutdownChan:
			d.retryMutex.Lock()
			d.retryHeap.PushItem(item)
			d.retryMutex.Unlock()
			return true
		}
	}
	return false
}

// retryWorker reads retry jobs from the queue and processes them.
// Several workers run at the same time to handle retries more quickly.
//
// Takes id (int) which identifies this worker for logging.
func (d *EmailDispatcher) retryWorker(ctx context.Context, id int) {
	ctx, l := logger_domain.From(ctx, log)
	defer d.wg.Done()
	defer goroutine.RecoverPanic(ctx, "email.retryWorker")
	workerLog := l.With(logger_domain.Int(logFieldRetryWorkerID, id))
	workerLog.Internal("Retry worker started")

	for item := range d.retryJobsChan {
		d.processRetryItem(ctx, item)
	}

	workerLog.Internal("Retry worker stopped")
}

// calculateRetrySleep works out how long to wait before the next retry is due.
//
// Returns time.Duration which is the time to wait, or zero if an item is ready
// to process now.
//
// Safe for concurrent use; protects access to the retry heap with a mutex.
func (d *EmailDispatcher) calculateRetrySleep() time.Duration {
	d.retryMutex.Lock()
	defer d.retryMutex.Unlock()

	if d.retryHeap.Len() == 0 {
		return emptyQueueSleep
	}

	nextItem, _ := d.retryHeap.Peek()
	sleepDuration := nextItem.priority.Sub(d.clock.Now())
	if sleepDuration < 0 {
		return 0
	}
	return sleepDuration
}

// collectReadyRetryItems removes all items from the heap that are ready to
// retry.
//
// Returns []*retryItem which contains items whose retry time has passed.
//
// Safe for concurrent use. Holds retryMutex for the whole operation.
func (d *EmailDispatcher) collectReadyRetryItems() []*retryItem {
	d.retryMutex.Lock()
	defer d.retryMutex.Unlock()

	var readyItems []*retryItem
	now := d.clock.Now()
	for d.retryHeap.Len() > 0 {
		top, _ := d.retryHeap.Peek()
		if top.priority.After(now) {
			break
		}
		item, ok := d.retryHeap.PopItem()
		if !ok {
			break
		}
		readyItems = append(readyItems, item)
	}
	return readyItems
}

// processRetryItem attempts to send a single retried email and handles
// the outcome.
//
// Takes item (*retryItem) which contains the email to retry.
func (d *EmailDispatcher) processRetryItem(ctx context.Context, item *retryItem) {
	ctx, l := logger_domain.From(ctx, log)
	emailError := item.emailError
	atomic.AddInt64(&d.totalRetries, 1)
	retryAttemptCount.Add(ctx, 1)

	if err := d.sendWithCircuitBreaker(ctx, &emailError.Email); err != nil {
		meta := failureMeta{
			attempt:      emailError.Attempt + 1,
			firstAttempt: emailError.FirstAttempt,
			email:        &emailError.Email,
			sendErr:      err,
		}
		d.handleFailedEmail(ctx, meta)
	} else {
		atomic.AddInt64(&d.totalSuccessful, 1)
		atomic.AddInt64(&d.totalProcessed, 1)
		l.Trace("Email retry successful",
			logger_domain.String(logFieldRecipients, fmt.Sprintf(fmtListRecipients, emailError.Email.To)),
			logger_domain.Int(logFieldAttempt, emailError.Attempt+1),
		)
	}
}

// handleFailedEmail processes a failed email by deciding whether to retry or
// send it to the dead-letter queue. It acts as a high-level dispatcher for
// failure handling.
//
// Takes meta (failureMeta) which contains the failed email and retry state.
func (d *EmailDispatcher) handleFailedEmail(ctx context.Context, meta failureMeta) {
	ctx, l := logger_domain.From(ctx, log)
	attempt := meta.attempt
	firstAttempt := meta.firstAttempt
	email := meta.email
	sendErr := meta.sendErr

	if attempt == 1 && firstAttempt.IsZero() {
		firstAttempt = d.clock.Now()
		meta.firstAttempt = firstAttempt
	}

	if !d.retryConfig.ShouldRetry(attempt) {
		dlqCtx := deadLetterContext{
			email:         *email,
			originalError: sendErr,
			totalAttempts: attempt,
			firstAttempt:  firstAttempt,
			lastAttempt:   d.clock.Now(),
		}
		d.sendToDeadLetterQueue(ctx, &dlqCtx)
		return
	}

	if d.isRetryHeapFull() {
		heapSize, maxSize := d.retryHeap.Len(), d.maxRetryHeapSize
		l.ReportError(nil, sendErr, "Retry heap is full, dropping email (fail fast)",
			logger_domain.String(logFieldRecipients, fmt.Sprintf(fmtListRecipients, email.To)),
			logger_domain.String(logFieldSubject, email.Subject),
			logger_domain.Int(logFieldAttempt, attempt),
			logger_domain.Int("heap_size", heapSize),
			logger_domain.Int("max_heap_size", maxSize),
		)
		dlqCtx := deadLetterContext{
			email:         *email,
			originalError: fmt.Errorf("retry heap full (size: %d, max: %d): %w", heapSize, maxSize, sendErr),
			totalAttempts: attempt,
			firstAttempt:  firstAttempt,
			lastAttempt:   d.clock.Now(),
		}
		d.sendToDeadLetterQueue(ctx, &dlqCtx)
		return
	}

	d.queueEmailForRetry(ctx, meta)
}

// isRetryHeapFull checks if the retry heap has reached its capacity limit.
//
// Returns bool which is true when the heap is at or above the maximum size.
//
// Safe for concurrent use; acquires the retry mutex before reading.
func (d *EmailDispatcher) isRetryHeapFull() bool {
	d.retryMutex.Lock()
	defer d.retryMutex.Unlock()
	return d.retryHeap.Len() >= d.maxRetryHeapSize
}

// queueEmailForRetry adds a failed email to the retry heap.
//
// Takes meta (failureMeta) which holds the failed email and error details.
//
// Safe for concurrent use. Uses a mutex to protect the retry heap and sends a
// non-blocking signal to wake the producer loop.
func (d *EmailDispatcher) queueEmailForRetry(ctx context.Context, meta failureMeta) {
	ctx, l := logger_domain.From(ctx, log)
	now := d.clock.Now()
	nextRetry := d.retryConfig.CalculateNextRetry(meta.attempt, now)
	emailError := EmailError{
		Email:        *meta.email,
		Error:        meta.sendErr,
		Attempt:      meta.attempt,
		FirstAttempt: meta.firstAttempt,
		LastAttempt:  now,
		NextRetry:    nextRetry,
	}

	d.retryMutex.Lock()
	d.retryHeap.PushItem(&retryItem{emailError: emailError, priority: nextRetry})
	d.retryMutex.Unlock()

	retryScheduledCount.Add(ctx, 1)

	select {
	case d.retrySignal <- struct{}{}:
	default:
	}

	l.Trace("Email queued for retry",
		logger_domain.String(logFieldRecipients, fmt.Sprintf(fmtListRecipients, meta.email.To)),
		logger_domain.Int(logFieldAttempt, meta.attempt),
		logger_domain.Time(logFieldNextRetry, nextRetry),
		logger_domain.String(logFieldError, meta.sendErr.Error()),
	)
}

// sendToDeadLetterQueue sends a failed email to the dead letter queue.
//
// Takes dlqCtx (*deadLetterContext) which holds the failed email and retry
// details.
func (d *EmailDispatcher) sendToDeadLetterQueue(ctx context.Context, dlqCtx *deadLetterContext) {
	ctx, l := logger_domain.From(ctx, log)
	atomic.AddInt64(&d.totalFailed, 1)
	atomic.AddInt64(&d.totalProcessed, 1)
	deadLetterCount.Add(ctx, 1)

	if d.deadLetterQueue == nil || !d.retryConfig.DeadLetterQueue {
		l.ReportError(nil, dlqCtx.originalError, "Email permanently failed after max retries (no DLQ)",
			logger_domain.String(logFieldRecipients, fmt.Sprintf(fmtListRecipients, dlqCtx.email.To)),
			logger_domain.String(logFieldSubject, dlqCtx.email.Subject),
			logger_domain.Int(logFieldTotalAttempts, dlqCtx.totalAttempts),
		)
		return
	}

	entry := email_dto.DeadLetterEntry{
		Email:         dlqCtx.email,
		OriginalError: dlqCtx.originalError.Error(),
		TotalAttempts: dlqCtx.totalAttempts,
		FirstAttempt:  dlqCtx.firstAttempt,
		LastAttempt:   dlqCtx.lastAttempt,
	}

	if err := d.deadLetterQueue.Add(ctx, &entry); err != nil {
		l.ReportError(nil, err, "Failed to add email to dead letter queue",
			logger_domain.String(logFieldRecipients, fmt.Sprintf(fmtListRecipients, dlqCtx.email.To)),
			logger_domain.String(logFieldSubject, dlqCtx.email.Subject),
		)
	}
}

// processingLoop runs the main loop that batches and sends emails from the
// queue. It acts as a state machine, passing actions to helper methods.
func (d *EmailDispatcher) processingLoop(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	defer d.wg.Done()
	defer goroutine.RecoverPanic(ctx, "email.processingLoop")

	ticker := d.clock.NewTicker(d.flushInterval)
	defer ticker.Stop()

	batch := make([]*email_dto.SendParams, 0, d.batchSize)

	for {
		select {
		case email := <-d.queue:
			batch = d.processNewEmail(ctx, batch, email)

		case <-ticker.C():
			batch = d.flushBatchIfNotEmpty(ctx, batch)

		case <-d.flushChan:
			batch = d.drainQueueAndFlush(ctx, batch)

		case <-d.shutdownChan:
			d.drainQueueOnShutdown(ctx, batch)
			return

		case <-ctx.Done():
			l.Internal("Context cancelled, shutting down main processing loop")
			return
		}
	}
}

// processNewEmail adds an email to the batch and sends it when the batch is
// full.
//
// Takes batch ([]*email_dto.SendParams) which is the current batch of emails.
// Takes email (*email_dto.SendParams) which is the email to add.
//
// Returns []*email_dto.SendParams which is the updated batch, or an empty
// batch if the batch was sent.
func (d *EmailDispatcher) processNewEmail(ctx context.Context, batch []*email_dto.SendParams, email *email_dto.SendParams) []*email_dto.SendParams {
	batch = append(batch, email)
	if len(batch) >= d.batchSize {
		d.sendBatch(ctx, batch)
		return make([]*email_dto.SendParams, 0, d.batchSize)
	}
	return batch
}

// flushBatchIfNotEmpty sends the current batch if it has any emails.
//
// Takes batch ([]*email_dto.SendParams) which holds the emails to send.
//
// Returns []*email_dto.SendParams which is a new empty batch if emails were
// sent, or the same batch if it was already empty.
func (d *EmailDispatcher) flushBatchIfNotEmpty(ctx context.Context, batch []*email_dto.SendParams) []*email_dto.SendParams {
	if len(batch) > 0 {
		d.sendBatch(ctx, batch)
		return make([]*email_dto.SendParams, 0, d.batchSize)
	}
	return batch
}

// drainQueueAndFlush empties all queued emails and sends any remaining
// partial batch. Used for manual flush operations to ensure predictable
// behaviour.
//
// Takes batch ([]*email_dto.SendParams) which holds the current partial batch.
//
// Returns []*email_dto.SendParams which is the updated batch after sending.
func (d *EmailDispatcher) drainQueueAndFlush(ctx context.Context, batch []*email_dto.SendParams) []*email_dto.SendParams {
	for {
		select {
		case email := <-d.queue:
			batch = d.processNewEmail(ctx, batch, email)
		default:
			return d.flushBatchIfNotEmpty(ctx, batch)
		}
	}
}

// drainQueueOnShutdown sends all remaining emails during graceful shutdown.
//
// Takes batch ([]*email_dto.SendParams) which contains any pending emails to
// send before draining the queue.
func (d *EmailDispatcher) drainQueueOnShutdown(ctx context.Context, batch []*email_dto.SendParams) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Starting final flush on shutdown...")
	batch = d.flushBatchIfNotEmpty(ctx, batch)

	l.Internal("Draining email queue...", logger_domain.Int("items_in_queue", len(d.queue)))
	close(d.queue)
	for email := range d.queue {
		batch = d.processNewEmail(ctx, batch, email)
	}

	_ = d.flushBatchIfNotEmpty(ctx, batch)
	l.Internal("Queue draining complete.")
}

// sendBatch sends a batch of emails, using bulk sending if the provider
// supports it. Falls back to sending emails one at a time if bulk sending
// fails.
//
// Takes batch ([]*email_dto.SendParams) which contains the emails to send.
func (d *EmailDispatcher) sendBatch(ctx context.Context, batch []*email_dto.SendParams) {
	ctx, l := logger_domain.From(ctx, log)
	if len(batch) == 0 {
		return
	}

	_ = l.RunInSpan(ctx, "EmailDispatcher.sendBatch", func(spanCtx context.Context, spanLog logger_domain.Logger) error {
		batchSentCount.Add(spanCtx, 1)
		batchSizeMetric.Record(spanCtx, int64(len(batch)))
		spanLog.Trace("Sending email batch", logger_domain.Int(logFieldBatchSize, len(batch)))

		start := d.clock.Now()
		if d.provider.SupportsBulkSending() {
			if err := d.provider.SendBulk(spanCtx, batch); err != nil {
				spanLog.ReportError(nil, err, "Bulk send failed, falling back to individual sends",
					logger_domain.Int(logFieldBatchSize, len(batch)))
				d.sendBatchIndividually(spanCtx, batch)
			} else {
				durMs := float64(d.clock.Now().Sub(start).Milliseconds())
				emailSendDuration.Record(spanCtx, durMs)
				emailSentCount.Add(spanCtx, int64(len(batch)))
				atomic.AddInt64(&d.totalSuccessful, int64(len(batch)))
				atomic.AddInt64(&d.totalProcessed, int64(len(batch)))
			}
		} else {
			d.sendBatchIndividually(spanCtx, batch)
		}
		return nil
	}, logger_domain.Int(logFieldBatchSize, len(batch)))
}

// sendWithCircuitBreaker wraps the provider's Send method with circuit
// breaker logic and records metrics.
//
// Takes email (*email_dto.SendParams) which contains the email to send.
//
// Returns error when the circuit breaker is open or sending fails.
func (d *EmailDispatcher) sendWithCircuitBreaker(ctx context.Context, email *email_dto.SendParams) error {
	ctx, l := logger_domain.From(ctx, log)
	return l.RunInSpan(ctx, "EmailDispatcher.Send", func(spanCtx context.Context, _ logger_domain.Logger) error {
		start := d.clock.Now()
		var err error
		if d.circuitBreaker != nil {
			_, err = d.circuitBreaker.Execute(func() (any, error) {
				return nil, d.provider.Send(spanCtx, email)
			})
		} else {
			err = d.provider.Send(spanCtx, email)
		}

		durMs := float64(d.clock.Now().Sub(start).Milliseconds())
		emailSendDuration.Record(spanCtx, durMs)
		if err != nil {
			emailSendErrorCount.Add(spanCtx, 1)
			return fmt.Errorf("sending email via provider: %w", err)
		}
		emailSentCount.Add(spanCtx, 1)
		return nil
	}, logger_domain.Int("recipient_count", len(email.To)))
}

// sendBatchIndividually sends each email in the batch one at a time.
// Failed emails are passed to handleFailedEmail for retry handling.
//
// Takes batch ([]*email_dto.SendParams) which contains the emails to send.
func (d *EmailDispatcher) sendBatchIndividually(ctx context.Context, batch []*email_dto.SendParams) {
	for _, email := range batch {
		if ctx.Err() != nil {
			return
		}
		if err := d.sendWithCircuitBreaker(ctx, email); err != nil {
			meta := failureMeta{
				attempt:      1,
				firstAttempt: d.clock.Now(),
				email:        email,
				sendErr:      err,
			}
			d.handleFailedEmail(ctx, meta)
		} else {
			atomic.AddInt64(&d.totalSuccessful, 1)
			atomic.AddInt64(&d.totalProcessed, 1)
		}
	}
}

// shutdownCleanup saves any pending work when the application stops.
//
// Returns error when the stop operation fails.
func (d *EmailDispatcher) shutdownCleanup(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)
	l.Internal("Executing email dispatcher shutdown cleanup")

	shutdownCtx, cancel := context.WithTimeoutCause(context.Background(), 30*time.Second,
		errors.New("email dispatcher shutdown exceeded 30s timeout"))
	defer cancel()

	return d.Stop(shutdownCtx)
}

// persistRetryQueueOnShutdown moves all pending retry items to the dead-letter
// queue to prevent data loss during shutdown.
//
// Not safe for concurrent use. Acquires retryMutex lock within the method.
func (d *EmailDispatcher) persistRetryQueueOnShutdown(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	d.retryMutex.Lock()
	defer d.retryMutex.Unlock()

	if d.retryHeap.Len() == 0 {
		return
	}

	l.Internal("Persisting retry queue to dead-letter queue on shutdown", logger_domain.Int(logFieldItems, d.retryHeap.Len()))
	for d.retryHeap.Len() > 0 {
		if ctx.Err() != nil {
			l.Warn("Shutdown timeout reached, abandoning remaining retry items",
				logger_domain.Int("remaining", d.retryHeap.Len()))
			return
		}

		item, ok := d.retryHeap.PopItem()
		if !ok {
			break
		}

		dlqCtx := deadLetterContext{
			email:         item.emailError.Email,
			originalError: fmt.Errorf("service shutdown during retry: %w", item.emailError.Error),
			totalAttempts: item.emailError.Attempt,
			lastAttempt:   item.emailError.LastAttempt,
		}
		d.sendToDeadLetterQueue(ctx, &dlqCtx)
	}
}

// applyDispatcherConfigDefaults fills in missing or invalid fields in the
// config with sensible default values.
//
// Takes config (*email_dto.DispatcherConfig) which holds the dispatcher
// settings to fill in.
func applyDispatcherConfigDefaults(config *email_dto.DispatcherConfig) {
	if config.BatchSize <= 0 {
		config.BatchSize = defaultBatchSize
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = defaultFlushInterval
	}
	if config.QueueSize <= 0 {
		config.QueueSize = defaultQueueSize
	}
	if config.RetryQueueSize <= 0 {
		config.RetryQueueSize = defaultRetryQueueSize
	}
	if config.MaxRetries < 0 {
		config.MaxRetries = defaultMaxRetries
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = defaultInitialDelay
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = defaultMaxDelay
	}
	if config.BackoffFactor <= 0 {
		config.BackoffFactor = defaultBackoffFactor
	}
	if config.RetryWorkerCount <= 0 {
		config.RetryWorkerCount = runtime.NumCPU()
	}
	if config.MaxRetryHeapSize <= 0 {
		config.MaxRetryHeapSize = defaultMaxRetryHeapSize
	}
}

// drainTimer removes any pending event from a timer channel.
//
// Takes t (clock.ChannelTimer) which is the timer to drain.
func drainTimer(t clock.ChannelTimer) {
	if !t.Stop() {
		select {
		case <-t.C():
		default:
		}
	}
}
