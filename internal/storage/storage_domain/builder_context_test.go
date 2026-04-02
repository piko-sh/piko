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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_dto"
)

func TestUploadBuilder_Do_CancelledContext(t *testing.T) {
	t.Parallel()

	service, mock := setupTestService(t)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	content := []byte("test data")
	err := service.NewUpload(bytes.NewReader(content)).
		Key("test.txt").
		ContentType("text/plain").
		Size(int64(len(content))).
		Do(ctx)

	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, mock.GetPutCalls(), "provider should not be called")
}

func TestUploadBuilder_Do_ExpiredContext(t *testing.T) {
	t.Parallel()

	service, mock := setupTestService(t)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 0, fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	content := []byte("test data")
	err := service.NewUpload(bytes.NewReader(content)).
		Key("test.txt").
		ContentType("text/plain").
		Size(int64(len(content))).
		Do(ctx)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Empty(t, mock.GetPutCalls(), "provider should not be called")
}

func TestRequestBuilder_Get_CancelledContext(t *testing.T) {
	t.Parallel()

	service, _ := setupTestService(t)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	_, err := service.NewRequest(storage_dto.StorageRepositoryDefault, "test.txt").Get(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestRequestBuilder_Get_ExpiredContext(t *testing.T) {
	t.Parallel()

	service, _ := setupTestService(t)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 0, fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	_, err := service.NewRequest(storage_dto.StorageRepositoryDefault, "test.txt").Get(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestRequestBuilder_Stat_CancelledContext(t *testing.T) {
	t.Parallel()

	service, _ := setupTestService(t)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	_, err := service.NewRequest(storage_dto.StorageRepositoryDefault, "test.txt").Stat(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestRequestBuilder_Stat_ExpiredContext(t *testing.T) {
	t.Parallel()

	service, _ := setupTestService(t)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 0, fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	_, err := service.NewRequest(storage_dto.StorageRepositoryDefault, "test.txt").Stat(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestRequestBuilder_Remove_CancelledContext(t *testing.T) {
	t.Parallel()

	service, _ := setupTestService(t)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	err := service.NewRequest(storage_dto.StorageRepositoryDefault, "test.txt").Remove(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestRequestBuilder_Remove_ExpiredContext(t *testing.T) {
	t.Parallel()

	service, _ := setupTestService(t)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 0, fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	err := service.NewRequest(storage_dto.StorageRepositoryDefault, "test.txt").Remove(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestRequestBuilder_Hash_CancelledContext(t *testing.T) {
	t.Parallel()

	service, _ := setupTestService(t)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	_, err := service.NewRequest(storage_dto.StorageRepositoryDefault, "test.txt").Hash(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestRequestBuilder_Hash_ExpiredContext(t *testing.T) {
	t.Parallel()

	service, _ := setupTestService(t)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 0, fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	_, err := service.NewRequest(storage_dto.StorageRepositoryDefault, "test.txt").Hash(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
