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
//
// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package storage_domain_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_adapters/provider_mock"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

func TestDispatcher_QueueRemove_Success(t *testing.T) {
	ctx := context.Background()

	mock := provider_mock.NewMockStorageProvider()
	d := newTestDispatcher(t, mock, func(config *storage_domain.DispatcherConfig) {
		config.FlushInterval = 50 * time.Millisecond
		config.BatchSize = 10
		config.QueueSize = 100
		config.MaxRetries = 3
	})

	dCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(fmt.Errorf("test: cleanup"))

	err := d.Start(dCtx)
	require.NoError(t, err)

	err = mock.Put(ctx, &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "remove-me.txt",
		Reader:      bytes.NewReader([]byte("seed data")),
		Size:        9,
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	err = d.QueueRemove(ctx, storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "remove-me.txt",
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		_ = d.Flush(ctx)
		return len(mock.GetRemoveCalls()) >= 1
	}, 2*time.Second, 20*time.Millisecond)

	calls := mock.GetRemoveCalls()
	require.GreaterOrEqual(t, len(calls), 1, "expected at least one remove call")

	err = d.Stop(ctx)
	require.NoError(t, err)
}

func TestDispatcher_QueueRemove_FailureWithRetry(t *testing.T) {
	ctx := context.Background()

	mock := provider_mock.NewMockStorageProvider()

	mock.SetRemoveError(errors.New("connection refused"))

	d := newTestDispatcher(t, mock, func(config *storage_domain.DispatcherConfig) {
		config.MaxRetries = 2
		config.FlushInterval = 50 * time.Millisecond
		config.BatchSize = 1
	})

	dCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(fmt.Errorf("test: cleanup"))

	err := d.Start(dCtx)
	require.NoError(t, err)

	err = d.QueueRemove(ctx, storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "retry-remove.txt",
	})
	require.NoError(t, err)

	_ = d.Flush(ctx)
	time.Sleep(200 * time.Millisecond)

	err = d.Stop(ctx)
	require.NoError(t, err)

	calls := mock.GetRemoveCalls()
	assert.GreaterOrEqual(t, len(calls), 2, "expected at least 2 remove calls (initial + retry)")
}

func TestDispatcher_QueueRemove_DLQAfterMaxRetries(t *testing.T) {
	ctx := context.Background()

	mock := provider_mock.NewMockStorageProvider()

	mock.SetRemoveError(errors.New("connection refused"))

	d := newTestDispatcher(t, mock, func(config *storage_domain.DispatcherConfig) {
		config.MaxRetries = 1
		config.FlushInterval = 50 * time.Millisecond
		config.BatchSize = 1
	})

	dCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(fmt.Errorf("test: cleanup"))

	err := d.Start(dCtx)
	require.NoError(t, err)

	err = d.QueueRemove(ctx, storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "doomed-remove.txt",
	})
	require.NoError(t, err)

	_ = d.Flush(ctx)

	require.Eventually(t, func() bool {
		stats := d.GetStats()
		return stats.TotalFailed >= 1
	}, 3*time.Second, 20*time.Millisecond)

	err = d.Stop(ctx)
	require.NoError(t, err)

	stats := d.GetStats()
	assert.GreaterOrEqual(t, stats.TotalFailed, int64(1), "expected at least one DLQ entry")
}

func TestDispatcher_Flush_ProcessesBothQueues(t *testing.T) {
	ctx := context.Background()

	mock := provider_mock.NewMockStorageProvider()
	d := newTestDispatcher(t, mock, func(config *storage_domain.DispatcherConfig) {
		config.FlushInterval = 50 * time.Millisecond
		config.BatchSize = 10
	})

	dCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(fmt.Errorf("test: cleanup"))

	err := d.Start(dCtx)
	require.NoError(t, err)

	err = mock.Put(ctx, &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "both-queues.txt",
		Reader:      bytes.NewReader([]byte("seed")),
		Size:        4,
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	err = d.QueuePut(ctx, &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "queued.txt",
		Reader:      bytes.NewReader([]byte("data")),
		Size:        4,
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	err = d.QueueRemove(ctx, storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "both-queues.txt",
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		_ = d.Flush(ctx)
		putCalls := mock.GetPutCalls()
		removeCalls := mock.GetRemoveCalls()

		return len(putCalls) >= 2 && len(removeCalls) >= 1
	}, 2*time.Second, 20*time.Millisecond)

	err = d.Stop(ctx)
	require.NoError(t, err)
}

func TestDispatcher_StopDrainsQueues(t *testing.T) {
	ctx := context.Background()

	mock := provider_mock.NewMockStorageProvider()
	d := newTestDispatcher(t, mock, func(config *storage_domain.DispatcherConfig) {

		config.FlushInterval = 10 * time.Second
		config.BatchSize = 100
	})

	dCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(fmt.Errorf("test: cleanup"))

	err := d.Start(dCtx)
	require.NoError(t, err)

	for i := range 3 {
		err = d.QueuePut(ctx, &storage_dto.PutParams{
			Repository:  storage_dto.StorageRepositoryDefault,
			Key:         "drain-" + string(rune('a'+i)) + ".txt",
			Reader:      bytes.NewReader([]byte("data")),
			Size:        4,
			ContentType: "text/plain",
		})
		require.NoError(t, err)
	}

	err = d.Stop(ctx)
	require.NoError(t, err)

	calls := mock.GetPutCalls()
	assert.Len(t, calls, 3, "all three queued puts should have been drained on stop")
}

func TestService_RegisterDispatcher(t *testing.T) {
	ctx := context.Background()

	service, mock := setupManagementTestService(t)

	dispatcher := storage_domain.NewStorageDispatcher(mock, "default", storage_domain.DefaultDispatcherConfig())

	err := service.RegisterDispatcher(ctx, dispatcher)
	require.NoError(t, err)
	defer func() { _ = dispatcher.Stop(ctx) }()

	err = service.FlushDispatcher(ctx)
	require.NoError(t, err)
}

func TestService_RegisterDispatcher_Nil(t *testing.T) {
	service, _ := setupManagementTestService(t)

	err := service.RegisterDispatcher(context.Background(), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}
