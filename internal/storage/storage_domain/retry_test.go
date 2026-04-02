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
	"io"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_adapters/provider_mock"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

func retryTestConfig() storage_domain.RetryConfig {
	return storage_domain.RetryConfig{
		MaxRetries:    3,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      50 * time.Millisecond,
		BackoffFactor: 2.0,
	}
}

func setupRetryTestService(t *testing.T) (storage_domain.Service, *provider_mock.MockStorageProvider) {
	t.Helper()

	mock := provider_mock.NewMockStorageProvider()

	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(true),
		storage_domain.WithRetryConfig(retryTestConfig()),
		storage_domain.WithCircuitBreakerEnabled(false),
		storage_domain.WithSingleflightEnabled(false),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	err := service.RegisterProvider(context.Background(), "default", mock)
	require.NoError(t, err)

	return service, mock
}

func seedMockObject(ctx context.Context, t *testing.T, mock *provider_mock.MockStorageProvider, repo, key, content string) {
	t.Helper()

	params := &storage_dto.PutParams{
		Repository:  repo,
		Key:         key,
		Reader:      bytes.NewReader([]byte(content)),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	}
	err := mock.Put(ctx, params)
	require.NoError(t, err)
}

func TestStorageErrorClassifier_PermanentErrors(t *testing.T) {
	testCases := []struct {
		err  error
		name string
	}{
		{name: "os.ErrNotExist", err: os.ErrNotExist},
		{name: "os.ErrPermission", err: os.ErrPermission},
		{name: "io.EOF", err: io.EOF},
		{name: "io.ErrUnexpectedEOF", err: io.ErrUnexpectedEOF},
		{name: "context.Canceled", err: context.Canceled},
		{name: "context.DeadlineExceeded", err: context.DeadlineExceeded},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.False(t, storage_domain.IsRetryableError(tc.err),
				"error %q should be permanent (not retryable)", tc.err)
		})
	}
}

func TestDefaultRetryConfig_Values(t *testing.T) {
	config := storage_domain.DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxRetries, "default max retries")
	assert.Equal(t, 1*time.Second, config.InitialDelay, "default initial delay")
	assert.Equal(t, 30*time.Second, config.MaxDelay, "default max delay")
	assert.Equal(t, 2.0, config.BackoffFactor, "default backoff factor")
}

func TestRetryWrapper_Put_Success(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	params := &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "test-file.txt",
		Reader:      bytes.NewReader([]byte("hello world")),
		Size:        11,
		ContentType: "text/plain",
	}

	err := service.PutObject(ctx, "default", params)
	require.NoError(t, err)

	calls := mock.GetPutCalls()
	assert.Len(t, calls, 1, "successful put should result in a single provider call")
	assert.Equal(t, "test-file.txt", calls[0].Key)
}

func TestRetryWrapper_Put_PermanentError_NoRetry(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	mock.SetPutError(context.Canceled)

	params := &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "perm-fail.txt",
		Reader:      bytes.NewReader([]byte("data")),
		Size:        4,
		ContentType: "text/plain",
	}

	err := service.PutObject(ctx, "default", params)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled,
		"permanent errors should be returned directly without wrapping")

	calls := mock.GetPutCalls()
	assert.Len(t, calls, 1, "permanent error should result in exactly 1 call (no retries)")
}

func TestRetryWrapper_Put_RetryableError_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	mock.SetPutError(errors.New("connection refused"))

	params := &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "retry-fail.txt",
		Reader:      bytes.NewReader([]byte("data")),
		Size:        4,
		ContentType: "text/plain",
	}

	err := service.PutObject(ctx, "default", params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed after",
		"error should indicate retry exhaustion")
	assert.Contains(t, err.Error(), "connection refused",
		"error should contain the original error message")

	calls := mock.GetPutCalls()
	assert.Len(t, calls, 4, "retryable error should result in 1 initial + 3 retries = 4 total calls")
}

func TestRetryWrapper_Put_ContextCancellation(t *testing.T) {
	service, mock := setupRetryTestService(t)

	mock.SetPutError(errors.New("connection refused"))

	ctx, cancel := context.WithCancelCause(context.Background())

	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel(fmt.Errorf("test: simulating cancelled context"))
	}()

	params := &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "cancel-test.txt",
		Reader:      bytes.NewReader([]byte("data")),
		Size:        4,
		ContentType: "text/plain",
	}

	err := service.PutObject(ctx, "default", params)
	require.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "cancelled") || errors.Is(err, context.Canceled),
		"error should indicate cancellation, got: %v", err)
}

func TestRetryWrapper_Get_Success(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	seedMockObject(ctx, t, mock, storage_dto.StorageRepositoryDefault, "get-test.txt", "hello world")

	mock.Reset()
	seedMockObject(ctx, t, mock, storage_dto.StorageRepositoryDefault, "get-test.txt", "hello world")

	reader, err := service.GetObject(ctx, "default", storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "get-test.txt",
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
}

func TestRetryWrapper_Get_PermanentError_NoRetry(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	seedMockObject(ctx, t, mock, storage_dto.StorageRepositoryDefault, "get-perm.txt", "data")
	mock.SetGetError(context.Canceled)

	_, err := service.GetObject(ctx, "default", storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "get-perm.txt",
	})
	require.Error(t, err)

	getCalls := mock.GetGetCalls()
	assert.Len(t, getCalls, 1, "permanent Get error should result in exactly 1 Get call")
}

func TestRetryWrapper_Get_RetryableError_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	seedMockObject(ctx, t, mock, storage_dto.StorageRepositoryDefault, "get-retry.txt", "data")
	mock.SetGetError(errors.New("connection refused"))

	_, err := service.GetObject(ctx, "default", storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "get-retry.txt",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed after",
		"error should indicate retry exhaustion")

	getCalls := mock.GetGetCalls()
	assert.Len(t, getCalls, 4, "retryable Get error should produce 4 total Get calls")
}

func TestRetryWrapper_Remove_Success(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	seedMockObject(ctx, t, mock, storage_dto.StorageRepositoryDefault, "remove-test.txt", "data")

	err := service.RemoveObject(ctx, "default", storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "remove-test.txt",
	})
	require.NoError(t, err)

	removeCalls := mock.GetRemoveCalls()
	assert.Len(t, removeCalls, 1, "successful remove should be a single call")
	assert.Equal(t, "remove-test.txt", removeCalls[0].Key)
}

func TestRetryWrapper_Remove_PermanentError_NoRetry(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	mock.SetRemoveError(context.DeadlineExceeded)

	err := service.RemoveObject(ctx, "default", storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "remove-perm.txt",
	})
	require.Error(t, err)

	removeCalls := mock.GetRemoveCalls()
	assert.Len(t, removeCalls, 1, "permanent Remove error should result in exactly 1 call")
}

func TestRetryWrapper_Remove_RetryableError_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	mock.SetRemoveError(errors.New("connection reset"))

	err := service.RemoveObject(ctx, "default", storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "remove-retry.txt",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed after")

	removeCalls := mock.GetRemoveCalls()
	assert.Len(t, removeCalls, 4, "retryable Remove error should produce 4 total calls")
}

func TestRetryWrapper_Passthrough_Stat(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	seedMockObject(ctx, t, mock, storage_dto.StorageRepositoryDefault, "stat-test.txt", "hello")

	info, err := service.StatObject(ctx, "default", storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "stat-test.txt",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(5), info.Size)
	assert.Equal(t, "text/plain", info.ContentType)
}

func TestRetryWrapper_Passthrough_Stat_Error_NoRetry(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	mock.SetStatError(errors.New("connection refused"))

	_, err := service.StatObject(ctx, "default", storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "stat-fail.txt",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")

	statCalls := mock.GetStatCalls()
	assert.Len(t, statCalls, 1, "Stat passthrough should not retry on error")
}

func TestRetryWrapper_Passthrough_Copy(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	seedMockObject(ctx, t, mock, "src-repo", "src-key.txt", "data to copy")

	err := service.CopyObject(ctx, "default", storage_dto.CopyParams{
		SourceRepository:      "src-repo",
		SourceKey:             "src-key.txt",
		DestinationRepository: "src-repo",
		DestinationKey:        "dst-key.txt",
	})
	require.NoError(t, err)

	copyCalls := mock.GetCopyCalls()
	assert.Len(t, copyCalls, 1, "Copy should delegate directly")
	assert.Equal(t, "src-key.txt", copyCalls[0].SourceKey)
	assert.Equal(t, "dst-key.txt", copyCalls[0].DestinationKey)
}

func TestRetryWrapper_Passthrough_Copy_Error_NoRetry(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	mock.SetCopyError(errors.New("connection refused"))

	err := service.CopyObject(ctx, "default", storage_dto.CopyParams{
		SourceRepository:      "src-repo",
		SourceKey:             "src-key.txt",
		DestinationRepository: "src-repo",
		DestinationKey:        "dst-key.txt",
	})
	require.Error(t, err)

	copyCalls := mock.GetCopyCalls()
	assert.Len(t, copyCalls, 1, "Copy passthrough should not retry on error")
}

func TestRetryWrapper_Passthrough_Close(t *testing.T) {
	ctx := context.Background()
	service, _ := setupRetryTestService(t)

	err := service.Close(ctx)
	require.NoError(t, err)
}

func TestRetryWrapper_Passthrough_CapabilityChecks(t *testing.T) {
	service, _ := setupRetryTestService(t)

	assert.True(t, service.HasProvider("default"),
		"registered provider should be discoverable")
	providers := service.GetProviders(context.Background())
	assert.Contains(t, providers, "default",
		"providers list should include the default provider")
}

func TestRetryWrapper_Passthrough_GetHash_Success(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	seedMockObject(ctx, t, mock, storage_dto.StorageRepositoryDefault, "hash-test.txt", "data")

	hash, err := service.GetObjectHash(ctx, "default", storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "hash-test.txt",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, hash, "GetHash should return a non-empty hash")

	hashCalls := mock.GetGetHashCalls()
	assert.Len(t, hashCalls, 1, "GetHash should be a single passthrough call")
}

func TestRetryWrapper_Passthrough_GetHash_Error_NoRetry(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	mock.SetGetHashError(errors.New("connection refused"))

	_, err := service.GetObjectHash(ctx, "default", storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "hash-fail.txt",
	})
	require.Error(t, err)

	hashCalls := mock.GetGetHashCalls()
	assert.Len(t, hashCalls, 1, "GetHash passthrough should not retry")
}

func TestRetryWrapper_Passthrough_PresignURL_Success(t *testing.T) {
	ctx := context.Background()
	service, _ := setupRetryTestService(t)

	url, err := service.GeneratePresignedUploadURL(ctx, "default", storage_dto.PresignParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "presign-test.txt",
		ContentType: "text/plain",
		ExpiresIn:   15 * time.Minute,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, url, "PresignURL should return a non-empty URL")
	assert.Contains(t, url, "presign", "URL should contain presign indicator")
}

func TestRetryWrapper_Passthrough_PresignURL_Error_NoRetry(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	mock.SetPresignError(errors.New("connection refused"))

	_, err := service.GeneratePresignedUploadURL(ctx, "default", storage_dto.PresignParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "presign-fail.txt",
		ContentType: "text/plain",
		ExpiresIn:   15 * time.Minute,
	})
	require.Error(t, err)

	presignCalls := mock.GetPresignURLCalls()
	assert.Len(t, presignCalls, 1, "PresignURL passthrough should not retry")
}

func TestRetryWrapper_SyscallError_IsRetried(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	mock.SetRemoveError(syscall.ECONNRESET)

	err := service.RemoveObject(ctx, "default", storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "syscall-test.txt",
	})
	require.Error(t, err)

	removeCalls := mock.GetRemoveCalls()
	assert.Len(t, removeCalls, 4,
		"syscall ECONNRESET should be retried: 1 initial + 3 retries = 4 total")
}

func TestRetryWrapper_WrappedRetryableError_IsRetried(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	wrappedErr := fmt.Errorf("storage operation failed: %w", errors.New("connection refused"))
	mock.SetRemoveError(wrappedErr)

	err := service.RemoveObject(ctx, "default", storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "wrapped-test.txt",
	})
	require.Error(t, err)

	removeCalls := mock.GetRemoveCalls()
	assert.Len(t, removeCalls, 4,
		"wrapped retryable error should be retried")
}

func TestRetryDisabled_NoRetryOnRetryableError(t *testing.T) {
	ctx := context.Background()

	mock := provider_mock.NewMockStorageProvider()
	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })
	err := service.RegisterProvider(context.Background(), "default", mock)
	require.NoError(t, err)

	mock.SetPutError(errors.New("connection refused"))

	params := &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "no-retry.txt",
		Reader:      bytes.NewReader([]byte("data")),
		Size:        4,
		ContentType: "text/plain",
	}

	err = service.PutObject(ctx, "default", params)
	require.Error(t, err)

	calls := mock.GetPutCalls()
	assert.Len(t, calls, 1,
		"with retry disabled, retryable errors should produce exactly 1 call")
}

func TestRetryWrapper_SequentialOperations_IndependentRetries(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	seedMockObject(ctx, t, mock, storage_dto.StorageRepositoryDefault, "seq-test.txt", "hello")

	putParams := &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "seq-put.txt",
		Reader:      bytes.NewReader([]byte("data")),
		Size:        4,
		ContentType: "text/plain",
	}
	err := service.PutObject(ctx, "default", putParams)
	require.NoError(t, err)

	reader, err := service.GetObject(ctx, "default", storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "seq-test.txt",
	})
	require.NoError(t, err)
	_ = reader.Close()

	err = service.RemoveObject(ctx, "default", storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "seq-test.txt",
	})
	require.NoError(t, err)

	putCalls := mock.GetPutCalls()
	assert.GreaterOrEqual(t, len(putCalls), 2, "should have seed Put + test Put")

	getCalls := mock.GetGetCalls()
	assert.Len(t, getCalls, 1, "Get should have 1 call")

	removeCalls := mock.GetRemoveCalls()
	assert.Len(t, removeCalls, 1, "Remove should have 1 call")
}

func TestRetryWrapper_PutObjects_Success(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	params := &storage_dto.PutManyParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Objects: []storage_dto.PutObjectSpec{
			{
				Key:         "batch-1.txt",
				Reader:      bytes.NewReader([]byte("data1")),
				Size:        5,
				ContentType: "text/plain",
			},
			{
				Key:         "batch-2.txt",
				Reader:      bytes.NewReader([]byte("data2")),
				Size:        5,
				ContentType: "text/plain",
			},
		},
		Concurrency:     1,
		ContinueOnError: true,
	}

	err := service.PutObjects(ctx, "default", params)
	require.NoError(t, err)

	putManyCalls := mock.GetPutManyCalls()
	assert.Len(t, putManyCalls, 1, "batch put should result in a single PutMany call")
}

func TestRetryWrapper_RemoveObjects_Success(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	seedMockObject(ctx, t, mock, storage_dto.StorageRepositoryDefault, "rm-1.txt", "data1")
	seedMockObject(ctx, t, mock, storage_dto.StorageRepositoryDefault, "rm-2.txt", "data2")

	params := storage_dto.RemoveManyParams{
		Repository:      storage_dto.StorageRepositoryDefault,
		Keys:            []string{"rm-1.txt", "rm-2.txt"},
		Concurrency:     1,
		ContinueOnError: true,
	}

	err := service.RemoveObjects(ctx, "default", params)
	require.NoError(t, err)

	removeManyCalls := mock.GetRemoveManyCalls()
	assert.Len(t, removeManyCalls, 1, "batch remove should result in a single RemoveMany call")
}

func TestRetryWrapper_ErrorMessageFormat(t *testing.T) {
	ctx := context.Background()
	service, mock := setupRetryTestService(t)

	mock.SetPutError(errors.New("temporary failure"))

	params := &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "err-fmt.txt",
		Reader:      bytes.NewReader([]byte("data")),
		Size:        4,
		ContentType: "text/plain",
	}

	err := service.PutObject(ctx, "default", params)
	require.Error(t, err)

	errMessage := err.Error()
	assert.Contains(t, errMessage, "Put",
		"error should contain the operation name")
	assert.Contains(t, errMessage, "err-fmt.txt",
		"error should contain the key")
	assert.Contains(t, errMessage, "4 attempts",
		"error should contain the total attempt count")
	assert.Contains(t, errMessage, "temporary failure",
		"error should contain the underlying error")
}

func TestRetryWrapper_PermanentErrors_AllSingleCall(t *testing.T) {

	testCases := []struct {
		err  error
		name string
	}{
		{name: "context cancelled", err: context.Canceled},
		{name: "deadline exceeded", err: context.DeadlineExceeded},
		{name: "file not found", err: os.ErrNotExist},
		{name: "permission denied", err: os.ErrPermission},
		{name: "io EOF", err: io.EOF},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			service, mock := setupRetryTestService(t)
			mock.SetRemoveError(tc.err)

			err := service.RemoveObject(ctx, "default", storage_dto.GetParams{
				Repository: storage_dto.StorageRepositoryDefault,
				Key:        "perm-test.txt",
			})
			require.Error(t, err)

			removeCalls := mock.GetRemoveCalls()
			assert.Len(t, removeCalls, 1,
				"permanent error %q should result in exactly 1 call", tc.name)
		})
	}
}

func TestRetryWrapper_RetryableErrors_AllRetried(t *testing.T) {

	testCases := []struct {
		err  error
		name string
	}{
		{name: "connection refused", err: errors.New("connection refused")},
		{name: "connection reset", err: errors.New("connection reset")},
		{name: "timeout", err: errors.New("timeout")},
		{name: "temporary failure", err: errors.New("temporary failure")},
		{name: "rate limit", err: errors.New("rate limit exceeded")},
		{name: "HTTP 503", err: errors.New("HTTP 503")},
		{name: "syscall ECONNRESET", err: syscall.ECONNRESET},
		{name: "syscall ETIMEDOUT", err: syscall.ETIMEDOUT},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			service, mock := setupRetryTestService(t)
			mock.SetRemoveError(tc.err)

			err := service.RemoveObject(ctx, "default", storage_dto.GetParams{
				Repository: storage_dto.StorageRepositoryDefault,
				Key:        "retry-test.txt",
			})
			require.Error(t, err)

			removeCalls := mock.GetRemoveCalls()
			assert.Len(t, removeCalls, 4,
				"retryable error %q should produce 4 calls (1 initial + 3 retries)", tc.name)
		})
	}
}

func TestRetryWrapper_Passthrough_PresignDownloadURL(t *testing.T) {
	ctx := context.Background()
	service, _ := setupRetryTestService(t)

	url, err := service.GeneratePresignedDownloadURL(ctx, "default", storage_dto.PresignDownloadParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "download-test.txt",
		ExpiresIn:  15 * time.Minute,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, url, "PresignDownloadURL should return a non-empty URL")
}
