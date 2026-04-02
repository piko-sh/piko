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
	"time"

	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_adapters/provider_mock"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

func setupWrapper(t *testing.T) (*storage_domain.CircuitBreakerWrapper, *provider_mock.MockStorageProvider, storage_domain.CircuitBreakerConfig) {
	mockProvider := provider_mock.NewMockStorageProvider()

	config := storage_domain.CircuitBreakerConfig{
		MaxConsecutiveFailures: 2,
		Timeout:                100 * time.Millisecond,
		Interval:               0,
	}

	wrapper := storage_domain.NewCircuitBreakerWrapper(context.Background(), mockProvider, config, "test-provider")
	require.NotNil(t, wrapper)

	return wrapper, mockProvider, config
}

func TestCircuitBreakerWrapper_New(t *testing.T) {
	wrapper, _, _ := setupWrapper(t)
	assert.Equal(t, gobreaker.StateClosed, wrapper.GetState(), "Initial state should be Closed")
}

func TestCircuitBreakerWrapper_HappyPath_StaysClosed(t *testing.T) {
	ctx := context.Background()
	testRepo := storage_dto.StorageRepositoryDefault
	testKey := "path/to/object.txt"
	testData := []byte("hello world")

	testCases := []struct {
		operation   func(w *storage_domain.CircuitBreakerWrapper) error
		setupState  func(m *provider_mock.MockStorageProvider)
		assertMock  func(t *testing.T, m *provider_mock.MockStorageProvider)
		setupResult func(m *provider_mock.MockStorageProvider)
		name        string
	}{
		{
			name: "Put succeeds",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				return w.Put(ctx, &storage_dto.PutParams{
					Repository: testRepo,
					Key:        testKey,
					Reader:     bytes.NewReader(testData),
				})
			},
			setupState: nil,
			assertMock: func(t *testing.T, m *provider_mock.MockStorageProvider) {
				assert.Len(t, m.GetPutCalls(), 1)
			},
		},
		{
			name: "Get succeeds",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				rc, err := w.Get(ctx, storage_dto.GetParams{Repository: testRepo, Key: testKey})
				if err == nil {
					_ = rc.Close()
				}
				return err
			},

			setupState: func(m *provider_mock.MockStorageProvider) {
				_ = m.Put(ctx, &storage_dto.PutParams{Repository: testRepo, Key: testKey, Reader: bytes.NewReader(testData)})
			},
			assertMock: func(t *testing.T, m *provider_mock.MockStorageProvider) {
				assert.Len(t, m.GetGetCalls(), 1)
			},
		},
		{
			name: "Stat succeeds",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				_, err := w.Stat(ctx, storage_dto.GetParams{Repository: testRepo, Key: testKey})
				return err
			},

			setupState: func(m *provider_mock.MockStorageProvider) {
				_ = m.Put(ctx, &storage_dto.PutParams{Repository: testRepo, Key: testKey, Reader: bytes.NewReader(testData)})
			},
			assertMock: func(t *testing.T, m *provider_mock.MockStorageProvider) {
				assert.Len(t, m.GetStatCalls(), 1)
			},
		},
		{
			name: "Remove succeeds",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				return w.Remove(ctx, storage_dto.GetParams{Repository: testRepo, Key: testKey})
			},

			setupState: func(m *provider_mock.MockStorageProvider) {
				_ = m.Put(ctx, &storage_dto.PutParams{Repository: testRepo, Key: testKey, Reader: bytes.NewReader(testData)})
			},
			assertMock: func(t *testing.T, m *provider_mock.MockStorageProvider) {
				assert.Len(t, m.GetRemoveCalls(), 1)
			},
		},
		{
			name: "Copy succeeds",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				return w.Copy(ctx, testRepo, testKey, "path/to/copy.txt")
			},

			setupState: func(m *provider_mock.MockStorageProvider) {
				_ = m.Put(ctx, &storage_dto.PutParams{Repository: testRepo, Key: testKey, Reader: bytes.NewReader(testData)})
			},
			assertMock: func(t *testing.T, m *provider_mock.MockStorageProvider) {
				assert.Len(t, m.GetCopyCalls(), 1)
			},
		},
		{
			name: "CopyToAnotherRepository succeeds",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				return w.CopyToAnotherRepository(ctx, testRepo, testKey, "test", "path/to/copy.txt")
			},

			setupState: func(m *provider_mock.MockStorageProvider) {
				_ = m.Put(ctx, &storage_dto.PutParams{Repository: testRepo, Key: testKey, Reader: bytes.NewReader(testData)})
			},
			assertMock: func(t *testing.T, m *provider_mock.MockStorageProvider) {
				assert.Len(t, m.GetCopyCalls(), 1)
			},
		},
		{
			name: "GetHash succeeds",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				_, err := w.GetHash(ctx, storage_dto.GetParams{Repository: testRepo, Key: testKey})
				return err
			},

			setupState: func(m *provider_mock.MockStorageProvider) {
				_ = m.Put(ctx, &storage_dto.PutParams{Repository: testRepo, Key: testKey, Reader: bytes.NewReader(testData)})
			},
			assertMock: func(t *testing.T, m *provider_mock.MockStorageProvider) {
				assert.Len(t, m.GetGetHashCalls(), 1)
			},
		},
		{
			name: "PresignURL succeeds",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				_, err := w.PresignURL(ctx, storage_dto.PresignParams{Repository: testRepo, Key: testKey})
				return err
			},
			setupState: nil,
			assertMock: func(t *testing.T, m *provider_mock.MockStorageProvider) {
				assert.Len(t, m.GetPresignURLCalls(), 1)
			},
		},
		{
			name: "PutMany succeeds",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				_, err := w.PutMany(ctx, &storage_dto.PutManyParams{Repository: testRepo})
				return err
			},
			setupState: nil,
			assertMock: func(t *testing.T, m *provider_mock.MockStorageProvider) {
				assert.Len(t, m.GetPutManyCalls(), 1)
			},
		},
		{
			name: "RemoveMany succeeds",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				_, err := w.RemoveMany(ctx, storage_dto.RemoveManyParams{Repository: testRepo})
				return err
			},
			setupState: nil,
			assertMock: func(t *testing.T, m *provider_mock.MockStorageProvider) {
				assert.Len(t, m.GetRemoveManyCalls(), 1)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wrapper, mock, _ := setupWrapper(t)

			if tc.setupState != nil {
				tc.setupState(mock)
			}

			mock.SetError(nil)
			if tc.setupResult != nil {
				tc.setupResult(mock)
			}

			err := tc.operation(wrapper)

			assert.NoError(t, err, "Operation should succeed")
			assert.Equal(t, gobreaker.StateClosed, wrapper.GetState(), "State should remain Closed")
			tc.assertMock(t, mock)
			counts := wrapper.GetCounts()
			assert.Equal(t, uint32(1), counts.Requests, "Should be one request")
			assert.Equal(t, uint32(1), counts.TotalSuccesses, "Should be one success")
			assert.Equal(t, uint32(0), counts.TotalFailures, "Should be zero failures")
		})
	}
}

func TestCircuitBreakerWrapper_TripsToOpenState(t *testing.T) {
	wrapper, mock, config := setupWrapper(t)
	failErr := errors.New("provider failure")
	mock.SetError(failErr)
	ctx := context.Background()
	putParams := &storage_dto.PutParams{Reader: bytes.NewReader(nil)}

	err := wrapper.Put(ctx, putParams)
	require.ErrorIs(t, err, failErr, "Should return the underlying error")
	assert.Equal(t, gobreaker.StateClosed, wrapper.GetState(), "State should be Closed after 1st failure")
	assert.Equal(t, uint32(1), wrapper.GetCounts().ConsecutiveFailures)
	assert.Len(t, mock.GetPutCalls(), 1, "Mock should be called for the 1st failure")

	err = wrapper.Put(ctx, putParams)
	require.ErrorIs(t, err, failErr, "Should return the underlying error")
	assert.Equal(t, gobreaker.StateOpen, wrapper.GetState(), "State should be Open after 2nd failure")

	assert.Len(t, mock.GetPutCalls(), int(config.MaxConsecutiveFailures), "Mock should have been called for each failure leading to trip")
}

func TestCircuitBreakerWrapper_BlocksCallsInOpenState(t *testing.T) {
	wrapper, mock, _ := setupWrapper(t)
	failErr := errors.New("provider failure")
	mock.SetError(failErr)
	ctx := context.Background()

	for range 2 {
		_ = wrapper.Put(ctx, &storage_dto.PutParams{Reader: bytes.NewReader(nil)})
	}
	require.Equal(t, gobreaker.StateOpen, wrapper.GetState())

	initialCallCount := mock.GetTotalCallCount()

	testCases := []struct {
		operation func(w *storage_domain.CircuitBreakerWrapper) error
		name      string
	}{
		{
			name: "Put is blocked",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				return w.Put(ctx, &storage_dto.PutParams{Reader: bytes.NewReader(nil)})
			},
		},
		{
			name: "Get is blocked",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				_, err := w.Get(ctx, storage_dto.GetParams{})
				return err
			},
		},
		{
			name: "Stat is blocked",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				_, err := w.Stat(ctx, storage_dto.GetParams{})
				return err
			},
		},
		{
			name: "Remove is blocked",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				return w.Remove(ctx, storage_dto.GetParams{})
			},
		},
		{
			name: "Copy is blocked",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				return w.Copy(ctx, "media", "a", "b")
			},
		},
		{
			name: "CopyToAnotherRepository is blocked",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				return w.CopyToAnotherRepository(ctx, "media", "a", "test", "b")
			},
		},
		{
			name: "GetHash is blocked",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				_, err := w.GetHash(ctx, storage_dto.GetParams{})
				return err
			},
		},
		{
			name: "PresignURL is blocked",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				_, err := w.PresignURL(ctx, storage_dto.PresignParams{})
				return err
			},
		},
		{
			name: "PutMany is blocked",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				_, err := w.PutMany(ctx, &storage_dto.PutManyParams{})
				return err
			},
		},
		{
			name: "RemoveMany is blocked",
			operation: func(w *storage_domain.CircuitBreakerWrapper) error {
				_, err := w.RemoveMany(ctx, storage_dto.RemoveManyParams{})
				return err
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.operation(wrapper)
			assert.Error(t, err)
			assert.ErrorIs(t, err, gobreaker.ErrOpenState, "Error should be ErrOpenState")
		})
	}

	assert.Equal(t, initialCallCount, mock.GetTotalCallCount(), "Underlying provider should not be called in Open state")
}

func TestCircuitBreakerWrapper_TransitionsToHalfOpenAndRecovers(t *testing.T) {
	wrapper, mock, config := setupWrapper(t)
	failErr := errors.New("provider failure")
	ctx := context.Background()
	putParams := &storage_dto.PutParams{Reader: bytes.NewReader(nil)}

	mock.SetError(failErr)
	for range 2 {
		_ = wrapper.Put(ctx, putParams)
	}
	require.Equal(t, gobreaker.StateOpen, wrapper.GetState())
	initialCallCount := mock.GetTotalCallCount()

	time.Sleep(config.Timeout + 10*time.Millisecond)

	mock.SetError(nil)

	err := wrapper.Put(ctx, putParams)
	require.NoError(t, err)

	assert.Equal(t, gobreaker.StateClosed, wrapper.GetState())
	assert.Equal(t, initialCallCount+1, mock.GetTotalCallCount(), "Mock should have been called once for recovery")
	assert.Equal(t, uint32(0), wrapper.GetCounts().ConsecutiveFailures, "Consecutive failures should be reset")

	err = wrapper.Put(ctx, putParams)
	require.NoError(t, err)
	assert.Equal(t, gobreaker.StateClosed, wrapper.GetState())
	assert.Equal(t, initialCallCount+2, mock.GetTotalCallCount())
}

func TestCircuitBreakerWrapper_TransitionsToHalfOpenAndFails(t *testing.T) {
	wrapper, mock, config := setupWrapper(t)
	failErr := errors.New("provider failure")
	ctx := context.Background()
	putParams := &storage_dto.PutParams{Reader: bytes.NewReader(nil)}

	mock.SetError(failErr)
	for range 2 {
		_ = wrapper.Put(ctx, putParams)
	}
	require.Equal(t, gobreaker.StateOpen, wrapper.GetState())
	initialCallCount := mock.GetTotalCallCount()

	time.Sleep(config.Timeout + 10*time.Millisecond)

	err := wrapper.Put(ctx, putParams)
	require.ErrorIs(t, err, failErr, "Should return the underlying provider's error")

	assert.Equal(t, gobreaker.StateOpen, wrapper.GetState())
	assert.Equal(t, initialCallCount+1, mock.GetTotalCallCount(), "Mock should have been called once for the failed recovery")

	err = wrapper.Put(ctx, putParams)
	require.ErrorIs(t, err, gobreaker.ErrOpenState)
	assert.Equal(t, initialCallCount+1, mock.GetTotalCallCount(), "Mock should NOT be called again")
}

func TestCircuitBreakerWrapper_NonConsecutiveFailuresDoNotTrip(t *testing.T) {
	wrapper, mock, _ := setupWrapper(t)
	failErr := errors.New("provider failure")
	ctx := context.Background()
	putParams := &storage_dto.PutParams{Reader: bytes.NewReader(nil)}

	mock.SetError(failErr)
	err := wrapper.Put(ctx, putParams)
	require.Error(t, err)
	assert.Equal(t, gobreaker.StateClosed, wrapper.GetState())
	assert.Equal(t, uint32(1), wrapper.GetCounts().ConsecutiveFailures)

	mock.SetError(nil)
	err = wrapper.Put(ctx, putParams)
	require.NoError(t, err)
	assert.Equal(t, gobreaker.StateClosed, wrapper.GetState())
	assert.Equal(t, uint32(0), wrapper.GetCounts().ConsecutiveFailures)

	mock.SetError(failErr)
	err = wrapper.Put(ctx, putParams)
	require.Error(t, err)
	assert.Equal(t, gobreaker.StateClosed, wrapper.GetState())
	assert.Equal(t, uint32(1), wrapper.GetCounts().ConsecutiveFailures)

	assert.Equal(t, 3, mock.GetTotalCallCount(), "Mock should have been called three times")
}

func TestCircuitBreakerWrapper_PassthroughMethods(t *testing.T) {
	wrapper, mock, _ := setupWrapper(t)
	ctx := context.Background()

	mock.SetError(errors.New("fail"))
	for range 2 {
		_ = wrapper.Put(ctx, &storage_dto.PutParams{Reader: bytes.NewReader(nil)})
	}
	require.Equal(t, gobreaker.StateOpen, wrapper.GetState())

	t.Run("Close is unaffected", func(t *testing.T) {
		err := wrapper.Close(ctx)
		assert.NoError(t, err)
	})

	t.Run("SupportsMultipart is unaffected", func(t *testing.T) {
		mock.SetError(nil)
		assert.True(t, wrapper.SupportsMultipart())
	})

	t.Run("SupportsBatchOperations is unaffected", func(t *testing.T) {
		mock.SetError(nil)
		assert.True(t, wrapper.SupportsBatchOperations())
	})
}

func TestCircuitBreaker_PresignDownloadURL(t *testing.T) {
	ctx := context.Background()

	t.Run("successful call through circuit breaker", func(t *testing.T) {
		wrapper, _, _ := setupWrapper(t)

		result, err := wrapper.PresignDownloadURL(ctx, storage_dto.PresignDownloadParams{
			Repository: "media",
			Key:        "file.pdf",
		})

		require.NoError(t, err)
		assert.NotEmpty(t, result)
	})

	t.Run("blocked when circuit is open", func(t *testing.T) {
		wrapper, mock, _ := setupWrapper(t)

		mock.SetError(errors.New("fail"))
		for range 2 {
			_ = wrapper.Put(ctx, &storage_dto.PutParams{Reader: bytes.NewReader(nil)})
		}
		require.Equal(t, gobreaker.StateOpen, wrapper.GetState())

		_, err := wrapper.PresignDownloadURL(ctx, storage_dto.PresignDownloadParams{
			Repository: "media",
			Key:        "file.pdf",
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, gobreaker.ErrOpenState)
	})
}

func TestCircuitBreaker_Rename(t *testing.T) {
	ctx := context.Background()

	t.Run("successful rename through circuit breaker", func(t *testing.T) {
		wrapper, mock, _ := setupWrapper(t)

		_ = mock.Put(ctx, &storage_dto.PutParams{
			Repository: "media",
			Key:        "old-name.txt",
			Reader:     bytes.NewReader([]byte("data")),
		})

		err := wrapper.Rename(ctx, "media", "old-name.txt", "new-name.txt")

		assert.NoError(t, err)
	})

	t.Run("blocked when circuit is open", func(t *testing.T) {
		wrapper, mock, _ := setupWrapper(t)

		mock.SetError(errors.New("fail"))
		for range 2 {
			_ = wrapper.Put(ctx, &storage_dto.PutParams{Reader: bytes.NewReader(nil)})
		}
		require.Equal(t, gobreaker.StateOpen, wrapper.GetState())

		err := wrapper.Rename(ctx, "media", "old.txt", "new.txt")

		require.Error(t, err)
		assert.ErrorIs(t, err, gobreaker.ErrOpenState)
	})
}

func TestCircuitBreaker_Exists(t *testing.T) {
	ctx := context.Background()

	t.Run("successful exists check through circuit breaker", func(t *testing.T) {
		wrapper, mock, _ := setupWrapper(t)

		_ = mock.Put(ctx, &storage_dto.PutParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Key:        "existing.txt",
			Reader:     bytes.NewReader([]byte("data")),
		})

		exists, err := wrapper.Exists(ctx, storage_dto.GetParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Key:        "existing.txt",
		})

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("non-existent object returns false", func(t *testing.T) {
		wrapper, _, _ := setupWrapper(t)

		exists, err := wrapper.Exists(ctx, storage_dto.GetParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Key:        "nonexistent.txt",
		})

		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("blocked when circuit is open", func(t *testing.T) {
		wrapper, mock, _ := setupWrapper(t)

		mock.SetError(errors.New("fail"))
		for range 2 {
			_ = wrapper.Put(ctx, &storage_dto.PutParams{Reader: bytes.NewReader(nil)})
		}
		require.Equal(t, gobreaker.StateOpen, wrapper.GetState())

		_, err := wrapper.Exists(ctx, storage_dto.GetParams{
			Repository: storage_dto.StorageRepositoryDefault,
			Key:        "test.txt",
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, gobreaker.ErrOpenState)
	})
}

func TestCircuitBreaker_SupportsRetry(t *testing.T) {
	wrapper, _, _ := setupWrapper(t)

	result := wrapper.SupportsRetry()

	assert.False(t, result, "should delegate to the underlying provider's SupportsRetry")
}

func TestCircuitBreaker_SupportsCircuitBreaking(t *testing.T) {
	wrapper, _, _ := setupWrapper(t)

	result := wrapper.SupportsCircuitBreaking()

	assert.False(t, result, "should delegate to the underlying provider's SupportsCircuitBreaking")
}

func TestCircuitBreaker_SupportsRateLimiting(t *testing.T) {
	wrapper, _, _ := setupWrapper(t)

	result := wrapper.SupportsRateLimiting()

	assert.False(t, result, "should delegate to the underlying provider's SupportsRateLimiting")
}

func TestCircuitBreaker_SupportsPresignedURLs(t *testing.T) {
	wrapper, _, _ := setupWrapper(t)

	result := wrapper.SupportsPresignedURLs()

	assert.True(t, result, "should delegate to the underlying provider's SupportsPresignedURLs")
}
