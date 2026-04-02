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

package notification_domain

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sony/gobreaker/v2"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/notification/notification_dto"
	"piko.sh/piko/internal/retry"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safeconv"
)

var _ NotificationDispatcherPort = (*NotificationDispatcher)(nil)

// retryProducerAction represents the outcome of a retry producer step.
type retryProducerAction int

const (
	// retryActionContinue tells the retry producer to continue processing.
	retryActionContinue retryProducerAction = iota

	// retryActionShutdown signals that the retry producer should stop.
	retryActionShutdown
)

const (
	// defaultBatchSize is the default number of notifications to process in one
	// batch.
	defaultBatchSize = 10

	// defaultFlushInterval is the default time between notification flushes.
	defaultFlushInterval = 30 * time.Second

	// defaultQueueSize is the default size of the notification dispatch queue.
	defaultQueueSize = 1000

	// defaultRetryQueueSize is the default buffer size for the retry queue.
	defaultRetryQueueSize = 500

	// defaultMaxRetries is the default number of retry attempts for notifications.
	defaultMaxRetries = 3

	// defaultInitialDelay is the wait time before the first retry of a
	// notification.
	defaultInitialDelay = 5 * time.Second

	// defaultMaxDelay is the longest wait time between retry attempts.
	defaultMaxDelay = 5 * time.Minute

	// defaultBackoffFactor is the multiplier for exponential backoff delays.
	defaultBackoffFactor = 2.0

	// defaultMaxRetryHeapSize is the largest number of items the retry heap can
	// hold.
	defaultMaxRetryHeapSize = 50000

	// defaultCircuitBreakerInterval is the default time between circuit breaker
	// state checks.
	defaultCircuitBreakerInterval = 60 * time.Second

	// defaultCircuitBreakerTimeout is the default time limit for circuit breaker
	// operations.
	defaultCircuitBreakerTimeout = 30 * time.Second

	// circuitBreakerBucketPeriod is the time window for counting circuit breaker
	// failures.
	circuitBreakerBucketPeriod = 10 * time.Second

	// defaultMaxConsecutiveFailures is the default circuit breaker failure
	// threshold.
	defaultMaxConsecutiveFailures = 5

	// emptyQueueSleep is the duration to wait before rechecking when the retry
	// queue is empty.
	emptyQueueSleep = 10 * time.Minute
)

// queuedNotification represents a notification queued for sending with
// multi-cast and retry tracking.
type queuedNotification struct {
	// firstAttempt is when the first send was tried.
	firstAttempt time.Time

	// nextRetryTime is when this notification should next be retried.
	nextRetryTime time.Time

	// params holds the notification content and delivery settings.
	params *notification_dto.SendParams

	// targetProviders lists the providers to try; an empty slice defaults to a
	// single empty string which selects the default provider.
	targetProviders []string

	// failedProviders tracks providers that failed during the current send
	// attempt.
	failedProviders []string

	// attempt is the current retry attempt number; starts at 0.
	attempt int
}

// NotificationDispatcher provides batched notification sending with retry,
// dead-letter queue, and per-provider circuit breaker capabilities. It
// implements NotificationDispatcherPort and DispatcherPort.
type NotificationDispatcher struct {
	// service provides access to notification providers.
	service *service

	// clock provides time functions for testing purposes.
	clock clock.Clock

	// startTime records when the dispatcher started; used to work out uptime.
	startTime time.Time

	// deadLetterQueue stores failed messages for later retry or inspection.
	deadLetterQueue DeadLetterPort

	// circuitBreakers maps provider names to their circuit breaker instances.
	circuitBreakers map[string]*gobreaker.CircuitBreaker[any]

	// queue holds notifications waiting to be processed in batches.
	queue chan *notification_dto.SendParams

	// retryHeap holds notifications waiting to be retried, sorted by next retry
	// time.
	retryHeap *retry.Heap[*queuedNotification]

	// retryJobsChan sends retry items to retry workers for processing.
	retryJobsChan chan *retryItem

	// flushChan signals when to flush queued notifications immediately.
	flushChan chan struct{}

	// retrySignal signals the retry producer to check for new items.
	retrySignal chan struct{}

	// shutdownChan signals all goroutines to stop processing.
	shutdownChan chan struct{}

	// shutdownName is the name used to register this dispatcher for graceful
	// shutdown.
	shutdownName string

	// retryConfig controls retry behaviour for failed notifications.
	retryConfig RetryConfig

	// wg tracks running goroutines to allow a clean shutdown.
	wg sync.WaitGroup

	// flushInterval is the time between automatic batch flushes.
	flushInterval time.Duration

	// batchSize is the number of notifications to gather before processing them.
	batchSize int

	// retryWorkerCount is the number of goroutines that process failed
	// notifications.
	retryWorkerCount int

	// maxRetryHeapSize is the maximum number of entries allowed in the retry heap.
	maxRetryHeapSize int

	// totalProcessed counts all notifications handled; accessed atomically.
	totalProcessed int64

	// totalSuccessful counts notifications sent to all providers without error.
	totalSuccessful int64

	// totalFailed counts notifications that were sent to the dead letter queue.
	totalFailed int64

	// totalRetries is the total number of retry attempts made.
	totalRetries int64

	// mu guards concurrent access to isRunning.
	mu sync.RWMutex

	// retryMutex guards access to the retryHeap.
	retryMutex sync.Mutex

	// isRunning indicates whether the dispatcher is active.
	isRunning bool

	// cbConfig holds the circuit breaker settings for notification providers.
	cbConfig circuitBreakerConfig
}

// circuitBreakerConfig holds settings for the circuit breaker pattern.
type circuitBreakerConfig struct {
	// maxConsecutiveFailures is the number of failures in a row that triggers the
	// circuit breaker to open.
	maxConsecutiveFailures int

	// interval is the time window for counting failures; 0 uses the default.
	interval time.Duration

	// timeout is how long to wait before trying again after the circuit opens.
	timeout time.Duration
}

// retryItem holds a notification that is ready to be sent again.
type retryItem struct {
	// notification is the queued notification awaiting retry.
	notification *queuedNotification
}

// NewNotificationDispatcher creates a new NotificationDispatcher.
//
// Takes notificationService (Service) which provides the notification sending
// capabilities.
// Takes deadLetterQueue (DeadLetterPort) which stores failed notifications.
// Takes config (*notification_dto.DispatcherConfig) which configures dispatcher
// behaviour.
//
// Returns *NotificationDispatcher which is ready to start, or nil
// if service is not a valid service implementation.
func NewNotificationDispatcher(
	notificationService Service,
	deadLetterQueue DeadLetterPort,
	config *notification_dto.DispatcherConfig,
) *NotificationDispatcher {
	svcImpl, ok := notificationService.(*service)
	if !ok {
		return nil
	}

	applyDispatcherConfigDefaults(config)

	clk := clock.RealClock()

	priorityQueue := retry.NewHeap(func(qn *queuedNotification) time.Time { return qn.nextRetryTime })

	retryWorkerCount := runtime.NumCPU()
	queueSize := defaultQueueSize
	retryQueueSize := defaultRetryQueueSize
	maxRetryHeapSize := defaultMaxRetryHeapSize

	return &NotificationDispatcher{
		service:          svcImpl,
		clock:            clk,
		queue:            make(chan *notification_dto.SendParams, queueSize),
		retryHeap:        priorityQueue,
		retryJobsChan:    make(chan *retryItem, retryQueueSize),
		retrySignal:      make(chan struct{}, 1),
		deadLetterQueue:  deadLetterQueue,
		circuitBreakers:  make(map[string]*gobreaker.CircuitBreaker[any]),
		batchSize:        config.BatchSize,
		flushInterval:    config.FlushInterval,
		retryWorkerCount: retryWorkerCount,
		maxRetryHeapSize: maxRetryHeapSize,
		retryConfig: RetryConfig{
			MaxRetries:    config.MaxRetries,
			InitialDelay:  config.InitialDelay,
			MaxDelay:      config.MaxDelay,
			BackoffFactor: config.BackoffFactor,
		},
		cbConfig: circuitBreakerConfig{
			maxConsecutiveFailures: config.CircuitBreakerThreshold,
			interval:               config.CircuitBreakerInterval,
			timeout:                config.CircuitBreakerTimeout,
		},
		shutdownChan: make(chan struct{}),
		flushChan:    make(chan struct{}, 1),
		shutdownName: "notification-dispatcher",
	}
}

// Queue adds a notification to the batch queue for later sending.
//
// Takes params (*notification_dto.SendParams) which specifies the notification
// details to queue.
//
// Returns error when the dispatcher is not running or the context is cancelled.
//
// Safe for concurrent use.
func (d *NotificationDispatcher) Queue(ctx context.Context, params *notification_dto.SendParams) error {
	d.mu.RLock()
	if !d.isRunning {
		d.mu.RUnlock()
		return ErrDispatcherNotRunning
	}
	d.mu.RUnlock()

	select {
	case d.queue <- params:
		notificationQueuedCount.Add(ctx, 1)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("queueing notification: %w", ctx.Err())
	}
}

// Flush sends all queued notifications straight away.
//
// Returns error when the dispatcher is not running or the context is
// cancelled.
//
// Safe for concurrent use.
func (d *NotificationDispatcher) Flush(ctx context.Context) error {
	d.mu.RLock()
	if !d.isRunning {
		d.mu.RUnlock()
		return ErrDispatcherNotRunning
	}
	d.mu.RUnlock()

	select {
	case d.flushChan <- struct{}{}:
		flushCount.Add(ctx, 1)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("flushing notifications: %w", ctx.Err())
	}
}

// Start begins the dispatcher processing loops and registers for shutdown.
//
// Returns error when the dispatcher is already running.
//
// Spawns goroutines for the main processing loop, retry producer, and retry
// workers. These run until Stop is called.
func (d *NotificationDispatcher) Start(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)

	d.mu.Lock()
	if d.isRunning {
		d.mu.Unlock()
		return ErrDispatcherAlreadyRunning
	}
	d.isRunning = true
	d.startTime = d.clock.Now()
	d.mu.Unlock()

	dispatcherStartCount.Add(ctx, 1)

	detachedCtx := context.WithoutCancel(ctx)

	d.wg.Add(1)
	go d.processQueue(detachedCtx)

	d.wg.Add(1)
	go d.produceRetryJobs(detachedCtx)

	for range d.retryWorkerCount {
		d.wg.Add(1)
		go d.retryWorker(detachedCtx)
	}

	shutdown.Register(ctx, d.shutdownName, func(shutdownCtx context.Context) error {
		return d.Stop(shutdownCtx)
	})

	l.Internal("Notification dispatcher started",
		logger_domain.Int("batch_size", d.batchSize),
		logger_domain.Duration("flush_interval", d.flushInterval),
		logger_domain.Int("max_retries", d.retryConfig.MaxRetries))

	return nil
}

// Stop halts the dispatcher gracefully.
//
// Returns error when the dispatcher is not running.
//
// Safe for concurrent use. Signals shutdown to all spawned goroutines and
// waits for them to complete before returning.
func (d *NotificationDispatcher) Stop(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)

	d.mu.Lock()
	if !d.isRunning {
		d.mu.Unlock()
		return ErrDispatcherNotRunning
	}
	d.isRunning = false
	d.mu.Unlock()

	dispatcherStopCount.Add(ctx, 1)

	close(d.shutdownChan)

	d.wg.Wait()

	l.Internal("Notification dispatcher stopped")

	return nil
}

// SetBatchSize sets the number of notifications to process in each batch.
//
// Takes size (int) which specifies the batch size.
//
// Safe for concurrent use.
func (d *NotificationDispatcher) SetBatchSize(size int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.batchSize = size
}

// SetFlushInterval sets the time between flushes.
//
// Takes interval (time.Duration) which specifies the new flush interval.
//
// Safe for concurrent use.
func (d *NotificationDispatcher) SetFlushInterval(interval time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.flushInterval = interval
}

// SetRetryConfig sets the retry configuration.
//
// Takes config (RetryConfig) which specifies the retry behaviour settings.
//
// Safe for concurrent use.
func (d *NotificationDispatcher) SetRetryConfig(config RetryConfig) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.retryConfig = config
}

// GetRetryConfig returns the retry configuration.
//
// Returns RetryConfig which contains the current retry settings.
//
// Safe for concurrent use.
func (d *NotificationDispatcher) GetRetryConfig() RetryConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.retryConfig
}

// GetDeadLetterQueue returns the dead letter queue.
//
// Returns DeadLetterPort which provides access to failed notifications.
func (d *NotificationDispatcher) GetDeadLetterQueue() DeadLetterPort {
	return d.deadLetterQueue
}

// GetDeadLetterCount returns the number of messages in the dead letter queue.
//
// Returns int which is the number of messages in the dead letter queue.
// Returns error when the count operation fails.
func (d *NotificationDispatcher) GetDeadLetterCount(ctx context.Context) (int, error) {
	if d.deadLetterQueue == nil {
		return 0, nil
	}
	return d.deadLetterQueue.Count(ctx)
}

// ClearDeadLetterQueue removes all messages from the dead letter queue.
//
// Returns error when clearing the queue fails.
func (d *NotificationDispatcher) ClearDeadLetterQueue(ctx context.Context) error {
	if d.deadLetterQueue == nil {
		return nil
	}
	return d.deadLetterQueue.Clear(ctx)
}

// GetRetryQueueSize returns the number of items in the retry queue.
//
// Returns int which is the count of items waiting to be retried.
// Returns error which is always nil.
//
// Safe for concurrent use; protected by a mutex.
func (d *NotificationDispatcher) GetRetryQueueSize(_ context.Context) (int, error) {
	d.retryMutex.Lock()
	defer d.retryMutex.Unlock()
	return d.retryHeap.Len(), nil
}

// GetProcessingStats retrieves the current processing statistics.
//
// Returns DispatcherStats which contains the current processing metrics.
// Returns error when the statistics cannot be retrieved.
//
// Safe for concurrent use. Uses read locks to access shared state.
func (d *NotificationDispatcher) GetProcessingStats(ctx context.Context) (DispatcherStats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := DispatcherStats{
		QueuedNotifications: len(d.queue),
		TotalProcessed:      atomic.LoadInt64(&d.totalProcessed),
		TotalSuccessful:     atomic.LoadInt64(&d.totalSuccessful),
		TotalFailed:         atomic.LoadInt64(&d.totalFailed),
		TotalRetries:        atomic.LoadInt64(&d.totalRetries),
	}

	d.retryMutex.Lock()
	stats.RetryQueueSize = d.retryHeap.Len()
	d.retryMutex.Unlock()

	if d.deadLetterQueue != nil {
		count, err := d.deadLetterQueue.Count(ctx)
		if err == nil {
			stats.DeadLetterCount = count
		}
	}

	if !d.startTime.IsZero() {
		stats.Uptime = d.clock.Now().Sub(d.startTime)
	}

	return stats, nil
}

// getOrCreateCircuitBreaker gets or creates a circuit breaker for a
// provider.
//
// Takes ctx (context.Context) which carries logging context for the state
// change callback.
// Takes providerName (string) which identifies the notification provider.
//
// Returns *gobreaker.CircuitBreaker[any] which is the circuit breaker for
// the provider.
//
// Safe for concurrent use. Access is serialised by an internal
// mutex.
func (d *NotificationDispatcher) getOrCreateCircuitBreaker(ctx context.Context, providerName string) *gobreaker.CircuitBreaker[any] {
	d.mu.Lock()
	defer d.mu.Unlock()

	if circuitBreaker, exists := d.circuitBreakers[providerName]; exists {
		return circuitBreaker
	}

	cbSettings := gobreaker.Settings{
		Name:         fmt.Sprintf("notification-provider-%s", providerName),
		MaxRequests:  1,
		Interval:     d.cbConfig.interval,
		Timeout:      d.cbConfig.timeout,
		BucketPeriod: circuitBreakerBucketPeriod,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= safeconv.IntToUint32(d.cbConfig.maxConsecutiveFailures)
		},
		IsExcluded: func(err error) bool {
			return errors.Is(err, context.Canceled) ||
				errors.Is(err, context.DeadlineExceeded)
		},
		OnStateChange: func(_ string, from gobreaker.State, to gobreaker.State) {
			_, cbL := logger_domain.From(ctx, log)
			cbL.Internal("Circuit breaker state changed",
				logger_domain.String("provider", providerName),
				logger_domain.String("from", from.String()),
				logger_domain.String("to", to.String()))
			circuitBreakerStateChangeCount.Add(ctx, 1)
		},
	}

	circuitBreaker := gobreaker.NewCircuitBreaker[any](cbSettings)
	d.circuitBreakers[providerName] = circuitBreaker
	return circuitBreaker
}

// processQueue runs the main loop that collects and sends notifications.
func (d *NotificationDispatcher) processQueue(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	_ = l

	defer d.wg.Done()
	defer goroutine.RecoverPanic(ctx, "notification.processQueue")

	ticker := d.clock.NewTicker(d.flushInterval)
	defer ticker.Stop()

	batch := make([]*notification_dto.SendParams, 0, d.batchSize)

	for {
		select {
		case <-d.shutdownChan:
			if len(batch) > 0 {
				d.processBatch(ctx, batch)
			}
			return

		case <-d.flushChan:
			if len(batch) > 0 {
				d.processBatch(ctx, batch)
				batch = batch[:0]
			}

		case <-ticker.C():
			if len(batch) > 0 {
				d.processBatch(ctx, batch)
				batch = batch[:0]
			}

		case params := <-d.queue:
			batch = append(batch, params)
			if len(batch) >= d.batchSize {
				d.processBatch(ctx, batch)
				batch = batch[:0]
			}
		}
	}
}

// processBatch sends a batch of notifications, handling multi-cast and
// partial failures.
//
// Takes batch ([]*notification_dto.SendParams) which contains the
// notifications to send.
func (d *NotificationDispatcher) processBatch(ctx context.Context, batch []*notification_dto.SendParams) {
	if len(batch) == 0 {
		return
	}

	batchSentCount.Add(ctx, 1)
	batchSizeMetric.Record(ctx, int64(len(batch)))

	for _, params := range batch {
		if ctx.Err() != nil {
			return
		}

		atomic.AddInt64(&d.totalProcessed, 1)

		qn := &queuedNotification{
			params:          params,
			targetProviders: params.Providers,
			attempt:         1,
			firstAttempt:    d.clock.Now(),
		}

		if len(qn.targetProviders) == 0 {
			qn.targetProviders = []string{""}
		}

		d.sendToProviders(ctx, qn)
	}
}

// sendToProviders sends a notification to its target providers and tracks
// partial failures.
//
// Takes qn (*queuedNotification) which contains the notification and targets.
func (d *NotificationDispatcher) sendToProviders(ctx context.Context, qn *queuedNotification) {
	qn.failedProviders = nil

	for _, providerName := range qn.targetProviders {
		if ctx.Err() != nil {
			break
		}
		if err := d.sendToSingleProvider(ctx, providerName, qn.params); err != nil {
			qn.failedProviders = append(qn.failedProviders, providerName)
		}
	}

	if len(qn.failedProviders) == 0 {
		atomic.AddInt64(&d.totalSuccessful, 1)
		notificationSentCount.Add(ctx, 1)
	} else if len(qn.failedProviders) < len(qn.targetProviders) {
		partialFailureCount.Add(ctx, 1)
		d.scheduleRetry(ctx, qn)
	} else {
		d.scheduleRetry(ctx, qn)
	}
}

// sendToSingleProvider sends to one provider with circuit breaker
// protection.
//
// Takes providerName (string) which identifies the target provider.
// Takes params (*notification_dto.SendParams) which contains the notification
// content and recipients.
//
// Returns error when the provider is not found or sending fails.
func (d *NotificationDispatcher) sendToSingleProvider(ctx context.Context, providerName string, params *notification_dto.SendParams) error {
	ctx, l := logger_domain.From(ctx, log)

	provider, err := d.service.getProvider(providerName)
	if err != nil {
		return fmt.Errorf("resolving notification provider %q: %w", providerName, err)
	}

	circuitBreaker := d.getOrCreateCircuitBreaker(ctx, providerName)

	start := d.clock.Now()
	_, sendErr := circuitBreaker.Execute(func() (any, error) {
		return nil, provider.Send(ctx, params)
	})

	duration := d.clock.Now().Sub(start).Milliseconds()
	notificationSendDuration.Record(ctx, float64(duration))

	if sendErr != nil {
		notificationSendErrorCount.Add(ctx, 1)
		l.Warn("Failed to send notification",
			logger_domain.String("provider", providerName),
			logger_domain.String("title", params.Content.Title),
			logger_domain.Error(sendErr))
		return sendErr
	}

	return nil
}

// scheduleRetry adds a failed notification to the retry queue.
//
// Takes qn (*queuedNotification) which is the notification to schedule for
// retry.
//
// Safe for concurrent use. Uses retryMutex to protect the retry
// heap. Sends a non-blocking signal to wake the retry producer goroutine.
func (d *NotificationDispatcher) scheduleRetry(ctx context.Context, qn *queuedNotification) {
	ctx, l := logger_domain.From(ctx, log)

	qn.targetProviders = qn.failedProviders
	qn.failedProviders = nil
	qn.attempt++

	if !d.retryConfig.ShouldRetry(qn.attempt) {
		d.sendToDeadLetter(ctx, qn)
		return
	}

	qn.nextRetryTime = d.retryConfig.CalculateNextRetry(qn.attempt, d.clock.Now())

	d.retryMutex.Lock()
	if d.retryHeap.Len() >= d.maxRetryHeapSize {
		d.retryMutex.Unlock()
		l.Warn("Retry heap full, sending to DLQ",
			logger_domain.Int("heap_size", d.retryHeap.Len()))
		d.sendToDeadLetter(ctx, qn)
		return
	}
	d.retryHeap.PushItem(qn)
	d.retryMutex.Unlock()

	retryScheduledCount.Add(ctx, 1)

	select {
	case d.retrySignal <- struct{}{}:
	default:
	}
}

// sendToDeadLetter sends a failed notification to the dead letter queue.
//
// Takes qn (*queuedNotification) which is the failed notification to store.
func (d *NotificationDispatcher) sendToDeadLetter(ctx context.Context, qn *queuedNotification) {
	ctx, l := logger_domain.From(ctx, log)

	atomic.AddInt64(&d.totalFailed, 1)
	deadLetterCount.Add(ctx, 1)

	if d.deadLetterQueue == nil {
		l.Warn("No dead letter queue configured, notification lost",
			logger_domain.String("title", qn.params.Content.Title),
			logger_domain.Int("attempts", qn.attempt))
		return
	}

	entry := &notification_dto.DeadLetterEntry{
		Params:        *qn.params,
		Providers:     qn.targetProviders,
		TotalAttempts: qn.attempt,
		FirstAttempt:  qn.firstAttempt,
		LastAttempt:   d.clock.Now(),
	}

	if err := d.deadLetterQueue.Add(ctx, entry); err != nil {
		l.ReportError(nil, err, "Failed to add notification to dead letter queue")
	}
}

// produceRetryJobs monitors the retry heap and schedules items for retry.
//
// Takes ctx (context.Context) which carries tracing values with cancellation
// already detached by Start.
//
// Runs until shutdown is signalled. Blocks on the retry signal channel
// when the heap is empty.
func (d *NotificationDispatcher) produceRetryJobs(ctx context.Context) {
	defer d.wg.Done()
	defer goroutine.RecoverPanic(ctx, "notification.produceRetryJobs")

	for {
		action := d.produceRetryJobsStep()
		if action == retryActionShutdown {
			return
		}
	}
}

// produceRetryJobsStep performs a single step of the retry producer loop.
//
// Returns retryProducerAction which indicates what to do next based on the
// retry heap state.
//
// Safe for concurrent use. Acquires the retry mutex to check the heap state.
func (d *NotificationDispatcher) produceRetryJobsStep() retryProducerAction {
	d.retryMutex.Lock()
	if d.retryHeap.Len() == 0 {
		d.retryMutex.Unlock()
		return d.waitForRetrySignal()
	}

	nextItem, _ := d.retryHeap.Peek()
	waitDuration := nextItem.nextRetryTime.Sub(d.clock.Now())
	d.retryMutex.Unlock()

	if waitDuration <= 0 {
		return d.dispatchReadyRetryItem()
	}
	return d.waitForNextRetryTime(waitDuration)
}

// waitForRetrySignal waits until a signal arrives, a timeout passes, or
// shutdown occurs when the heap is empty.
//
// Returns retryProducerAction which shows whether to carry on or shut down.
func (d *NotificationDispatcher) waitForRetrySignal() retryProducerAction {
	sleepTimer := d.clock.NewTimer(emptyQueueSleep)
	select {
	case <-d.shutdownChan:
		sleepTimer.Stop()
		return retryActionShutdown
	case <-d.retrySignal:
		sleepTimer.Stop()
		return retryActionContinue
	case <-sleepTimer.C():
		return retryActionContinue
	}
}

// dispatchReadyRetryItem pops a ready item from the heap and sends it.
//
// Returns retryProducerAction which indicates whether to continue or shutdown.
//
// Safe for concurrent use; protects the retry heap with a mutex.
func (d *NotificationDispatcher) dispatchReadyRetryItem() retryProducerAction {
	d.retryMutex.Lock()
	item, ok := d.retryHeap.PopItem()
	d.retryMutex.Unlock()

	if !ok {
		return retryActionContinue
	}

	select {
	case d.retryJobsChan <- &retryItem{notification: item}:
		return retryActionContinue
	case <-d.shutdownChan:
		return retryActionShutdown
	}
}

// waitForNextRetryTime waits until the next item is ready or an interrupt
// occurs.
//
// Takes waitDuration (time.Duration) which specifies how long to wait before
// the next retry attempt.
//
// Returns retryProducerAction which indicates whether to continue processing
// or shut down.
func (d *NotificationDispatcher) waitForNextRetryTime(waitDuration time.Duration) retryProducerAction {
	waitTimer := d.clock.NewTimer(waitDuration)
	select {
	case <-d.shutdownChan:
		waitTimer.Stop()
		return retryActionShutdown
	case <-d.retrySignal:
		waitTimer.Stop()
		return retryActionContinue
	case <-waitTimer.C():
		return retryActionContinue
	}
}

// retryWorker processes retry jobs from the retry channel.
//
// Runs until shutdown is signalled. Each job increments the retry counter
// and sends the notification to providers.
//
// processing.
func (d *NotificationDispatcher) retryWorker(ctx context.Context) {
	defer d.wg.Done()
	defer goroutine.RecoverPanic(ctx, "notification.retryWorker")

	for {
		select {
		case <-d.shutdownChan:
			return

		case job := <-d.retryJobsChan:
			atomic.AddInt64(&d.totalRetries, 1)
			retryAttemptCount.Add(ctx, 1)

			d.sendToProviders(ctx, job.notification)
		}
	}
}

// applyDispatcherConfigDefaults sets default values for any zero-value fields
// in the config.
//
// Takes config (*notification_dto.DispatcherConfig) which is changed in place
// to have sensible defaults for any unset fields.
func applyDispatcherConfigDefaults(config *notification_dto.DispatcherConfig) {
	if config.BatchSize <= 0 {
		config.BatchSize = defaultBatchSize
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = defaultFlushInterval
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
	if config.CircuitBreakerThreshold <= 0 {
		config.CircuitBreakerThreshold = defaultMaxConsecutiveFailures
	}
	if config.CircuitBreakerTimeout <= 0 {
		config.CircuitBreakerTimeout = defaultCircuitBreakerTimeout
	}
	if config.CircuitBreakerInterval <= 0 {
		config.CircuitBreakerInterval = defaultCircuitBreakerInterval
	}
}
