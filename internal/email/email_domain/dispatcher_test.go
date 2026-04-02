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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/email/email_dto"
)

type controlledBulkFakeProvider struct {
	controlledFakeProvider
	bulkSendCalls       [][]*email_dto.SendParams
	individualSendCount int64
	mu                  sync.Mutex
	supportsBulk        bool
	failBulk            bool
}

func newControlledBulkFakeProvider(supportsBulk bool) *controlledBulkFakeProvider {
	return &controlledBulkFakeProvider{
		controlledFakeProvider: *newControlledFakeProvider(),
		supportsBulk:           supportsBulk,
	}
}

func (p *controlledBulkFakeProvider) Send(ctx context.Context, params *email_dto.SendParams) error {
	atomic.AddInt64(&p.individualSendCount, 1)
	return p.controlledFakeProvider.Send(ctx, params)
}

func (p *controlledBulkFakeProvider) SendBulk(ctx context.Context, emails []*email_dto.SendParams) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.bulkSendCalls = append(p.bulkSendCalls, emails)

	if p.failBulk {
		return errors.New("simulated bulk send failure")
	}

	for _, email := range emails {
		key := email.Subject
		p.sendAttempts[key] = append(p.sendAttempts[key], time.Now())
	}
	return nil
}

func (p *controlledBulkFakeProvider) SupportsBulkSending() bool {
	return p.supportsBulk
}

func (p *controlledBulkFakeProvider) setFailBulk(fail bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failBulk = fail
}

func (p *controlledBulkFakeProvider) getBulkSendCallCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.bulkSendCalls)
}

func (p *controlledBulkFakeProvider) getIndividualSendCount() int64 {
	return atomic.LoadInt64(&p.individualSendCount)
}

func setupDispatcherTestWithBulk(t *testing.T, config *email_dto.DispatcherConfig, provider *controlledBulkFakeProvider) (d *EmailDispatcher, dlq *controlledFakeDLQ, ctx context.Context, cleanup func()) {
	config.JitterFunc = func(_ time.Duration) time.Duration { return 0 }

	dlq = newControlledFakeDLQ()

	d = NewEmailDispatcher(provider, dlq, config)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 10*time.Second, fmt.Errorf("test: dispatcher setup exceeded %s timeout", 10*time.Second))
	require.NoError(t, d.Start(ctx))

	cleanup = func() {
		_ = d.Stop(ctx)
		cancel()
	}

	return d, dlq, ctx, cleanup
}

func TestDispatcher_BatchingAndFlushing(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                  string
		batchSize             int
		flushInterval         time.Duration
		emailsToQueue         int
		manualFlush           bool
		expectBulkCalls       int
		expectIndividualCalls int
	}{
		{
			name:                  "Fills a batch exactly and sends",
			batchSize:             5,
			flushInterval:         1 * time.Second,
			emailsToQueue:         5,
			manualFlush:           false,
			expectBulkCalls:       1,
			expectIndividualCalls: 0,
		},
		{
			name:                  "Sends a partial batch when flush interval is reached",
			batchSize:             10,
			flushInterval:         50 * time.Millisecond,
			emailsToQueue:         3,
			manualFlush:           false,
			expectBulkCalls:       1,
			expectIndividualCalls: 0,
		},
		{
			name:                  "Sends a partial batch on manual flush",
			batchSize:             10,
			flushInterval:         1 * time.Second,
			emailsToQueue:         4,
			manualFlush:           true,
			expectBulkCalls:       1,
			expectIndividualCalls: 0,
		},
		{
			name:                  "Sends multiple full batches",
			batchSize:             3,
			flushInterval:         1 * time.Second,
			emailsToQueue:         7,
			manualFlush:           true,
			expectBulkCalls:       3,
			expectIndividualCalls: 0,
		},
		{
			name:                  "Sends individually when bulk is not supported",
			batchSize:             5,
			flushInterval:         100 * time.Millisecond,
			emailsToQueue:         5,
			manualFlush:           false,
			expectBulkCalls:       0,
			expectIndividualCalls: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := newControlledBulkFakeProvider(tc.expectIndividualCalls == 0)
			config := email_dto.DispatcherConfig{
				BatchSize:     tc.batchSize,
				FlushInterval: tc.flushInterval,
				MaxRetries:    0,
			}
			d, _, ctx, cleanup := setupDispatcherTestWithBulk(t, &config, provider)
			defer cleanup()

			for i := 0; i < tc.emailsToQueue; i++ {
				email := &email_dto.SendParams{To: []string{"test@example.com"}, Subject: fmt.Sprintf("email-%d", i)}
				require.NoError(t, d.Queue(ctx, email))
			}

			if tc.manualFlush {
				require.NoError(t, d.Flush(ctx))
			}

			require.Eventually(t, func() bool {
				stats, _ := d.GetProcessingStats(ctx)
				return stats.TotalSuccessful == int64(tc.emailsToQueue)
			}, 3*time.Second, 20*time.Millisecond, "All queued emails should have been processed")

			require.Equal(t, tc.expectBulkCalls, provider.getBulkSendCallCount(), "Incorrect number of bulk send calls")
			require.Equal(t, int64(tc.expectIndividualCalls), provider.getIndividualSendCount(), "Incorrect number of individual send calls")
		})
	}
}

func TestDispatcher_BulkSendFailure_FallsBackToIndividualSends(t *testing.T) {
	t.Parallel()
	provider := newControlledBulkFakeProvider(true)
	provider.setFailBulk(true)

	config := email_dto.DispatcherConfig{
		BatchSize:     5,
		FlushInterval: 100 * time.Millisecond,
		MaxRetries:    0,
	}
	d, _, ctx, cleanup := setupDispatcherTestWithBulk(t, &config, provider)
	defer cleanup()

	emailsToQueue := 3
	for i := range emailsToQueue {
		email := &email_dto.SendParams{To: []string{"test@example.com"}, Subject: fmt.Sprintf("email-%d", i)}
		require.NoError(t, d.Queue(ctx, email))
	}

	require.Eventually(t, func() bool {
		stats, _ := d.GetProcessingStats(ctx)
		return stats.TotalSuccessful == int64(emailsToQueue)
	}, 3*time.Second, 20*time.Millisecond, "All emails should be processed individually after bulk failure")

	require.Equal(t, 1, provider.getBulkSendCallCount(), "Should have attempted one bulk send")
	require.Equal(t, int64(emailsToQueue), provider.getIndividualSendCount(), "Should have fallen back to 3 individual sends")
}

func TestDispatcher_GracefulShutdown_DrainsQueue(t *testing.T) {
	t.Parallel()
	provider := newControlledBulkFakeProvider(true)
	config := email_dto.DispatcherConfig{
		BatchSize:     10,
		QueueSize:     50,
		FlushInterval: 5 * time.Second,
	}
	d, _, ctx, cleanup := setupDispatcherTestWithBulk(t, &config, provider)
	defer cleanup()

	emailsToQueue := 35
	for i := range emailsToQueue {
		email := &email_dto.SendParams{To: []string{"test@example.com"}, Subject: fmt.Sprintf("shutdown-%d", i)}
		select {
		case d.queue <- email:
		default:
			t.Fatalf("Queue filled up unexpectedly")
		}
	}

	time.Sleep(50 * time.Millisecond)
	require.Less(t, provider.getBulkSendCallCount(), 4, "No more than the final number of batches should be sent before shutdown")

	require.NoError(t, d.Stop(ctx))

	stats, err := d.GetProcessingStats(ctx)
	require.NoError(t, err)
	require.False(t, d.isRunning, "Dispatcher should be marked as not running")
	require.Equal(t, int64(emailsToQueue), stats.TotalSuccessful, "All queued emails should be processed on shutdown")
	require.Equal(t, 4, provider.getBulkSendCallCount(), "Should have sent 4 batches (10, 10, 10, 5)")
}

func TestDispatcher_Queue_RespectsContextCancellation(t *testing.T) {
	t.Parallel()
	provider := newControlledBulkFakeProvider(true)
	config := email_dto.DispatcherConfig{
		QueueSize: 1,
	}
	d, _, ctx, cleanup := setupDispatcherTestWithBulk(t, &config, provider)
	defer cleanup()

	require.NoError(t, d.Queue(ctx, &email_dto.SendParams{To: []string{"a@a.com"}, Subject: "first"}))

	cancelledCtx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	err := d.Queue(cancelledCtx, &email_dto.SendParams{To: []string{"b@b.com"}, Subject: "second"})

	require.Error(t, err, "Queueing should fail when context is cancelled")
	require.ErrorIs(t, err, context.Canceled, "Error should be context.Canceled")
}

func TestDispatcher_Start_AlreadyRunningReturnsError(t *testing.T) {
	t.Parallel()
	provider := newControlledBulkFakeProvider(true)
	config := email_dto.DispatcherConfig{}
	d, _, ctx, cleanup := setupDispatcherTestWithBulk(t, &config, provider)
	defer cleanup()

	err := d.Start(ctx)

	require.Error(t, err, "Calling Start on a running dispatcher should return an error")
	require.Contains(t, err.Error(), "dispatcher already running")
}

func TestDispatcher_Stop_NotRunningIsNoOp(t *testing.T) {
	t.Parallel()
	provider := newControlledBulkFakeProvider(true)
	d := NewEmailDispatcher(provider, nil, &email_dto.DispatcherConfig{})

	err := d.Stop(context.Background())

	require.NoError(t, err, "Calling Stop on a non-running dispatcher should be a no-op")
}

func TestDispatcher_GetProcessingStats(t *testing.T) {
	t.Parallel()
	provider := newControlledBulkFakeProvider(false)
	provider.setFailNTimes("fail-once", 1)
	provider.setPermanentFailure("fail-always", true)

	config := email_dto.DispatcherConfig{
		BatchSize:     1,
		FlushInterval: 20 * time.Millisecond,
		MaxRetries:    1,
		InitialDelay:  10 * time.Millisecond,
	}
	d, dlq, ctx, cleanup := setupDispatcherTestWithBulk(t, &config, provider)
	defer cleanup()

	require.NoError(t, d.Queue(ctx, &email_dto.SendParams{To: []string{"a@a.com"}, Subject: "success-1"}))
	require.NoError(t, d.Queue(ctx, &email_dto.SendParams{To: []string{"b@b.com"}, Subject: "success-2"}))
	require.NoError(t, d.Queue(ctx, &email_dto.SendParams{To: []string{"c@c.com"}, Subject: "fail-once"}))
	require.NoError(t, d.Queue(ctx, &email_dto.SendParams{To: []string{"d@d.com"}, Subject: "fail-always"}))

	require.Eventually(t, func() bool {
		stats, _ := d.GetProcessingStats(ctx)
		return stats.TotalProcessed == 4
	}, 3*time.Second, 50*time.Millisecond, "Processing should complete for all 4 emails")

	stats, err := d.GetProcessingStats(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(4), stats.TotalProcessed, "Total processed should be 4")
	require.Equal(t, int64(3), stats.TotalSuccessful, "Total successful should be 3")
	require.Equal(t, int64(1), stats.TotalFailed, "Total failed should be 1")
	require.Equal(t, int64(2), stats.TotalRetries, "Total retries should be 2 (1 for fail-once, 1 for fail-always)")
	require.Equal(t, 1, dlq.getAddCallCount(), "DLQ Add should have been called once")
	require.True(t, stats.Uptime > 0, "Uptime should be positive")
	require.Equal(t, 0, len(d.queue), "Processing queue should be empty")
}
