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
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/healthprobe/healthprobe_domain"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/storage/storage_adapters/provider_mock"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/wdk/clock"
)

func setupTestService(t *testing.T, opts ...storage_domain.ServiceOption) (storage_domain.Service, *provider_mock.MockStorageProvider) {
	mock := provider_mock.NewMockStorageProvider()

	allOpts := make([]storage_domain.ServiceOption, 0, 2+len(opts))
	allOpts = append(allOpts,
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	allOpts = append(allOpts, opts...)
	service := storage_domain.NewService(context.Background(), allOpts...)
	t.Cleanup(func() { _ = service.Close(context.Background()) })
	err := service.RegisterProvider(context.Background(), "default", mock)
	require.NoError(t, err)
	return service, mock
}

func TestService_PutObject(t *testing.T) {
	ctx := context.Background()

	t.Run("successful put with small file", func(t *testing.T) {
		service, mock := setupTestService(t)

		params := &storage_dto.PutParams{
			Repository:  storage_dto.StorageRepositoryDefault,
			Key:         "test.txt",
			Reader:      bytes.NewReader([]byte("test data")),
			Size:        9,
			ContentType: "text/plain",
		}

		err := service.PutObject(ctx, "default", params)
		require.NoError(t, err)

		calls := mock.GetPutCalls()
		require.Len(t, calls, 1)
		assert.Equal(t, "test.txt", calls[0].Key)
		assert.Equal(t, int64(9), calls[0].Size)
		assert.Equal(t, "text/plain", calls[0].ContentType)
	})

	t.Run("successful put with large file triggers multipart", func(t *testing.T) {

		service, mock := setupTestService(t, storage_domain.WithMaxUploadSizeBytes(200*1024*1024))

		largeSize := int64(150 * 1024 * 1024)
		params := &storage_dto.PutParams{
			Repository:  storage_dto.StorageRepositoryDefault,
			Key:         "large.bin",
			Reader:      bytes.NewReader(make([]byte, 1024)),
			Size:        largeSize,
			ContentType: "application/octet-stream",
		}

		err := service.PutObject(ctx, "default", params)
		require.NoError(t, err)

		calls := mock.GetPutCalls()
		require.Len(t, calls, 1)

		assert.NotNil(t, calls[0].MultipartConfig)
	})

	t.Run("put with explicit multipart config", func(t *testing.T) {
		service, mock := setupTestService(t)

		multipartConfig := storage_dto.MultipartUploadConfig{
			PartSize:    10 * 1024 * 1024,
			Concurrency: 3,
		}

		params := &storage_dto.PutParams{
			Repository:      storage_dto.StorageRepositoryDefault,
			Key:             "test.txt",
			Reader:          bytes.NewReader([]byte("test data")),
			Size:            9,
			ContentType:     "text/plain",
			MultipartConfig: &multipartConfig,
		}

		err := service.PutObject(ctx, "default", params)
		require.NoError(t, err)

		calls := mock.GetPutCalls()
		require.Len(t, calls, 1)
		assert.NotNil(t, calls[0].MultipartConfig)
		assert.Equal(t, int64(10*1024*1024), calls[0].MultipartConfig.PartSize)
	})

	t.Run("put with metadata", func(t *testing.T) {
		service, mock := setupTestService(t)

		metadata := map[string]string{
			"author": "test-user",
			"type":   "document",
		}

		params := &storage_dto.PutParams{
			Repository:  storage_dto.StorageRepositoryDefault,
			Key:         "test.txt",
			Reader:      bytes.NewReader([]byte("test data")),
			Size:        9,
			ContentType: "text/plain",
			Metadata:    metadata,
		}

		err := service.PutObject(ctx, "default", params)
		require.NoError(t, err)

		calls := mock.GetPutCalls()
		require.Len(t, calls, 1)
		assert.Equal(t, metadata, calls[0].Metadata)
	})

	t.Run("put with validation error", func(t *testing.T) {
		service, _ := setupTestService(t)

		params := &storage_dto.PutParams{
			Repository:  storage_dto.StorageRepositoryDefault,
			Key:         "",
			Reader:      bytes.NewReader([]byte("test data")),
			Size:        9,
			ContentType: "text/plain",
		}

		err := service.PutObject(ctx, "default", params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "key cannot be empty")
	})

	t.Run("put with provider error", func(t *testing.T) {
		service, mock := setupTestService(t)
		mock.SetPutError(errors.New("disk full"))

		params := &storage_dto.PutParams{
			Repository:  storage_dto.StorageRepositoryDefault,
			Key:         "test.txt",
			Reader:      bytes.NewReader([]byte("test data")),
			Size:        9,
			ContentType: "text/plain",
		}

		err := service.PutObject(ctx, "default", params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "disk full")
	})

	t.Run("put with non-existent provider", func(t *testing.T) {
		service, _ := setupTestService(t)

		params := &storage_dto.PutParams{
			Repository:  storage_dto.StorageRepositoryDefault,
			Key:         "test.txt",
			Reader:      bytes.NewReader([]byte("test data")),
			Size:        9,
			ContentType: "text/plain",
		}

		err := service.PutObject(ctx, "nonexistent", params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestService_GetObject(t *testing.T) {
	ctx := context.Background()

	t.Run("successful get", func(t *testing.T) {
		service, mock := setupTestService(t)

		testData := []byte("test data")

		err := mock.Put(ctx, &storage_dto.PutParams{
			Repository:  storage_dto.StorageRepositoryDefault,
			Key:         "test.txt",
			Reader:      bytes.NewReader(testData),
			Size:        int64(len(testData)),
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		reader, err := service.GetObject(ctx, "default", storage_dto.GetParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Key:        "test.txt",
		})
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, testData, data)
	})

	t.Run("get with byte range", func(t *testing.T) {
		service, mock := setupTestService(t)

		testData := []byte("0123456789")
		err := mock.Put(ctx, &storage_dto.PutParams{
			Repository:  storage_dto.StorageRepositoryDefault,
			Key:         "test.txt",
			Reader:      bytes.NewReader(testData),
			Size:        int64(len(testData)),
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		reader, err := service.GetObject(ctx, "default", storage_dto.GetParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Key:        "test.txt",
			ByteRange: &storage_dto.ByteRange{
				Start: 2,
				End:   5,
			},
		})
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, err := io.ReadAll(reader)
		require.NoError(t, err)

		assert.Equal(t, []byte("2345"), data)
	})

	t.Run("get with singleflight for small file", func(t *testing.T) {
		service, mock := setupTestService(t)

		testData := []byte("small test data")
		err := mock.Put(ctx, &storage_dto.PutParams{
			Repository:  storage_dto.StorageRepositoryDefault,
			Key:         "small.txt",
			Reader:      bytes.NewReader(testData),
			Size:        int64(len(testData)),
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		reader, err := service.GetObject(ctx, "default", storage_dto.GetParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Key:        "small.txt",
		})
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, testData, data)
	})

	t.Run("get non-existent object", func(t *testing.T) {
		service, _ := setupTestService(t)

		_, err := service.GetObject(ctx, "default", storage_dto.GetParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Key:        "nonexistent.txt",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("get with validation error", func(t *testing.T) {
		service, _ := setupTestService(t)

		_, err := service.GetObject(ctx, "default", storage_dto.GetParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Key:        "",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "key cannot be empty")
	})
}

func TestService_GetObject_Concurrent(t *testing.T) {
	ctx := context.Background()

	service, mock := setupTestService(t)

	testData := []byte("concurrent test data")
	err := mock.Put(ctx, &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "concurrent.txt",
		Reader:      bytes.NewReader(testData),
		Size:        int64(len(testData)),
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	numGoroutines := 10
	var wg sync.WaitGroup
	errs := make(chan error, numGoroutines)

	for range numGoroutines {
		wg.Go(func() {
			reader, err := service.GetObject(ctx, "default", storage_dto.GetParams{
				Repository: storage_dto.StorageRepositoryDefault,
				Key:        "concurrent.txt",
			})
			if err != nil {
				errs <- err
				return
			}
			defer func() { _ = reader.Close() }()

			data, err := io.ReadAll(reader)
			if err != nil {
				errs <- err
				return
			}
			if !bytes.Equal(data, testData) {
				errs <- errors.New("data mismatch")
			}
		})
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("Concurrent get failed: %v", err)
	}
}

func TestService_StatObject(t *testing.T) {
	ctx := context.Background()

	t.Run("successful stat", func(t *testing.T) {
		service, mock := setupTestService(t)

		testData := []byte("test data")
		metadata := map[string]string{"key": "value"}
		err := mock.Put(ctx, &storage_dto.PutParams{
			Repository:  storage_dto.StorageRepositoryDefault,
			Key:         "test.txt",
			Reader:      bytes.NewReader(testData),
			Size:        int64(len(testData)),
			ContentType: "text/plain",
			Metadata:    metadata,
		})
		require.NoError(t, err)

		info, err := service.StatObject(ctx, "default", storage_dto.GetParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Key:        "test.txt",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(len(testData)), info.Size)
		assert.Equal(t, "text/plain", info.ContentType)
		assert.Equal(t, metadata, info.Metadata)
	})

	t.Run("stat non-existent object", func(t *testing.T) {
		service, _ := setupTestService(t)

		_, err := service.StatObject(ctx, "default", storage_dto.GetParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Key:        "nonexistent.txt",
		})
		require.Error(t, err)
	})
}

func TestService_CopyObject(t *testing.T) {
	ctx := context.Background()

	t.Run("copy within same repository", func(t *testing.T) {
		service, mock := setupTestService(t)

		testData := []byte("test data")
		err := mock.Put(ctx, &storage_dto.PutParams{
			Repository:  storage_dto.StorageRepositoryDefault,
			Key:         "source.txt",
			Reader:      bytes.NewReader(testData),
			Size:        int64(len(testData)),
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		err = service.CopyObject(ctx, "default", storage_dto.CopyParams{
			SourceRepository:      storage_dto.StorageRepositoryDefault,
			SourceKey:             "source.txt",
			DestinationRepository: storage_dto.StorageRepositoryDefault,
			DestinationKey:        "dest.txt",
		})
		require.NoError(t, err)

		copyCalls := mock.GetCopyCalls()
		require.Len(t, copyCalls, 1)
		assert.Equal(t, "source.txt", copyCalls[0].SourceKey)
		assert.Equal(t, "dest.txt", copyCalls[0].DestinationKey)
	})

	t.Run("copy across repositories", func(t *testing.T) {
		service, mock := setupTestService(t)

		testData := []byte("test data")
		err := mock.Put(ctx, &storage_dto.PutParams{
			Repository:  "repo1",
			Key:         "source.txt",
			Reader:      bytes.NewReader(testData),
			Size:        int64(len(testData)),
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		err = service.CopyObject(ctx, "default", storage_dto.CopyParams{
			SourceRepository:      "repo1",
			SourceKey:             "source.txt",
			DestinationRepository: "repo2",
			DestinationKey:        "dest.txt",
		})
		require.NoError(t, err)

		copyCalls := mock.GetCopyCalls()
		require.Len(t, copyCalls, 1)
		assert.Equal(t, "repo1", copyCalls[0].SourceRepository)
		assert.Equal(t, "repo2", copyCalls[0].DestinationRepository)
	})

	t.Run("copy with validation error", func(t *testing.T) {
		service, _ := setupTestService(t)

		err := service.CopyObject(ctx, "default", storage_dto.CopyParams{
			SourceRepository:      storage_dto.StorageRepositoryDefault,
			SourceKey:             "",
			DestinationRepository: storage_dto.StorageRepositoryDefault,
			DestinationKey:        "dest.txt",
		})
		require.Error(t, err)
	})
}

func TestService_RemoveObject(t *testing.T) {
	ctx := context.Background()

	t.Run("successful remove", func(t *testing.T) {
		service, mock := setupTestService(t)

		testData := []byte("test data")
		err := mock.Put(ctx, &storage_dto.PutParams{
			Repository:  storage_dto.StorageRepositoryDefault,
			Key:         "test.txt",
			Reader:      bytes.NewReader(testData),
			Size:        int64(len(testData)),
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		err = service.RemoveObject(ctx, "default", storage_dto.GetParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Key:        "test.txt",
		})
		require.NoError(t, err)

		removeCalls := mock.GetRemoveCalls()
		require.Len(t, removeCalls, 1)
		assert.Equal(t, "test.txt", removeCalls[0].Key)
	})

	t.Run("remove non-existent object is idempotent", func(t *testing.T) {
		service, _ := setupTestService(t)

		err := service.RemoveObject(ctx, "default", storage_dto.GetParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Key:        "nonexistent.txt",
		})

		require.NoError(t, err)
	})

	t.Run("remove with validation error", func(t *testing.T) {
		service, _ := setupTestService(t)

		err := service.RemoveObject(ctx, "default", storage_dto.GetParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Key:        "",
		})
		require.Error(t, err)
	})
}

func TestService_GetObjectHash(t *testing.T) {
	ctx := context.Background()

	t.Run("successful hash retrieval", func(t *testing.T) {
		service, mock := setupTestService(t)

		testData := []byte("test data")
		err := mock.Put(ctx, &storage_dto.PutParams{
			Repository:  storage_dto.StorageRepositoryDefault,
			Key:         "test.txt",
			Reader:      bytes.NewReader(testData),
			Size:        int64(len(testData)),
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		hash, err := service.GetObjectHash(ctx, "default", storage_dto.GetParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Key:        "test.txt",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, hash)

		assert.Len(t, hash, 64)
	})

	t.Run("hash of non-existent object", func(t *testing.T) {
		service, _ := setupTestService(t)

		_, err := service.GetObjectHash(ctx, "default", storage_dto.GetParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Key:        "nonexistent.txt",
		})
		require.Error(t, err)
	})
}

func TestService_GeneratePresignedUploadURL(t *testing.T) {
	ctx := context.Background()

	t.Run("successful presigned URL generation", func(t *testing.T) {
		service, mock := setupTestService(t)

		expectedURL := "https://mock.storage.local/test/test.txt?presigned=true"
		mock.SetPresignedURL(expectedURL)

		url, err := service.GeneratePresignedUploadURL(ctx, "default", storage_dto.PresignParams{
			Repository:  "test",
			Key:         "test.txt",
			ContentType: "text/plain",
			ExpiresIn:   15 * time.Minute,
		})
		require.NoError(t, err)
		assert.Equal(t, expectedURL, url)

		presignCalls := mock.GetPresignURLCalls()
		require.Len(t, presignCalls, 1)
		assert.Equal(t, "test.txt", presignCalls[0].Key)
		assert.Equal(t, "text/plain", presignCalls[0].ContentType)
	})

	t.Run("presigned URL with provider error", func(t *testing.T) {
		service, mock := setupTestService(t)
		mock.SetPresignError(errors.New("presign not supported"))

		_, err := service.GeneratePresignedUploadURL(ctx, "default", storage_dto.PresignParams{
			Repository:  "test",
			Key:         "test.txt",
			ContentType: "text/plain",
			ExpiresIn:   15 * time.Minute,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "presign not supported")
	})
}

func TestService_WithMockClock(t *testing.T) {
	ctx := context.Background()
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithClock(mockClock),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	require.NotNil(t, service)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	mock := provider_mock.NewMockStorageProvider()
	err := service.RegisterProvider(context.Background(), "default", mock)
	require.NoError(t, err)

	stats := service.GetStats(ctx)
	assert.Equal(t, startTime, stats.StartTime)

	mockClock.Advance(1 * time.Hour)
	uptime := stats.UptimeAt(mockClock.Now())
	assert.Equal(t, 1*time.Hour, uptime)
}

func TestService_MetricsTimingWithMockClock(t *testing.T) {
	ctx := context.Background()
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithClock(mockClock),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	mock := provider_mock.NewMockStorageProvider()
	err := service.RegisterProvider(context.Background(), "default", mock)
	require.NoError(t, err)

	testData := []byte("test data")
	params := &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "test.txt",
		Reader:      bytes.NewReader(testData),
		Size:        int64(len(testData)),
		ContentType: "text/plain",
	}

	err = service.PutObject(ctx, "default", params)
	require.NoError(t, err)

	stats := service.GetStats(ctx)
	assert.Equal(t, int64(1), stats.TotalOperations)
	assert.Equal(t, int64(1), stats.SuccessfulOperations)
}

func TestService_GeneratePresignedDownloadURL(t *testing.T) {
	ctx := context.Background()

	t.Run("delegates to provider with native presigned URL support", func(t *testing.T) {
		service, mock := setupTestService(t)

		expectedURL := "https://mock.storage.local/download/test.txt"
		mock.SetPresignedURL(expectedURL)

		url, err := service.GeneratePresignedDownloadURL(ctx, "default", storage_dto.PresignDownloadParams{
			Repository: "test",
			Key:        "test.txt",
			ExpiresIn:  15 * time.Minute,
		})
		require.NoError(t, err)
		assert.Equal(t, expectedURL, url)
	})

	t.Run("returns an error when provider fails", func(t *testing.T) {
		service, mock := setupTestService(t)
		mock.SetPresignError(errors.New("presign download failed"))

		_, err := service.GeneratePresignedDownloadURL(ctx, "default", storage_dto.PresignDownloadParams{
			Repository: "test",
			Key:        "test.txt",
			ExpiresIn:  15 * time.Minute,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "presign download failed")
	})

	t.Run("returns an error for a non-existent provider", func(t *testing.T) {
		service, _ := setupTestService(t)

		_, err := service.GeneratePresignedDownloadURL(ctx, "nonexistent", storage_dto.PresignDownloadParams{
			Repository: "test",
			Key:        "test.txt",
			ExpiresIn:  15 * time.Minute,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestService_Check_Liveness(t *testing.T) {
	t.Run("healthy when providers are registered", func(t *testing.T) {
		service, _ := setupTestService(t)
		ctx := context.Background()

		probe, ok := service.(healthprobe_domain.Probe)
		require.True(t, ok, "service should implement the Probe interface")

		status := probe.Check(ctx, healthprobe_dto.CheckTypeLiveness)
		assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
		assert.Equal(t, "StorageService", status.Name)
		assert.Contains(t, status.Message, "running")
	})

	t.Run("unhealthy when no providers are registered", func(t *testing.T) {
		service := storage_domain.NewService(
			context.Background(),
			storage_domain.WithRetryEnabled(false),
			storage_domain.WithCircuitBreakerEnabled(false),
		)
		t.Cleanup(func() { _ = service.Close(context.Background()) })
		ctx := context.Background()

		probe, ok := service.(healthprobe_domain.Probe)
		require.True(t, ok, "service should implement the Probe interface")

		status := probe.Check(ctx, healthprobe_dto.CheckTypeLiveness)
		assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
		assert.Contains(t, status.Message, "No storage providers registered")
	})
}

func TestService_Check_Readiness(t *testing.T) {
	t.Run("reports readiness status with providers", func(t *testing.T) {
		service, _ := setupTestService(t)
		ctx := context.Background()

		probe, ok := service.(healthprobe_domain.Probe)
		require.True(t, ok, "service should implement the Probe interface")

		status := probe.Check(ctx, healthprobe_dto.CheckTypeReadiness)
		assert.Equal(t, "StorageService", status.Name)

		assert.NotEmpty(t, status.Message)
	})
}
