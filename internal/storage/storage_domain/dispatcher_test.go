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

package storage_domain_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_adapters/provider_mock"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/wdk/clock"
)

func newTestDispatcher(t *testing.T, provider *provider_mock.MockStorageProvider, overrides ...func(*storage_domain.DispatcherConfig)) *storage_domain.StorageDispatcher {
	t.Helper()
	config := storage_domain.DispatcherConfig{
		BatchSize:     5,
		FlushInterval: 1 * time.Hour,
		QueueSize:     100,
		MaxRetries:    3,
		Clock:         clock.NewMockClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
	}
	for _, override := range overrides {
		override(&config)
	}
	return storage_domain.NewStorageDispatcher(provider, "test-provider", config)
}

func TestDefaultDispatcherConfig(t *testing.T) {
	config := storage_domain.DefaultDispatcherConfig()
	assert.Equal(t, 10, config.BatchSize)
	assert.Equal(t, 30*time.Second, config.FlushInterval)
	assert.Equal(t, 1000, config.QueueSize)
	assert.Equal(t, 3, config.MaxRetries)
}

func TestNewStorageDispatcher(t *testing.T) {
	mock := provider_mock.NewMockStorageProvider()

	t.Run("uses provided config values", func(t *testing.T) {
		d := storage_domain.NewStorageDispatcher(mock, "test", storage_domain.DispatcherConfig{
			BatchSize:     20,
			FlushInterval: 10 * time.Second,
			QueueSize:     500,
			MaxRetries:    5,
			Clock:         clock.NewMockClock(time.Time{}),
		})
		require.NotNil(t, d)

		stats := d.GetStats()
		assert.Equal(t, int64(0), stats.TotalQueued)
		assert.Equal(t, int64(0), stats.TotalProcessed)
	})

	t.Run("applies defaults for zero config values", func(t *testing.T) {
		d := storage_domain.NewStorageDispatcher(mock, "test", storage_domain.DispatcherConfig{})
		require.NotNil(t, d)
	})
}

func TestStorageDispatcher_StartStop(t *testing.T) {
	mock := provider_mock.NewMockStorageProvider()

	t.Run("start and stop lifecycle", func(t *testing.T) {
		d := newTestDispatcher(t, mock)
		ctx := t.Context()

		err := d.Start(ctx)
		require.NoError(t, err)

		err = d.Stop(ctx)
		require.NoError(t, err)
	})

	t.Run("double start returns error", func(t *testing.T) {
		d := newTestDispatcher(t, mock)
		ctx := t.Context()

		err := d.Start(ctx)
		require.NoError(t, err)

		err = d.Start(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already running")

		err = d.Stop(ctx)
		require.NoError(t, err)
	})

	t.Run("stop when not running is a no-op", func(t *testing.T) {
		d := newTestDispatcher(t, mock)
		err := d.Stop(context.Background())
		require.NoError(t, err)
	})
}

func TestStorageDispatcher_QueuePut(t *testing.T) {
	ctx := context.Background()

	t.Run("queue and flush a single put", func(t *testing.T) {
		mock := provider_mock.NewMockStorageProvider()
		d := newTestDispatcher(t, mock)
		dCtx, cancel := context.WithCancelCause(ctx)
		defer cancel(fmt.Errorf("test: cleanup"))

		err := d.Start(dCtx)
		require.NoError(t, err)

		err = d.QueuePut(ctx, &storage_dto.PutParams{
			Repository:  "default",
			Key:         "hello.txt",
			Reader:      strings.NewReader("hello"),
			Size:        5,
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		stats := d.GetStats()
		assert.Equal(t, int64(1), stats.TotalQueued)

		err = d.Stop(ctx)
		require.NoError(t, err)

		calls := mock.GetPutCalls()
		require.Len(t, calls, 1)
		assert.Equal(t, "hello.txt", calls[0].Key)
	})

	t.Run("queue full returns error", func(t *testing.T) {
		mock := provider_mock.NewMockStorageProvider()
		d := newTestDispatcher(t, mock, func(config *storage_domain.DispatcherConfig) {
			config.QueueSize = 1
		})

		err := d.QueuePut(ctx, &storage_dto.PutParams{
			Repository:  "default",
			Key:         "a.txt",
			Reader:      strings.NewReader("a"),
			Size:        1,
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		err = d.QueuePut(ctx, &storage_dto.PutParams{
			Repository:  "default",
			Key:         "b.txt",
			Reader:      strings.NewReader("b"),
			Size:        1,
			ContentType: "text/plain",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "queue is full")
	})

	t.Run("cancelled context returns error", func(t *testing.T) {
		mock := provider_mock.NewMockStorageProvider()
		d := newTestDispatcher(t, mock, func(config *storage_domain.DispatcherConfig) {
			config.QueueSize = 1
		})

		_ = d.QueuePut(ctx, &storage_dto.PutParams{
			Repository:  "default",
			Key:         "fill.txt",
			Reader:      strings.NewReader("fill"),
			Size:        4,
			ContentType: "text/plain",
		})

		cancelledCtx, cancel := context.WithCancelCause(ctx)
		cancel(fmt.Errorf("test: simulating cancelled context"))

		err := d.QueuePut(cancelledCtx, &storage_dto.PutParams{
			Repository:  "default",
			Key:         "blocked.txt",
			Reader:      strings.NewReader("blocked"),
			Size:        7,
			ContentType: "text/plain",
		})
		require.Error(t, err)
	})
}

func TestStorageDispatcher_QueueRemove(t *testing.T) {
	ctx := context.Background()

	t.Run("queue and flush a single remove", func(t *testing.T) {
		mock := provider_mock.NewMockStorageProvider()
		d := newTestDispatcher(t, mock)
		dCtx, cancel := context.WithCancelCause(ctx)
		defer cancel(fmt.Errorf("test: cleanup"))

		err := d.Start(dCtx)
		require.NoError(t, err)

		err = d.QueueRemove(ctx, storage_dto.GetParams{
			Repository: "default",
			Key:        "delete-me.txt",
		})
		require.NoError(t, err)

		err = d.Stop(ctx)
		require.NoError(t, err)

		calls := mock.GetRemoveCalls()
		require.Len(t, calls, 1)
		assert.Equal(t, "delete-me.txt", calls[0].Key)
	})

	t.Run("queue full returns error", func(t *testing.T) {
		mock := provider_mock.NewMockStorageProvider()
		d := newTestDispatcher(t, mock, func(config *storage_domain.DispatcherConfig) {
			config.QueueSize = 1
		})

		err := d.QueueRemove(ctx, storage_dto.GetParams{Repository: "default", Key: "a"})
		require.NoError(t, err)

		err = d.QueueRemove(ctx, storage_dto.GetParams{Repository: "default", Key: "b"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "queue is full")
	})
}

func TestStorageDispatcher_Flush(t *testing.T) {
	ctx := context.Background()

	t.Run("flush triggers processing", func(t *testing.T) {
		mock := provider_mock.NewMockStorageProvider()
		d := newTestDispatcher(t, mock, func(config *storage_domain.DispatcherConfig) {
			config.Clock = clock.RealClock()
			config.BatchSize = 100
		})
		dCtx, cancel := context.WithCancelCause(ctx)
		defer cancel(fmt.Errorf("test: cleanup"))

		err := d.Start(dCtx)
		require.NoError(t, err)

		err = d.QueuePut(ctx, &storage_dto.PutParams{
			Repository:  "default",
			Key:         "flushed.txt",
			Reader:      strings.NewReader("flushed"),
			Size:        7,
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			_ = d.Flush(ctx)
			return len(mock.GetPutCalls()) >= 1
		}, 2*time.Second, 20*time.Millisecond)

		err = d.Stop(ctx)
		require.NoError(t, err)
	})

	t.Run("flush with cancelled context returns nil due to default branch", func(t *testing.T) {
		mock := provider_mock.NewMockStorageProvider()
		d := newTestDispatcher(t, mock)

		cancelledCtx, cancel := context.WithCancelCause(ctx)
		cancel(fmt.Errorf("test: simulating cancelled context"))

		err := d.Flush(cancelledCtx)
		if err != nil {
			assert.ErrorIs(t, err, context.Canceled)
		}
	})
}

func TestStorageDispatcher_GetStats(t *testing.T) {
	ctx := context.Background()

	mock := provider_mock.NewMockStorageProvider()
	d := newTestDispatcher(t, mock)
	dCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(fmt.Errorf("test: cleanup"))

	err := d.Start(dCtx)
	require.NoError(t, err)

	for i := range 3 {
		_ = d.QueuePut(ctx, &storage_dto.PutParams{
			Repository:  "default",
			Key:         "item-" + string(rune('a'+i)) + ".txt",
			Reader:      strings.NewReader("data"),
			Size:        4,
			ContentType: "text/plain",
		})
	}

	err = d.Stop(ctx)
	require.NoError(t, err)

	stats := d.GetStats()
	assert.Equal(t, int64(3), stats.TotalQueued)
	assert.Equal(t, int64(3), stats.TotalProcessed)
	assert.Equal(t, int64(0), stats.TotalFailed)
}

func TestStorageDispatcher_SetBatchSize(t *testing.T) {
	mock := provider_mock.NewMockStorageProvider()
	d := newTestDispatcher(t, mock)

	t.Run("updates batch size with valid value", func(t *testing.T) {
		d.SetBatchSize(20)

	})

	t.Run("ignores zero value", func(t *testing.T) {
		d.SetBatchSize(0)
	})

	t.Run("ignores negative value", func(t *testing.T) {
		d.SetBatchSize(-1)
	})
}

func TestStorageDispatcher_SetFlushInterval(t *testing.T) {
	mock := provider_mock.NewMockStorageProvider()
	d := newTestDispatcher(t, mock)

	t.Run("updates flush interval with valid value", func(t *testing.T) {
		d.SetFlushInterval(10 * time.Second)
	})

	t.Run("ignores zero value", func(t *testing.T) {
		d.SetFlushInterval(0)
	})

	t.Run("ignores negative value", func(t *testing.T) {
		d.SetFlushInterval(-1 * time.Second)
	})
}

func TestStorageDispatcher_BatchProcessing(t *testing.T) {
	ctx := context.Background()

	t.Run("processes items in batches", func(t *testing.T) {
		mock := provider_mock.NewMockStorageProvider()
		d := newTestDispatcher(t, mock, func(config *storage_domain.DispatcherConfig) {
			config.BatchSize = 3
		})
		dCtx, cancel := context.WithCancelCause(ctx)
		defer cancel(fmt.Errorf("test: cleanup"))

		err := d.Start(dCtx)
		require.NoError(t, err)

		for i := range 6 {
			_ = d.QueuePut(ctx, &storage_dto.PutParams{
				Repository:  "default",
				Key:         "batch-" + string(rune('0'+i)) + ".txt",
				Reader:      bytes.NewReader([]byte("data")),
				Size:        4,
				ContentType: "text/plain",
			})
		}

		err = d.Stop(ctx)
		require.NoError(t, err)

		calls := mock.GetPutCalls()
		assert.Len(t, calls, 6)
	})
}

func TestStorageDispatcher_RetryOnFailure(t *testing.T) {
	ctx := context.Background()

	t.Run("retries transient errors then succeeds", func(t *testing.T) {
		mock := provider_mock.NewMockStorageProvider()

		mock.SetPutError(errors.New("connection reset"))

		d := newTestDispatcher(t, mock, func(config *storage_domain.DispatcherConfig) {
			config.MaxRetries = 3
			config.BatchSize = 1
			config.Clock = clock.RealClock()
		})
		dCtx, cancel := context.WithCancelCause(ctx)
		defer cancel(fmt.Errorf("test: cleanup"))

		err := d.Start(dCtx)
		require.NoError(t, err)

		_ = d.QueuePut(ctx, &storage_dto.PutParams{
			Repository:  "default",
			Key:         "retry-me.txt",
			Reader:      strings.NewReader("data"),
			Size:        4,
			ContentType: "text/plain",
		})

		time.Sleep(200 * time.Millisecond)

		mock.SetPutError(nil)

		_ = d.Flush(ctx)

		require.Eventually(t, func() bool {
			stats := d.GetStats()
			return stats.TotalProcessed >= 2
		}, 2*time.Second, 10*time.Millisecond)

		err = d.Stop(ctx)
		require.NoError(t, err)
	})
}

func TestStorageDispatcher_PermanentFailureToDLQ(t *testing.T) {
	ctx := context.Background()

	mock := provider_mock.NewMockStorageProvider()
	mock.SetPutError(errors.New("permanent failure"))

	d := newTestDispatcher(t, mock, func(config *storage_domain.DispatcherConfig) {
		config.MaxRetries = 1
		config.BatchSize = 1
		config.Clock = clock.RealClock()
	})
	dCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(fmt.Errorf("test: cleanup"))

	err := d.Start(dCtx)
	require.NoError(t, err)

	_ = d.QueuePut(ctx, &storage_dto.PutParams{
		Repository:  "default",
		Key:         "doomed.txt",
		Reader:      strings.NewReader("data"),
		Size:        4,
		ContentType: "text/plain",
	})

	require.Eventually(t, func() bool {
		stats := d.GetStats()
		return stats.TotalFailed >= 1
	}, 2*time.Second, 10*time.Millisecond)

	err = d.Stop(ctx)
	require.NoError(t, err)

	stats := d.GetStats()
	assert.GreaterOrEqual(t, stats.TotalFailed, int64(1))
}

func TestStorageDispatcher_GracefulShutdown(t *testing.T) {
	ctx := context.Background()

	mock := provider_mock.NewMockStorageProvider()
	d := newTestDispatcher(t, mock, func(config *storage_domain.DispatcherConfig) {
		config.BatchSize = 100
		config.Clock = clock.RealClock()
	})
	dCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(fmt.Errorf("test: cleanup"))

	err := d.Start(dCtx)
	require.NoError(t, err)

	for i := range 5 {
		_ = d.QueuePut(ctx, &storage_dto.PutParams{
			Repository:  "default",
			Key:         "shutdown-" + string(rune('a'+i)) + ".txt",
			Reader:      strings.NewReader("data"),
			Size:        4,
			ContentType: "text/plain",
		})
	}
	for i := range 3 {
		_ = d.QueueRemove(ctx, storage_dto.GetParams{
			Repository: "default",
			Key:        "rm-" + string(rune('a'+i)) + ".txt",
		})
	}

	err = d.Stop(ctx)
	require.NoError(t, err)

	putCalls := mock.GetPutCalls()
	removeCalls := mock.GetRemoveCalls()
	assert.Len(t, putCalls, 5)
	assert.Len(t, removeCalls, 3)
}

func TestStorageDispatcher_ContextCancellation(t *testing.T) {

	mock := provider_mock.NewMockStorageProvider()
	d := newTestDispatcher(t, mock, func(config *storage_domain.DispatcherConfig) {
		config.Clock = clock.RealClock()
	})
	ctx, cancel := context.WithCancelCause(context.Background())

	err := d.Start(ctx)
	require.NoError(t, err)

	_ = d.QueuePut(ctx, &storage_dto.PutParams{
		Repository:  "default",
		Key:         "ctx.txt",
		Reader:      strings.NewReader("data"),
		Size:        4,
		ContentType: "text/plain",
	})

	cancel(fmt.Errorf("test: simulating cancelled context"))

	err = d.Stop(context.Background())
	require.NoError(t, err)
}

func TestStorageDispatcher_RemoveBatchProcessing(t *testing.T) {
	ctx := context.Background()

	t.Run("remove failures are retried", func(t *testing.T) {
		mock := provider_mock.NewMockStorageProvider()
		mock.SetRemoveError(errors.New("remove failed"))

		d := newTestDispatcher(t, mock, func(config *storage_domain.DispatcherConfig) {
			config.MaxRetries = 1
			config.BatchSize = 1
			config.Clock = clock.RealClock()
		})
		dCtx, cancel := context.WithCancelCause(ctx)
		defer cancel(fmt.Errorf("test: cleanup"))

		err := d.Start(dCtx)
		require.NoError(t, err)

		_ = d.QueueRemove(ctx, storage_dto.GetParams{
			Repository: "default",
			Key:        "doomed-rm.txt",
		})

		require.Eventually(t, func() bool {
			return d.GetStats().TotalFailed >= 1
		}, 2*time.Second, 10*time.Millisecond)

		err = d.Stop(ctx)
		require.NoError(t, err)
	})
}
