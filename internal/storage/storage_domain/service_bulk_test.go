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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_adapters/provider_mock"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

func TestService_PutObjects(t *testing.T) {
	ctx := context.Background()

	t.Run("successful batch put with native batch support", func(t *testing.T) {
		service, mock := setupTestService(t)

		objects := []storage_dto.PutObjectSpec{
			{
				Key:         "file1.txt",
				Reader:      bytes.NewReader([]byte("data1")),
				Size:        5,
				ContentType: "text/plain",
			},
			{
				Key:         "file2.txt",
				Reader:      bytes.NewReader([]byte("data2")),
				Size:        5,
				ContentType: "text/plain",
			},
			{
				Key:         "file3.txt",
				Reader:      bytes.NewReader([]byte("data3")),
				Size:        5,
				ContentType: "text/plain",
			},
		}

		params := &storage_dto.PutManyParams{
			Repository:      storage_dto.StorageRepositoryDefault,
			Objects:         objects,
			Concurrency:     2,
			ContinueOnError: true,
		}

		err := service.PutObjects(ctx, "default", params)
		require.NoError(t, err)

		putManyCalls := mock.GetPutManyCalls()
		require.Len(t, putManyCalls, 1)
		assert.Len(t, putManyCalls[0].Objects, 3)
	})

	t.Run("batch put with all objects succeeding", func(t *testing.T) {
		service, mock := setupTestService(t)

		objects := []storage_dto.PutObjectSpec{
			{
				Key:         "file1.txt",
				Reader:      bytes.NewReader([]byte("data1")),
				Size:        5,
				ContentType: "text/plain",
			},
			{
				Key:         "file2.txt",
				Reader:      bytes.NewReader([]byte("data2")),
				Size:        5,
				ContentType: "text/plain",
			},
		}

		params := &storage_dto.PutManyParams{
			Repository:      storage_dto.StorageRepositoryDefault,
			Objects:         objects,
			ContinueOnError: true,
		}

		err := service.PutObjects(ctx, "default", params)
		require.NoError(t, err)

		putManyCalls := mock.GetPutManyCalls()
		require.Len(t, putManyCalls, 1)
		assert.Len(t, putManyCalls[0].Objects, 2)
	})

	t.Run("empty batch is handled gracefully", func(t *testing.T) {
		service, _ := setupTestService(t)

		params := &storage_dto.PutManyParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Objects:    []storage_dto.PutObjectSpec{},
		}

		err := service.PutObjects(ctx, "default", params)
		require.NoError(t, err)
	})

	t.Run("validation error before upload", func(t *testing.T) {
		service, _ := setupTestService(t)

		objects := []storage_dto.PutObjectSpec{
			{
				Key:         "",
				Reader:      bytes.NewReader([]byte("data1")),
				Size:        5,
				ContentType: "text/plain",
			},
		}

		params := &storage_dto.PutManyParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Objects:    objects,
		}

		err := service.PutObjects(ctx, "default", params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation")
	})

	t.Run("batch size exceeds limit", func(t *testing.T) {
		service, _ := setupTestService(t)

		objects := make([]storage_dto.PutObjectSpec, 1001)
		for i := range objects {
			objects[i] = storage_dto.PutObjectSpec{
				Key:         "file.txt",
				Reader:      bytes.NewReader([]byte("data")),
				Size:        4,
				ContentType: "text/plain",
			}
		}

		params := &storage_dto.PutManyParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Objects:    objects,
		}

		err := service.PutObjects(ctx, "default", params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum")
	})

	t.Run("negative concurrency is rejected", func(t *testing.T) {
		service, _ := setupTestService(t)

		objects := []storage_dto.PutObjectSpec{
			{
				Key:         "file1.txt",
				Reader:      bytes.NewReader([]byte("data1")),
				Size:        5,
				ContentType: "text/plain",
			},
		}

		params := &storage_dto.PutManyParams{
			Repository:  storage_dto.StorageRepositoryDefault,
			Objects:     objects,
			Concurrency: -1,
		}

		err := service.PutObjects(ctx, "default", params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "concurrency cannot be negative")
	})
}

func TestService_RemoveObjects(t *testing.T) {
	ctx := context.Background()

	t.Run("successful batch remove with native batch support", func(t *testing.T) {
		service, mock := setupTestService(t)

		for i := 1; i <= 3; i++ {
			err := mock.Put(ctx, &storage_dto.PutParams{
				Repository:  storage_dto.StorageRepositoryDefault,
				Key:         "file" + string(rune('0'+i)) + ".txt",
				Reader:      bytes.NewReader([]byte("data")),
				Size:        4,
				ContentType: "text/plain",
			})
			require.NoError(t, err)
		}

		keys := []string{"file1.txt", "file2.txt", "file3.txt"}
		params := storage_dto.RemoveManyParams{
			Repository:      storage_dto.StorageRepositoryDefault,
			Keys:            keys,
			Concurrency:     2,
			ContinueOnError: true,
		}

		err := service.RemoveObjects(ctx, "default", params)
		require.NoError(t, err)

		removeManyCalls := mock.GetRemoveManyCalls()
		require.Len(t, removeManyCalls, 1)
		assert.Len(t, removeManyCalls[0].Keys, 3)
	})

	t.Run("remove with all keys succeeding", func(t *testing.T) {
		service, mock := setupTestService(t)

		keys := []string{"file1.txt", "file2.txt", "file3.txt"}

		params := storage_dto.RemoveManyParams{
			Repository:      storage_dto.StorageRepositoryDefault,
			Keys:            keys,
			ContinueOnError: true,
		}

		err := service.RemoveObjects(ctx, "default", params)
		require.NoError(t, err)

		removeManyCalls := mock.GetRemoveManyCalls()
		require.Len(t, removeManyCalls, 1)
		assert.Len(t, removeManyCalls[0].Keys, 3)
	})

	t.Run("empty key list is handled gracefully", func(t *testing.T) {
		service, _ := setupTestService(t)

		params := storage_dto.RemoveManyParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Keys:       []string{},
		}

		err := service.RemoveObjects(ctx, "default", params)
		require.NoError(t, err)
	})

	t.Run("validation error for invalid keys", func(t *testing.T) {
		service, _ := setupTestService(t)

		keys := []string{"../etc/passwd"}
		params := storage_dto.RemoveManyParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Keys:       keys,
		}

		err := service.RemoveObjects(ctx, "default", params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation")
	})

	t.Run("batch size exceeds limit", func(t *testing.T) {
		service, _ := setupTestService(t)

		keys := make([]string, 1001)
		for i := range keys {
			keys[i] = "file.txt"
		}

		params := storage_dto.RemoveManyParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Keys:       keys,
		}

		err := service.RemoveObjects(ctx, "default", params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum")
	})

	t.Run("idempotent removal of non-existent objects", func(t *testing.T) {
		service, _ := setupTestService(t)

		keys := []string{"nonexistent1.txt", "nonexistent2.txt"}
		params := storage_dto.RemoveManyParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Keys:       keys,
		}

		err := service.RemoveObjects(ctx, "default", params)

		require.NoError(t, err)
	})
}

func TestService_PutObjects_SequentialFallback(t *testing.T) {
	ctx := context.Background()

	t.Run("sequential fallback for provider without batch support", func(t *testing.T) {

		customMock := &nonBatchMockProvider{
			MockStorageProvider: provider_mock.NewMockStorageProvider(),
		}

		service := storage_domain.NewService(
			context.Background(),
			storage_domain.WithRetryEnabled(false),
			storage_domain.WithCircuitBreakerEnabled(false),
		)
		t.Cleanup(func() { _ = service.Close(context.Background()) })
		err := service.RegisterProvider(context.Background(), "default", customMock)
		require.NoError(t, err)

		objects := []storage_dto.PutObjectSpec{
			{
				Key:         "file1.txt",
				Reader:      bytes.NewReader([]byte("data1")),
				Size:        5,
				ContentType: "text/plain",
			},
			{
				Key:         "file2.txt",
				Reader:      bytes.NewReader([]byte("data2")),
				Size:        5,
				ContentType: "text/plain",
			},
		}

		params := &storage_dto.PutManyParams{
			Repository:      storage_dto.StorageRepositoryDefault,
			Objects:         objects,
			ContinueOnError: true,
		}

		err = service.PutObjects(ctx, "default", params)
		require.NoError(t, err)

		putCalls := customMock.GetPutCalls()
		assert.Len(t, putCalls, 2)
	})

	t.Run("sequential fallback with error", func(t *testing.T) {
		customMock := &nonBatchMockProvider{
			MockStorageProvider: provider_mock.NewMockStorageProvider(),
		}

		service := storage_domain.NewService(
			context.Background(),
			storage_domain.WithRetryEnabled(false),
			storage_domain.WithCircuitBreakerEnabled(false),
		)
		t.Cleanup(func() { _ = service.Close(context.Background()) })
		err := service.RegisterProvider(context.Background(), "default", customMock)
		require.NoError(t, err)

		customMock.SetPutError(errors.New("disk full"))

		objects := []storage_dto.PutObjectSpec{
			{
				Key:         "file1.txt",
				Reader:      bytes.NewReader([]byte("data1")),
				Size:        5,
				ContentType: "text/plain",
			},
		}

		params := &storage_dto.PutManyParams{
			Repository:      storage_dto.StorageRepositoryDefault,
			Objects:         objects,
			ContinueOnError: false,
		}

		err = service.PutObjects(ctx, "default", params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "disk full")
	})
}

func TestService_RemoveObjects_SequentialFallback(t *testing.T) {
	ctx := context.Background()

	t.Run("sequential fallback for provider without batch support", func(t *testing.T) {
		customMock := &nonBatchMockProvider{
			MockStorageProvider: provider_mock.NewMockStorageProvider(),
		}

		service := storage_domain.NewService(
			context.Background(),
			storage_domain.WithRetryEnabled(false),
			storage_domain.WithCircuitBreakerEnabled(false),
		)
		t.Cleanup(func() { _ = service.Close(context.Background()) })
		err := service.RegisterProvider(context.Background(), "default", customMock)
		require.NoError(t, err)

		keys := []string{"file1.txt", "file2.txt", "file3.txt"}
		params := storage_dto.RemoveManyParams{
			Repository:      storage_dto.StorageRepositoryDefault,
			Keys:            keys,
			ContinueOnError: true,
		}

		err = service.RemoveObjects(ctx, "default", params)
		require.NoError(t, err)

		removeCalls := customMock.GetRemoveCalls()
		assert.Len(t, removeCalls, 3)
	})
}

type nonBatchMockProvider struct {
	*provider_mock.MockStorageProvider
}

func (n *nonBatchMockProvider) SupportsBatchOperations() bool {
	return false
}
