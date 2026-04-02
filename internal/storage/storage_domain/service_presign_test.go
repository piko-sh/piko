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
	"crypto/rand"
	"io"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_adapters/provider_mock"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/wdk/clock"
)

type noPresignMockProvider struct {
	*provider_mock.MockStorageProvider
}

func (m *noPresignMockProvider) SupportsPresignedURLs() bool {
	return false
}

func setupPresignFallbackService(t *testing.T) (storage_domain.Service, *provider_mock.MockStorageProvider) {
	t.Helper()

	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	require.NoError(t, err)

	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
		storage_domain.WithPresignConfig(storage_domain.PresignConfig{
			Secret:             secret,
			DefaultExpiry:      15 * time.Minute,
			MaxExpiry:          1 * time.Hour,
			DefaultMaxSize:     100 * 1024 * 1024,
			MaxMaxSize:         1024 * 1024 * 1024,
			RateLimitPerMinute: 50,
		}),
		storage_domain.WithPresignFallbackBaseURL("https://example.com"),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	mock := provider_mock.NewMockStorageProvider()
	wrapper := &noPresignMockProvider{MockStorageProvider: mock}
	err = service.RegisterProvider(context.Background(), "default", wrapper)
	require.NoError(t, err)

	return service, mock
}

func TestPresignFallback_UploadURL(t *testing.T) {
	ctx := context.Background()

	t.Run("generates a valid fallback upload URL", func(t *testing.T) {
		service, _ := setupPresignFallbackService(t)

		result, err := service.GeneratePresignedUploadURL(ctx, "default", storage_dto.PresignParams{
			Repository:  "test",
			Key:         "upload.txt",
			ContentType: "text/plain",
			ExpiresIn:   10 * time.Minute,
		})
		require.NoError(t, err)
		assert.Contains(t, result, "/_piko/storage/upload")
		assert.Contains(t, result, "token=")
		assert.Contains(t, result, "provider=default")
		assert.True(t, strings.HasPrefix(result, "https://example.com"),
			"URL should start with the configured base URL")
	})

	t.Run("generates a fallback upload URL without base URL", func(t *testing.T) {
		secret := make([]byte, 32)
		_, err := rand.Read(secret)
		require.NoError(t, err)

		service := storage_domain.NewService(
			context.Background(),
			storage_domain.WithRetryEnabled(false),
			storage_domain.WithCircuitBreakerEnabled(false),
			storage_domain.WithPresignConfig(storage_domain.PresignConfig{
				Secret:         secret,
				DefaultExpiry:  15 * time.Minute,
				MaxExpiry:      1 * time.Hour,
				DefaultMaxSize: 100 * 1024 * 1024,
				MaxMaxSize:     1024 * 1024 * 1024,
			}),
		)
		t.Cleanup(func() { _ = service.Close(context.Background()) })
		mock := provider_mock.NewMockStorageProvider()
		wrapper := &noPresignMockProvider{MockStorageProvider: mock}
		require.NoError(t, service.RegisterProvider(context.Background(), "default", wrapper))

		result, err := service.GeneratePresignedUploadURL(ctx, "default", storage_dto.PresignParams{
			Repository:  "test",
			Key:         "upload.txt",
			ContentType: "text/plain",
			ExpiresIn:   10 * time.Minute,
		})
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(result, "/_piko/storage/upload?"),
			"URL should be a relative path when no base URL is set")
	})

	t.Run("upload URL contains a valid parseable token", func(t *testing.T) {
		service, _ := setupPresignFallbackService(t)

		result, err := service.GeneratePresignedUploadURL(ctx, "default", storage_dto.PresignParams{
			Repository:  "media",
			Key:         "image.png",
			ContentType: "image/png",
			ExpiresIn:   5 * time.Minute,
		})
		require.NoError(t, err)

		parsed, err := url.Parse(result)
		require.NoError(t, err)
		tok := parsed.Query().Get("token")
		assert.NotEmpty(t, tok, "token query parameter should be present")
		assert.Contains(t, tok, ".", "token should have payload.signature format")
	})
}

func TestPresignFallback_DownloadURL(t *testing.T) {
	ctx := context.Background()

	t.Run("generates a valid fallback download URL", func(t *testing.T) {
		service, _ := setupPresignFallbackService(t)

		result, err := service.GeneratePresignedDownloadURL(ctx, "default", storage_dto.PresignDownloadParams{
			Repository:  "test",
			Key:         "download.txt",
			FileName:    "download.txt",
			ContentType: "text/plain",
			ExpiresIn:   10 * time.Minute,
		})
		require.NoError(t, err)
		assert.Contains(t, result, "/_piko/storage/download")
		assert.Contains(t, result, "token=")
		assert.Contains(t, result, "provider=default")
		assert.True(t, strings.HasPrefix(result, "https://example.com"),
			"URL should start with the configured base URL")
	})

	t.Run("generates a fallback download URL without base URL", func(t *testing.T) {
		secret := make([]byte, 32)
		_, err := rand.Read(secret)
		require.NoError(t, err)

		service := storage_domain.NewService(
			context.Background(),
			storage_domain.WithRetryEnabled(false),
			storage_domain.WithCircuitBreakerEnabled(false),
			storage_domain.WithPresignConfig(storage_domain.PresignConfig{
				Secret:         secret,
				DefaultExpiry:  15 * time.Minute,
				MaxExpiry:      1 * time.Hour,
				DefaultMaxSize: 100 * 1024 * 1024,
				MaxMaxSize:     1024 * 1024 * 1024,
			}),
		)
		t.Cleanup(func() { _ = service.Close(context.Background()) })
		mock := provider_mock.NewMockStorageProvider()
		wrapper := &noPresignMockProvider{MockStorageProvider: mock}
		require.NoError(t, service.RegisterProvider(context.Background(), "default", wrapper))

		result, err := service.GeneratePresignedDownloadURL(ctx, "default", storage_dto.PresignDownloadParams{
			Repository: "test",
			Key:        "download.txt",
			ExpiresIn:  10 * time.Minute,
		})
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(result, "/_piko/storage/download?"),
			"URL should be a relative path when no base URL is set")
	})
}

func TestGeneratePresignDownloadToken(t *testing.T) {
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	require.NoError(t, err)

	t.Run("generates and verifies a valid download token", func(t *testing.T) {
		data := storage_domain.PresignDownloadTokenData{
			Key:         "files/report.pdf",
			Repository:  "documents",
			FileName:    "report.pdf",
			ContentType: "application/pdf",
			ExpiresAt:   time.Now().Add(15 * time.Minute).Unix(),
			RID:         "test-rid-123",
		}

		tok, err := storage_domain.GeneratePresignDownloadToken(secret, data)
		require.NoError(t, err)
		assert.NotEmpty(t, tok)
		assert.Contains(t, tok, ".", "token should contain a delimiter")

		parsed, err := storage_domain.ParseAndVerifyPresignDownloadToken(secret, tok)
		require.NoError(t, err)
		assert.Equal(t, data.Key, parsed.Key)
		assert.Equal(t, data.Repository, parsed.Repository)
		assert.Equal(t, data.FileName, parsed.FileName)
		assert.Equal(t, data.ContentType, parsed.ContentType)
		assert.Equal(t, data.RID, parsed.RID)
	})

	t.Run("rejects token with wrong secret", func(t *testing.T) {
		data := storage_domain.PresignDownloadTokenData{
			Key:        "files/report.pdf",
			Repository: "documents",
			ExpiresAt:  time.Now().Add(15 * time.Minute).Unix(),
			RID:        "test-rid",
		}

		tok, err := storage_domain.GeneratePresignDownloadToken(secret, data)
		require.NoError(t, err)

		wrongSecret := make([]byte, 32)
		_, err = rand.Read(wrongSecret)
		require.NoError(t, err)

		_, err = storage_domain.ParseAndVerifyPresignDownloadToken(wrongSecret, tok)
		require.Error(t, err)
	})

	t.Run("rejects expired download token", func(t *testing.T) {
		data := storage_domain.PresignDownloadTokenData{
			Key:        "old-file.txt",
			Repository: "archive",
			ExpiresAt:  time.Now().Add(-1 * time.Hour).Unix(),
			RID:        "expired-rid",
		}

		tok, err := storage_domain.GeneratePresignDownloadToken(secret, data)
		require.NoError(t, err)

		_, err = storage_domain.ParseAndVerifyPresignDownloadToken(secret, tok)
		require.Error(t, err)
	})

	t.Run("rejects too-short secret", func(t *testing.T) {
		shortSecret := make([]byte, 16)
		data := storage_domain.PresignDownloadTokenData{
			Key:        "file.txt",
			Repository: "test",
			ExpiresAt:  time.Now().Add(15 * time.Minute).Unix(),
			RID:        "rid",
		}

		_, err := storage_domain.GeneratePresignDownloadToken(shortSecret, data)
		require.Error(t, err)
	})

	t.Run("rejects malformed download token", func(t *testing.T) {
		_, err := storage_domain.ParseAndVerifyPresignDownloadToken(secret, "not-a-valid-token")
		require.Error(t, err)
	})
}

func TestPresignDownloadTokenData_IsExpired(t *testing.T) {
	t.Run("returns false for future expiry", func(t *testing.T) {
		data := &storage_domain.PresignDownloadTokenData{
			ExpiresAt: time.Now().Add(10 * time.Minute).Unix(),
		}
		assert.False(t, data.IsExpired())
	})

	t.Run("returns true for past expiry", func(t *testing.T) {
		data := &storage_domain.PresignDownloadTokenData{
			ExpiresAt: time.Now().Add(-10 * time.Minute).Unix(),
		}
		assert.True(t, data.IsExpired())
	})
}

func TestService_GetObject_Singleflight(t *testing.T) {
	ctx := context.Background()

	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
		storage_domain.WithSingleflightEnabled(true),
		storage_domain.WithSingleflightMemoryThreshold(1024*1024),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	mock := provider_mock.NewMockStorageProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "default", mock))

	testData := []byte("singleflight test data")
	putParams := &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "sf-test.txt",
		Reader:      bytes.NewReader(testData),
		Size:        int64(len(testData)),
		ContentType: "text/plain",
	}
	require.NoError(t, service.PutObject(ctx, "default", putParams))

	getParams := storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "sf-test.txt",
	}

	t.Run("returns correct content via singleflight", func(t *testing.T) {
		reader, err := service.GetObject(ctx, "default", getParams)
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, testData, data)
	})

	t.Run("concurrent reads use singleflight deduplication", func(t *testing.T) {
		const concurrency = 10
		var wg sync.WaitGroup
		results := make([][]byte, concurrency)
		errs := make([]error, concurrency)

		wg.Add(concurrency)
		for i := range concurrency {
			go func(index int) {
				defer wg.Done()
				reader, err := service.GetObject(ctx, "default", getParams)
				if err != nil {
					errs[index] = err
					return
				}
				defer func() { _ = reader.Close() }()
				data, err := io.ReadAll(reader)
				if err != nil {
					errs[index] = err
					return
				}
				results[index] = data
			}(i)
		}
		wg.Wait()

		for i := range concurrency {
			require.NoError(t, errs[i], "goroutine %d should not fail", i)
			assert.Equal(t, testData, results[i], "goroutine %d should get correct data", i)
		}
	})
}

func TestNewServiceWithDefaultProvider(t *testing.T) {
	service := storage_domain.NewServiceWithDefaultProvider("primary",
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	require.NotNil(t, service)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	mock := provider_mock.NewMockStorageProvider()
	err := service.RegisterProvider(context.Background(), "primary", mock)
	require.NoError(t, err)

	ctx := context.Background()

	testData := []byte("hello")
	putParams := &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "test.txt",
		Reader:      bytes.NewReader(testData),
		Size:        int64(len(testData)),
		ContentType: "text/plain",
	}
	err = service.PutObject(ctx, "primary", putParams)
	require.NoError(t, err)

	calls := mock.GetPutCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "test.txt", calls[0].Key)
}

func TestService_Name(t *testing.T) {
	service, _ := setupTestService(t)

	type namer interface {
		Name() string
	}
	probe, ok := service.(namer)
	require.True(t, ok, "service should expose Name()")
	assert.Equal(t, "StorageService", probe.Name())
}

func TestService_GetPresignConfig(t *testing.T) {
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	require.NoError(t, err)

	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
		storage_domain.WithPresignConfig(storage_domain.PresignConfig{
			Secret:         secret,
			DefaultExpiry:  20 * time.Minute,
			MaxExpiry:      45 * time.Minute,
			DefaultMaxSize: 50 * 1024 * 1024,
			MaxMaxSize:     512 * 1024 * 1024,
		}),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	type presignConfigGetter interface {
		GetPresignConfig() storage_domain.PresignConfig
	}
	getter, ok := service.(presignConfigGetter)
	require.True(t, ok, "service should expose GetPresignConfig()")

	config := getter.GetPresignConfig()
	assert.Equal(t, secret, config.Secret)
	assert.Equal(t, 20*time.Minute, config.DefaultExpiry)
	assert.Equal(t, 45*time.Minute, config.MaxExpiry)
}

func TestServiceOptions_PresignAndClock(t *testing.T) {
	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	require.NoError(t, err)

	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithClock(mockClock),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
		storage_domain.WithPresignConfig(storage_domain.PresignConfig{
			Secret:         secret,
			DefaultExpiry:  10 * time.Minute,
			MaxExpiry:      30 * time.Minute,
			DefaultMaxSize: 50 * 1024 * 1024,
			MaxMaxSize:     512 * 1024 * 1024,
		}),
		storage_domain.WithPresignFallbackBaseURL("https://cdn.example.com"),
	)
	require.NotNil(t, service)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	mock := provider_mock.NewMockStorageProvider()
	wrapper := &noPresignMockProvider{MockStorageProvider: mock}
	require.NoError(t, service.RegisterProvider(context.Background(), "default", wrapper))

	ctx := context.Background()

	result, err := service.GeneratePresignedUploadURL(ctx, "default", storage_dto.PresignParams{
		Repository:  "test",
		Key:         "timed.txt",
		ContentType: "text/plain",
		ExpiresIn:   5 * time.Minute,
	})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(result, "https://cdn.example.com"),
		"should use the configured presign fallback base URL")
}
