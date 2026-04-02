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
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"piko.sh/piko/internal/email/email_dto"
)

func TestDispatcher_Retry_Succeeds(t *testing.T) {
	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: Retry_Succeeds exceeded %s timeout", 5*time.Second))
	defer cancel()

	provider := newFakeProvider(false)
	dlq := &emailDLQ{}
	config := email_dto.DispatcherConfig{
		JitterFunc:      func(_ time.Duration) time.Duration { return 0 },
		BatchSize:       10,
		FlushInterval:   50 * time.Millisecond,
		QueueSize:       10,
		RetryQueueSize:  10,
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        20 * time.Millisecond,
		BackoffFactor:   1.0,
		DeadLetterQueue: true,
	}
	d := NewEmailDispatcher(provider, dlq, &config)
	if err := d.Start(ctx); err != nil {
		t.Fatalf("start dispatcher: %v", err)
	}
	defer func() { _ = d.Stop(ctx) }()

	subject := "retry-me"
	provider.failNTimes[subject] = 2

	e := &email_dto.SendParams{To: []string{"x@example.com"}, Subject: subject}
	if err := d.Queue(ctx, e); err != nil {
		t.Fatalf("queue: %v", err)
	}

	if err := d.Flush(ctx); err != nil {
		t.Fatalf("flush: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		stats, _ := d.GetProcessingStats(ctx)
		if provider.attemptsFor(subject) >= 3 && stats.TotalSuccessful >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	stats, _ := d.GetProcessingStats(ctx)
	if provider.attemptsFor(subject) < 3 {
		t.Fatalf("expected at least 3 attempts, got %d", provider.attemptsFor(subject))
	}
	if stats.TotalSuccessful < 1 {
		t.Fatalf("expected success after retries, got successful=%d", stats.TotalSuccessful)
	}
	if stats.TotalFailed != 0 {
		t.Fatalf("expected no DLQ failures, got %d", stats.TotalFailed)
	}
}

func TestDispatcher_DeadLetter_OnPermanentFailure(t *testing.T) {
	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: DeadLetter_OnPermanentFailure exceeded %s timeout", 5*time.Second))
	defer cancel()

	provider := newFakeProvider(false)
	dlq := &emailDLQ{}
	dlq.CountFunc = func(_ context.Context) (int, error) {
		return int(atomic.LoadInt64(&dlq.AddCallCount)), nil
	}
	config := email_dto.DispatcherConfig{
		JitterFunc:      func(_ time.Duration) time.Duration { return 0 },
		BatchSize:       10,
		FlushInterval:   50 * time.Millisecond,
		QueueSize:       10,
		RetryQueueSize:  10,
		MaxRetries:      1,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        20 * time.Millisecond,
		BackoffFactor:   1.0,
		DeadLetterQueue: true,
	}
	d := NewEmailDispatcher(provider, dlq, &config)
	if err := d.Start(ctx); err != nil {
		t.Fatalf("start dispatcher: %v", err)
	}
	defer func() { _ = d.Stop(ctx) }()

	subject := "always-fail"
	provider.permanent[subject] = true
	e := &email_dto.SendParams{To: []string{"y@example.com"}, Subject: subject}
	if err := d.Queue(ctx, e); err != nil {
		t.Fatalf("queue: %v", err)
	}

	if err := d.Flush(ctx); err != nil {
		t.Fatalf("flush: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		count, _ := d.GetDeadLetterCount(ctx)
		if count >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	count, _ := d.GetDeadLetterCount(ctx)
	if count < 1 {
		t.Fatalf("expected at least 1 DLQ entry, got %d", count)
	}
	stats, _ := d.GetProcessingStats(ctx)
	if stats.TotalFailed < 1 {
		t.Fatalf("expected failure count to increase, got %d", stats.TotalFailed)
	}
}
