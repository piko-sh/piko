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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/retry"
	"piko.sh/piko/wdk/clock"
)

type controlledFakeProvider struct {
	failNTimes        map[string]int
	permanentFailures map[string]bool
	sendAttempts      map[string][]time.Time
	mu                sync.Mutex
}

func newControlledFakeProvider() *controlledFakeProvider {
	return &controlledFakeProvider{
		failNTimes:        make(map[string]int),
		permanentFailures: make(map[string]bool),
		sendAttempts:      make(map[string][]time.Time),
	}
}

func (p *controlledFakeProvider) Send(ctx context.Context, params *email_dto.SendParams) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := params.Subject
	p.sendAttempts[key] = append(p.sendAttempts[key], time.Now())

	if p.permanentFailures[key] {
		return errors.New("permanent provider failure")
	}

	if n, ok := p.failNTimes[key]; ok && n > 0 {
		p.failNTimes[key] = n - 1
		return fmt.Errorf("transient provider failure (remaining: %d)", n-1)
	}

	return nil
}

func (p *controlledFakeProvider) SendBulk(ctx context.Context, emails []*email_dto.SendParams) error {
	return errors.New("SendBulk not implemented for controlledFakeProvider")
}

func (p *controlledFakeProvider) SupportsBulkSending() bool { return false }
func (p *controlledFakeProvider) Close(_ context.Context) error {
	return nil
}

func (p *controlledFakeProvider) setFailNTimes(subject string, n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failNTimes[subject] = n
}

func (p *controlledFakeProvider) setPermanentFailure(subject string, fail bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.permanentFailures[subject] = fail
}

func (p *controlledFakeProvider) getSendAttempts(subject string) []time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	attemptsCopy := make([]time.Time, len(p.sendAttempts[subject]))
	copy(attemptsCopy, p.sendAttempts[subject])
	return attemptsCopy
}

type controlledFakeDLQ struct {
	entries      []*email_dto.DeadLetterEntry
	addCallCount int
	mu           sync.Mutex
	failAdd      bool
}

func newControlledFakeDLQ() *controlledFakeDLQ {
	return &controlledFakeDLQ{
		entries: make([]*email_dto.DeadLetterEntry, 0),
	}
}

func (d *controlledFakeDLQ) Add(_ context.Context, entry *email_dto.DeadLetterEntry) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.addCallCount++
	if d.failAdd {
		return errors.New("failed to add to DLQ")
	}
	d.entries = append(d.entries, entry)
	return nil
}

func (d *controlledFakeDLQ) Count(_ context.Context) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.entries), nil
}

func (d *controlledFakeDLQ) Get(_ context.Context, _ int) ([]*email_dto.DeadLetterEntry, error) {
	return nil, nil
}
func (d *controlledFakeDLQ) Remove(_ context.Context, _ []*email_dto.DeadLetterEntry) error {
	return nil
}
func (d *controlledFakeDLQ) Clear(_ context.Context) error { return nil }
func (d *controlledFakeDLQ) GetOlderThan(_ context.Context, _ time.Duration) ([]*email_dto.DeadLetterEntry, error) {
	return nil, nil
}

func (d *controlledFakeDLQ) getEntries() []*email_dto.DeadLetterEntry {
	d.mu.Lock()
	defer d.mu.Unlock()
	entriesCopy := make([]*email_dto.DeadLetterEntry, len(d.entries))
	copy(entriesCopy, d.entries)
	return entriesCopy
}

func (d *controlledFakeDLQ) getAddCallCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.addCallCount
}

func noJitter(_ time.Duration) time.Duration { return 0 }

func setupDispatcherTest(t *testing.T, config *email_dto.DispatcherConfig) (d *EmailDispatcher, provider *controlledFakeProvider, dlq *controlledFakeDLQ, ctx context.Context, cleanup func()) {
	config.JitterFunc = noJitter

	provider = newControlledFakeProvider()
	dlq = newControlledFakeDLQ()

	config.DeadLetterQueue = true
	d = NewEmailDispatcher(provider, dlq, config)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 10*time.Second, fmt.Errorf("test: retry setup exceeded %s timeout", 10*time.Second))
	require.NoError(t, d.Start(ctx))

	cleanup = func() {
		require.NoError(t, d.Stop(ctx))
		cancel()
	}

	return d, provider, dlq, ctx, cleanup
}

func TestDispatcher_Retry_SuccessAfterTransientFailures(t *testing.T) {
	t.Parallel()
	config := email_dto.DispatcherConfig{
		BatchSize:     1,
		FlushInterval: 100 * time.Millisecond,
		MaxRetries:    3,
		InitialDelay:  10 * time.Millisecond,
	}
	d, provider, dlq, ctx, cleanup := setupDispatcherTest(t, &config)
	defer cleanup()

	subject := "retry-success"
	provider.setFailNTimes(subject, 2)
	email := &email_dto.SendParams{To: []string{"test@example.com"}, Subject: subject}

	require.NoError(t, d.Queue(ctx, email))

	require.Eventually(t, func() bool {
		stats, _ := d.GetProcessingStats(ctx)
		return stats.TotalSuccessful == 1
	}, 3*time.Second, 20*time.Millisecond, "Email should have been successfully sent after retries")

	stats, err := d.GetProcessingStats(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), stats.TotalSuccessful, "Should have 1 successful email")
	require.Equal(t, int64(0), stats.TotalFailed, "Should have 0 failed emails")
	require.Equal(t, int64(2), stats.TotalRetries, "Should have made 2 retry attempts")
	require.Len(t, provider.getSendAttempts(subject), 3, "Provider's Send method should have been called 3 times")
	require.Len(t, dlq.getEntries(), 0, "DLQ should be empty")
}

func TestDispatcher_Retry_PermanentFailureSendsToDLQ(t *testing.T) {
	t.Parallel()
	config := email_dto.DispatcherConfig{
		BatchSize:     1,
		FlushInterval: 100 * time.Millisecond,
		MaxRetries:    2,
		InitialDelay:  10 * time.Millisecond,
	}
	d, provider, dlq, ctx, cleanup := setupDispatcherTest(t, &config)
	defer cleanup()

	subject := "permanent-failure"
	provider.setPermanentFailure(subject, true)
	email := &email_dto.SendParams{To: []string{"test@example.com"}, Subject: subject}

	require.NoError(t, d.Queue(ctx, email))

	require.Eventually(t, func() bool {
		return len(dlq.getEntries()) == 1
	}, 3*time.Second, 20*time.Millisecond, "Email should have been sent to the DLQ")

	stats, err := d.GetProcessingStats(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(0), stats.TotalSuccessful, "Should have 0 successful emails")
	require.Equal(t, int64(1), stats.TotalFailed, "Should have 1 failed email")
	require.Equal(t, int64(2), stats.TotalRetries, "Should have made 2 retry attempts")
	require.Len(t, provider.getSendAttempts(subject), 3, "Provider should be called 3 times")

	dlqEntries := dlq.getEntries()
	require.Len(t, dlqEntries, 1)
	require.Equal(t, subject, dlqEntries[0].Email.Subject)
	require.Equal(t, 3, dlqEntries[0].TotalAttempts, "DLQ entry should record 3 total attempts")
	require.Contains(t, dlqEntries[0].OriginalError, "permanent provider failure")
}

func TestDispatcher_Retry_MaxRetriesZero_FailsImmediatelyToDLQ(t *testing.T) {
	t.Parallel()
	config := email_dto.DispatcherConfig{
		BatchSize:     1,
		FlushInterval: 100 * time.Millisecond,
		MaxRetries:    0,
	}
	d, provider, dlq, ctx, cleanup := setupDispatcherTest(t, &config)
	defer cleanup()

	subject := "no-retries"
	provider.setFailNTimes(subject, 5)
	email := &email_dto.SendParams{To: []string{"test@example.com"}, Subject: subject}

	require.NoError(t, d.Queue(ctx, email))

	require.Eventually(t, func() bool {
		return len(dlq.getEntries()) == 1
	}, 3*time.Second, 20*time.Millisecond, "Email should go to DLQ without retries")

	stats, err := d.GetProcessingStats(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(0), stats.TotalRetries, "Should be exactly 0 retries")
	require.Len(t, provider.getSendAttempts(subject), 1, "Provider should be called only once")
	require.Equal(t, 1, dlq.getEntries()[0].TotalAttempts, "DLQ entry should record 1 total attempt")
}

func TestDispatcher_Retry_ExponentialBackoffTimingIsCorrect(t *testing.T) {
	t.Parallel()

	mockClk := clock.NewMockClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	config := email_dto.DispatcherConfig{
		BatchSize:     1,
		FlushInterval: 100 * time.Millisecond,
		MaxRetries:    3,
		InitialDelay:  50 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
		Clock:         mockClk,
	}
	d, provider, _, ctx, cleanup := setupDispatcherTest(t, &config)
	defer cleanup()

	subject := "backoff-test"
	provider.setFailNTimes(subject, 3)
	email := &email_dto.SendParams{To: []string{"test@example.com"}, Subject: subject}

	require.NoError(t, d.Queue(ctx, email))

	mockClk.Advance(config.FlushInterval)
	waitForAttempts(t, provider, subject, 1, "First send attempt")
	waitForRetryReady(t, ctx, d, mockClk)

	mockClk.Advance(config.InitialDelay)
	waitForAttempts(t, provider, subject, 2, "First retry (InitialDelay)")
	waitForRetryReady(t, ctx, d, mockClk)

	mockClk.Advance(time.Duration(float64(config.InitialDelay) * config.BackoffFactor))
	waitForAttempts(t, provider, subject, 3, "Second retry (InitialDelay * BackoffFactor)")
	waitForRetryReady(t, ctx, d, mockClk)

	mockClk.Advance(time.Duration(float64(config.InitialDelay) * config.BackoffFactor * config.BackoffFactor))
	waitForAttempts(t, provider, subject, 4, "Third retry (InitialDelay * BackoffFactor^2)")
}

func waitForRetryReady(t *testing.T, ctx context.Context, d *EmailDispatcher, clk *clock.MockClock) {
	t.Helper()
	require.Eventually(t, func() bool {
		stats, err := d.GetProcessingStats(ctx)
		return err == nil && stats.RetryQueueSize > 0
	}, 5*time.Second, time.Millisecond, "retry item should be queued")

	snap := clk.TimerCount()
	clk.AwaitTimerSetup(snap, 2*time.Second)
}

func waitForAttempts(t *testing.T, provider *controlledFakeProvider, subject string, expected int, message string) {
	t.Helper()
	require.Eventually(t, func() bool {
		return len(provider.getSendAttempts(subject)) >= expected
	}, 5*time.Second, 5*time.Millisecond, message)
}

func TestDispatcher_Retry_ConcurrencyWithMultipleFailingEmails(t *testing.T) {
	t.Parallel()
	config := email_dto.DispatcherConfig{
		BatchSize:        5,
		FlushInterval:    50 * time.Millisecond,
		RetryWorkerCount: 4,
		MaxRetries:       1,
		InitialDelay:     20 * time.Millisecond,
	}
	d, provider, dlq, ctx, cleanup := setupDispatcherTest(t, &config)
	defer cleanup()

	numEmails := 10
	for i := range numEmails {
		subject := fmt.Sprintf("concurrent-fail-%d", i)
		provider.setPermanentFailure(subject, true)
		email := &email_dto.SendParams{To: []string{"test@example.com"}, Subject: subject}
		require.NoError(t, d.Queue(ctx, email))
	}

	require.Eventually(t, func() bool {
		return len(dlq.getEntries()) == numEmails
	}, 5*time.Second, 50*time.Millisecond, "All %d emails should end up in the DLQ", numEmails)

	stats, err := d.GetProcessingStats(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(numEmails), stats.TotalFailed)
	require.Equal(t, int64(numEmails), stats.TotalRetries, "Each email should have been retried once")
}

func TestDispatcher_Retry_HeapFullFailFast_SendsToDLQ(t *testing.T) {
	t.Parallel()
	config := email_dto.DispatcherConfig{
		BatchSize:        1,
		FlushInterval:    20 * time.Millisecond,
		MaxRetries:       5,
		InitialDelay:     500 * time.Millisecond,
		MaxRetryHeapSize: 2,
	}
	d, provider, dlq, ctx, cleanup := setupDispatcherTest(t, &config)
	defer cleanup()

	for i := 0; i < config.MaxRetryHeapSize; i++ {
		subject := fmt.Sprintf("fill-heap-%d", i)
		provider.setPermanentFailure(subject, true)
		require.NoError(t, d.Queue(ctx, &email_dto.SendParams{Subject: subject}))
	}

	require.Eventually(t, func() bool {
		size, _ := d.GetRetryQueueSize(ctx)
		return size == config.MaxRetryHeapSize
	}, 3*time.Second, 20*time.Millisecond, "Retry heap should fill up")

	failFastSubject := "fail-fast-email"
	provider.setPermanentFailure(failFastSubject, true)
	require.NoError(t, d.Queue(ctx, &email_dto.SendParams{Subject: failFastSubject}))

	require.Eventually(t, func() bool {
		return len(dlq.getEntries()) == 1
	}, 3*time.Second, 20*time.Millisecond, "The extra email should fail-fast to the DLQ")

	dlqEntries := dlq.getEntries()
	require.Len(t, dlqEntries, 1)
	require.Equal(t, failFastSubject, dlqEntries[0].Email.Subject)
	require.Equal(t, 1, dlqEntries[0].TotalAttempts, "Fail-fast email should have only 1 attempt")
	require.Contains(t, dlqEntries[0].OriginalError, "retry heap full", "Error message should indicate fail-fast reason")

	retryQueueSize, _ := d.GetRetryQueueSize(ctx)
	require.Equal(t, config.MaxRetryHeapSize, retryQueueSize, "Retry heap should remain full")
}

func TestDispatcher_Retry_ShutdownPersistsPendingRetriesToDLQ(t *testing.T) {
	t.Parallel()
	config := email_dto.DispatcherConfig{
		BatchSize:     1,
		FlushInterval: 20 * time.Millisecond,
		MaxRetries:    5,
		InitialDelay:  2 * time.Second,
	}
	d, provider, dlq, ctx, cleanup := setupDispatcherTest(t, &config)
	defer cleanup()

	subject := "shutdown-test"
	provider.setPermanentFailure(subject, true)
	email := &email_dto.SendParams{To: []string{"test@example.com"}, Subject: subject}

	require.NoError(t, d.Queue(ctx, email))
	require.Eventually(t, func() bool {
		size, _ := d.GetRetryQueueSize(ctx)
		return size == 1
	}, 3*time.Second, 20*time.Millisecond, "Email should enter the retry queue")

	require.NoError(t, d.Stop(ctx))

	dlqEntries := dlq.getEntries()
	require.Len(t, dlqEntries, 1, "Pending retry should be persisted to DLQ on shutdown")
	require.Equal(t, subject, dlqEntries[0].Email.Subject)
	require.Contains(t, dlqEntries[0].OriginalError, "service shutdown during retry")
}

func TestRetryConfig_ShouldRetry(t *testing.T) {
	t.Parallel()
	config := RetryConfig{Config: retry.Config{MaxRetries: 3}}

	testCases := []struct {
		attempt int
		want    bool
	}{
		{attempt: 0, want: true},
		{attempt: 1, want: true},
		{attempt: 2, want: true},
		{attempt: 3, want: true},
		{attempt: 4, want: false},
		{attempt: 5, want: false},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("attempt_%d", tc.attempt), func(t *testing.T) {
			got := config.ShouldRetry(tc.attempt)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestRetryHeap_Order(t *testing.T) {
	h := retry.NewHeap(func(item *retryItem) time.Time { return item.priority })

	now := time.Now()
	items := []*retryItem{
		{priority: now.Add(50 * time.Millisecond)},
		{priority: now.Add(10 * time.Millisecond)},
		{priority: now.Add(30 * time.Millisecond)},
	}

	for _, it := range items {
		h.PushItem(it)
	}

	var last time.Time
	for h.Len() > 0 {
		poppedItem, ok := h.PopItem()
		if !ok {
			t.Fatal("expected PopItem to succeed")
		}
		if !last.IsZero() && poppedItem.priority.Before(last) {
			t.Fatalf("heap popped out of order: %v before %v", poppedItem.priority, last)
		}
		last = poppedItem.priority
	}
}
