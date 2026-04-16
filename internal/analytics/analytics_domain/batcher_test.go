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
	"testing"
	"time"

	"piko.sh/piko/internal/retry"
)

func TestBatcher_BasicFlush(t *testing.T) {
	var mu sync.Mutex
	var received [][]string

	batcher := newTestBatcher[string](
		BatcherConfig{
			Name:          "test",
			BatchSize:     3,
			FlushInterval: 1 * time.Hour,
		},
		func(_ context.Context, batch []string) error {
			mu.Lock()
			copied := make([]string, len(batch))
			copy(copied, batch)
			received = append(received, copied)
			mu.Unlock()
			return nil
		},
	)

	batcher.Add("a")
	batcher.Add("b")
	batcher.Add("c")

	_ = batcher.Flush(context.Background())
	_ = batcher.Close()

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(received))
	}
	if len(received[0]) != 3 {
		t.Fatalf("expected 3 items, got %d", len(received[0]))
	}
}

func TestBatcher_RetryOnTransientError(t *testing.T) {
	var attempts atomic.Int32

	batcher := newTestBatcher[string](
		BatcherConfig{
			Name:          "retry-test",
			BatchSize:     1,
			FlushInterval: 1 * time.Hour,
			Retry: &retry.Config{
				MaxRetries:    3,
				InitialDelay:  10 * time.Millisecond,
				MaxDelay:      50 * time.Millisecond,
				BackoffFactor: 2.0,
			},
		},
		func(_ context.Context, _ []string) error {
			count := attempts.Add(1)
			if count <= 2 {
				return fmt.Errorf("connection refused")
			}
			return nil
		},
	)

	batcher.Add("item")
	err := batcher.Flush(context.Background())
	_ = batcher.Close()

	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestBatcher_NoRetryOnPermanentError(t *testing.T) {
	var attempts atomic.Int32

	batcher := newTestBatcher[string](
		BatcherConfig{
			Name:          "permanent-error-test",
			BatchSize:     1,
			FlushInterval: 1 * time.Hour,
			Retry: &retry.Config{
				MaxRetries:    3,
				InitialDelay:  10 * time.Millisecond,
				MaxDelay:      50 * time.Millisecond,
				BackoffFactor: 2.0,
			},
		},
		func(_ context.Context, _ []string) error {
			attempts.Add(1)
			return context.Canceled
		},
	)

	batcher.Add("item")
	err := batcher.Flush(context.Background())
	_ = batcher.Close()

	if err == nil {
		t.Fatal("expected error from permanent failure")
	}
	if attempts.Load() != 1 {
		t.Errorf("expected 1 attempt (no retry), got %d", attempts.Load())
	}
}

func TestBatcher_RetryExhausted(t *testing.T) {
	var attempts atomic.Int32

	batcher := newTestBatcher[string](
		BatcherConfig{
			Name:          "exhaust-test",
			BatchSize:     1,
			FlushInterval: 1 * time.Hour,
			Retry: &retry.Config{
				MaxRetries:    2,
				InitialDelay:  10 * time.Millisecond,
				MaxDelay:      50 * time.Millisecond,
				BackoffFactor: 2.0,
			},
		},
		func(_ context.Context, _ []string) error {
			attempts.Add(1)
			return fmt.Errorf("503 service unavailable")
		},
	)

	batcher.Add("item")
	err := batcher.Flush(context.Background())
	_ = batcher.Close()

	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}

	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts (initial + 2 retries), got %d", attempts.Load())
	}
}

func TestBatcher_CircuitBreakerOpens(t *testing.T) {
	var attempts atomic.Int32

	batcher := newTestBatcher[string](
		BatcherConfig{
			Name:          "breaker-test",
			BatchSize:     1,
			FlushInterval: 1 * time.Hour,
			CircuitBreaker: &CircuitBreakerConfig{
				MaxConsecutiveFailures: 3,
				Timeout:                5 * time.Second,
				Interval:               0,
			},
		},
		func(_ context.Context, _ []string) error {
			attempts.Add(1)
			return fmt.Errorf("server error")
		},
	)

	for range 3 {
		batcher.Add("item")
		_ = batcher.Flush(context.Background())
	}

	countBefore := attempts.Load()
	batcher.Add("item")
	err := batcher.Flush(context.Background())
	_ = batcher.Close()

	if err == nil {
		t.Fatal("expected error from open circuit breaker")
	}

	if attempts.Load() != countBefore {
		t.Errorf("expected circuit breaker to block send, but sendFunc was called")
	}
}

func TestBatcher_CircuitBreakerWithRetry(t *testing.T) {
	var attempts atomic.Int32

	batcher := newTestBatcher[string](
		BatcherConfig{
			Name:          "breaker-retry-test",
			BatchSize:     1,
			FlushInterval: 1 * time.Hour,
			Retry: &retry.Config{
				MaxRetries:    2,
				InitialDelay:  10 * time.Millisecond,
				MaxDelay:      50 * time.Millisecond,
				BackoffFactor: 2.0,
			},
			CircuitBreaker: &CircuitBreakerConfig{
				MaxConsecutiveFailures: 10,
				Timeout:                5 * time.Second,
				Interval:               0,
			},
		},
		func(_ context.Context, _ []string) error {
			count := attempts.Add(1)
			if count <= 1 {
				return fmt.Errorf("timeout")
			}
			return nil
		},
	)

	batcher.Add("item")
	err := batcher.Flush(context.Background())
	_ = batcher.Close()

	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if attempts.Load() != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts.Load())
	}
}

func TestBatcher_NoResilienceByDefault(t *testing.T) {
	var attempts atomic.Int32

	batcher := newTestBatcher[string](
		BatcherConfig{
			Name:          "no-resilience",
			BatchSize:     1,
			FlushInterval: 1 * time.Hour,
		},
		func(_ context.Context, _ []string) error {
			attempts.Add(1)
			return errors.New("fail")
		},
	)

	batcher.Add("item")
	err := batcher.Flush(context.Background())
	_ = batcher.Close()

	if err == nil {
		t.Fatal("expected error")
	}
	if attempts.Load() != 1 {
		t.Errorf("expected exactly 1 attempt (no retry), got %d", attempts.Load())
	}
}

func TestBatcher_EmptyFlush(t *testing.T) {
	called := false
	batcher := newTestBatcher[string](
		BatcherConfig{
			Name:          "empty-test",
			BatchSize:     10,
			FlushInterval: 1 * time.Hour,
		},
		func(_ context.Context, _ []string) error {
			called = true
			return nil
		},
	)

	_ = batcher.Flush(context.Background())
	_ = batcher.Close()

	if called {
		t.Error("sendFunc should not be called for empty buffer")
	}
}

func TestBatcher_DoubleClose(t *testing.T) {
	batcher := newTestBatcher[string](
		BatcherConfig{
			Name:          "close-test",
			BatchSize:     10,
			FlushInterval: 1 * time.Hour,
		},
		func(_ context.Context, _ []string) error { return nil },
	)

	if err := batcher.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := batcher.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestNewBatcher_ErrorOnZeroBatchSize(t *testing.T) {
	_, err := NewBatcher[string](BatcherConfig{
		Name:          "error-test",
		BatchSize:     0,
		FlushInterval: time.Second,
	}, func(_ context.Context, _ []string) error { return nil })
	if err == nil {
		t.Fatal("expected error for zero BatchSize")
	}
}

func TestNewBatcher_ErrorOnZeroFlushInterval(t *testing.T) {
	_, err := NewBatcher[string](BatcherConfig{
		Name:          "error-test",
		BatchSize:     10,
		FlushInterval: 0,
	}, func(_ context.Context, _ []string) error { return nil })
	if err == nil {
		t.Fatal("expected error for zero FlushInterval")
	}
}

func TestBatcher_FlushRespectsContextCancellation(t *testing.T) {

	blockCh := make(chan struct{})

	batcher := newTestBatcher[string](
		BatcherConfig{
			Name:          "ctx-cancel-test",
			BatchSize:     10,
			FlushInterval: 1 * time.Hour,
		},
		func(ctx context.Context, _ []string) error {
			select {
			case <-blockCh:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	)

	batcher.Add("item")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := batcher.Flush(ctx)
	if err == nil {
		t.Fatal("expected Flush to return an error when context is cancelled")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}

	close(blockCh)
	_ = batcher.Close()
}

func TestBatcher_AddAfterClose(t *testing.T) {
	batcher := newTestBatcher[string](
		BatcherConfig{
			Name:          "add-after-close",
			BatchSize:     10,
			FlushInterval: 1 * time.Hour,
		},
		func(_ context.Context, _ []string) error { return nil },
	)

	_ = batcher.Close()

	batcher.Add("late-item")

	batcher.mu.Lock()
	bufferLength := len(batcher.buffer)
	batcher.mu.Unlock()

	if bufferLength != 0 {
		t.Errorf("expected empty buffer after Close, got %d items", bufferLength)
	}
}

func newTestBatcher[T any](config BatcherConfig, sendFunc BatchSendFunc[T]) *Batcher[T] {
	b, err := NewBatcher[T](config, sendFunc)
	if err != nil {
		panic(fmt.Sprintf("newTestBatcher: %v", err))
	}
	b.Start(context.Background())
	return b
}
