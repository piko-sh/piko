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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/deadletter/deadletter_domain"
	"piko.sh/piko/internal/notification/notification_dto"
	"piko.sh/piko/internal/retry"
	"piko.sh/piko/wdk/clock"
)

type mockDLQ = deadletter_domain.MockDeadLetterPort[*notification_dto.DeadLetterEntry]

func defaultTestConfig() *notification_dto.DispatcherConfig {
	return &notification_dto.DispatcherConfig{
		BatchSize:               2,
		FlushInterval:           100 * time.Millisecond,
		MaxRetries:              2,
		InitialDelay:            10 * time.Millisecond,
		MaxDelay:                100 * time.Millisecond,
		BackoffFactor:           2.0,
		CircuitBreakerThreshold: 5,
		CircuitBreakerTimeout:   1 * time.Second,
		CircuitBreakerInterval:  1 * time.Second,
	}
}

func newTestDispatcher(t *testing.T, provider *mockProvider, dlq *mockDLQ) (*NotificationDispatcher, Service) {
	t.Helper()
	service := NewService()
	if provider != nil {
		if err := service.RegisterProvider("test", provider); err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}
	}

	var dlqPort DeadLetterPort
	if dlq != nil {
		dlqPort = dlq
	}

	disp := NewNotificationDispatcher(service, dlqPort, defaultTestConfig())
	if disp == nil {
		t.Fatal("expected non-nil dispatcher")
	}
	return disp, service
}

func TestNewNotificationDispatcher_ValidService(t *testing.T) {
	service := NewService()
	disp := NewNotificationDispatcher(service, nil, defaultTestConfig())
	if disp == nil {
		t.Fatal("expected non-nil dispatcher")
	}
}

func TestNewNotificationDispatcher_NilService(t *testing.T) {
	disp := NewNotificationDispatcher(nil, nil, defaultTestConfig())
	if disp != nil {
		t.Error("expected nil for nil service")
	}
}

type fakeService struct{}

func (*fakeService) NewNotification() *NotificationBuilder { return nil }
func (*fakeService) SendBulk(context.Context, []*notification_dto.SendParams) error {
	return nil
}
func (*fakeService) SendBulkWithProvider(context.Context, string, []*notification_dto.SendParams) error {
	return nil
}
func (*fakeService) SendToProviders(context.Context, *notification_dto.SendParams, []string) error {
	return nil
}
func (*fakeService) RegisterProvider(string, NotificationProviderPort) error { return nil }
func (*fakeService) SetDefaultProvider(string) error                         { return nil }
func (*fakeService) GetProviders() []string                                  { return nil }
func (*fakeService) HasProvider(string) bool                                 { return false }
func (*fakeService) RegisterDispatcher(NotificationDispatcherPort) error     { return nil }
func (*fakeService) FlushDispatcher(context.Context) error {
	return nil
}
func (*fakeService) Close(context.Context) error { return nil }

var _ Service = (*fakeService)(nil)

func TestNewNotificationDispatcher_NonServiceImpl(t *testing.T) {
	disp := NewNotificationDispatcher(&fakeService{}, nil, defaultTestConfig())
	if disp != nil {
		t.Error("expected nil for non-*service implementation")
	}
}

func TestRetryHeap_Len(t *testing.T) {
	h := retry.NewHeap(func(qn *queuedNotification) time.Time { return qn.nextRetryTime })
	if h.Len() != 0 {
		t.Errorf("expected 0, got %d", h.Len())
	}
	h.PushItem(&queuedNotification{})
	if h.Len() != 1 {
		t.Errorf("expected 1, got %d", h.Len())
	}
}

func TestRetryHeap_Ordering(t *testing.T) {
	now := time.Now()
	h := retry.NewHeap(func(qn *queuedNotification) time.Time { return qn.nextRetryTime })

	h.PushItem(&queuedNotification{nextRetryTime: now.Add(2 * time.Second)})
	h.PushItem(&queuedNotification{nextRetryTime: now.Add(1 * time.Second)})

	first, ok := h.Peek()
	if !ok {
		t.Fatal("expected Peek to succeed")
	}
	if !first.nextRetryTime.Equal(now.Add(1 * time.Second)) {
		t.Error("expected earliest item first")
	}
}

func TestRetryHeap_PushPop(t *testing.T) {
	h := retry.NewHeap(func(qn *queuedNotification) time.Time { return qn.nextRetryTime })

	now := time.Now()
	items := []*queuedNotification{
		{nextRetryTime: now.Add(3 * time.Second), params: testParams("third")},
		{nextRetryTime: now.Add(1 * time.Second), params: testParams("first")},
		{nextRetryTime: now.Add(2 * time.Second), params: testParams("second")},
	}

	for _, item := range items {
		h.PushItem(item)
	}

	if h.Len() != 3 {
		t.Fatalf("expected 3 items, got %d", h.Len())
	}

	first, ok := h.PopItem()
	require.True(t, ok, "expected PopItem to succeed")
	if first.params.Content.Title != "first" {
		t.Errorf("expected 'first', got %q", first.params.Content.Title)
	}
}

func TestRetryHeap_PopEmpty(t *testing.T) {
	h := retry.NewHeap(func(qn *queuedNotification) time.Time { return qn.nextRetryTime })
	_, ok := h.PopItem()
	if ok {
		t.Error("expected PopItem on empty heap to return false")
	}
}

func TestApplyDispatcherConfigDefaults_AllZeros(t *testing.T) {
	config := &notification_dto.DispatcherConfig{}
	applyDispatcherConfigDefaults(config)

	if config.BatchSize != defaultBatchSize {
		t.Errorf("expected batch size %d, got %d", defaultBatchSize, config.BatchSize)
	}
	if config.FlushInterval != defaultFlushInterval {
		t.Errorf("expected flush interval %v, got %v", defaultFlushInterval, config.FlushInterval)
	}

	if config.MaxRetries != 0 {
		t.Errorf("expected max retries 0 (zero is valid), got %d", config.MaxRetries)
	}
	if config.InitialDelay != defaultInitialDelay {
		t.Errorf("expected initial delay %v, got %v", defaultInitialDelay, config.InitialDelay)
	}
	if config.MaxDelay != defaultMaxDelay {
		t.Errorf("expected max delay %v, got %v", defaultMaxDelay, config.MaxDelay)
	}
	if config.BackoffFactor != defaultBackoffFactor {
		t.Errorf("expected backoff factor %v, got %v", defaultBackoffFactor, config.BackoffFactor)
	}
	if config.CircuitBreakerThreshold != defaultMaxConsecutiveFailures {
		t.Errorf("expected CB threshold %d, got %d", defaultMaxConsecutiveFailures, config.CircuitBreakerThreshold)
	}
}

func TestApplyDispatcherConfigDefaults_PreservesOverrides(t *testing.T) {
	config := &notification_dto.DispatcherConfig{
		BatchSize:     42,
		FlushInterval: 5 * time.Second,
		MaxRetries:    10,
	}
	applyDispatcherConfigDefaults(config)

	if config.BatchSize != 42 {
		t.Errorf("expected 42, got %d", config.BatchSize)
	}
	if config.FlushInterval != 5*time.Second {
		t.Errorf("expected 5s, got %v", config.FlushInterval)
	}
	if config.MaxRetries != 10 {
		t.Errorf("expected 10, got %d", config.MaxRetries)
	}

	if config.InitialDelay != defaultInitialDelay {
		t.Errorf("expected default initial delay, got %v", config.InitialDelay)
	}
}

func TestQueue_NotRunning(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	ctx := context.Background()
	err := disp.Queue(ctx, testParams("test"))
	if !errors.Is(err, ErrDispatcherNotRunning) {
		t.Errorf("expected ErrDispatcherNotRunning, got %v", err)
	}
}

func TestFlush_NotRunning(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	ctx := context.Background()
	err := disp.Flush(ctx)
	if !errors.Is(err, ErrDispatcherNotRunning) {
		t.Errorf("expected ErrDispatcherNotRunning, got %v", err)
	}
}

func TestQueue_ContextCancelled(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)

	disp.mu.Lock()
	disp.isRunning = true
	disp.mu.Unlock()
	disp.queue = make(chan *notification_dto.SendParams)

	cancelCtx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	err := disp.Queue(cancelCtx, testParams("cancelled"))
	if err == nil {
		t.Error("expected error from cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestStart_Success(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	ctx := context.Background()
	if err := disp.Start(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = disp.Stop(ctx) })
}

func TestStart_AlreadyRunning(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	ctx := context.Background()
	_ = disp.Start(ctx)
	t.Cleanup(func() { _ = disp.Stop(ctx) })

	err := disp.Start(ctx)
	if !errors.Is(err, ErrDispatcherAlreadyRunning) {
		t.Errorf("expected ErrDispatcherAlreadyRunning, got %v", err)
	}
}

func TestStop_NotRunning(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	ctx := context.Background()
	err := disp.Stop(ctx)
	if !errors.Is(err, ErrDispatcherNotRunning) {
		t.Errorf("expected ErrDispatcherNotRunning, got %v", err)
	}
}

func TestStop_Success(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	ctx := context.Background()
	_ = disp.Start(ctx)

	err := disp.Stop(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetBatchSize(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	disp.SetBatchSize(42)
	if disp.batchSize != 42 {
		t.Errorf("expected 42, got %d", disp.batchSize)
	}
}

func TestSetFlushInterval(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	disp.SetFlushInterval(5 * time.Minute)
	if disp.flushInterval != 5*time.Minute {
		t.Errorf("expected 5m, got %v", disp.flushInterval)
	}
}

func TestSetRetryConfig_GetRetryConfig(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	rc := RetryConfig{
		MaxRetries:    10,
		InitialDelay:  1 * time.Second,
		MaxDelay:      1 * time.Minute,
		BackoffFactor: 3.0,
	}
	disp.SetRetryConfig(rc)
	got := disp.GetRetryConfig()
	if got.MaxRetries != 10 || got.BackoffFactor != 3.0 {
		t.Errorf("unexpected retry config: %+v", got)
	}
}

func TestGetDeadLetterQueue(t *testing.T) {
	dlq := &mockDLQ{}
	disp, _ := newTestDispatcher(t, &mockProvider{}, dlq)
	if disp.GetDeadLetterQueue() != dlq {
		t.Error("expected the mock DLQ to be returned")
	}
}

func TestGetDeadLetterCount_NilDLQ(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	ctx := context.Background()
	count, err := disp.GetDeadLetterCount(ctx)
	if err != nil || count != 0 {
		t.Errorf("expected (0, nil), got (%d, %v)", count, err)
	}
}

func TestGetDeadLetterCount_WithDLQ(t *testing.T) {
	dlq := &mockDLQ{
		CountFunc: func(_ context.Context) (int, error) {
			return 5, nil
		},
	}
	disp, _ := newTestDispatcher(t, &mockProvider{}, dlq)
	ctx := context.Background()
	count, err := disp.GetDeadLetterCount(ctx)
	if err != nil || count != 5 {
		t.Errorf("expected (5, nil), got (%d, %v)", count, err)
	}
}

func TestClearDeadLetterQueue_NilDLQ(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	ctx := context.Background()
	err := disp.ClearDeadLetterQueue(ctx)
	if err != nil {
		t.Errorf("expected nil error for nil DLQ, got %v", err)
	}
}

func TestClearDeadLetterQueue_WithDLQ(t *testing.T) {
	cleared := false
	dlq := &mockDLQ{
		ClearFunc: func(_ context.Context) error {
			cleared = true
			return nil
		},
	}
	disp, _ := newTestDispatcher(t, &mockProvider{}, dlq)
	ctx := context.Background()
	_ = disp.ClearDeadLetterQueue(ctx)
	if !cleared {
		t.Error("expected Clear to be called")
	}
}

func TestGetRetryQueueSize(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	ctx := context.Background()
	size, err := disp.GetRetryQueueSize(ctx)
	if err != nil || size != 0 {
		t.Errorf("expected (0, nil), got (%d, %v)", size, err)
	}
}

func TestGetProcessingStats(t *testing.T) {
	mockClk := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	dlq := &mockDLQ{
		CountFunc: func(_ context.Context) (int, error) {
			return 3, nil
		},
	}
	disp, _ := newTestDispatcher(t, &mockProvider{}, dlq)
	disp.clock = mockClk

	ctx := context.Background()

	if err := disp.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = disp.Stop(ctx) })

	mockClk.Advance(5 * time.Minute)

	stats, err := disp.GetProcessingStats(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.DeadLetterCount != 3 {
		t.Errorf("expected DLQ count 3, got %d", stats.DeadLetterCount)
	}
	if stats.Uptime != 5*time.Minute {
		t.Errorf("expected 5m uptime, got %v", stats.Uptime)
	}
}

func TestGetProcessingStats_NilDLQ(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	ctx := context.Background()
	stats, err := disp.GetProcessingStats(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.DeadLetterCount != 0 {
		t.Errorf("expected 0, got %d", stats.DeadLetterCount)
	}
}

func TestQueue_Success(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	ctx := context.Background()
	_ = disp.Start(ctx)
	t.Cleanup(func() { _ = disp.Stop(ctx) })

	err := disp.Queue(ctx, testParams("test"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFlush_Success(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	ctx := context.Background()
	_ = disp.Start(ctx)
	t.Cleanup(func() { _ = disp.Stop(ctx) })

	err := disp.Flush(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDispatcher_ProcessesBatchOnSize(t *testing.T) {
	var sendCount int64
	provider := &mockProvider{
		SendFunc: func(_ context.Context, _ *notification_dto.SendParams) error {
			atomic.AddInt64(&sendCount, 1)
			return nil
		},
	}

	disp, _ := newTestDispatcher(t, provider, nil)
	ctx := context.Background()
	_ = disp.Start(ctx)
	t.Cleanup(func() { _ = disp.Stop(ctx) })

	_ = disp.Queue(ctx, testParams("one"))
	_ = disp.Queue(ctx, testParams("two"))

	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for batch processing, send count: %d", atomic.LoadInt64(&sendCount))
		default:
			if atomic.LoadInt64(&sendCount) >= 2 {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestDispatcher_ProcessesOnFlush(t *testing.T) {
	var sendCount int64
	provider := &mockProvider{
		SendFunc: func(_ context.Context, _ *notification_dto.SendParams) error {
			atomic.AddInt64(&sendCount, 1)
			return nil
		},
	}

	disp, _ := newTestDispatcher(t, provider, nil)
	disp.batchSize = 100
	ctx := context.Background()
	_ = disp.Start(ctx)
	t.Cleanup(func() { _ = disp.Stop(ctx) })

	_ = disp.Queue(ctx, testParams("one"))

	time.Sleep(20 * time.Millisecond)
	_ = disp.Flush(ctx)

	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for flush processing, send count: %d", atomic.LoadInt64(&sendCount))
		default:
			if atomic.LoadInt64(&sendCount) >= 1 {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestDispatcher_SuccessfulSendIncrementsStats(t *testing.T) {
	provider := &mockProvider{}
	disp, _ := newTestDispatcher(t, provider, nil)
	ctx := context.Background()
	_ = disp.Start(ctx)
	t.Cleanup(func() { _ = disp.Stop(ctx) })

	_ = disp.Queue(ctx, testParams("test"))
	_ = disp.Queue(ctx, testParams("test2"))

	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for stats")
		default:
			if atomic.LoadInt64(&disp.totalProcessed) >= 2 && atomic.LoadInt64(&disp.totalSuccessful) >= 2 {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestDispatcher_FailingSendGoesToDeadLetter(t *testing.T) {
	provider := &mockProvider{
		SendFunc: func(_ context.Context, _ *notification_dto.SendParams) error {
			return errors.New("always fail")
		},
	}
	dlq := &mockDLQ{}

	service := NewService()
	_ = service.RegisterProvider("test", provider)
	config := defaultTestConfig()
	config.MaxRetries = 0
	disp := NewNotificationDispatcher(service, dlq, config)
	if disp == nil {
		t.Fatal("nil dispatcher")
	}

	ctx := context.Background()
	_ = disp.Start(ctx)
	t.Cleanup(func() { _ = disp.Stop(ctx) })

	_ = disp.Queue(ctx, testParams("fail"))

	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for DLQ, add count: %d, totalFailed: %d",
				atomic.LoadInt64(&dlq.AddCallCount), atomic.LoadInt64(&disp.totalFailed))
		default:
			if atomic.LoadInt64(&dlq.AddCallCount) >= 1 {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestDispatcher_FailingSendNoDLQ(t *testing.T) {
	provider := &mockProvider{
		SendFunc: func(_ context.Context, _ *notification_dto.SendParams) error {
			return errors.New("always fail")
		},
	}

	service := NewService()
	_ = service.RegisterProvider("test", provider)
	config := defaultTestConfig()
	config.MaxRetries = 0
	disp := NewNotificationDispatcher(service, nil, config)
	if disp == nil {
		t.Fatal("nil dispatcher")
	}

	ctx := context.Background()
	_ = disp.Start(ctx)
	t.Cleanup(func() { _ = disp.Stop(ctx) })

	_ = disp.Queue(ctx, testParams("fail"))

	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out, totalFailed: %d", atomic.LoadInt64(&disp.totalFailed))
		default:
			if atomic.LoadInt64(&disp.totalFailed) >= 1 {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestGetOrCreateCircuitBreaker(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)

	ctx := context.Background()

	cb1 := disp.getOrCreateCircuitBreaker(ctx, "test-provider")
	if cb1 == nil {
		t.Fatal("expected non-nil circuit breaker")
	}

	cb2 := disp.getOrCreateCircuitBreaker(ctx, "test-provider")
	if cb1 != cb2 {
		t.Error("expected same circuit breaker for same provider")
	}

	cb3 := disp.getOrCreateCircuitBreaker(ctx, "other-provider")
	if cb1 == cb3 {
		t.Error("expected different circuit breaker for different provider")
	}
}

func TestScheduleRetry_WithinMaxRetries(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)

	qn := &queuedNotification{
		params:          testParams("retry-me"),
		targetProviders: []string{"test"},
		failedProviders: []string{"test"},
		attempt:         1,
		firstAttempt:    time.Now(),
	}

	disp.scheduleRetry(context.Background(), qn)

	disp.retryMutex.Lock()
	size := disp.retryHeap.Len()
	disp.retryMutex.Unlock()

	if size != 1 {
		t.Errorf("expected 1 item in retry heap, got %d", size)
	}
}

func TestScheduleRetry_ExceedsMaxRetries(t *testing.T) {
	dlq := &mockDLQ{}
	disp, _ := newTestDispatcher(t, &mockProvider{}, dlq)

	qn := &queuedNotification{
		params:          testParams("dead"),
		targetProviders: []string{"test"},
		failedProviders: []string{"test"},
		attempt:         disp.retryConfig.MaxRetries,
		firstAttempt:    time.Now(),
	}

	disp.scheduleRetry(context.Background(), qn)

	if atomic.LoadInt64(&dlq.AddCallCount) != 1 {
		t.Errorf("expected 1 DLQ add call, got %d", atomic.LoadInt64(&dlq.AddCallCount))
	}
}

func TestScheduleRetry_HeapFull(t *testing.T) {
	dlq := &mockDLQ{}
	disp, _ := newTestDispatcher(t, &mockProvider{}, dlq)
	disp.maxRetryHeapSize = 0

	qn := &queuedNotification{
		params:          testParams("overflow"),
		targetProviders: []string{"test"},
		failedProviders: []string{"test"},
		attempt:         0,
		firstAttempt:    time.Now(),
	}

	disp.scheduleRetry(context.Background(), qn)

	if atomic.LoadInt64(&dlq.AddCallCount) != 1 {
		t.Errorf("expected 1 DLQ add, got %d", atomic.LoadInt64(&dlq.AddCallCount))
	}
}

func TestSendToDeadLetter_WithDLQ(t *testing.T) {
	var addedEntry *notification_dto.DeadLetterEntry
	dlq := &mockDLQ{
		AddFunc: func(_ context.Context, entry *notification_dto.DeadLetterEntry) error {
			addedEntry = entry
			return nil
		},
	}
	disp, _ := newTestDispatcher(t, &mockProvider{}, dlq)

	qn := &queuedNotification{
		params:          testParams("dead-letter"),
		targetProviders: []string{"test"},
		attempt:         3,
		firstAttempt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	disp.sendToDeadLetter(context.Background(), qn)

	if addedEntry == nil {
		t.Fatal("expected entry to be added to DLQ")
	}
	if addedEntry.TotalAttempts != 3 {
		t.Errorf("expected 3 total attempts, got %d", addedEntry.TotalAttempts)
	}
	if addedEntry.Params.Content.Title != "dead-letter" {
		t.Errorf("expected title %q, got %q", "dead-letter", addedEntry.Params.Content.Title)
	}
}

func TestSendToDeadLetter_DLQError(t *testing.T) {
	dlq := &mockDLQ{
		AddFunc: func(_ context.Context, _ *notification_dto.DeadLetterEntry) error {
			return errors.New("dlq full")
		},
	}
	disp, _ := newTestDispatcher(t, &mockProvider{}, dlq)

	qn := &queuedNotification{
		params:          testParams("err"),
		targetProviders: []string{"test"},
		attempt:         1,
		firstAttempt:    time.Now(),
	}

	disp.sendToDeadLetter(context.Background(), qn)

	if atomic.LoadInt64(&disp.totalFailed) != 1 {
		t.Errorf("expected totalFailed=1, got %d", atomic.LoadInt64(&disp.totalFailed))
	}
}

func TestSendToDeadLetter_NilDLQ(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)

	qn := &queuedNotification{
		params:          testParams("lost"),
		targetProviders: []string{"test"},
		attempt:         1,
		firstAttempt:    time.Now(),
	}

	disp.sendToDeadLetter(context.Background(), qn)

	if atomic.LoadInt64(&disp.totalFailed) != 1 {
		t.Errorf("expected totalFailed=1, got %d", atomic.LoadInt64(&disp.totalFailed))
	}
}

func TestProduceRetryJobsStep_EmptyHeap(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	disp.shutdownChan = make(chan struct{})

	go func() {
		time.Sleep(10 * time.Millisecond)
		disp.retrySignal <- struct{}{}
	}()

	action := disp.produceRetryJobsStep()
	if action != retryActionContinue {
		t.Errorf("expected retryActionContinue, got %v", action)
	}
}

func TestProduceRetryJobsStep_Shutdown(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	disp.shutdownChan = make(chan struct{})

	go func() {
		time.Sleep(10 * time.Millisecond)
		close(disp.shutdownChan)
	}()

	action := disp.produceRetryJobsStep()
	if action != retryActionShutdown {
		t.Errorf("expected retryActionShutdown, got %v", action)
	}
}

func TestDispatchReadyRetryItem(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	disp.shutdownChan = make(chan struct{})
	disp.retryJobsChan = make(chan *retryItem, 10)

	qn := &queuedNotification{
		params:          testParams("ready"),
		targetProviders: []string{"test"},
		nextRetryTime:   time.Now().Add(-1 * time.Second),
	}
	disp.retryHeap.PushItem(qn)

	action := disp.dispatchReadyRetryItem()
	if action != retryActionContinue {
		t.Errorf("expected retryActionContinue, got %v", action)
	}

	select {
	case job := <-disp.retryJobsChan:
		if job.notification.params.Content.Title != "ready" {
			t.Errorf("expected 'ready', got %q", job.notification.params.Content.Title)
		}
	default:
		t.Error("expected item on retry jobs channel")
	}
}

func TestFlush_ContextCancelledBeforeStart(t *testing.T) {
	disp, _ := newTestDispatcher(t, &mockProvider{}, nil)
	ctx := context.Background()

	cancelCtx, cancel := context.WithCancelCause(ctx)
	cancel(fmt.Errorf("test: simulating cancelled context"))
	err := disp.Flush(cancelCtx)
	if !errors.Is(err, ErrDispatcherNotRunning) {
		t.Errorf("expected ErrDispatcherNotRunning, got %v", err)
	}
}
