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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/email/email_dto"
)

func TestDispatcher_CircuitBreaker_TripsAndShortCircuits(t *testing.T) {
	t.Parallel()

	config := email_dto.DispatcherConfig{
		BatchSize:              1,
		FlushInterval:          10 * time.Millisecond,
		MaxRetries:             0,
		DeadLetterQueue:        true,
		MaxConsecutiveFailures: 2,
		CircuitBreakerTimeout:  200 * time.Millisecond,
		CircuitBreakerInterval: 50 * time.Millisecond,
	}
	d, provider, dlq, ctx, cleanup := setupDispatcherTest(t, &config)
	defer cleanup()

	provider.setPermanentFailure("cb-1", true)
	provider.setPermanentFailure("cb-2", true)
	provider.setPermanentFailure("cb-3", true)

	require.NoError(t, d.Queue(ctx, &email_dto.SendParams{To: []string{"a@a.com"}, Subject: "cb-1"}))
	require.NoError(t, d.Queue(ctx, &email_dto.SendParams{To: []string{"b@b.com"}, Subject: "cb-2"}))
	require.NoError(t, d.Queue(ctx, &email_dto.SendParams{To: []string{"c@c.com"}, Subject: "cb-3"}))

	require.Eventually(t, func() bool { return len(dlq.getEntries()) == 3 }, 3*time.Second, 20*time.Millisecond)

	require.Len(t, provider.getSendAttempts("cb-1"), 1)
	require.Len(t, provider.getSendAttempts("cb-2"), 1)

	require.Len(t, provider.getSendAttempts("cb-3"), 0)
}

func TestDispatcher_CircuitBreaker_RecoversAfterTimeout(t *testing.T) {
	t.Parallel()

	config := email_dto.DispatcherConfig{
		BatchSize:              1,
		FlushInterval:          10 * time.Millisecond,
		MaxRetries:             0,
		DeadLetterQueue:        true,
		MaxConsecutiveFailures: 2,
		CircuitBreakerTimeout:  150 * time.Millisecond,
		CircuitBreakerInterval: 50 * time.Millisecond,
	}
	d, provider, dlq, ctx, cleanup := setupDispatcherTest(t, &config)
	defer cleanup()

	provider.setPermanentFailure("fail-1", true)
	provider.setPermanentFailure("fail-2", true)
	require.NoError(t, d.Queue(ctx, &email_dto.SendParams{To: []string{"a@a.com"}, Subject: "fail-1"}))
	require.NoError(t, d.Queue(ctx, &email_dto.SendParams{To: []string{"b@b.com"}, Subject: "fail-2"}))

	require.Eventually(t, func() bool { return len(dlq.getEntries()) >= 2 }, 3*time.Second, 20*time.Millisecond)

	shortCircuitSubject := "open-drop"
	provider.setPermanentFailure(shortCircuitSubject, false)
	require.NoError(t, d.Queue(ctx, &email_dto.SendParams{To: []string{"c@c.com"}, Subject: shortCircuitSubject}))

	time.Sleep(50 * time.Millisecond)
	require.Len(t, provider.getSendAttempts(shortCircuitSubject), 0, "provider should not be called while CB is open")
	require.Eventually(t, func() bool { return len(dlq.getEntries()) >= 3 }, 3*time.Second, 20*time.Millisecond)

	recoverySubject := "recover-ok"

	time.Sleep(200 * time.Millisecond)
	require.NoError(t, d.Queue(ctx, &email_dto.SendParams{To: []string{"d@d.com"}, Subject: recoverySubject}))

	require.Eventually(t, func() bool { return len(provider.getSendAttempts(recoverySubject)) == 1 }, 3*time.Second, 20*time.Millisecond)

	require.Eventually(t, func() bool {
		stats, _ := d.GetProcessingStats(ctx)
		return stats.TotalSuccessful >= 1
	}, 3*time.Second, 20*time.Millisecond)
}
