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
	"testing"
	"time"

	"piko.sh/piko/internal/email/email_dto"
)

func TestDispatcher_BatchAndFlush_SuccessBulk(t *testing.T) {
	ctx, cancel := context.WithTimeoutCause(context.Background(), 3*time.Second, fmt.Errorf("test: BatchAndFlush_SuccessBulk exceeded %s timeout", 3*time.Second))
	defer cancel()

	provider := newFakeProvider(true)
	dlq := &emailDLQ{}
	config := email_dto.DispatcherConfig{
		BatchSize:       3,
		FlushInterval:   500 * time.Millisecond,
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

	emails := []*email_dto.SendParams{
		{To: []string{"a@example.com"}, Subject: "s1"},
		{To: []string{"b@example.com"}, Subject: "s2"},
	}
	for _, e := range emails {
		if err := d.Queue(ctx, e); err != nil {
			t.Fatalf("queue: %v", err)
		}
	}

	if err := d.Flush(ctx); err != nil {
		t.Fatalf("flush: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if provider.totalBulkCalls() >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if provider.totalBulkCalls() == 0 {
		t.Fatalf("expected bulk send to be called at least once")
	}

	stats, err := d.GetProcessingStats(ctx)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.TotalSuccessful != 2 {
		t.Fatalf("expected 2 successful, got %d", stats.TotalSuccessful)
	}
	if stats.TotalFailed != 0 {
		t.Fatalf("expected 0 failed, got %d", stats.TotalFailed)
	}
}
